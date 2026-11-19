package notifications

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sentrix/server/internal/auth"
)

func isDisallowedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	// IPv4 & IPv6 Loopback (127.0.0.0/8, ::1)
	if ip.IsLoopback() {
		return true
	}
	// Cloud metadata (169.254.169.254) & Link-local (169.254.0.0/16, fe80::/10)
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	// Private network ranges (10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16, fc00::/7)
	if ip.IsPrivate() {
		return true
	}
	// Unspecified (0.0.0.0, ::)
	if ip.IsUnspecified() {
		return true
	}
	return false
}

// ValidateSafeWebhookURL validates that a URL is safe from SSRF attacks (blocks private IPs, loopback, cloud metadata)
func ValidateSafeWebhookURL(rawURL string) (*url.URL, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, errors.New("invalid webhook URL")
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, errors.New("webhook URL scheme must be http or https")
	}

	hostname := parsedURL.Hostname()
	if hostname == "" {
		return nil, errors.New("webhook URL must include a valid hostname")
	}

	// Reject raw IP representations of cloud metadata directly
	if hostname == "169.254.169.254" || hostname == "metadata.google.internal" || hostname == "localhost" {
		return nil, errors.New("SSRF protection: prohibited metadata or localhost destination")
	}

	// Resolve IPs and verify none are private / loopback / metadata
	ips, err := net.LookupIP(hostname)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve webhook host: %w", err)
	}

	if len(ips) == 0 {
		return nil, errors.New("webhook host could not be resolved to any IP address")
	}

	for _, ip := range ips {
		if isDisallowedIP(ip) {
			return nil, fmt.Errorf("SSRF protection: webhook URL resolves to prohibited IP address %s (internal, private, or cloud metadata ranges are blocked)", ip.String())
		}
	}

	return parsedURL, nil
}

// MaskWebhookURL masks sensitive paths, keys, and tokens in webhook URLs for API responses
func MaskWebhookURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return "https://***"
	}

	// Mask query parameters (often contain access tokens or webhook keys)
	q := u.Query()
	for k := range q {
		q.Set(k, "••••••")
	}
	u.RawQuery = q.Encode()

	// Mask long path segments that typically contain secret tokens
	parts := strings.Split(u.Path, "/")
	for i, part := range parts {
		if len(part) > 10 {
			parts[i] = part[:4] + "••••••"
		}
	}
	u.Path = strings.Join(parts, "/")

	return u.String()
}

