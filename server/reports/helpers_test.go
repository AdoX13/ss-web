package reports

import (
	"testing"
	"time"
)

func TestReportFormattingHelpers(t *testing.T) {
	if got := cleanControlType("Control Periodic"); got != "Periodic" {
		t.Fatalf("cleanControlType: got %q", got)
	}
	if got := cleanOpinion("apt conditionat"); got != "APT Conditionat" {
		t.Fatalf("cleanOpinion: got %q", got)
	}
	if got := percent(1, 3); got != 33.33 {
		t.Fatalf("percent: got %v", got)
	}
	month := formatMonth(time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC))
	if month != "2026-05" {
		t.Fatalf("formatMonth: got %q", month)
	}
}
