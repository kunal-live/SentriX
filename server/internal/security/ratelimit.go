package security

import (
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type client struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type RateLimiter struct {
	mu      sync.Mutex
	rate    rate.Limit
	burst   int
	clients map[string]*client
}

func NewRateLimiter(perSecond float64, burst int) *RateLimiter {
	rl := &RateLimiter{
		rate:    rate.Limit(perSecond),
		burst:   burst,
		clients: make(map[string]*client),
	}

	go rl.cleanupLoop()

	return rl
}

func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()

		rl.mu.Lock()

		for key, c := range rl.clients {
			if now.Sub(c.lastSeen) > 10*time.Minute {
				delete(rl.clients, key)
			}
		}

		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	c, ok := rl.clients[key]
	if !ok {
		c = &client{
			limiter: rate.NewLimiter(rl.rate, rl.burst),
		}
		rl.clients[key] = c
	}

	c.lastSeen = time.Now()

	return c.limiter.Allow()
}

func HTTPRateLimit(
	limiter *RateLimiter,
	keyFunc func(*http.Request) string,
	message string,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFunc(r)

			if !limiter.Allow(key) {
				w.Header().Set("Retry-After", "60")
				http.Error(w, message, http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func IPKey(r *http.Request) string {
	return ClientIP(r)
}

func ClientIP(r *http.Request) string {
	remoteHost, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		remoteHost = r.RemoteAddr
	}

	// Only honor X-Forwarded-For if request came from a trusted reverse proxy
	if !IsTrustedProxy(remoteHost) {
		return remoteHost
	}

	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded == "" {
		return remoteHost
	}

	parts := strings.Split(forwarded, ",")
	// Parse right to left: the client IP is the rightmost IP that is NOT a trusted proxy.
	// This prevents attackers from injecting arbitrary spoofed IPs at the beginning of the chain.
	var candidate string
	for i := len(parts) - 1; i >= 0; i-- {
		ipStr := strings.TrimSpace(parts[i])
		if ipStr == "" || net.ParseIP(ipStr) == nil {
			continue
		}
		if !IsTrustedProxy(ipStr) {
			return ipStr
		}
		candidate = ipStr
	}

	if candidate != "" {
		return candidate
	}

	return remoteHost
}

func IsTrustedProxy(remoteHost string) bool {
	ip := net.ParseIP(remoteHost)
	if ip == nil {
		return false
	}

	// Loopback is trusted only if explicitly requested via env or configured in trusted CIDRs
	if ip.IsLoopback() && os.Getenv("SENTRIX_TRUST_LOOPBACK_PROXY") == "true" {
		return true
	}

	trustedEnv := strings.TrimSpace(os.Getenv("SENTRIX_TRUSTED_PROXIES"))
	if trustedEnv == "" {
		trustedEnv = strings.TrimSpace(os.Getenv("TRUSTED_PROXIES"))
	}
	if trustedEnv == "" {
		return false
	}

	for _, entry := range strings.Split(trustedEnv, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		if _, cidr, err := net.ParseCIDR(entry); err == nil {
			if cidr.Contains(ip) {
				return true
			}
		} else if directIP := net.ParseIP(entry); directIP != nil {
			if directIP.Equal(ip) {
				return true
			}
		}
	}

	return false
}
