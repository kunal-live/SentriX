package metrics

import (
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sentrix/server/internal/auth"
)

func TestHashCredential(t *testing.T) {
	raw := "stx-agent-credential-secret-key-12345"
	expectedSum := sha256.Sum256([]byte(raw))
	expected := base64.StdEncoding.EncodeToString(expectedSum[:])

	result := hashCredential(raw)
	if result != expected {
		t.Fatalf("hashCredential failed: expected %s, got %s", expected, result)
	}
}

func TestTelemetryPayloadStructure(t *testing.T) {
	payload := TelemetryPayload{
		AgentID:   "agt-001",
		Timestamp: time.Now(),
		Sequence:  42,
		Metrics: []Metric{
			{Name: "cpu_usage", Value: 45.2, Unit: "percent", Labels: map[string]string{"core": "0"}},
			{Name: "memory_usage", Value: 72.1, Unit: "percent"},
		},
	}

	if len(payload.Metrics) != 2 {
		t.Fatalf("Expected 2 metrics, got %d", len(payload.Metrics))
	}
	if payload.Metrics[0].Name != "cpu_usage" {
		t.Errorf("Expected cpu_usage, got %s", payload.Metrics[0].Name)
	}
	if payload.Metrics[0].Labels["core"] != "0" {
		t.Errorf("Expected core 0 label, got %s", payload.Metrics[0].Labels["core"])
	}
}

func TestIngestAuthMiddlewareRoleCheck(t *testing.T) {
	t.Setenv("SENTRIX_DEMO_MODE", "true")
	auth.InitJWT()

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})
	handler := IngestAuthMiddleware(nil)(next)

	cases := []struct {
		name     string
		role     string
		wantCode int
	}{
		{"viewer_blocked", "VIEWER", http.StatusForbidden},
		{"operator_allowed", "OPERATOR", http.StatusAccepted},
		{"admin_allowed", "ADMIN", http.StatusAccepted},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tok, _, err := auth.GenerateAccessToken("usr-1", "test@sentrix.local", tc.role)
			if err != nil {
				t.Fatalf("GenerateAccessToken failed: %v", err)
			}
			req := httptest.NewRequest(http.MethodPost, "/api/v1/kubernetes/ingest", nil)
			req.Header.Set("Authorization", "Bearer "+tok)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			if rr.Code != tc.wantCode {
				t.Errorf("role %s: got %d, want %d", tc.role, rr.Code, tc.wantCode)
			}
		})
	}
}
