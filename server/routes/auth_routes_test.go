package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/mock/gomock"

	"mqtt-streaming-server/auth"
	"mqtt-streaming-server/domain"
	mock_domain "mqtt-streaming-server/mocks"
)

const authTestSecret = "test-jwt-secret-for-auth-routes-32b!"

func newTestAuthCtrl(users domain.UserRepository, refreshTokens domain.RefreshTokenRepository) *authController {
	return &authController{
		users:         users,
		refreshTokens: refreshTokens,
		jwtSecret:     authTestSecret,
		rateLimiter:   auth.NewRateLimiter(100, 1000),
	}
}

func hashForTest(t *testing.T, pw string) string {
	t.Helper()
	h, err := auth.HashPassword(pw)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	return h
}

// ── register ─────────────────────────────────────────────────────────────────

func TestAuthRegister_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	users := mock_domain.NewMockUserRepository(ctrl)
	users.EXPECT().FindByEmail(gomock.Any(), "new@example.com").Return(nil, mongo.ErrNoDocuments)
	users.EXPECT().Save(gomock.Any(), "new@example.com", gomock.Any()).Return(nil)

	c := newTestAuthCtrl(users, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register",
		strings.NewReader(`{"email":"new@example.com","password":"password123"}`))
	rr := httptest.NewRecorder()
	c.register(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAuthRegister_MethodNotAllowed(t *testing.T) {
	c := newTestAuthCtrl(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/register", nil)
	rr := httptest.NewRecorder()
	c.register(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("want 405, got %d", rr.Code)
	}
}

func TestAuthRegister_InvalidJSON(t *testing.T) {
	c := newTestAuthCtrl(nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register",
		strings.NewReader(`{bad`))
	rr := httptest.NewRecorder()
	c.register(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rr.Code)
	}
}

func TestAuthRegister_ShortPassword(t *testing.T) {
	c := newTestAuthCtrl(nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register",
		strings.NewReader(`{"email":"x@x.com","password":"short"}`))
	rr := httptest.NewRecorder()
	c.register(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rr.Code)
	}
}

func TestAuthRegister_EmailRequired(t *testing.T) {
	c := newTestAuthCtrl(nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register",
		strings.NewReader(`{"email":"","password":"password123"}`))
	rr := httptest.NewRecorder()
	c.register(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rr.Code)
	}
}

func TestAuthRegister_AlreadyExists(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	users := mock_domain.NewMockUserRepository(ctrl)
	users.EXPECT().FindByEmail(gomock.Any(), "dup@example.com").
		Return(&domain.User{Email: "dup@example.com"}, nil)

	c := newTestAuthCtrl(users, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register",
		strings.NewReader(`{"email":"dup@example.com","password":"password123"}`))
	rr := httptest.NewRecorder()
	c.register(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("want 409, got %d", rr.Code)
	}
}

// ── login ─────────────────────────────────────────────────────────────────────

func TestAuthLogin_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	hash := hashForTest(t, "password123")
	users := mock_domain.NewMockUserRepository(ctrl)
	users.EXPECT().FindByEmail(gomock.Any(), "u@example.com").
		Return(&domain.User{Email: "u@example.com", Password: hash, Role: "doctor"}, nil)

	tokens := mock_domain.NewMockRefreshTokenRepository(ctrl)
	tokens.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)

	c := newTestAuthCtrl(users, tokens)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
		strings.NewReader(`{"email":"u@example.com","password":"password123"}`))
	rr := httptest.NewRecorder()
	c.login(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, "access_token") || !strings.Contains(body, "refresh_token") {
		t.Errorf("response missing token fields: %s", body)
	}
}

func TestAuthLogin_WrongPassword(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	hash := hashForTest(t, "password123")
	users := mock_domain.NewMockUserRepository(ctrl)
	users.EXPECT().FindByEmail(gomock.Any(), "u@example.com").
		Return(&domain.User{Email: "u@example.com", Password: hash}, nil)

	c := newTestAuthCtrl(users, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
		strings.NewReader(`{"email":"u@example.com","password":"wrongpass1"}`))
	rr := httptest.NewRecorder()
	c.login(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rr.Code)
	}
}

