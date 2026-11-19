package security

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestCORSMiddleware(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	handler := CORSMiddleware(nextHandler)

	t.Run("same-origin or non-CORS request passes through", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/servers", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
		if rec.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Errorf("expected no CORS header for non-CORS request")
		}
	})

	t.Run("localhost dev origin allowed in non-production", func(t *testing.T) {
		os.Setenv("SENTRIX_ENV", "development")
		defer os.Unsetenv("SENTRIX_ENV")

		req := httptest.NewRequest("GET", "/api/v1/servers", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
		if rec.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
			t.Errorf("expected allow origin http://localhost:3000, got %s", rec.Header().Get("Access-Control-Allow-Origin"))
		}
		if rec.Header().Get("Access-Control-Allow-Credentials") != "true" {
			t.Errorf("expected Access-Control-Allow-Credentials: true")
		}
	})

	t.Run("unauthorized origin preflight rejected with 403 in production", func(t *testing.T) {
		os.Setenv("SENTRIX_ENV", "production")
		os.Setenv("WEB_ORIGIN", "https://dashboard.sentrix.io")
		defer os.Unsetenv("SENTRIX_ENV")
		defer os.Unsetenv("WEB_ORIGIN")

		prodHandler := CORSMiddleware(nextHandler)

		req := httptest.NewRequest("OPTIONS", "/api/v1/servers", nil)
		req.Header.Set("Origin", "https://attacker.com")
		rec := httptest.NewRecorder()

		prodHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden for unauthorized origin preflight, got %d", rec.Code)
		}
	})

	t.Run("allowlisted origin in production gets CORS headers and 204 on OPTIONS", func(t *testing.T) {
		os.Setenv("SENTRIX_ENV", "production")
		os.Setenv("WEB_ORIGIN", "https://dashboard.sentrix.io")
		defer os.Unsetenv("SENTRIX_ENV")
		defer os.Unsetenv("WEB_ORIGIN")

		prodHandler := CORSMiddleware(nextHandler)

		req := httptest.NewRequest("OPTIONS", "/api/v1/servers", nil)
		req.Header.Set("Origin", "https://dashboard.sentrix.io")
		rec := httptest.NewRecorder()

		prodHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Errorf("expected 204 No Content, got %d", rec.Code)
		}
		if rec.Header().Get("Access-Control-Allow-Origin") != "https://dashboard.sentrix.io" {
			t.Errorf("expected allow origin https://dashboard.sentrix.io, got %s", rec.Header().Get("Access-Control-Allow-Origin"))
		}
		if rec.Header().Get("Access-Control-Allow-Credentials") != "true" {
			t.Errorf("expected Access-Control-Allow-Credentials: true")
		}
	})
}
