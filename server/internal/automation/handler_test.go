package automation

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/sentrix/server/internal/auth"
)

func TestHandleExecuteRunbookEnforcesRoleAndAllowedRoles(t *testing.T) {
	auth.InitJWT()

	r := chi.NewRouter()
	r.With(auth.RequireAuth).Post("/runbooks/{id}/execute", HandleExecuteRunbook(nil))

	viewerToken, _, err := auth.GenerateAccessToken("u1", "viewer@sentrix.local", "VIEWER")
	if err != nil {
		t.Fatalf("GenerateAccessToken viewer: %v", err)
	}
	opToken, _, err := auth.GenerateAccessToken("u2", "op@sentrix.local", "OPERATOR")
	if err != nil {
		t.Fatalf("GenerateAccessToken operator: %v", err)
	}

	// Unauthenticated -> 401
	req := httptest.NewRequest(http.MethodPost, "/runbooks/rb-disk-cleanup/execute", strings.NewReader(`{"approved":true}`))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated: expected 401, got %d", rr.Code)
	}

	// VIEWER -> 403 even with approved: true
	reqViewer := httptest.NewRequest(http.MethodPost, "/runbooks/rb-disk-cleanup/execute", strings.NewReader(`{"approved":true}`))
	reqViewer.Header.Set("Authorization", "Bearer "+viewerToken)
	rrViewer := httptest.NewRecorder()
	r.ServeHTTP(rrViewer, reqViewer)
	if rrViewer.Code != http.StatusForbidden {
		t.Fatalf("VIEWER: expected 403, got %d", rrViewer.Code)
	}

	// OPERATOR on rb-disk-cleanup (AllowedRoles: ["ADMIN"]) -> 403
	opReq := httptest.NewRequest(http.MethodPost, "/runbooks/rb-disk-cleanup/execute", strings.NewReader(`{"approved":true}`))
	opReq.Header.Set("Authorization", "Bearer "+opToken)
	rrOp := httptest.NewRecorder()
	r.ServeHTTP(rrOp, opReq)
	if rrOp.Code != http.StatusForbidden {
		t.Fatalf("OPERATOR on ADMIN-only runbook: expected 403, got %d", rrOp.Code)
	}

	// OPERATOR on rb-agent-flush (AllowedRoles: ["ADMIN", "OPERATOR"]) -> 202
	opAllowedReq := httptest.NewRequest(http.MethodPost, "/runbooks/rb-agent-flush/execute", strings.NewReader(`{"approved":true}`))
	opAllowedReq.Header.Set("Authorization", "Bearer "+opToken)
	rrOpAllowed := httptest.NewRecorder()
	r.ServeHTTP(rrOpAllowed, opAllowedReq)
	if rrOpAllowed.Code != http.StatusAccepted {
		t.Fatalf("OPERATOR on allowed runbook: expected 202, got %d", rrOpAllowed.Code)
	}
}
