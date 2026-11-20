package security

import (
	"net/http/httptest"
	"os"
	"testing"
)

func TestClientIP(t *testing.T) {
	t.Run("direct untrusted client cannot spoof X-Forwarded-For", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/auth/login", nil)
		req.RemoteAddr = "203.0.113.195:45821"
		req.Header.Set("X-Forwarded-For", "198.51.100.1")

		ip := ClientIP(req)
		if ip != "203.0.113.195" {
			t.Errorf("expected RemoteAddr host '203.0.113.195', got '%s'", ip)
		}
	})

	t.Run("loopback is not trusted by default without explicit configuration", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/auth/login", nil)
		req.RemoteAddr = "127.0.0.1:8080"
		req.Header.Set("X-Forwarded-For", "198.51.100.42")

		ip := ClientIP(req)
		if ip != "127.0.0.1" {
			t.Errorf("expected RemoteAddr '127.0.0.1' when untrusted, got '%s'", ip)
		}
	})

	t.Run("configured trusted proxy subnet forwards client IP with right-to-left parsing", func(t *testing.T) {
		os.Setenv("SENTRIX_TRUSTED_PROXIES", "10.0.0.0/8,172.16.0.0/12,127.0.0.1/32")
		defer os.Unsetenv("SENTRIX_TRUSTED_PROXIES")

		req := httptest.NewRequest("GET", "/api/v1/auth/login", nil)
		req.RemoteAddr = "10.0.5.20:41234"
		// Attacker forged 1.1.1.1, real client 203.0.113.50, internal proxy 10.0.0.1 appended
		req.Header.Set("X-Forwarded-For", "1.1.1.1, 203.0.113.50, 10.0.0.1")

		ip := ClientIP(req)
		if ip != "203.0.113.50" {
			t.Errorf("expected rightmost untrusted IP '203.0.113.50', got '%s'", ip)
		}
	})

	t.Run("loopback reverse proxy forwards client IP when explicitly enabled", func(t *testing.T) {
		os.Setenv("SENTRIX_TRUST_LOOPBACK_PROXY", "true")
		defer os.Unsetenv("SENTRIX_TRUST_LOOPBACK_PROXY")

		req := httptest.NewRequest("GET", "/api/v1/auth/login", nil)
		req.RemoteAddr = "127.0.0.1:8080"
		req.Header.Set("X-Forwarded-For", "198.51.100.42")

		ip := ClientIP(req)
		if ip != "198.51.100.42" {
			t.Errorf("expected forwarded client IP '198.51.100.42', got '%s'", ip)
		}
	})
}
