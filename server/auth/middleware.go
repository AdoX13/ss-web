package auth

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const (
	ContextEmail contextKey = "email"
	ContextRole  contextKey = "role"
)

// WithAuth validates the Bearer token from the Authorization header and
// stores the claims in the request context. Returns 401 on missing or
// invalid tokens.
func WithAuth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				http.Error(w, "authorization required", http.StatusUnauthorized)
				return
			}
			tokenStr := strings.TrimPrefix(header, "Bearer ")
			claims, err := ValidateAccessToken(tokenStr, jwtSecret)
			if err != nil {
				http.Error(w, "invalid or expired token", http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), ContextEmail, claims.Email)
			ctx = context.WithValue(ctx, ContextRole, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole returns 403 if the authenticated user's role is not in the
// allowed set. Must be chained after WithAuth.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, _ := r.Context().Value(ContextRole).(string)
			if !allowed[role] {
				http.Error(w, "insufficient permissions", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// SecureHeaders sets OWASP-recommended response headers on every response.
func SecureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'")
		// HSTS is only meaningful over HTTPS; set it unconditionally so it takes
		// effect once TLS is terminated upstream.
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		next.ServeHTTP(w, r)
	})
}

// RateLimit returns 429 if the per-IP token bucket is exhausted.
func RateLimit(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r)
			if !limiter.Allow(ip) {
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// EmailFromCtx extracts the authenticated email from the request context.
func EmailFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(ContextEmail).(string)
	return v
}

// RoleFromCtx extracts the authenticated role from the request context.
func RoleFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(ContextRole).(string)
	return v
}

func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		// Take the first (client) IP in a potentially comma-separated list.
		if idx := strings.Index(fwd, ","); idx != -1 {
			return strings.TrimSpace(fwd[:idx])
		}
		return strings.TrimSpace(fwd)
	}
	// RemoteAddr is host:port — strip the port.
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		return addr[:idx]
	}
	return addr
}
