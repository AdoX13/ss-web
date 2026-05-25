// Package audit provides PHI audit-log writers. The production writer is
// append-only by construction: it exposes Insert/List operations only.
package audit

import (
	"context"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	defaultAuditLimit = 100
	maxAuditLimit     = 500
	auditScanLimit    = 5000
)

// Entry is one audit log record.
type Entry struct {
	Timestamp    time.Time      `json:"ts" bson:"ts"`
	ActorEmail   string         `json:"actor_email" bson:"actor_email"`
	ActorIP      string         `json:"actor_ip" bson:"actor_ip"`
	Action       string         `json:"action" bson:"action"`
	ResourceType string         `json:"resource_type" bson:"resource_type"`
	ResourceID   string         `json:"resource_id" bson:"resource_id"`
	Details      map[string]any `json:"details,omitempty" bson:"details,omitempty"`
}

// Writer persists audit entries. Optional route integrations should still
// guard a nil interface; use Noop when a concrete no-op writer is desired.
type Writer interface {
	Write(ctx context.Context, e Entry) error
}

// Filter is used by the audit API to query recent records.
type Filter struct {
	ActorEmail   string
	Action       string
	ResourceType string
	ResourceID   string
	From         time.Time
	To           time.Time
	Limit        int64
}

// Reader lists audit entries without exposing update/delete paths.
type Reader interface {
	List(ctx context.Context, f Filter) ([]Entry, error)
}

// MongoWriter writes audit rows to MongoDB's audit_log collection.
type MongoWriter struct {
	collection *mongo.Collection
}

func NewMongoWriter(db *mongo.Database) *MongoWriter {
	return &MongoWriter{collection: db.Collection("audit_log")}
}

func (m *MongoWriter) Write(ctx context.Context, e Entry) error {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now().UTC()
	}
	if e.Details == nil {
		e.Details = map[string]any{}
	}
	_, err := m.collection.InsertOne(ctx, e)
	return err
}

func (m *MongoWriter) List(ctx context.Context, f Filter) ([]Entry, error) {
	limit := normalizeLimit(f.Limit)
	cursor, err := m.collection.Find(ctx, bson.D{}, options.Find().
		SetSort(bson.D{{Key: "ts", Value: -1}}).
		SetLimit(auditScanLimit))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	entries := make([]Entry, 0, int(limit))
	for cursor.Next(ctx) {
		var entry Entry
		if err := cursor.Decode(&entry); err != nil {
			return nil, err
		}
		if !matchesFilter(entry, f) {
			continue
		}
		entries = append(entries, entry)
		if int64(len(entries)) >= limit {
			break
		}
	}
	return entries, cursor.Err()
}

func normalizeLimit(limit int64) int64 {
	if limit <= 0 || limit > maxAuditLimit {
		return defaultAuditLimit
	}
	return limit
}

func matchesFilter(entry Entry, f Filter) bool {
	if f.ActorEmail != "" && entry.ActorEmail != f.ActorEmail {
		return false
	}
	if f.Action != "" && entry.Action != f.Action {
		return false
	}
	if f.ResourceType != "" && entry.ResourceType != f.ResourceType {
		return false
	}
	if f.ResourceID != "" && entry.ResourceID != f.ResourceID {
		return false
	}
	if !f.From.IsZero() && entry.Timestamp.Before(f.From) {
		return false
	}
	if !f.To.IsZero() && entry.Timestamp.After(f.To) {
		return false
	}
	return true
}

// Slog logs audit entries via slog. Use only when MongoDB is unavailable in
// tests or local experiments.
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
