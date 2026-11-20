package logs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sentrix/server/internal/api"
)

type StructuredLogEntry struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"` // ERROR, WARN, INFO, DEBUG
	Service   string    `json:"service"`
	Server    string    `json:"server"`
	ServerID  string    `json:"server_id"`
	Message   string    `json:"message"`
	TraceID   string    `json:"trace_id,omitempty"`
	RequestID string    `json:"request_id,omitempty"`
}

type BatchLogsRequest struct {
	ServerID string               `json:"server_id"`
	Logs     []StructuredLogEntry `json:"logs"`
}

var (
	logStoreMu sync.RWMutex
	logBuffer  []StructuredLogEntry
)

// RecordLog records a structured log entry into TimescaleDB/PostgreSQL system_logs table
// and preserves the latest entries in memory for instant retrieval.
func RecordLog(ctx context.Context, pool *pgxpool.Pool, entry StructuredLogEntry) {
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}
	if entry.Level == "" {
		entry.Level = "INFO"
	}
	if entry.Service == "" {
		entry.Service = "sentrix-server"
	}
	if entry.Server == "" {
		entry.Server = "sentrix-core"
	}

	// Update memory ring buffer
	logStoreMu.Lock()
	logBuffer = append([]StructuredLogEntry{entry}, logBuffer...)
	if len(logBuffer) > 2000 {
		logBuffer = logBuffer[:2000]
	}
	logStoreMu.Unlock()

	// Persist to database if pool is live
	if pool != nil {
		var serverIDParam any
		if entry.ServerID != "" {
			if _, err := uuid.Parse(entry.ServerID); err == nil {
				serverIDParam = entry.ServerID
			}
		}

		var traceParam any
		if entry.TraceID != "" {
			traceParam = entry.TraceID
		}

		var reqParam any
		if entry.RequestID != "" {
			reqParam = entry.RequestID
		}

		_, _ = pool.Exec(ctx, `
			INSERT INTO system_logs (id, timestamp, level, service, server, server_id, message, trace_id, request_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, entry.ID, entry.Timestamp, entry.Level, entry.Service, entry.Server, serverIDParam, entry.Message, traceParam, reqParam)
	}
}

// GET /api/v1/logs
func HandleSearchLogs(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := strings.TrimSpace(r.URL.Query().Get("query"))
		level := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("level")))
		service := strings.TrimSpace(r.URL.Query().Get("service"))
		serverID := strings.TrimSpace(r.URL.Query().Get("server_id"))
		traceID := strings.TrimSpace(r.URL.Query().Get("trace_id"))
		limitParam := r.URL.Query().Get("limit")

		limit := 100
		if limitParam != "" {
			if parsed, err := strconv.Atoi(limitParam); err == nil && parsed > 0 && parsed <= 500 {
				limit = parsed
			}
		}

		// When PostgreSQL pool is available, query real persistent system logs
		if pool != nil {
			ctx := r.Context()
			var whereClauses []string
			var args []any
			argIdx := 1

			if level != "" && level != "ALL" {
				whereClauses = append(whereClauses, fmt.Sprintf("level = $%d", argIdx))
				args = append(args, level)
				argIdx++
			}

			if service != "" {
				whereClauses = append(whereClauses, fmt.Sprintf("service ILIKE $%d", argIdx))
				args = append(args, "%"+service+"%")
				argIdx++
			}

			if serverID != "" {
				if _, err := uuid.Parse(serverID); err == nil {
					whereClauses = append(whereClauses, fmt.Sprintf("server_id = $%d", argIdx))
					args = append(args, serverID)
					argIdx++
				}
			}

			if traceID != "" {
				whereClauses = append(whereClauses, fmt.Sprintf("trace_id = $%d", argIdx))
				args = append(args, traceID)
				argIdx++
			}

			if query != "" {
				whereClauses = append(whereClauses, fmt.Sprintf("(message ILIKE $%d OR service ILIKE $%d OR server ILIKE $%d)", argIdx, argIdx, argIdx))
				args = append(args, "%"+query+"%")
				argIdx++
			}

			whereSQL := ""
			if len(whereClauses) > 0 {
				whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
			}

			sql := fmt.Sprintf(`
				SELECT id::text, timestamp, level, service, server, COALESCE(server_id::text, ''), message, COALESCE(trace_id, ''), COALESCE(request_id, '')
				FROM system_logs
				%s
				ORDER BY timestamp DESC
				LIMIT $%d
			`, whereSQL, argIdx)
			args = append(args, limit)

			rows, err := pool.Query(ctx, sql, args...)
			if err == nil {
				defer rows.Close()
				var results []StructuredLogEntry
				for rows.Next() {
					var item StructuredLogEntry
					if scanErr := rows.Scan(
						&item.ID,
						&item.Timestamp,
						&item.Level,
						&item.Service,
						&item.Server,
						&item.ServerID,
						&item.Message,
						&item.TraceID,
						&item.RequestID,
					); scanErr == nil {
						results = append(results, item)
					}
				}

				if results == nil {
					results = []StructuredLogEntry{}
				}

				api.RespondJSON(w, http.StatusOK, map[string]any{
					"total": len(results),
					"logs":  results,
				})
				return
			}
		}

		// Fallback to in-memory buffer if database is unavailable or query errored
		logStoreMu.RLock()
		defer logStoreMu.RUnlock()

		filtered := make([]StructuredLogEntry, 0)
		for _, item := range logBuffer {
			if level != "" && level != "ALL" && item.Level != level {
				continue
			}
			if service != "" && !strings.Contains(strings.ToLower(item.Service), strings.ToLower(service)) {
				continue
			}
			if serverID != "" && item.ServerID != serverID {
				continue
			}
			if traceID != "" && item.TraceID != traceID {
				continue
			}
			if query != "" {
				q := strings.ToLower(query)
				inMsg := strings.Contains(strings.ToLower(item.Message), q)
				inSvc := strings.Contains(strings.ToLower(item.Service), q)
				inSrv := strings.Contains(strings.ToLower(item.Server), q)
				if !inMsg && !inSvc && !inSrv {
					continue
				}
			}

			filtered = append(filtered, item)
			if len(filtered) >= limit {
				break
			}
		}

		api.RespondJSON(w, http.StatusOK, map[string]any{
			"total": len(filtered),
			"logs":  filtered,
		})
	}
}

// GET /api/v1/logs/services
func HandleGetServices(pool *pgxpool.Pool) http.HandlerFunc {
	defaultServices := []string{
		"sentrix-server",
		"sentrix-agent",
		"timescale-db",
		"telemetry-engine",
		"auth-gateway",
		"node-monitor",
		"synthetic-checker",
		"systemd",
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if pool != nil {
			rows, err := pool.Query(r.Context(), `
				SELECT DISTINCT service
				FROM system_logs
				ORDER BY service ASC
			`)
			if err == nil {
				defer rows.Close()
				var services []string
				for rows.Next() {
					var s string
					if err := rows.Scan(&s); err == nil && s != "" {
						services = append(services, s)
					}
				}
				if len(services) > 0 {
					api.RespondJSON(w, http.StatusOK, services)
					return
				}
			}
		}

		api.RespondJSON(w, http.StatusOK, defaultServices)
	}
}

// POST /api/v1/logs/batch
func HandleBatchLogs(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req BatchLogsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			api.RespondError(w, r, http.StatusBadRequest, "INVALID_BODY", "Invalid batch logs payload")
			return
		}

		if len(req.Logs) > 1000 {
			api.RespondError(w, r, http.StatusBadRequest, "BATCH_TOO_LARGE", "Maximum 1000 log records allowed per batch")
			return
		}

		ctx := r.Context()
		now := time.Now().UTC()
		for _, entry := range req.Logs {
			if entry.ID == "" {
				entry.ID = uuid.New().String()
			}
			if entry.Timestamp.IsZero() {
				entry.Timestamp = now
			}
			if entry.ServerID == "" {
				entry.ServerID = req.ServerID
			}
			RecordLog(ctx, pool, entry)
		}

		api.RespondJSON(w, http.StatusAccepted, map[string]any{
			"status":   "accepted",
			"ingested": len(req.Logs),
		})
	}
}
