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
	filter := bson.M{}
	if f.ActorEmail != "" {
		filter["actor_email"] = f.ActorEmail
	}
	if f.Action != "" {
		filter["action"] = f.Action
	}
	if f.ResourceType != "" {
		filter["resource_type"] = f.ResourceType
	}
	if f.ResourceID != "" {
		filter["resource_id"] = f.ResourceID
	}
	if !f.From.IsZero() || !f.To.IsZero() {
		ts := bson.M{}
		if !f.From.IsZero() {
			ts["$gte"] = f.From
		}
		if !f.To.IsZero() {
			ts["$lte"] = f.To
		}
		filter["ts"] = ts
	}

	limit := f.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	cursor, err := m.collection.Find(ctx, filter, options.Find().
		SetSort(bson.D{{Key: "ts", Value: -1}}).
		SetLimit(limit))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var entries []Entry
	if err := cursor.All(ctx, &entries); err != nil {
		return nil, err
	}
	return entries, nil
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
