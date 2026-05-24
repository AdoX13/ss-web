package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"mqtt-streaming-server/auth"
)

// sentinel handler that writes 200 + a marker body so tests know it was reached
func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("reached"))
	})
}

// ── WithAuth ──────────────────────────────────────────────────────────────────

func TestWithAuth_ValidToken(t *testing.T) {
	token, err := auth.GenerateAccessToken("u@test.com", "doctor", testSecret)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	auth.WithAuth(testSecret)(okHandler()).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestWithAuth_MissingHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	auth.WithAuth(testSecret)(okHandler()).ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rr.Code)
	}
}

func TestWithAuth_MalformedHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "NotBearer token")
	rr := httptest.NewRecorder()
	auth.WithAuth(testSecret)(okHandler()).ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rr.Code)
	}
}

func TestWithAuth_InvalidToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer not.a.valid.jwt")
	rr := httptest.NewRecorder()
	auth.WithAuth(testSecret)(okHandler()).ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rr.Code)
	}
}

func TestWithAuth_WrongSecret(t *testing.T) {
	token, _ := auth.GenerateAccessToken("u@test.com", "doctor", "other-secret")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	auth.WithAuth(testSecret)(okHandler()).ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rr.Code)
	}
}

// ── RequireRole ───────────────────────────────────────────────────────────────

func validRequest(t *testing.T, role string) *http.Request {
	t.Helper()
	token, err := auth.GenerateAccessToken("u@test.com", role, testSecret)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	// Run through WithAuth so the context is populated.
	var captured *http.Request
	capture := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		captured = r
	})
	rr := httptest.NewRecorder()
	auth.WithAuth(testSecret)(capture).ServeHTTP(rr, req)
	return captured
}

func TestRequireRole_Allowed(t *testing.T) {
	req := validRequest(t, "admin")
	rr := httptest.NewRecorder()
	auth.RequireRole("admin", "doctor")(okHandler()).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
}

func TestRequireRole_Denied(t *testing.T) {
	req := validRequest(t, "researcher")
	rr := httptest.NewRecorder()
	auth.RequireRole("admin", "doctor")(okHandler()).ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d", rr.Code)
	}
}

// ── SecureHeaders ─────────────────────────────────────────────────────────────

func TestSecureHeaders_Present(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	auth.SecureHeaders(okHandler()).ServeHTTP(rr, req)

	headers := rr.Result().Header
	checks := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
	}
	for header, want := range checks {
		if got := headers.Get(header); got != want {
			t.Errorf("%s: got %q, want %q", header, got, want)
		}
	}
	if headers.Get("Content-Security-Policy") == "" {
		t.Error("Content-Security-Policy header missing")
	}
}

// ── RateLimit ─────────────────────────────────────────────────────────────────

func TestRateLimit_Middleware_Allows(t *testing.T) {
	rl := auth.NewRateLimiter(100, 100)
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	rr := httptest.NewRecorder()
	auth.RateLimit(rl)(okHandler()).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
}

func TestRateLimit_Middleware_Blocks(t *testing.T) {
	rl := auth.NewRateLimiter(0.001, 1)
	// Exhaust the burst of 1.
	req1 := httptest.NewRequest(http.MethodPost, "/", nil)
	req1.RemoteAddr = "10.0.0.2:1234"
	auth.RateLimit(rl)(okHandler()).ServeHTTP(httptest.NewRecorder(), req1)

	// Second request should be blocked.
	req2 := httptest.NewRequest(http.MethodPost, "/", nil)
	req2.RemoteAddr = "10.0.0.2:1234"
	rr := httptest.NewRecorder()
	auth.RateLimit(rl)(okHandler()).ServeHTTP(rr, req2)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("want 429, got %d", rr.Code)
	}
}

// ── EmailFromCtx / RoleFromCtx ────────────────────────────────────────────────

func TestEmailFromCtx_Empty(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// No auth context set — should return empty string without panic.
	email := auth.EmailFromCtx(req.Context())
	if email != "" {
		t.Errorf("want empty, got %q", email)
	}
}

func TestRoleFromCtx_Empty(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	role := auth.RoleFromCtx(req.Context())
	if role != "" {
		t.Errorf("want empty, got %q", role)
	}
}

func TestEmailFromCtx_AfterWithAuth(t *testing.T) {
	token, _ := auth.GenerateAccessToken("e@test.com", "doctor", testSecret)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	var gotEmail string
	handler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		gotEmail = auth.EmailFromCtx(r.Context())
	})
	auth.WithAuth(testSecret)(handler).ServeHTTP(httptest.NewRecorder(), req)
	if gotEmail != "e@test.com" {
		t.Errorf("want e@test.com, got %q", gotEmail)
	}
}
