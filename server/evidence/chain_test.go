package evidence

import (
	"testing"
	"time"
)

func TestRecordHashIgnoresSignatureFields(t *testing.T) {
	rec := Record{
		Seq:        1,
		PrevHash:   "",
		ActorEmail: "doc@test.com",
		Action:     "review_approve",
		Payload:    map[string]any{"item_id": "abc"},
		CreatedAt:  time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC),
		ThisHash:   "ignored",
		Signature:  "ignored",
	}
	a, err := recordHash(rec)
	if err != nil {
		t.Fatalf("recordHash: %v", err)
	}
	rec.ThisHash = "tampered"
	rec.Signature = "also ignored"
	b, err := recordHash(rec)
	if err != nil {
		t.Fatalf("recordHash second: %v", err)
	}
	if a != b {
		t.Fatalf("hash should ignore this_hash/signature: %s != %s", a, b)
	}
}

func TestRecordHashDetectsPayloadChange(t *testing.T) {
	rec := Record{
		Seq:        1,
		ActorEmail: "doc@test.com",
		Action:     "report_run",
		Payload:    map[string]any{"report": "recent_exams", "rows": 3},
		CreatedAt:  time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC),
	}
	before, err := recordHash(rec)
	if err != nil {
		t.Fatalf("recordHash: %v", err)
	}
	rec.Payload["rows"] = 4
	after, err := recordHash(rec)
	if err != nil {
		t.Fatalf("recordHash after: %v", err)
	}
	if before == after {
		t.Fatal("payload mutation should change the record hash")
	}
}
