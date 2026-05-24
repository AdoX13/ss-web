package routes

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.mongodb.org/mongo-driver/mongo"

	"mqtt-streaming-server/audit"
	"mqtt-streaming-server/auth"
	"mqtt-streaming-server/domain"
	"mqtt-streaming-server/evidence"
	"mqtt-streaming-server/reports"
)

// Config bundles every dependency that route handlers need. Passed to
// InitRoutes so the HTTP layer stays decoupled from the wiring in main.go.
type Config struct {
	DB               *mongo.Database
	MQTTClient       mqtt.Client
	JWTSecret        string
	UserRepo         domain.UserRepository
	RefreshTokenRepo domain.RefreshTokenRepository
	ReviewItemRepo   domain.ReviewItemRepository
	AuditWriter      audit.Writer
	EvidenceChain    evidence.Chain
	ReportRegistry   *reports.Registry
	ReviewHub        *ReviewHub
	AuthRateLimiter  *auth.RateLimiter
	"strconv"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.mongodb.org/mongo-driver/mongo"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"mqtt-streaming-server/utils"
)

func InitRoutes(db *mongo.Database, mqttClient mqtt.Client) http.Handler {
	mux := http.NewServeMux()
	InitUserRoutes(db, mux)
	InitPhotoRoutes(db, mux)
	InitDeviceRoutes(db, mqttClient, mux)

	// Serve static files from ./uploads
	fs := http.FileServer(http.Dir("uploads"))
	mux.Handle("/uploads/", http.StripPrefix("/uploads/", fs))

	// Broker info endpoint
	mux.HandleFunc("/broker-info", handleBrokerInfo)

	// Prometheus metrics endpoint
	mux.Handle("/metrics", promhttp.Handler())

	corsHandler := withCORS(mux)
	metricsHandler := withMetrics(corsHandler)

	return metricsHandler
}

// customResponseWriter captures the HTTP status code
type customResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *customResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func withMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := &customResponseWriter{w, http.StatusOK}
		next.ServeHTTP(rw, r)
		utils.HttpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, strconv.Itoa(rw.statusCode)).Inc()
	})
}

func InitRoutes(cfg *Config) http.Handler {
	mux := http.NewServeMux()

	// ── Legacy skeleton routes (kept for backward compatibility) ─────────────
	InitUserRoutes(cfg.DB, mux)
	InitPhotoRoutes(cfg.DB, mux, cfg.JWTSecret, cfg.AuditWriter, cfg.EvidenceChain)
	InitDeviceRoutes(cfg.DB, cfg.MQTTClient, mux, cfg.JWTSecret)

	fs := http.FileServer(http.Dir("uploads"))
	uploads := http.StripPrefix("/uploads/", fs)
	mux.Handle("/uploads/", auth.WithAuth(cfg.JWTSecret)(
		auth.RequireRole(domain.RoleAdmin, domain.RoleDoctor)(uploads)))
	mux.HandleFunc("/broker-info", handleBrokerInfo)

	// ── v1 API ────────────────────────────────────────────────────────────────
	initAuthRoutes(cfg, mux)
	initReviewRoutes(cfg, mux)
	initReportRoutes(cfg, mux)
	initUsersRoutes(cfg, mux)
	initAuditRoutes(cfg, mux)
	initEvidenceRoutes(cfg, mux)

	// ── WebSocket ─────────────────────────────────────────────────────────────
	if cfg.ReviewHub != nil {
		withAuth := auth.WithAuth(cfg.JWTSecret)
		mux.Handle("/ws/review", withAuth(http.HandlerFunc(cfg.ReviewHub.HandleReviewWS)))
	}

	// ── Health & metrics ──────────────────────────────────────────────────────
	mux.HandleFunc("/health", handleHealth(cfg))
	mux.HandleFunc("/metrics", handleMetrics)
	// Get the server's local IP address
	ip := getOutboundIP()
	port := "8883" // mTLS MQTT port

	return auth.SecureHeaders(withCORS(mux))
}

// handleHealth returns JSON describing the service's readiness. Returns 200
// if everything is up, 503 if the database ping fails.
func handleHealth(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		dbStatus := "ok"
		if err := cfg.DB.Client().Ping(ctx, nil); err != nil {
			dbStatus = "error: " + err.Error()
		}

		brokerStatus := "ok"
		if cfg.MQTTClient != nil && !cfg.MQTTClient.IsConnected() {
			brokerStatus = "disconnected"
		}

		status := http.StatusOK
		if dbStatus != "ok" {
			status = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(map[string]string{
			"status": func() string {
				if status == http.StatusOK {
					return "healthy"
				}
				return "unhealthy"
			}(),
			"database": dbStatus,
			"broker":   brokerStatus,
		})
	}
}

// handleBrokerInfo returns the MQTT broker IP and port for client connections.
func handleBrokerInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"ip":   getOutboundIP(),
		"port": "8883",
	})
}

func getOutboundIP() string {
	if hostIP := os.Getenv("MQTT_HOST_IP"); hostIP != "" {
		if addrs, err := net.LookupHost(hostIP); err == nil && len(addrs) > 0 {
			return addrs[0]
		}
		if net.ParseIP(hostIP) != nil {
			return hostIP
		}
	}
	if addrs, err := net.LookupHost("host.docker.internal"); err == nil && len(addrs) > 0 {
		return addrs[0]
	}
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "localhost"
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).IP.String()
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// noAuth is kept for the legacy skeleton routes only.
func noAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), auth.ContextEmail, "guest@example.com")
		ctx = context.WithValue(ctx, auth.ContextRole, "guest")
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
