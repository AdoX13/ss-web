package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"mqtt-streaming-server/audit"
	"mqtt-streaming-server/auth"
	"mqtt-streaming-server/broker"
	medcrypto "mqtt-streaming-server/crypto"
	"mqtt-streaming-server/domain"
	"mqtt-streaming-server/evidence"
	"mqtt-streaming-server/ocr"
	"mqtt-streaming-server/reports"
	"mqtt-streaming-server/repository"
	"mqtt-streaming-server/routes"
)

func NewTLSConfig() *tls.Config {
	certpool := x509.NewCertPool()
	pemCerts, err := os.ReadFile("/run/secrets/ca.crt")
	if err != nil {
		panic(err)
	}
	certpool.AppendCertsFromPEM(pemCerts)

	cert, err := tls.LoadX509KeyPair("/run/secrets/web.crt", "/run/secrets/web.key")
	if err != nil {
		panic(err)
	}

	return &tls.Config{
		RootCAs:            certpool,
		ClientCAs:          certpool,
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: false,
	}
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	uri := fmt.Sprintf("mongodb://%s:%s@mongo-db:27017/?authSource=admin",
		os.Getenv("MONGO_INITDB_ROOT_USERNAME"),
		os.Getenv("MONGO_INITDB_ROOT_PASSWORD"),
	)
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		slog.Error("failed to connect to MongoDB", "err", err)
		panic(err)
	}
	defer func() {
		if err := mongoClient.Disconnect(context.Background()); err != nil {
			panic(err)
		}
	}()
	db := mongoClient.Database("mqtt-streaming-server")
	slog.Info("connected to MongoDB")
	if err := repository.EnsureSchema(context.Background(), db); err != nil {
		slog.Warn("MongoDB schema/index bootstrap failed", "err", err)
	}

	jwtSecret := jwtSecretFromEnv()

	userRepo := repository.NewUserRepository(db)
	refreshTokenRepo := repository.NewRefreshTokenRepository(db)
	reviewItemRepo := repository.NewReviewItemRepository(db)

	auditWriter := audit.NewMongoWriter(db)
	evidenceChain := newEvidenceChain(db)
	reportRegistry := reports.DefaultRegistry()

	rateLimiter := auth.NewRateLimiter(5, 10)

	reviewNotifyCh := make(chan *domain.ReviewItem, 64)
	hub := routes.NewReviewHub(reviewNotifyCh)
	go hub.Run()

	socketPath := os.Getenv("OCR_SOCKET_PATH")
	if socketPath == "" {
		socketPath = "/run/ocr/ocr.sock"
	}
	ocrClient := ocr.NewUnixSocketClient(socketPath)
	defer ocrClient.Close()

	tlsconfig := NewTLSConfig()
	opts := mqtt.NewClientOptions()
	opts.AddBroker("ssl://broker:8883")
	opts.SetClientID("web").SetTLSConfig(tlsconfig)

	mqttClient := mqtt.NewClient(opts)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	brokerHandler := broker.NewBrokerHandler(db, ocrClient, reviewNotifyCh)

	if token := mqttClient.Subscribe("ssproject/images/#", 0, brokerHandler.HandlePhoto); token.Wait() && token.Error() != nil {
		slog.Error("subscribe failed", "topic", "ssproject/images/#", "err", token.Error())
		os.Exit(1)
	}
	if token := mqttClient.Subscribe("register/#", 0, brokerHandler.RegisterDevice); token.Wait() && token.Error() != nil {
		slog.Error("subscribe failed", "topic", "register/#", "err", token.Error())
		os.Exit(1)
	}
	if token := mqttClient.Subscribe("device/id/#", 0, brokerHandler.DisconnectDevice); token.Wait() && token.Error() != nil {
		slog.Error("subscribe failed", "topic", "device/id/#", "err", token.Error())
		os.Exit(1)
	}

	handler := routes.InitRoutes(&routes.Config{
		DB:               db,
		MQTTClient:       mqttClient,
		JWTSecret:        jwtSecret,
		UserRepo:         userRepo,
		RefreshTokenRepo: refreshTokenRepo,
		ReviewItemRepo:   reviewItemRepo,
		AuditWriter:      auditWriter,
		EvidenceChain:    evidenceChain,
		ReportRegistry:   reportRegistry,
		ReviewHub:        hub,
		AuthRateLimiter:  rateLimiter,
	})

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	go func() {
		slog.Info("starting HTTP server", "port", 8080)
		if err := http.ListenAndServe(":8080", handler); err != nil {
			panic(err)
		}
	}()

	<-sig
	slog.Info("shutting down")
}

func jwtSecretFromEnv() string {
	if s := os.Getenv("JWT_SECRET"); s != "" {
		return s
	}
	return "dev-secret-change-in-production"
}

func newEvidenceChain(db *mongo.Database) evidence.Chain {
	privateKey, err := medcrypto.LoadEd25519PrivateKeyFromEnv()
	if err != nil {
		slog.Warn("using ephemeral evidence signing key; set EVIDENCE_ED25519_PRIVATE_KEY for stable verification", "err", err)
		_, generatedPrivate, genErr := medcrypto.GenerateEd25519Key()
		if genErr != nil {
			slog.Error("failed to generate evidence signing key", "err", genErr)
			return &evidence.Noop{}
		}
		privateKey = generatedPrivate
	}
	chain, err := evidence.NewMongoChain(db, privateKey)
	if err != nil {
		slog.Error("failed to initialize evidence chain", "err", err)
		return &evidence.Noop{}
	}
	return chain
}
