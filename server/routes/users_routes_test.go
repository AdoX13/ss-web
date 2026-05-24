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
	mock_domain "mqtt-streaming-server/mocks"
)

func newTestUsersCtrl(users domain.UserRepository) *usersController {
	return &usersController{
		users:   users,
		auditor: &audit.Noop{},
	}
}

func ctxWithRole(r *http.Request, email, role string) *http.Request {
	ctx := context.WithValue(r.Context(), auth.ContextEmail, email)
	ctx = context.WithValue(ctx, auth.ContextRole, role)
	return r.WithContext(ctx)
}

// ── listUsers ─────────────────────────────────────────────────────────────────

func TestListUsers_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	users := mock_domain.NewMockUserRepository(ctrl)
	users.EXPECT().GetAll(gomock.Any()).Return([]*domain.User{
		{Email: "a@a.com", Role: "doctor"},
	}, nil)

	c := newTestUsersCtrl(users)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	rr := httptest.NewRecorder()
	c.listUsers(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "a@a.com") {
		t.Errorf("response missing user: %s", rr.Body.String())
	}
}

func TestListUsers_DBError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	users := mock_domain.NewMockUserRepository(ctrl)
	users.EXPECT().GetAll(gomock.Any()).Return(nil, errors.New("db error"))

	c := newTestUsersCtrl(users)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	rr := httptest.NewRecorder()
	c.listUsers(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", rr.Code)
	}
}

// ── createUser ────────────────────────────────────────────────────────────────

func TestCreateUser_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	users := mock_domain.NewMockUserRepository(ctrl)
	users.EXPECT().Save(gomock.Any(), "new@example.com", gomock.Any()).Return(nil)

	c := newTestUsersCtrl(users)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users",
		strings.NewReader(`{"email":"new@example.com","password":"password123","role":"doctor"}`))
	req = ctxWithRole(req, "admin@example.com", domain.RoleAdmin)
	rr := httptest.NewRecorder()
	c.createUser(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestCreateUser_WithNonDefaultRole(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	users := mock_domain.NewMockUserRepository(ctrl)
	users.EXPECT().Save(gomock.Any(), "r@example.com", gomock.Any()).Return(nil)
	users.EXPECT().UpdateRole(gomock.Any(), "r@example.com", domain.RoleResearcher).Return(nil)

	c := newTestUsersCtrl(users)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users",
		strings.NewReader(`{"email":"r@example.com","password":"password123","role":"researcher"}`))
	req = ctxWithRole(req, "admin@example.com", domain.RoleAdmin)
	rr := httptest.NewRecorder()
	c.createUser(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestCreateUser_ShortPassword(t *testing.T) {
	c := newTestUsersCtrl(nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users",
		strings.NewReader(`{"email":"x@x.com","password":"short"}`))
	rr := httptest.NewRecorder()
	c.createUser(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rr.Code)
	}
}

func TestCreateUser_InvalidJSON(t *testing.T) {
	c := newTestUsersCtrl(nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users",
		strings.NewReader(`{bad`))
	rr := httptest.NewRecorder()
	c.createUser(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rr.Code)
	}
}

func TestCreateUser_DBError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	users := mock_domain.NewMockUserRepository(ctrl)
	users.EXPECT().Save(gomock.Any(), "x@x.com", gomock.Any()).Return(errors.New("db error"))

	c := newTestUsersCtrl(users)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users",
		strings.NewReader(`{"email":"x@x.com","password":"password123"}`))
	rr := httptest.NewRecorder()
	c.createUser(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", rr.Code)
	}
}

// ── updateUser ────────────────────────────────────────────────────────────────

func TestUpdateUser_Deactivate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	users := mock_domain.NewMockUserRepository(ctrl)
	users.EXPECT().Deactivate(gomock.Any(), "target@example.com").Return(nil)

	c := newTestUsersCtrl(users)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/target@example.com",
		strings.NewReader(`{"deactivate":true}`))
	req = ctxWithRole(req, "admin@example.com", domain.RoleAdmin)
	rr := httptest.NewRecorder()
	c.updateUser(rr, req, "target@example.com")
	if rr.Code != http.StatusNoContent {
		t.Fatalf("want 204, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestUpdateUser_ChangeRole(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	users := mock_domain.NewMockUserRepository(ctrl)
	users.EXPECT().UpdateRole(gomock.Any(), "target@example.com", domain.RoleAuditor).Return(nil)

	c := newTestUsersCtrl(users)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/target@example.com",
		strings.NewReader(`{"role":"auditor"}`))
	req = ctxWithRole(req, "admin@example.com", domain.RoleAdmin)
	rr := httptest.NewRecorder()
	c.updateUser(rr, req, "target@example.com")
	if rr.Code != http.StatusNoContent {
		t.Fatalf("want 204, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestUpdateUser_InvalidRole(t *testing.T) {
	c := newTestUsersCtrl(nil)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/x@x.com",
		strings.NewReader(`{"role":"superuser"}`))
	req = ctxWithRole(req, "admin@example.com", domain.RoleAdmin)
	rr := httptest.NewRecorder()
	c.updateUser(rr, req, "x@x.com")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rr.Code)
	}
}

func TestUpdateUser_InvalidJSON(t *testing.T) {
	c := newTestUsersCtrl(nil)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/x@x.com",
		strings.NewReader(`{bad`))
	rr := httptest.NewRecorder()
	c.updateUser(rr, req, "x@x.com")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rr.Code)
	}
}

func TestHandleItem_MethodNotAllowed(t *testing.T) {
	c := newTestUsersCtrl(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/x@x.com", nil)
	rr := httptest.NewRecorder()
	c.handleItem(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("want 405, got %d", rr.Code)
	}
}

func TestHandleItem_MissingEmail(t *testing.T) {
	c := newTestUsersCtrl(nil)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/", nil)
	rr := httptest.NewRecorder()
	c.handleItem(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rr.Code)
	}
}

func TestIsValidRole(t *testing.T) {
	valid := []string{domain.RoleAdmin, domain.RoleDoctor, domain.RoleResearcher, domain.RoleAuditor}
	for _, r := range valid {
		if !isValidRole(r) {
			t.Errorf("isValidRole(%q) = false, want true", r)
		}
	}
	if isValidRole("superuser") {
		t.Error("isValidRole(superuser) = true, want false")
	}
}