func TestAuthLogin_UserNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	users := mock_domain.NewMockUserRepository(ctrl)
	users.EXPECT().FindByEmail(gomock.Any(), "ghost@example.com").
		Return(nil, mongo.ErrNoDocuments)

	c := newTestAuthCtrl(users, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
		strings.NewReader(`{"email":"ghost@example.com","password":"password123"}`))
	rr := httptest.NewRecorder()
	c.login(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rr.Code)
	}
}

func TestAuthLogin_MethodNotAllowed(t *testing.T) {
	c := newTestAuthCtrl(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/login", nil)
	rr := httptest.NewRecorder()
	c.login(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("want 405, got %d", rr.Code)
	}
}

func TestAuthLogin_InvalidJSON(t *testing.T) {
	c := newTestAuthCtrl(nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
		strings.NewReader(`{bad`))
	rr := httptest.NewRecorder()
	c.login(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rr.Code)
	}
}

// ── refresh ───────────────────────────────────────────────────────────────────

func TestAuthRefresh_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	future := time.Now().UTC().Add(7 * 24 * time.Hour)
	existing := &domain.RefreshToken{Token: "valid-rt", Email: "u@example.com", ExpiresAt: future}

	users := mock_domain.NewMockUserRepository(ctrl)
	users.EXPECT().FindByEmail(gomock.Any(), "u@example.com").
		Return(&domain.User{Email: "u@example.com", Role: "doctor"}, nil)

	tokens := mock_domain.NewMockRefreshTokenRepository(ctrl)
	tokens.EXPECT().FindByToken(gomock.Any(), "valid-rt").Return(existing, nil)
	tokens.EXPECT().Revoke(gomock.Any(), "valid-rt").Return(nil)
	tokens.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)

	c := newTestAuthCtrl(users, tokens)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh",
		strings.NewReader(`{"refresh_token":"valid-rt"}`))
	rr := httptest.NewRecorder()
	c.refresh(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAuthRefresh_Expired(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	past := time.Now().UTC().Add(-1 * time.Hour)
	existing := &domain.RefreshToken{Token: "expired-rt", Email: "u@example.com", ExpiresAt: past}

	tokens := mock_domain.NewMockRefreshTokenRepository(ctrl)
	tokens.EXPECT().FindByToken(gomock.Any(), "expired-rt").Return(existing, nil)
	tokens.EXPECT().Revoke(gomock.Any(), "expired-rt").Return(nil)

	c := newTestAuthCtrl(nil, tokens)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh",
		strings.NewReader(`{"refresh_token":"expired-rt"}`))
	rr := httptest.NewRecorder()
	c.refresh(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rr.Code)
	}
}

func TestAuthRefresh_TokenNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tokens := mock_domain.NewMockRefreshTokenRepository(ctrl)
	tokens.EXPECT().FindByToken(gomock.Any(), "bad-rt").Return(nil, mongo.ErrNoDocuments)

	c := newTestAuthCtrl(nil, tokens)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh",
		strings.NewReader(`{"refresh_token":"bad-rt"}`))
	rr := httptest.NewRecorder()
	c.refresh(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rr.Code)
	}
}

func TestAuthRefresh_MissingToken(t *testing.T) {
	c := newTestAuthCtrl(nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh",
		strings.NewReader(`{}`))
	rr := httptest.NewRecorder()
	c.refresh(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rr.Code)
	}
}

func TestAuthRefresh_MethodNotAllowed(t *testing.T) {
	c := newTestAuthCtrl(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/refresh", nil)
	rr := httptest.NewRecorder()
	c.refresh(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("want 405, got %d", rr.Code)
	}
}

// ── logout ────────────────────────────────────────────────────────────────────

func TestAuthLogout_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tokens := mock_domain.NewMockRefreshTokenRepository(ctrl)
	tokens.EXPECT().RevokeAllForEmail(gomock.Any(), "u@example.com").Return(nil)

	c := newTestAuthCtrl(nil, tokens)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	ctx := context.WithValue(req.Context(), auth.ContextEmail, "u@example.com")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	c.logout(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("want 204, got %d", rr.Code)
	}
}

func TestAuthLogout_MethodNotAllowed(t *testing.T) {
	c := newTestAuthCtrl(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/logout", nil)
	rr := httptest.NewRecorder()
	c.logout(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("want 405, got %d", rr.Code)
	}
}
