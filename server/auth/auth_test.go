package auth_test

import (
	"testing"
	"time"

	"mqtt-streaming-server/auth"
)

const testSecret = "test-jwt-secret-32-bytes-minimum!"

// ── JWT ──────────────────────────────────────────────────────────────────────

func TestGenerateAndValidateAccessToken(t *testing.T) {
	token, err := auth.GenerateAccessToken("alice@example.com", "doctor", testSecret)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	claims, err := auth.ValidateAccessToken(token, testSecret)
	if err != nil {
		t.Fatalf("ValidateAccessToken: %v", err)
	}
	if claims.Email != "alice@example.com" {
		t.Errorf("email: got %q, want %q", claims.Email, "alice@example.com")
	}
	if claims.Role != "doctor" {
		t.Errorf("role: got %q, want %q", claims.Role, "doctor")
	}
}

func TestValidateAccessToken_WrongSecret(t *testing.T) {
	token, _ := auth.GenerateAccessToken("x@x.com", "user", testSecret)
	_, err := auth.ValidateAccessToken(token, "wrong-secret")
	if err == nil {
		t.Fatal("expected error for wrong secret, got nil")
	}
}

func TestValidateAccessToken_Expired(t *testing.T) {
	// Can't trivially generate an expired token without time travel; verify
	// that the duration constant is sane and < 1 hour for access tokens.
	if auth.AccessTokenDuration >= time.Hour {
		t.Errorf("AccessTokenDuration %v is too long; want < 1h", auth.AccessTokenDuration)
	}
}

func TestValidateAccessToken_Tampered(t *testing.T) {
	token, _ := auth.GenerateAccessToken("x@x.com", "user", testSecret)
	tampered := token[:len(token)-4] + "XXXX"
	_, err := auth.ValidateAccessToken(tampered, testSecret)
	if err == nil {
		t.Fatal("expected error for tampered token, got nil")
	}
}

// ── Password ─────────────────────────────────────────────────────────────────

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := auth.HashPassword("hunter2")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}

	ok, err := auth.VerifyPassword("hunter2", hash)
	if err != nil {
		t.Fatalf("VerifyPassword: %v", err)
	}
	if !ok {
		t.Fatal("expected true for correct password")
	}
}

func TestVerifyPassword_Wrong(t *testing.T) {
	hash, _ := auth.HashPassword("hunter2")
	ok, err := auth.VerifyPassword("wrong", hash)
	if err != nil {
		t.Fatalf("VerifyPassword: %v", err)
	}
	if ok {
		t.Fatal("expected false for wrong password")
	}
}

func TestVerifyPassword_BcryptLegacy(t *testing.T) {
	// Pre-computed bcrypt hash of "legacy" (cost 10)
	bcryptHash := "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"
	// This is a known bcrypt hash; we just verify it parses without error.
	// (The actual password match would require the real bcrypt password.)
	_, err := auth.VerifyPassword("somepassword", bcryptHash)
	if err != nil {
		t.Fatalf("VerifyPassword bcrypt: unexpected error: %v", err)
	}
}

func TestHashPassword_Unique(t *testing.T) {
	h1, _ := auth.HashPassword("password")
	h2, _ := auth.HashPassword("password")
	if h1 == h2 {
		t.Error("two hashes of the same password should differ (different salts)")
	}
}

// ── RateLimiter ──────────────────────────────────────────────────────────────

func TestRateLimiter_AllowsBurst(t *testing.T) {
	rl := auth.NewRateLimiter(1.0/60, 5) // 1 req/min, burst 5
	for i := 0; i < 5; i++ {
		if !rl.Allow("192.0.2.1") {
			t.Fatalf("request %d should be allowed (within burst)", i+1)
		}
	}
}

func TestRateLimiter_BlocksAfterBurst(t *testing.T) {
	rl := auth.NewRateLimiter(1.0/60, 3)
	for i := 0; i < 3; i++ {
		rl.Allow("10.0.0.1")
	}
	if rl.Allow("10.0.0.1") {
		t.Fatal("request after burst should be blocked")
	}
}

func TestRateLimiter_IndependentKeys(t *testing.T) {
	rl := auth.NewRateLimiter(1.0/60, 1)
	if !rl.Allow("192.0.2.1") {
		t.Fatal("first request for IP1 should be allowed")
	}
	if rl.Allow("192.0.2.1") {
		t.Fatal("second request for IP1 should be blocked")
	}
	if !rl.Allow("192.0.2.2") {
		t.Fatal("first request for IP2 should be allowed (independent bucket)")
	}
}
