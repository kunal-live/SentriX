package security

import (
	"net/http"
	"net/url"
	"os"
	"strings"
)

// CORSMiddleware restricts Cross-Origin Resource Sharing (CORS) strictly to authorized origins.
// In production, wildcard '*' is forbidden, and credentials (cookies/auth) are only shared with allowlisted origins.
func CORSMiddleware(next http.Handler) http.Handler {
	allowedOrigins := parseAllowedOrigins()
	isProd := isProduction()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Not a cross-origin request
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}

		if isOriginAllowed(origin, r.Host, allowedOrigins, isProd) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, Accept, Origin")
			w.Header().Set("Access-Control-Max-Age", "86400")
			w.Header().Set("Vary", "Origin")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		} else if r.Method == http.MethodOptions {
			// Explicitly reject preflight request from unauthorized cross-origin
			http.Error(w, "CORS origin forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func parseAllowedOrigins() map[string]bool {
	allowed := make(map[string]bool)

	// Read primary WEB_ORIGIN
	if origin := strings.TrimSpace(os.Getenv("WEB_ORIGIN")); origin != "" {
		for _, o := range strings.Split(origin, ",") {
			trimmed := strings.TrimRight(strings.TrimSpace(o), "/")
			if trimmed != "" {
				allowed[trimmed] = true
			}
		}
	}

	// Read comma-separated CORS_ALLOWED_ORIGINS
	if extra := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS")); extra != "" {
		for _, o := range strings.Split(extra, ",") {
			trimmed := strings.TrimRight(strings.TrimSpace(o), "/")
			if trimmed != "" {
				allowed[trimmed] = true
			}
		}
	}

	return allowed
}

func isOriginAllowed(origin string, requestHost string, allowedOrigins map[string]bool, isProd bool) bool {
	originTrimmed := strings.TrimRight(strings.TrimSpace(origin), "/")

	// Direct allowlist match
	if allowedOrigins[originTrimmed] {
		return true
	}

	parsed, err := url.Parse(originTrimmed)
	if err != nil {
		return false
	}

	// Check if origin host matches request Host (same host, different port/scheme)
	if requestHost != "" && parsed.Host == requestHost {
		return true
	}

	// In non-production only, allow standard local dev servers
	if !isProd {
		hostOnly := parsed.Hostname()
		if hostOnly == "localhost" || hostOnly == "127.0.0.1" {
			return true
		}
	}

	return false
}

func isProduction() bool {
	env := strings.ToLower(os.Getenv("SENTRIX_ENV"))
	if env == "" {
		env = strings.ToLower(os.Getenv("ENV"))
	}
	return env == "production" || env == "prod"
}
