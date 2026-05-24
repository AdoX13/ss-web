package audit_test

import (
	"context"
	"testing"
	"time"

	"mqtt-streaming-server/audit"
)

func testEntry() audit.Entry {
	return audit.Entry{
		Timestamp:    time.Now().UTC(),
		ActorEmail:   "actor@test.com",
		ActorIP:      "127.0.0.1",
		Action:       "test_action",
		ResourceType: "photo",
		ResourceID:   "abc123",
		Details:      map[string]any{"key": "value"},
	}
}

func TestSlog_Write(t *testing.T) {
	w := &audit.Slog{}
	if err := w.Write(context.Background(), testEntry()); err != nil {
		t.Fatalf("Slog.Write: unexpected error: %v", err)
	}
}

func TestNoop_Write(t *testing.T) {
	w := &audit.Noop{}
	if err := w.Write(context.Background(), testEntry()); err != nil {
		t.Fatalf("Noop.Write: unexpected error: %v", err)
	}
}

func TestSlog_Write_EmptyEntry(t *testing.T) {
	w := &audit.Slog{}
	if err := w.Write(context.Background(), audit.Entry{}); err != nil {
		t.Fatalf("Slog.Write empty entry: %v", err)
	}
}

func TestNoop_Write_NilDetails(t *testing.T) {
	w := &audit.Noop{}
	e := testEntry()
	e.Details = nil
	if err := w.Write(context.Background(), e); err != nil {
		t.Fatalf("Noop.Write nil details: %v", err)
	}
}
