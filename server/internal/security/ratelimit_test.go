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

	t.Run("loopback reverse proxy forwards client IP", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/auth/login", nil)
		req.RemoteAddr = "127.0.0.1:8080"
		req.Header.Set("X-Forwarded-For", "198.51.100.42, 10.0.0.1")

		ip := ClientIP(req)
		if ip != "198.51.100.42" {
			t.Errorf("expected forwarded client IP '198.51.100.42', got '%s'", ip)
		}
	})

	t.Run("configured trusted proxy subnet forwards client IP", func(t *testing.T) {
		os.Setenv("TRUSTED_PROXIES", "10.0.0.0/8,172.16.0.0/12")
		defer os.Unsetenv("TRUSTED_PROXIES")

		req := httptest.NewRequest("GET", "/api/v1/auth/login", nil)
		req.RemoteAddr = "10.0.5.20:41234"
		req.Header.Set("X-Forwarded-For", "203.0.113.50")

		ip := ClientIP(req)
		if ip != "203.0.113.50" {
			t.Errorf("expected forwarded client IP '203.0.113.50', got '%s'", ip)
		}
	})
}
