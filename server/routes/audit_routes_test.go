package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseAuditFilterRejectsUnsupportedCharacters(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit?action=report_run%7B", nil)
	if _, err := parseAuditFilter(req); err == nil {
		t.Fatal("expected invalid audit filter error")
	}
}

func TestParseAuditFilterAcceptsSafeTokens(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit?action=report_run&resource_type=report&resource_id=recent_exams&limit=10", nil)
	filter, err := parseAuditFilter(req)
	if err != nil {
		t.Fatalf("parseAuditFilter: %v", err)
	}
	if filter.Action != "report_run" || filter.ResourceID != "recent_exams" || filter.Limit != 10 {
		t.Fatalf("unexpected filter: %+v", filter)
	}
}
