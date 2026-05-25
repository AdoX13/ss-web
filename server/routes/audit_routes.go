package routes

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"mqtt-streaming-server/audit"
	"mqtt-streaming-server/auth"
	"mqtt-streaming-server/domain"
)

func initAuditRoutes(cfg *Config, mux *http.ServeMux) {
	withAuth := auth.WithAuth(cfg.JWTSecret)
	allowed := auth.RequireRole(domain.RoleAdmin, domain.RoleAuditor)
	mux.Handle("/api/v1/audit", withAuth(allowed(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		reader, ok := cfg.AuditWriter.(audit.Reader)
		if !ok {
			http.Error(w, "audit reader unavailable", http.StatusNotImplemented)
			return
		}
		filter, err := parseAuditFilter(r)
		if err != nil {
			http.Error(w, "invalid params: "+err.Error(), http.StatusBadRequest)
			return
		}
		entries, err := reader.List(r.Context(), filter)
		if err != nil {
			http.Error(w, "failed to query audit log", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(entries)
	}))))
}

func parseAuditFilter(r *http.Request) (audit.Filter, error) {
	q := r.URL.Query()
	filter := audit.Filter{
		ActorEmail:   q.Get("actor_email"),
		Action:       q.Get("action"),
		ResourceType: q.Get("resource_type"),
		ResourceID:   q.Get("resource_id"),
	}
	if s := q.Get("from"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return filter, err
		}
		filter.From = t
	}
	if s := q.Get("to"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return filter, err
		}
		filter.To = t
	}
	if s := q.Get("limit"); s != "" {
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return filter, err
		}
		filter.Limit = n
	}
	return filter, nil
}
