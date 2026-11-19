package notifications

import (
	"testing"
)

func TestSSRFPrevention(t *testing.T) {
	blockedURLs := []string{
		"http://169.254.169.254/latest/meta-data/", // AWS/OpenStack metadata
		"http://127.0.0.1:8080/internal",           // IPv4 Loopback
		"http://localhost:5432/admin",              // Localhost
		"http://[::1]/secret",                      // IPv6 Loopback
		"http://10.0.0.5/api/vault",                // Private Class A
		"http://192.168.1.1/admin",                 // Private Class C
		"http://172.16.0.1/status",                 // Private Class B
		"file:///etc/passwd",                       // Arbitrary scheme
		"ftp://internal.backup/files",              // Non-HTTP scheme
		"http://metadata.google.internal/compute",  // GCP metadata
	}

	for _, rawURL := range blockedURLs {
		_, err := ValidateSafeWebhookURL(rawURL)
		if err == nil {
			t.Errorf("Expected SSRF protection to block %s, but validation succeeded", rawURL)
		}
	}
}

func TestMaskWebhookURL(t *testing.T) {
	raw := "https://example.com/api/v1/webhook/receiver/channel0001/token987654321?secret=supersecret123"
	masked := maskWebhookURL(raw)

	if masked == raw {
		t.Errorf("Expected URL to be masked, got original %s", masked)
	}

	if contains(masked, "supersecret123") {
		t.Errorf("Masked URL leaked query parameter token: %s", masked)
	}

	if contains(masked, "token987654321") {
		t.Errorf("Masked URL leaked path token: %s", masked)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) > 0 && findSubstr(s, substr))
}

func findSubstr(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
