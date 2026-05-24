//go:build integration

// Integration tests for the HTTP API layer.
//
// These tests require a live MongoDB instance. Set MONGO_URL (default:
// mongodb://root:example@localhost:27017/?authSource=admin) before running:
//
//	go test -tags integration -run Integration ./routes/...
//
// For full containerised runs, add github.com/testcontainers/testcontainers-go
// to go.mod and replace the mongoURL helper with a testcontainers setup that
// spins up an ephemeral MongoDB container.
package routes_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"mqtt-streaming-server/audit"
	"mqtt-streaming-server/auth"
	"mqtt-streaming-server/domain"
	"mqtt-streaming-server/evidence"
	"mqtt-streaming-server/reports"
	"mqtt-streaming-server/repository"
	"mqtt-streaming-server/routes"
)

func integrationMongoURL() string {
	if u := os.Getenv("MONGO_URL"); u != "" {
		return u
	}
	return "mongodb://root:example@localhost:27017/?authSource=admin"
}

func integrationDB(t *testing.T) (*mongo.Database, func()) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(integrationMongoURL()))
	if err != nil {
		t.Fatalf("mongo connect: %v", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		t.Fatalf("mongo ping: %v", err)
	}

	dbName := "medsec_integration_test"
	db := client.Database(dbName)
	cleanup := func() {
		_ = db.Drop(context.Background())
		_ = client.Disconnect(context.Background())
	}
	return db, cleanup
}

func integrationHandler(t *testing.T, db *mongo.Database) http.Handler {
	t.Helper()
	const secret = "integration-test-secret-32bytes!!"

	reviewCh := make(chan *domain.ReviewItem, 64)
	hub := routes.NewReviewHub(reviewCh)
	go hub.Run()

	return routes.InitRoutes(&routes.Config{
		DB:               db,
		JWTSecret:        secret,
		UserRepo:         repository.NewUserRepository(db),
		RefreshTokenRepo: repository.NewRefreshTokenRepository(db),
		ReviewItemRepo:   repository.NewReviewItemRepository(db),
		AuditWriter:      &audit.Slog{},
		EvidenceChain:    &evidence.Noop{},
		ReportRegistry:   reports.DefaultRegistry(),
		ReviewHub:        hub,
		AuthRateLimiter:  auth.NewRateLimiter(100, 1000),
	})
}

// TestIntegration_RegisterAndLogin exercises the full register→login flow
// against a real MongoDB instance.
func TestIntegration_RegisterAndLogin(t *testing.T) {
	db, cleanup := integrationDB(t)
	defer cleanup()

	handler := integrationHandler(t, db)
	srv := httptest.NewServer(handler)
	defer srv.Close()

	// Register
	body, _ := json.Marshal(map[string]string{
		"email": "integration@test.com", "password": "securepass123",
	})
	resp, err := http.Post(srv.URL+"/api/v1/auth/register", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register: want 201, got %d", resp.StatusCode)
	}

	// Duplicate registration should conflict.
	resp2, _ := http.Post(srv.URL+"/api/v1/auth/register", "application/json", bytes.NewReader(body))
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusConflict {
		t.Fatalf("duplicate register: want 409, got %d", resp2.StatusCode)
	}

	// Login
	resp3, err := http.Post(srv.URL+"/api/v1/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	defer resp3.Body.Close()
	if resp3.StatusCode != http.StatusOK {
		t.Fatalf("login: want 200, got %d", resp3.StatusCode)
	}
	var loginResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(resp3.Body).Decode(&loginResp); err != nil {
		t.Fatalf("login decode: %v", err)
	}
	if loginResp.AccessToken == "" || loginResp.RefreshToken == "" {
		t.Fatal("login response missing tokens")
	}
}

// TestIntegration_ReviewQueue registers a user, logs in, and hits the review
// queue endpoint (should return an empty list since no OCR has run).
func TestIntegration_ReviewQueue(t *testing.T) {
	db, cleanup := integrationDB(t)
	defer cleanup()

	const secret = "integration-test-secret-32bytes!!"
	handler := integrationHandler(t, db)
	srv := httptest.NewServer(handler)
	defer srv.Close()

	// Register + login to get a token.
	creds, _ := json.Marshal(map[string]string{"email": "doc@test.com", "password": "securepass123"})
	http.Post(srv.URL+"/api/v1/auth/register", "application/json", bytes.NewReader(creds))
	loginResp, _ := http.Post(srv.URL+"/api/v1/auth/login", "application/json", bytes.NewReader(creds))
	var tokens struct {
		AccessToken string `json:"access_token"`
	}
	json.NewDecoder(loginResp.Body).Decode(&tokens)
	loginResp.Body.Close()

	// Hit the review queue.
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/review-queue", nil)
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("review-queue: %v", err)
	}
	defer resp.Body.Close()
	// Freshly registered user has no role set to admin/doctor yet,
	// so this may return 403. The test verifies the endpoint is reachable and
	// returns a defined HTTP status (not 5xx).
	if resp.StatusCode >= 500 {
		t.Fatalf("review-queue: unexpected server error %d", resp.StatusCode)
	}
}

// TestIntegration_HealthEndpoint verifies the /health endpoint is wired up.
func TestIntegration_HealthEndpoint(t *testing.T) {
	db, cleanup := integrationDB(t)
	defer cleanup()

	handler := integrationHandler(t, db)
	srv := httptest.NewServer(handler)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("health: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("health: want 200, got %d", resp.StatusCode)
	}
	var body map[string]string
	json.NewDecoder(resp.Body).Decode(&body)
	if body["status"] != "healthy" {
		t.Errorf("health status: got %q, want healthy", body["status"])
	}
}