type channelListItem struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Name      string    `json:"name"`
	Enabled   bool      `json:"enabled"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
}

type createChannelRequest struct {
	Name string `json:"name"`
	Type string `json:"type"`
	URL  string `json:"url"`
}

// GET /api/v1/notifications/channels
func HandleListChannels(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := pool.Query(r.Context(), `
			SELECT
				id,
				type,
				name,
				enabled,
				COALESCE(config->>'url', '') AS url,
				created_at
			FROM notification_channels
			ORDER BY created_at DESC
		`)

		if err != nil {
			http.Error(w, "database query failed", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		channels := []channelListItem{}

		for rows.Next() {
			var channel channelListItem

			err := rows.Scan(
				&channel.ID,
				&channel.Type,
				&channel.Name,
				&channel.Enabled,
				&channel.URL,
				&channel.CreatedAt,
			)

			if err != nil {
				http.Error(w, "scan failed", http.StatusInternalServerError)
				return
			}

			// Mask webhook URL to prevent secret/token exposure in list API
			channel.URL = MaskWebhookURL(channel.URL)

			channels = append(channels, channel)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(channels)
	}
}

// POST /api/v1/notifications/channels
func HandleCreateChannel(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createChannelRequest

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		if req.Name == "" {
			http.Error(w, "name is required", http.StatusBadRequest)
			return
		}

		if req.Type == "" {
			req.Type = "WEBHOOK"
		}

		if req.Type != "WEBHOOK" {
			http.Error(w, "only WEBHOOK channels are supported in V1", http.StatusBadRequest)
			return
		}

		if req.URL == "" {
			http.Error(w, "url is required", http.StatusBadRequest)
			return
		}

		// Enforce strict SSRF protection: reject private IP ranges, loopback, and cloud metadata
		if _, err := ValidateSafeWebhookURL(req.URL); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		config := map[string]string{
			"url": req.URL,
		}

		configJSON, err := json.Marshal(config)
		if err != nil {
			http.Error(w, "invalid configuration", http.StatusInternalServerError)
			return
		}

		id := uuid.New()

		_, err = pool.Exec(r.Context(), `
			INSERT INTO notification_channels (
				id,
				type,
				name,
				enabled,
				config
			) VALUES (
				$1, $2, $3, TRUE, $4
			)
		`, id, req.Type, req.Name, configJSON)

		if err != nil {
			http.Error(w, "database insert failed", http.StatusInternalServerError)
			return
		}

		actor := auth.GetUser(r)
		var actorID *string
		if actor != nil {
			actorID = &actor.ID
		}

		auth.WriteAudit(
			r.Context(),
			pool,
			actorID,
			"notification_channel.create",
			"notification_channel",
			id.String(),
			auth.ClientIP(r),
			map[string]any{
				"name": req.Name,
				"type": req.Type,
			},
		)

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"id": id.String(),
		})
	}
}

// DELETE /api/v1/notifications/channels/{channelID}
func HandleDeleteChannel(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		channelID := chi.URLParam(r, "channelID")
		if _, err := uuid.Parse(channelID); err != nil {
			http.Error(w, "invalid channel ID format", http.StatusBadRequest)
			return
		}

		_, err := pool.Exec(r.Context(), `
			DELETE FROM notification_channels
			WHERE id = $1
		`, channelID)

		if err != nil {
			http.Error(w, "database delete failed", http.StatusInternalServerError)
			return
		}

		actor := auth.GetUser(r)
		var actorID *string
		if actor != nil {
			actorID = &actor.ID
		}

		auth.WriteAudit(
			r.Context(),
			pool,
			actorID,
			"notification_channel.delete",
			"notification_channel",
			channelID,
			auth.ClientIP(r),
			nil,
		)

		w.WriteHeader(http.StatusNoContent)
	}
}

// GET /api/v1/notifications/jobs
func HandleListJobs(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := pool.Query(r.Context(), `
			SELECT
				id,
				channel_id,
				event_type,
				status,
				attempts,
				max_attempts,
				last_error,
				next_attempt_at,
				created_at
			FROM notification_jobs
			ORDER BY created_at DESC
			LIMIT 100
		`)

		if err != nil {
			http.Error(w, "database query failed", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type jobListItem struct {
			ID            string    `json:"id"`
			ChannelID     string    `json:"channel_id"`
			EventType     string    `json:"event_type"`
			Status        string    `json:"status"`
			Attempts      int       `json:"attempts"`
			MaxAttempts   int       `json:"max_attempts"`
			LastError     *string   `json:"last_error"`
			NextAttemptAt time.Time `json:"next_attempt_at"`
			CreatedAt     time.Time `json:"created_at"`
		}

		jobs := []jobListItem{}

		for rows.Next() {
			var job jobListItem

			err := rows.Scan(
				&job.ID,
				&job.ChannelID,
				&job.EventType,
				&job.Status,
				&job.Attempts,
				&job.MaxAttempts,
				&job.LastError,
				&job.NextAttemptAt,
				&job.CreatedAt,
			)

			if err != nil {
				http.Error(w, "scan failed", http.StatusInternalServerError)
				return
			}

			jobs = append(jobs, job)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jobs)
	}
}

// POST /api/v1/notifications/channels/{channelID}/test and /notifications/channels/test
func HandleTestChannel(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		channelID := chi.URLParam(r, "channelID")
		var rawURL string

		if channelID != "" {
			if _, err := uuid.Parse(channelID); err != nil {
				http.Error(w, "invalid channel ID format", http.StatusBadRequest)
				return
			}
			err := pool.QueryRow(r.Context(), `
				SELECT COALESCE(config->>'url', '')
				FROM notification_channels
				WHERE id = $1
			`, channelID).Scan(&rawURL)
			if err != nil {
				http.Error(w, "channel not found", http.StatusNotFound)
				return
			}
		} else {
			var body struct {
				URL string `json:"url"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			rawURL = body.URL
			if rawURL == "" {
				http.Error(w, "channel ID or webhook URL required", http.StatusBadRequest)
				return
			}
		}

		if _, err := ValidateSafeWebhookURL(rawURL); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		testPayload := []byte(`{"text":"[SentriX Test Alert] This is a verification test ping from your SentriX cluster."}`)
		status, err := sendWebhook(r.Context(), rawURL, testPayload)
		if err != nil {
			http.Error(w, fmt.Sprintf("test webhook delivery failed: %v", err), http.StatusBadGateway)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":      "delivered",
			"http_status": status,
		})
	}
}
