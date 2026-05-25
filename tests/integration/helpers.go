package integration

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// SetupTestEnv spins up MongoDB and Mosquitto via testcontainers
func SetupTestEnv(ctx context.Context) (testcontainers.Container, testcontainers.Container, error) {
	// Setup MongoDB
	reqMongo := testcontainers.ContainerRequest{
		Image:        "mongo:latest",
		ExposedPorts: []string{"27017/tcp"},
		Env: map[string]string{
			"MONGO_INITDB_ROOT_USERNAME": "admin",
			"MONGO_INITDB_ROOT_PASSWORD": "supersecret",
		},
		Cmd:        []string{"--auth"},
		WaitingFor: wait.ForLog("Waiting for connections"),
	}

	mongoC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: reqMongo,
		Started:          true,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to start mongo: %w", err)
	}

	// Setup Mosquitto (mounts local conf and secrets)
	cwd, _ := os.Getwd()
	projectRoot := filepath.Dir(filepath.Dir(cwd))

	reqMosq := testcontainers.ContainerRequest{
		Image:        "eclipse-mosquitto:latest",
		ExposedPorts: []string{"8883/tcp"},
		Files: []testcontainers.ContainerFile{
			{
				HostFilePath:      filepath.Join(projectRoot, "broker", "mosquitto.conf"),
				ContainerFilePath: "/mosquitto/config/mosquitto.conf",
				FileMode:          0444,
			},
			{
				HostFilePath:      filepath.Join(projectRoot, "secrets", "ca.crt"),
				ContainerFilePath: "/run/secrets/ca.crt",
				FileMode:          0444,
			},
			{
				HostFilePath:      filepath.Join(projectRoot, "secrets", "server.crt"),
				ContainerFilePath: "/run/secrets/server.crt",
				FileMode:          0444,
			},
			{
				HostFilePath:      filepath.Join(projectRoot, "secrets", "server.key"),
				ContainerFilePath: "/run/secrets/server.key",
				FileMode:          0400,
			},
		},
		WaitingFor: wait.ForListeningPort("8883/tcp").WithStartupTimeout(30 * time.Second),
	}

	mosqC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: reqMosq,
		Started:          true,
	})
	if err != nil {
		return mongoC, nil, fmt.Errorf("failed to start mosquitto: %w", err)
	}

	return mongoC, mosqC, nil
}

// GetTLSConfig creates a TLS configuration using the test certificates
func GetTLSConfig() (*tls.Config, error) {
	cwd, _ := os.Getwd()
	secretsDir := filepath.Join(filepath.Dir(filepath.Dir(cwd)), "secrets")

	certpool := x509.NewCertPool()
	ca, err := os.ReadFile(filepath.Join(secretsDir, "ca.crt"))
	if err != nil {
		return nil, err
	}
	certpool.AppendCertsFromPEM(ca)

	cert, err := tls.LoadX509KeyPair(
		filepath.Join(secretsDir, "web.crt"),
		filepath.Join(secretsDir, "web.key"),
	)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		RootCAs:      certpool,
		Certificates: []tls.Certificate{cert},
		// InsecureSkipVerify allows connecting to testcontainer mapped ports (localhost vs container IP)
		InsecureSkipVerify: true,
	}, nil
}

// NewTestMQTTClient creates an mTLS-enabled client connected to the provided broker port
func NewTestMQTTClient(brokerHost string, brokerPort string, clientID string) (mqtt.Client, error) {
	tlsConfig, err := GetTLSConfig()
	if err != nil {
		return nil, err
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("ssl://%s:%s", brokerHost, brokerPort))
	opts.SetClientID(clientID)
	opts.SetTLSConfig(tlsConfig)

	client := mqtt.NewClient(opts)
	token := client.Connect()
	token.WaitTimeout(5 * time.Second)
	
	if token.Error() != nil {
		return nil, token.Error()
	}

	return client, nil
}
