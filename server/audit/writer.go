// Package audit provides the PHI audit-log interface and a slog-based stub.
// P6 replaces the Slog stub with a MongoDB writer (server/audit/ is P6's lane).
package audit

import (
	"context"
	"log/slog"
	"time"
)

// Entry is one audit log record.
type Entry struct {
	Timestamp    time.Time
	ActorEmail   string
	ActorIP      string
	Action       string
	ResourceType string
	ResourceID   string
	Details      map[string]any
}

// Writer persists audit entries. The zero value (nil) is safe to use — Write
// is a no-op, so callers don't need nil-guards.
type Writer interface {
	Write(ctx context.Context, e Entry) error
}

// Slog logs audit entries via slog. P6 replaces this with the MongoDB writer.
type Slog struct{}

func (s *Slog) Write(_ context.Context, e Entry) error {
	slog.Info("audit",
		"actor", e.ActorEmail,
		"ip", e.ActorIP,
		"action", e.Action,
		"resource_type", e.ResourceType,
		"resource_id", e.ResourceID,
		"ts", e.Timestamp.Format(time.RFC3339),
	)
	return nil
}

// Noop discards every audit entry. Use in tests only.
type Noop struct{}

func (n *Noop) Write(_ context.Context, _ Entry) error { return nil }
