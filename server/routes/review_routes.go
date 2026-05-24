package routes

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"mqtt-streaming-server/auth"
	"mqtt-streaming-server/audit"
	"mqtt-streaming-server/domain"
	"mqtt-streaming-server/evidence"
)

type reviewController struct {
	items    domain.ReviewItemRepository
	auditor  audit.Writer
	evidence evidence.Chain
}

func initReviewRoutes(cfg *Config, mux *http.ServeMux) {
	c := &reviewController{
		items:    cfg.ReviewItemRepo,
		auditor:  cfg.AuditWriter,
		evidence: cfg.EvidenceChain,
	}
	withAuth := auth.WithAuth(cfg.JWTSecret)
	reviewers := auth.RequireRole(domain.RoleAdmin, domain.RoleDoctor)

	// List items and detail
	mux.Handle("/api/v1/review-queue",
		withAuth(reviewers(http.HandlerFunc(c.list))))
	// Approve / correct / reject — /api/v1/review-queue/{id}/{action}
	mux.Handle("/api/v1/review-queue/",
		withAuth(reviewers(http.HandlerFunc(c.action))))
}

func (c *reviewController) list(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	q := r.URL.Query()
	f := domain.ReviewItemFilters{
		Status:    domain.ReviewItemStatus(q.Get("status")),
		FieldName: q.Get("field_name"),
		ImageID:   q.Get("image_id"),
	}
	if f.Status == "" {
		f.Status = domain.ReviewItemPending
	}

	items, err := c.items.List(r.Context(), f)
	if err != nil {
		slog.Error("review list", "err", err)
		http.Error(w, "failed to fetch review items", http.StatusInternalServerError)
		return
	}

	_ = c.auditor.Write(r.Context(), audit.Entry{
		Timestamp:    time.Now().UTC(),
		ActorEmail:   auth.EmailFromCtx(r.Context()),
		ActorIP:      r.RemoteAddr,
		Action:       "review_queue_list",
		ResourceType: "review_items",
		Details:      map[string]any{"status": f.Status, "count": len(items)},
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

// action handles /api/v1/review-queue/{id}/approve|correct|reject
func (c *reviewController) action(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Path: /api/v1/review-queue/{id}/{action}
	trimmed := strings.TrimPrefix(r.URL.Path, "/api/v1/review-queue/")
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) != 2 {
		http.Error(w, "invalid path: use /{id}/approve|correct|reject", http.StatusBadRequest)
		return
	}
	itemID, action := parts[0], parts[1]

	item, err := c.items.GetByID(r.Context(), itemID)
	if err != nil {
		http.Error(w, "review item not found", http.StatusNotFound)
		return
	}
	if item.Status != domain.ReviewItemPending {
		http.Error(w, "item is not pending", http.StatusConflict)
		return
	}

	actorEmail := auth.EmailFromCtx(r.Context())
	now := time.Now().UTC()
	update := domain.ReviewItemUpdate{
		ReviewerEmail: actorEmail,
		ReviewedAt:    now,
	}

	switch action {
	case "approve":
		update.Status = domain.ReviewItemApproved
	case "correct":
		var req struct {
			CorrectedValue string `json:"corrected_value"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.CorrectedValue == "" {
			http.Error(w, "corrected_value required", http.StatusBadRequest)
			return
		}
		update.Status = domain.ReviewItemCorrected
		update.CorrectedValue = &req.CorrectedValue
	case "reject":
		update.Status = domain.ReviewItemRejected
	default:
		http.Error(w, "unknown action; use approve|correct|reject", http.StatusBadRequest)
		return
	}

	if err := c.items.UpdateStatus(r.Context(), itemID, update); err != nil {
		slog.Error("review action", "action", action, "id", itemID, "err", err)
		http.Error(w, "failed to update item", http.StatusInternalServerError)
		return
	}

	_ = c.auditor.Write(r.Context(), audit.Entry{
		Timestamp:    now,
		ActorEmail:   actorEmail,
		ActorIP:      r.RemoteAddr,
		Action:       "review_" + action,
		ResourceType: "review_item",
		ResourceID:   itemID,
	})
	_ = c.evidence.Append(r.Context(), evidence.Entry{
		ActorEmail: actorEmail,
		Action:     "review_" + action,
		Payload:    map[string]any{"item_id": itemID, "status": string(update.Status)},
	})

	w.WriteHeader(http.StatusNoContent)
}
