package metrics

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sentrix/server/internal/auth"
	"github.com/sentrix/server/internal/logs"
	"github.com/sentrix/server/internal/realtime"
	"github.com/sentrix/server/internal/selfmetrics"
)

type ctxKey string

const (
	agentIDKey    ctxKey = "agent_id"
	ingestRoleKey ctxKey = "ingest_role"
)

// AgentIDFromContext returns the authenticated agent ID from request context.
func AgentIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(agentIDKey).(string)
	return id, ok && id != ""
}

type telemetryLogEvent struct {
	serverID string
	seq      int64
	count    int
	ts       time.Time
}

var (
	telemetryLogOnce  sync.Once
	telemetryLogQueue = make(chan telemetryLogEvent, 256)
)

func startTelemetryLogWorker(pool *pgxpool.Pool) {
	telemetryLogOnce.Do(func() {
		go func() {
			for ev := range telemetryLogQueue {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				var srvName string
				_ = pool.QueryRow(ctx, "SELECT COALESCE(name, hostname) FROM servers WHERE id = $1", ev.serverID).Scan(&srvName)
				if srvName == "" && len(ev.serverID) >= 8 {
					srvName = "node-" + ev.serverID[:8]
				}
				logs.RecordLog(ctx, pool, logs.StructuredLogEntry{
					Timestamp: ev.ts,
					Level:     "INFO",
					Service:   "telemetry-engine",
					Server:    srvName,
					ServerID:  ev.serverID,
					Message:   fmt.Sprintf("Ingested %d metric datapoints (sequence %d) from %s into TimescaleDB.", ev.count, ev.seq, srvName),
				})
				cancel()
			}
		}()
	})
}

func hashCredential(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return base64.StdEncoding.EncodeToString(sum[:])
}

