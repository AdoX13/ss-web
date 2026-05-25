package routes

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/mongo"

	"mqtt-streaming-server/auth"
	"mqtt-streaming-server/domain"
)

type authController struct {
	users         domain.UserRepository
	refreshTokens domain.RefreshTokenRepository
	jwtSecret     string
	rateLimiter   *auth.RateLimiter
}

func initAuthRoutes(cfg *Config, mux *http.ServeMux) {
	c := &authController{
		users:         cfg.UserRepo,
		refreshTokens: cfg.RefreshTokenRepo,
		jwtSecret:     cfg.JWTSecret,
		rateLimiter:   cfg.AuthRateLimiter,
	}
	rl := auth.RateLimit(c.rateLimiter)

	mux.Handle("/api/v1/auth/register", rl(http.HandlerFunc(c.register)))
	mux.Handle("/api/v1/auth/login", rl(http.HandlerFunc(c.login)))
	mux.Handle("/api/v1/auth/refresh", rl(http.HandlerFunc(c.refresh)))
	mux.Handle("/api/v1/auth/logout", auth.WithAuth(c.jwtSecret)(http.HandlerFunc(c.logout)))
}

func (c *authController) register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Email == "" || len(req.Password) < 8 {
		http.Error(w, "email required and password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	existing, err := c.users.FindByEmail(r.Context(), req.Email)
	if err != nil && err != mongo.ErrNoDocuments {
		slog.Error("register: check user", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if existing != nil {
		http.Error(w, "email already registered", http.StatusConflict)
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		slog.Error("register: hash password", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := c.users.Save(r.Context(), req.Email, hash); err != nil {
		slog.Error("register: save user", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (c *authController) login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	user, err := c.users.FindByEmail(r.Context(), req.Email)
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	ok, err := auth.VerifyPassword(req.Password, user.Password)
	if err != nil || !ok {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	// Transparent bcrypt → Argon2id migration: re-hash on successful bcrypt login.
	if len(user.Password) > 3 && user.Password[:3] == "$2a" || len(user.Password) > 3 && user.Password[:3] == "$2b" {
		if newHash, err := auth.HashPassword(req.Password); err == nil {
			_ = c.users.UpdatePassword(r.Context(), user.Email, newHash)
		}
	}

	accessToken, err := auth.GenerateAccessToken(user.Email, user.Role, c.jwtSecret)
	if err != nil {
		slog.Error("login: generate access token", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	refreshToken, err := generateRefreshToken()
	if err != nil {
		slog.Error("login: generate refresh token", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	rt := &domain.RefreshToken{
		Token:     refreshToken,
		Email:     user.Email,
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(auth.RefreshTokenDuration),
	}
	if err := c.refreshTokens.Save(r.Context(), rt); err != nil {
		slog.Error("login: save refresh token", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "Bearer",
		"email":         user.Email,
		"role":          user.Role,
	})
}

func (c *authController) refresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		http.Error(w, "refresh_token required", http.StatusBadRequest)
		return
	}

	rt, err := c.refreshTokens.FindByToken(r.Context(), req.RefreshToken)
	if err != nil {
		http.Error(w, "invalid refresh token", http.StatusUnauthorized)
		return
	}
	if time.Now().UTC().After(rt.ExpiresAt) {
		_ = c.refreshTokens.Revoke(r.Context(), req.RefreshToken)
		http.Error(w, "refresh token expired", http.StatusUnauthorized)
		return
	}

	user, err := c.users.FindByEmail(r.Context(), rt.Email)
	if err != nil {
		http.Error(w, "user not found", http.StatusUnauthorized)
		return
	}

	// Token rotation: revoke old, issue new pair.
	_ = c.refreshTokens.Revoke(r.Context(), req.RefreshToken)

	accessToken, err := auth.GenerateAccessToken(user.Email, user.Role, c.jwtSecret)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	newRefresh, err := generateRefreshToken()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	newRT := &domain.RefreshToken{
		Token:     newRefresh,
		Email:     user.Email,
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(auth.RefreshTokenDuration),
	}
	_ = c.refreshTokens.Save(r.Context(), newRT)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"access_token":  accessToken,
		"refresh_token": newRefresh,
		"token_type":    "Bearer",
	})
}

func (c *authController) logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	email := auth.EmailFromCtx(r.Context())
	if err := c.refreshTokens.RevokeAllForEmail(r.Context(), email); err != nil {
		slog.Error("logout: revoke tokens", "email", email, "err", err)
	}
	w.WriteHeader(http.StatusNoContent)
}

func generateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
