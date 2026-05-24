package routes

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	"mqtt-streaming-server/audit"
	"mqtt-streaming-server/auth"
	"mqtt-streaming-server/domain"
	"mqtt-streaming-server/evidence"
	mock_domain "mqtt-streaming-server/mocks"
)

func newTestReviewCtrl(items domain.ReviewItemRepository) *reviewController {
	return &reviewController{
		items:    items,
		auditor:  &audit.Noop{},
		evidence: &evidence.Noop{},
	}
}

func ctxWithEmail(r *http.Request, email string) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), auth.ContextEmail, email))
}

// ── list ──────────────────────────────────────────────────────────────────────

func TestReviewList_DefaultsPending(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	items := mock_domain.NewMockReviewItemRepository(ctrl)
	items.EXPECT().List(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ interface{}, f domain.ReviewItemFilters) ([]*domain.ReviewItem, error) {
			if f.Status != domain.ReviewItemPending {
				t.Errorf("want default status=pending, got %q", f.Status)
			}
			return []*domain.ReviewItem{}, nil
		})

	c := newTestReviewCtrl(items)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/review-queue", nil)
	req = ctxWithEmail(req, "doc@example.com")
	rr := httptest.NewRecorder()
	c.list(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
}

func TestReviewList_MethodNotAllowed(t *testing.T) {
	c := newTestReviewCtrl(nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/review-queue", nil)
	rr := httptest.NewRecorder()
	c.list(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("want 405, got %d", rr.Code)
	}
}

func TestReviewList_DBError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	items := mock_domain.NewMockReviewItemRepository(ctrl)
	items.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, errors.New("not found"))

	c := newTestReviewCtrl(items)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/review-queue", nil)
	req = ctxWithEmail(req, "doc@example.com")
	rr := httptest.NewRecorder()
	c.list(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", rr.Code)
	}
}

// ── action ────────────────────────────────────────────────────────────────────

func pendingItem(id string) *domain.ReviewItem {
	return &domain.ReviewItem{
		FieldName: "patient_name",
		Status:    domain.ReviewItemPending,
	}
}

func TestReviewAction_Approve(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	items := mock_domain.NewMockReviewItemRepository(ctrl)
	items.EXPECT().GetByID(gomock.Any(), "abc123").Return(pendingItem("abc123"), nil)
	items.EXPECT().UpdateStatus(gomock.Any(), "abc123", gomock.Any()).
		DoAndReturn(func(_ interface{}, _ string, u domain.ReviewItemUpdate) error {
			if u.Status != domain.ReviewItemApproved {
				t.Errorf("want approved, got %q", u.Status)
			}
			return nil
		})

	c := newTestReviewCtrl(items)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/review-queue/abc123/approve", nil)
	req = ctxWithEmail(req, "doc@example.com")
	rr := httptest.NewRecorder()
	c.action(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("want 204, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestReviewAction_Correct(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	items := mock_domain.NewMockReviewItemRepository(ctrl)
	items.EXPECT().GetByID(gomock.Any(), "abc123").Return(pendingItem("abc123"), nil)
	items.EXPECT().UpdateStatus(gomock.Any(), "abc123", gomock.Any()).
		DoAndReturn(func(_ interface{}, _ string, u domain.ReviewItemUpdate) error {
			if u.Status != domain.ReviewItemCorrected {
				t.Errorf("want corrected, got %q", u.Status)
			}
			if u.CorrectedValue == nil || *u.CorrectedValue != "John Doe" {
				t.Errorf("want CorrectedValue=John Doe")
			}
			return nil
		})

	c := newTestReviewCtrl(items)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/review-queue/abc123/correct",
		strings.NewReader(`{"corrected_value":"John Doe"}`))
	req = ctxWithEmail(req, "doc@example.com")
	rr := httptest.NewRecorder()
	c.action(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("want 204, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestReviewAction_Reject(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	items := mock_domain.NewMockReviewItemRepository(ctrl)
	items.EXPECT().GetByID(gomock.Any(), "abc123").Return(pendingItem("abc123"), nil)
	items.EXPECT().UpdateStatus(gomock.Any(), "abc123", gomock.Any()).
		DoAndReturn(func(_ interface{}, _ string, u domain.ReviewItemUpdate) error {
			if u.Status != domain.ReviewItemRejected {
				t.Errorf("want rejected, got %q", u.Status)
			}
			return nil
		})

	c := newTestReviewCtrl(items)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/review-queue/abc123/reject", nil)
	req = ctxWithEmail(req, "doc@example.com")
	rr := httptest.NewRecorder()
	c.action(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("want 204, got %d", rr.Code)
	}
}

func TestReviewAction_ItemNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	items := mock_domain.NewMockReviewItemRepository(ctrl)
	items.EXPECT().GetByID(gomock.Any(), "missing").Return(nil, errors.New("not found"))

	c := newTestReviewCtrl(items)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/review-queue/missing/approve", nil)
	req = ctxWithEmail(req, "doc@example.com")
	rr := httptest.NewRecorder()
	c.action(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", rr.Code)
	}
}

func TestReviewAction_NotPending(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	alreadyApproved := &domain.ReviewItem{Status: domain.ReviewItemApproved}
	items := mock_domain.NewMockReviewItemRepository(ctrl)
	items.EXPECT().GetByID(gomock.Any(), "done").Return(alreadyApproved, nil)

	c := newTestReviewCtrl(items)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/review-queue/done/approve", nil)
	req = ctxWithEmail(req, "doc@example.com")
	rr := httptest.NewRecorder()
	c.action(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("want 409, got %d", rr.Code)
	}
}

func TestReviewAction_UnknownAction(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	items := mock_domain.NewMockReviewItemRepository(ctrl)
	items.EXPECT().GetByID(gomock.Any(), "abc123").Return(pendingItem("abc123"), nil)

	c := newTestReviewCtrl(items)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/review-queue/abc123/magic", nil)
	req = ctxWithEmail(req, "doc@example.com")
	rr := httptest.NewRecorder()
	c.action(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rr.Code)
	}
}

func TestReviewAction_MethodNotAllowed(t *testing.T) {
	c := newTestReviewCtrl(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/review-queue/abc123/approve", nil)
	rr := httptest.NewRecorder()
	c.action(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("want 405, got %d", rr.Code)
	}
}

func TestReviewAction_InvalidPath(t *testing.T) {
	c := newTestReviewCtrl(nil)
	// Only one path segment after the prefix — missing the action part
	req := httptest.NewRequest(http.MethodPost, "/api/v1/review-queue/abc123", nil)
	rr := httptest.NewRecorder()
	c.action(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rr.Code)
	}
}

func TestReviewAction_CorrectMissingValue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	items := mock_domain.NewMockReviewItemRepository(ctrl)
	items.EXPECT().GetByID(gomock.Any(), "abc123").Return(pendingItem("abc123"), nil)

	c := newTestReviewCtrl(items)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/review-queue/abc123/correct",
		strings.NewReader(`{"corrected_value":""}`))
	req = ctxWithEmail(req, "doc@example.com")
	rr := httptest.NewRecorder()
	c.action(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d: %s", rr.Code, rr.Body.String())
	}
}