// AgentAuthMiddleware validates agent bearer token.
func AgentAuthMiddleware(pool *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if len(authHeader) < 7 || !strings.EqualFold(authHeader[:7], "Bearer ") {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			token := strings.TrimSpace(authHeader[7:])
			hashed := hashCredential(token)

			var agentID string
			err := pool.QueryRow(r.Context(), "SELECT id FROM agents WHERE credential_hash = $1 AND status = 'ACTIVE'", hashed).Scan(&agentID)
			if err != nil {
				http.Error(w, "Invalid credential", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), agentIDKey, agentID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// IngestAuthMiddleware accepts either an active agent bearer token or a valid user JWT.
func IngestAuthMiddleware(pool *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if len(authHeader) < 7 || !strings.EqualFold(authHeader[:7], "Bearer ") {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			token := strings.TrimSpace(authHeader[7:])

			// 1. Check user JWT — require OPERATOR or ADMIN for ingestion
			if claims, err := auth.ValidateAccessToken(token); err == nil && claims != nil {
				if claims.Role != "ADMIN" && claims.Role != "OPERATOR" {
					http.Error(w, "Forbidden: ingestion requires OPERATOR role or above", http.StatusForbidden)
					return
				}
				ctx := context.WithValue(r.Context(), ingestRoleKey, claims.Role)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// 2. Check active agent credential
			if pool != nil {
				hashed := hashCredential(token)
				var agentID string
				if err := pool.QueryRow(r.Context(), "SELECT id FROM agents WHERE credential_hash = $1 AND status = 'ACTIVE'", hashed).Scan(&agentID); err == nil {
					ctx := context.WithValue(r.Context(), agentIDKey, agentID)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			http.Error(w, "Invalid credential", http.StatusUnauthorized)
		})
	}
}

type TelemetryPayload struct {
	AgentID      string    `json:"agent_id"`
	Timestamp    time.Time `json:"timestamp"`
	Sequence     int64     `json:"sequence"`
	RestartNonce string    `json:"restart_nonce,omitempty"`
	Metrics      []Metric  `json:"metrics"`
}

type Metric struct {
	Name   string            `json:"name"`
	Value  float64           `json:"value"`
	Unit   string            `json:"unit"`
	Labels map[string]string `json:"labels,omitempty"`
}

// POST /api/v1/agent/telemetry
func HandleTelemetry(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agentID, ok := AgentIDFromContext(r.Context())
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		selfmetrics.Global().IngestRequests.Add(1)

		var payload TelemetryPayload

		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			selfmetrics.Global().IngestFailures.Add(1)
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		if len(payload.Metrics) == 0 {
			selfmetrics.Global().IngestFailures.Add(1)
			http.Error(w, "no metrics provided", http.StatusBadRequest)
			return
		}

		if len(payload.Metrics) > 1000 {
			selfmetrics.Global().IngestFailures.Add(1)
			http.Error(w, "too many metrics in payload", http.StatusBadRequest)
			return
		}

		now := time.Now()

		if payload.Timestamp.IsZero() {
			payload.Timestamp = now
		}

		// Allow historical spooled telemetry up to 7 days old, and tolerate up to 5 minutes forward clock drift.
		if payload.Timestamp.After(now.Add(5*time.Minute)) || payload.Timestamp.Before(now.Add(-7*24*time.Hour)) {
			selfmetrics.Global().IngestFailures.Add(1)
			http.Error(w, "timestamp outside acceptance window", http.StatusBadRequest)
			return
		}

		if payload.Sequence < 0 {
			selfmetrics.Global().IngestFailures.Add(1)
			http.Error(w, "invalid sequence number", http.StatusBadRequest)
			return
		}

		ctx := r.Context()

		tx, err := pool.Begin(ctx)
		if err != nil {
			selfmetrics.Global().IngestFailures.Add(1)
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback(ctx)

		var lastSequence int64
		var lastTelemetryAt *time.Time
		var lastNonce string

		err = tx.QueryRow(ctx, `
			SELECT last_sequence, last_telemetry_at, COALESCE(last_restart_nonce, '')
			FROM agents
			WHERE id = $1
			FOR UPDATE
		`, agentID).Scan(&lastSequence, &lastTelemetryAt, &lastNonce)

		if err != nil {
			selfmetrics.Global().IngestFailures.Add(1)
			http.Error(w, "invalid agent", http.StatusUnauthorized)
			return
		}

		// Replay / duplicate / sequence verification.
		if payload.Sequence <= lastSequence {
			// Exact duplicate is accepted idempotently but not reprocessed.
			if payload.Sequence == lastSequence &&
				lastTelemetryAt != nil &&
				payload.Timestamp.Equal(*lastTelemetryAt) {

				w.WriteHeader(http.StatusAccepted)
				w.Write([]byte(`{"status":"duplicate"}`))
				return
			}

			// Allow agent restart sequence reset strictly at sequence == 1 when moving forward in time
			// and presenting a fresh, non-empty restart_nonce distinct from last_restart_nonce.
			allowReset := (lastTelemetryAt == nil) ||
				(payload.Sequence == 1 && payload.Timestamp.After(*lastTelemetryAt) && payload.RestartNonce != "" && payload.RestartNonce != lastNonce)

			if !allowReset {
				selfmetrics.Global().IngestFailures.Add(1)
				http.Error(w, "stale or replayed sequence", http.StatusConflict)
				return
			}
		}

		_, err = tx.Exec(ctx, `
			UPDATE agents
			SET
				last_sequence = $2,
				last_telemetry_at = $3,
				last_restart_nonce = COALESCE(NULLIF($4, ''), last_restart_nonce)
			WHERE id = $1
		`, agentID, payload.Sequence, payload.Timestamp, payload.RestartNonce)

		if err != nil {
			selfmetrics.Global().IngestFailures.Add(1)
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		var serverID string

		err = tx.QueryRow(ctx, `
			SELECT id
			FROM servers
			WHERE agent_id = $1
		`, agentID).Scan(&serverID)

		if err != nil {
			selfmetrics.Global().IngestFailures.Add(1)
			http.Error(w, "server not found for agent", http.StatusNotFound)
			return
		}

		batch := &pgx.Batch{}

		for _, m := range payload.Metrics {
			val := m.Value
			if math.IsNaN(val) || math.IsInf(val, 0) {
				continue
			}

			var tableName string

			switch m.Name {
			case "system.cpu.utilization":
				tableName = "metric_cpu"
			case "system.memory.utilization":
				tableName = "metric_memory"
			case "system.disk.utilization":
				tableName = "metric_disk"
			default:
				continue
			}

			if val < 0 {
				val = 0
			} else if val > 100 {
				val = 100
			}

			query := "INSERT INTO " + tableName + " (time, server_id, value) VALUES ($1, $2, $3)"
			batch.Queue(query, payload.Timestamp, serverID, val)
		}

		batch.Queue(`
			UPDATE servers
			SET
				last_seen_at = $1,
				status = 'ONLINE'
			WHERE id = $2
		`, now, serverID)

		br := tx.SendBatch(ctx, batch)

		if _, err := br.Exec(); err != nil {
			br.Close()
			selfmetrics.Global().IngestFailures.Add(1)
			http.Error(w, "database write error", http.StatusInternalServerError)
			return
		}

		br.Close()

		if err := tx.Commit(ctx); err != nil {
			selfmetrics.Global().IngestFailures.Add(1)
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		selfmetrics.Global().IngestSamples.Add(int64(len(payload.Metrics)))

		// Record live telemetry ingestion in system_logs periodically via bounded single worker
		if payload.Sequence%3 == 0 || payload.Sequence <= 2 {
			startTelemetryLogWorker(pool)
			select {
			case telemetryLogQueue <- telemetryLogEvent{
				serverID: serverID,
				seq:      payload.Sequence,
				count:    len(payload.Metrics),
				ts:       now,
			}:
			default:
			}
		}

		realtime.PublishServerUpdated(serverID)
		realtime.PublishServerTelemetry(serverID)
		realtime.PublishDashboardUpdated()

		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"status":"accepted"}`))
	}
}
