package audit

import (
	"testing"
	"time"
)

func TestNormalizeLimit(t *testing.T) {
	cases := []struct {
		name string
		in   int64
		want int64
	}{
		{name: "default for zero", in: 0, want: defaultAuditLimit},
		{name: "default for negative", in: -1, want: defaultAuditLimit},
		{name: "keeps valid limit", in: 25, want: 25},
		{name: "default for too large", in: maxAuditLimit + 1, want: defaultAuditLimit},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizeLimit(tc.in); got != tc.want {
				t.Fatalf("normalizeLimit(%d): got %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}

func TestMatchesFilter(t *testing.T) {
	ts := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	entry := Entry{
		Timestamp:    ts,
		ActorEmail:   "doctor@example.com",
		Action:       "report_run",
		ResourceType: "report",
		ResourceID:   "recent_exams",
	}

	if !matchesFilter(entry, Filter{
		ActorEmail:   "doctor@example.com",
		Action:       "report_run",
		ResourceType: "report",
		ResourceID:   "recent_exams",
		From:         ts.Add(-time.Minute),
		To:           ts.Add(time.Minute),
	}) {
		t.Fatal("expected filter to match")
	}
	if matchesFilter(entry, Filter{ActorEmail: "other@example.com"}) {
		t.Fatal("unexpected actor_email match")
	}
	if matchesFilter(entry, Filter{From: ts.Add(time.Second)}) {
		t.Fatal("unexpected from-range match")
	}
}
