package routes

import (
	"encoding/json"
	"net/http"

	"mqtt-streaming-server/auth"
	"mqtt-streaming-server/domain"
	"mqtt-streaming-server/evidence"
)

func initEvidenceRoutes(cfg *Config, mux *http.ServeMux) {
	withAuth := auth.WithAuth(cfg.JWTSecret)
	allowed := auth.RequireRole(domain.RoleAdmin, domain.RoleAuditor)
	mux.Handle("/api/v1/evidence/chain/verify", withAuth(allowed(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		verifier, ok := cfg.EvidenceChain.(evidence.Verifier)
		if !ok {
			http.Error(w, "evidence verifier unavailable", http.StatusNotImplemented)
			return
		}
		result, err := verifier.Verify(r.Context())
		if err != nil {
			http.Error(w, "failed to verify evidence chain", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}))))
}
