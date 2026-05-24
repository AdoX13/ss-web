package routes

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"mqtt-streaming-server/auth"
	"mqtt-streaming-server/audit"
	"mqtt-streaming-server/domain"
)

type usersController struct {
	users   domain.UserRepository
	auditor audit.Writer
}

func initUsersRoutes(cfg *Config, mux *http.ServeMux) {
	c := &usersController{
		users:   cfg.UserRepo,
		auditor: cfg.AuditWriter,
	}
	withAuth := auth.WithAuth(cfg.JWTSecret)
	adminOnly := auth.RequireRole(domain.RoleAdmin)

	mux.Handle("/api/v1/users", withAuth(adminOnly(http.HandlerFunc(c.handleCollection))))
	mux.Handle("/api/v1/users/", withAuth(adminOnly(http.HandlerFunc(c.handleItem))))
}

func (c *usersController) handleCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		c.listUsers(w, r)
	case http.MethodPost:
		c.createUser(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (c *usersController) handleItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// Path: /api/v1/users/{email}
	email := strings.TrimPrefix(r.URL.Path, "/api/v1/users/")
	if email == "" {
		http.Error(w, "email required in path", http.StatusBadRequest)
		return
	}
	c.updateUser(w, r, email)
}

func (c *usersController) listUsers(w http.ResponseWriter, r *http.Request) {
	users, err := c.users.GetAll(r.Context())
	if err != nil {
		slog.Error("list users", "err", err)
		http.Error(w, "failed to fetch users", http.StatusInternalServerError)
		return
	}
	// Redact passwords before sending.
	for _, u := range users {
		u.Password = ""
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func (c *usersController) createUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Email == "" || len(req.Password) < 8 {
		http.Error(w, "email required; password must be >= 8 chars", http.StatusBadRequest)
		return
	}
	if !isValidRole(req.Role) {
		req.Role = domain.RoleDoctor
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := c.users.Save(r.Context(), req.Email, hash); err != nil {
		slog.Error("create user", "err", err)
		http.Error(w, "failed to create user", http.StatusInternalServerError)
		return
	}
	// Set the requested role (Save defaults to doctor).
	if req.Role != domain.RoleDoctor {
		_ = c.users.UpdateRole(r.Context(), req.Email, req.Role)
	}

	_ = c.auditor.Write(r.Context(), audit.Entry{
		ActorEmail:   auth.EmailFromCtx(r.Context()),
		Action:       "create_user",
		ResourceType: "user",
		ResourceID:   req.Email,
	})

	w.WriteHeader(http.StatusCreated)
}

func (c *usersController) updateUser(w http.ResponseWriter, r *http.Request, email string) {
	var req struct {
		Role       string `json:"role"`
		Deactivate bool   `json:"deactivate"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	actor := auth.EmailFromCtx(r.Context())

	if req.Deactivate {
		if err := c.users.Deactivate(r.Context(), email); err != nil {
			slog.Error("deactivate user", "email", email, "err", err)
			http.Error(w, "failed to deactivate user", http.StatusInternalServerError)
			return
		}
		_ = c.auditor.Write(r.Context(), audit.Entry{
			ActorEmail: actor, Action: "deactivate_user",
			ResourceType: "user", ResourceID: email,
		})
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if req.Role != "" {
		if !isValidRole(req.Role) {
			http.Error(w, "invalid role", http.StatusBadRequest)
			return
		}
		if err := c.users.UpdateRole(r.Context(), email, req.Role); err != nil {
			slog.Error("update role", "email", email, "err", err)
			http.Error(w, "failed to update role", http.StatusInternalServerError)
			return
		}
		_ = c.auditor.Write(r.Context(), audit.Entry{
			ActorEmail: actor, Action: "update_role",
			ResourceType: "user", ResourceID: email,
			Details: map[string]any{"new_role": req.Role},
		})
	}

	w.WriteHeader(http.StatusNoContent)
}

func isValidRole(role string) bool {
	switch role {
	case domain.RoleAdmin, domain.RoleDoctor, domain.RoleResearcher, domain.RoleAuditor:
		return true
	}
	return false
}
