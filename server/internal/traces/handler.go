package traces

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sentrix/server/internal/api"
)

type SpanEvent struct {
	Name       string         `json:"name"`
	Timestamp  time.Time      `json:"timestamp"`
	Attributes map[string]any `json:"attributes,omitempty"`
}

type Span struct {
	SpanID       string         `json:"span_id"`
	ParentSpanID *string        `json:"parent_span_id,omitempty"`
	TraceID      string         `json:"trace_id"`
	ServiceName  string         `json:"service_name"`
	Operation    string         `json:"operation"`
	DurationMs   float64        `json:"duration_ms"`
	StartTime    time.Time      `json:"start_time"`
	Status       string         `json:"status"` // OK, ERROR
	StatusCode   int            `json:"status_code,omitempty"`
	Attributes   map[string]any `json:"attributes,omitempty"`
	Events       []SpanEvent    `json:"events,omitempty"`
}

type TraceDetail struct {
	TraceID       string    `json:"trace_id"`
	RootService   string    `json:"root_service"`
	RootOperation string    `json:"root_operation"`
	DurationMs    float64   `json:"duration_ms"`
	StartTime     time.Time `json:"start_time"`
	Status        string    `json:"status"`
	SpanCount     int       `json:"span_count"`
	Spans         []Span    `json:"spans"`
}

// OTLP / OpenTelemetry JSON specification models
type otlpAttributeValue struct {
	StringValue *string  `json:"stringValue,omitempty"`
	IntValue    *any     `json:"intValue,omitempty"`
	DoubleValue *float64 `json:"doubleValue,omitempty"`
	BoolValue   *bool    `json:"boolValue,omitempty"`
}

func (v otlpAttributeValue) ToValue() any {
	if v.StringValue != nil {
		return *v.StringValue
	}
	if v.IntValue != nil {
		switch val := (*v.IntValue).(type) {
		case string:
			if n, err := strconv.ParseInt(val, 10, 64); err == nil {
				return n
			}
			return val
		case float64:
			return int64(val)
		default:
			return val
		}
	}
	if v.DoubleValue != nil {
		return *v.DoubleValue
	}
	if v.BoolValue != nil {
		return *v.BoolValue
	}
	return nil
}

type otlpAttribute struct {
	Key   string             `json:"key"`
	Value otlpAttributeValue `json:"value"`
}

type otlpSpanStatus struct {
	Code    int    `json:"code,omitempty"` // 0: UNSET, 1: OK, 2: ERROR
	Message string `json:"message,omitempty"`
}

type otlpSpan struct {
	TraceID           string          `json:"traceId"`
	SpanID            string          `json:"spanId"`
	ParentSpanID      string          `json:"parentSpanId,omitempty"`
	Name              string          `json:"name"`
	Kind              int             `json:"kind,omitempty"`
	StartTimeUnixNano any             `json:"startTimeUnixNano"`
	EndTimeUnixNano   any             `json:"endTimeUnixNano"`
	Attributes        []otlpAttribute `json:"attributes,omitempty"`
	Status            *otlpSpanStatus `json:"status,omitempty"`
}

type otlpScopeSpan struct {
	Spans []otlpSpan `json:"spans"`
}

type otlpResource struct {
	Attributes []otlpAttribute `json:"attributes"`
}

type otlpResourceSpan struct {
	Resource   otlpResource    `json:"resource"`
	ScopeSpans []otlpScopeSpan `json:"scopeSpans"`
}

type otlpPayload struct {
	ResourceSpans []otlpResourceSpan `json:"resourceSpans"`
}

var (
	traceStoreMu sync.RWMutex
	traceBuffer  []TraceDetail
)

func init() {
	seedInMemoryTraces()
}

func seedInMemoryTraces() {
	now := time.Now().UTC()

	rootSpanID1 := "spn_root_001"
	childSpan1_1 := "spn_checkout_001"
	childSpan1_2 := "spn_db_pool_001"
	childSpan1_3 := "spn_redis_001"

	t1 := TraceDetail{
		TraceID:       "trc_9a8b7c6d5e4f",
		RootService:   "edge-ingress",
		RootOperation: "POST /v1/checkout/charge",
		DurationMs:    128.4,
		StartTime:     now.Add(-6 * time.Minute),
		Status:        "ERROR",
		SpanCount:     4,
		Spans: []Span{
			{
				SpanID:      rootSpanID1,
				TraceID:     "trc_9a8b7c6d5e4f",
				ServiceName: "edge-ingress",
				Operation:   "POST /v1/checkout/charge",
				DurationMs:  128.4,
				StartTime:   now.Add(-6 * time.Minute),
				Status:      "ERROR",
				StatusCode:  503,
				Attributes: map[string]any{
					"http.method":      "POST",
					"http.route":       "/v1/checkout/charge",
					"http.status_code": 503,
					"client.ip":        "203.0.113.88",
					"user_agent":       "Mozilla/5.0 (Macintosh)",
				},
			},
			{
				SpanID:       childSpan1_1,
				ParentSpanID: &rootSpanID1,
				TraceID:      "trc_9a8b7c6d5e4f",
				ServiceName:  "payments-service",
				Operation:    "AuthorizePaymentGateway",
				DurationMs:   112.1,
				StartTime:    now.Add(-6*time.Minute + 5*time.Millisecond),
				Status:       "ERROR",
				StatusCode:   503,
				Attributes: map[string]any{
					"rpc.system":       "grpc",
					"payment.currency": "USD",
					"payment.amount":   149.00,
				},
			},
			{
				SpanID:       childSpan1_2,
				ParentSpanID: &childSpan1_1,
				TraceID:      "trc_9a8b7c6d5e4f",
				ServiceName:  "timescale-db-cluster-01",
				Operation:    "SELECT acquire_db_connection_pool",
				DurationMs:   94.6,
				StartTime:    now.Add(-6*time.Minute + 12*time.Millisecond),
				Status:       "ERROR",
				StatusCode:   500,
				Attributes: map[string]any{
					"db.system":    "postgresql",
					"db.name":      "sentrix_telemetry",
					"error.reason": "connection buffer limit saturated (96%)",
				},
			},
			{
				SpanID:       childSpan1_3,
				ParentSpanID: &childSpan1_1,
				TraceID:      "trc_9a8b7c6d5e4f",
				ServiceName:  "cache-redis",
				Operation:    "GET idempotency_key",
				DurationMs:   3.8,
				StartTime:    now.Add(-6*time.Minute + 7*time.Millisecond),
				Status:       "OK",
				StatusCode:   200,
				Attributes: map[string]any{
					"db.system": "redis",
					"cache.hit": true,
				},
			},
		},
	}

	rootSpanID2 := "spn_root_002"
	childSpan2_1 := "spn_auth_001"
	childSpan2_2 := "spn_db_user_001"

	t2 := TraceDetail{
		TraceID:       "trc_1e2d3c4b5a6f",
		RootService:   "edge-ingress",
		RootOperation: "POST /api/v1/auth/token",
		DurationMs:    44.2,
		StartTime:     now.Add(-12 * time.Minute),
		Status:        "OK",
		SpanCount:     3,
		Spans: []Span{
			{
				SpanID:      rootSpanID2,
				TraceID:     "trc_1e2d3c4b5a6f",
				ServiceName: "edge-ingress",
				Operation:   "POST /api/v1/auth/token",
				DurationMs:  44.2,
				StartTime:   now.Add(-12 * time.Minute),
				Status:      "OK",
				StatusCode:  200,
				Attributes: map[string]any{
					"http.status_code": 200,
					"auth.grant_type":  "authorization_code",
				},
			},
			{
				SpanID:       childSpan2_1,
				ParentSpanID: &rootSpanID2,
				TraceID:      "trc_1e2d3c4b5a6f",
				ServiceName:  "auth-gateway",
				Operation:    "ValidateSessionTokens",
				DurationMs:   36.5,
				StartTime:    now.Add(-12*time.Minute + 3*time.Millisecond),
				Status:       "OK",
				StatusCode:   200,
			},
			{
				SpanID:       childSpan2_2,
				ParentSpanID: &childSpan2_1,
				TraceID:      "trc_1e2d3c4b5a6f",
				ServiceName:  "timescale-db-cluster-01",
				Operation:    "SELECT users WHERE email = $1",
				DurationMs:   14.2,
				StartTime:    now.Add(-12*time.Minute + 8*time.Millisecond),
				Status:       "OK",
				StatusCode:   200,
			},
		},
	}

	rootSpanID3 := "spn_root_003"
	childSpan3_1 := "spn_downsample_001"
	t3 := TraceDetail{
		TraceID:       "trc_5b4c3d2e1a90",
		RootService:   "worker-runner",
		RootOperation: "Job::DownsampleMetricsRollup",
		DurationMs:    184.0,
		StartTime:     now.Add(-18 * time.Minute),
		Status:        "OK",
		SpanCount:     2,
		Spans: []Span{
			{
				SpanID:      rootSpanID3,
				TraceID:     "trc_5b4c3d2e1a90",
				ServiceName: "worker-runner",
				Operation:   "Job::DownsampleMetricsRollup",
				DurationMs:  184.0,
				StartTime:   now.Add(-18 * time.Minute),
				Status:      "OK",
				Attributes: map[string]any{
					"batch.records": 14200,
					"job.interval":  "5m",
				},
			},
			{
				SpanID:       childSpan3_1,
				ParentSpanID: &rootSpanID3,
				TraceID:      "trc_5b4c3d2e1a90",
				ServiceName:  "telemetry-engine",
				Operation:    "TimescaleDB::ChunkDownsample",
				DurationMs:   148.2,
				StartTime:    now.Add(-18*time.Minute + 12*time.Millisecond),
				Status:       "OK",
			},
		},
	}

	traceBuffer = []TraceDetail{t1, t2, t3}
}

// SeedTracesDB ensures initial traces are present in TimescaleDB/PostgreSQL
func SeedTracesDB(ctx context.Context, pool *pgxpool.Pool) {
	if pool == nil {
		return
	}

	var count int
	err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM traces").Scan(&count)
	if err == nil && count > 0 {
		return
	}

	slog.Info("Seeding initial distributed traces into TimescaleDB...")
	for _, t := range traceBuffer {
		_, _ = pool.Exec(ctx, `
			INSERT INTO traces (trace_id, root_service, root_operation, duration_ms, start_time, status, span_count)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (trace_id) DO NOTHING
		`, t.TraceID, t.RootService, t.RootOperation, t.DurationMs, t.StartTime, t.Status, t.SpanCount)

		for _, s := range t.Spans {
			attrJSON, _ := json.Marshal(s.Attributes)
			_, _ = pool.Exec(ctx, `
				INSERT INTO spans (span_id, trace_id, parent_span_id, service_name, operation, duration_ms, start_time, status, status_code, attributes)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
				ON CONFLICT (span_id) DO NOTHING
			`, s.SpanID, s.TraceID, s.ParentSpanID, s.ServiceName, s.Operation, s.DurationMs, s.StartTime, s.Status, s.StatusCode, attrJSON)
		}
	}
}

// GET /api/v1/traces
func HandleListTraces(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		service := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("service")))
		operation := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("operation")))
		status := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("status")))
		minDuration := r.URL.Query().Get("min_duration_ms")

		var minDur float64
		if minDuration != "" {
			if parsed, err := strconv.ParseFloat(minDuration, 64); err == nil {
				minDur = parsed
			}
		}

		// Try database query if pool available
		if pool != nil {
			query := `SELECT trace_id, root_service, root_operation, duration_ms, start_time, status, span_count FROM traces WHERE 1=1`
			args := []any{}
			argIdx := 1

			if status != "" && status != "ALL" {
				query += fmt.Sprintf(" AND status = $%d", argIdx)
				args = append(args, status)
				argIdx++
			}
			if service != "" {
				query += fmt.Sprintf(" AND LOWER(root_service) LIKE $%d", argIdx)
				args = append(args, "%"+service+"%")
				argIdx++
			}
			if operation != "" {
				query += fmt.Sprintf(" AND LOWER(root_operation) LIKE $%d", argIdx)
				args = append(args, "%"+operation+"%")
				argIdx++
			}
			if minDur > 0 {
				query += fmt.Sprintf(" AND duration_ms >= $%d", argIdx)
				args = append(args, minDur)
				argIdx++
			}

			query += " ORDER BY start_time DESC LIMIT 100"

			rows, err := pool.Query(r.Context(), query, args...)
			if err == nil {
				defer rows.Close()
				dbTraces := make([]TraceDetail, 0)
				for rows.Next() {
					var t TraceDetail
					if err := rows.Scan(&t.TraceID, &t.RootService, &t.RootOperation, &t.DurationMs, &t.StartTime, &t.Status, &t.SpanCount); err == nil {
						// Load spans for each trace
						spanRows, sErr := pool.Query(r.Context(), `
							SELECT span_id, parent_span_id, service_name, operation, duration_ms, start_time, status, status_code, attributes
							FROM spans WHERE trace_id = $1 ORDER BY start_time ASC
						`, t.TraceID)
						if sErr == nil {
							for spanRows.Next() {
								var s Span
								var attrBytes []byte
								if err := spanRows.Scan(&s.SpanID, &s.ParentSpanID, &s.ServiceName, &s.Operation, &s.DurationMs, &s.StartTime, &s.Status, &s.StatusCode, &attrBytes); err == nil {
									s.TraceID = t.TraceID
									if len(attrBytes) > 0 {
										_ = json.Unmarshal(attrBytes, &s.Attributes)
									}
									t.Spans = append(t.Spans, s)
								}
							}
							spanRows.Close()
						}
						dbTraces = append(dbTraces, t)
					}
				}
				if len(dbTraces) > 0 {
					api.RespondJSON(w, http.StatusOK, dbTraces)
					return
				}
			}
		}

		// In-memory fallback
		traceStoreMu.RLock()
		defer traceStoreMu.RUnlock()

		filtered := make([]TraceDetail, 0)
		for _, item := range traceBuffer {
			if status != "" && status != "ALL" && item.Status != status {
				continue
			}
			if service != "" && !strings.Contains(strings.ToLower(item.RootService), service) {
				continue
			}
			if operation != "" && !strings.Contains(strings.ToLower(item.RootOperation), operation) {
				continue
			}
			if minDur > 0 && item.DurationMs < minDur {
				continue
			}
			filtered = append(filtered, item)
		}

		api.RespondJSON(w, http.StatusOK, filtered)
	}
}

// GET /api/v1/traces/{traceID}
func HandleGetTrace(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		traceID := chi.URLParam(r, "traceID")

		if pool != nil {
			var t TraceDetail
			err := pool.QueryRow(r.Context(), `
				SELECT trace_id, root_service, root_operation, duration_ms, start_time, status, span_count
				FROM traces WHERE trace_id = $1
			`, traceID).Scan(&t.TraceID, &t.RootService, &t.RootOperation, &t.DurationMs, &t.StartTime, &t.Status, &t.SpanCount)

			if err == nil {
				spanRows, sErr := pool.Query(r.Context(), `
					SELECT span_id, parent_span_id, service_name, operation, duration_ms, start_time, status, status_code, attributes
					FROM spans WHERE trace_id = $1 ORDER BY start_time ASC
				`, traceID)
				if sErr == nil {
					for spanRows.Next() {
						var s Span
						var attrBytes []byte
						if err := spanRows.Scan(&s.SpanID, &s.ParentSpanID, &s.ServiceName, &s.Operation, &s.DurationMs, &s.StartTime, &s.Status, &s.StatusCode, &attrBytes); err == nil {
							s.TraceID = traceID
							if len(attrBytes) > 0 {
								_ = json.Unmarshal(attrBytes, &s.Attributes)
							}
							t.Spans = append(t.Spans, s)
						}
					}
					spanRows.Close()
				}
				api.RespondJSON(w, http.StatusOK, t)
				return
			}
		}

		traceStoreMu.RLock()
		defer traceStoreMu.RUnlock()

		for _, item := range traceBuffer {
			if item.TraceID == traceID {
				api.RespondJSON(w, http.StatusOK, item)
				return
			}
		}

		api.RespondError(w, r, http.StatusNotFound, "NOT_FOUND", "Trace ID not found")
	}
}

// POST /api/v1/traces, POST /otlp/v1/traces, POST /v1/traces
// Supports both native SentriX TraceDetail format and standard OpenTelemetry OTLP/HTTP JSON
func HandleIngestTraces(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			api.RespondError(w, r, http.StatusBadRequest, "INVALID_BODY", "Failed to read request body")
			return
		}

		var tracesToSave []TraceDetail

		// 1. Try parsing standard OpenTelemetry OTLP payload
		var otlp otlpPayload
		if err := json.Unmarshal(bodyBytes, &otlp); err == nil && len(otlp.ResourceSpans) > 0 {
			tracesToSave = parseOTLPPayload(otlp)
		}

		// 2. If not OTLP, try single SentriX TraceDetail
		if len(tracesToSave) == 0 {
			var single TraceDetail
			if err := json.Unmarshal(bodyBytes, &single); err == nil && (single.TraceID != "" || len(single.Spans) > 0) {
				if single.TraceID == "" {
					single.TraceID = "trc_" + uuid.New().String()[:12]
				}
				if single.StartTime.IsZero() {
					single.StartTime = time.Now().UTC()
				}
				if single.SpanCount == 0 {
					single.SpanCount = len(single.Spans)
				}
				tracesToSave = append(tracesToSave, single)
			}
		}

		// 3. Try slice of TraceDetail
		if len(tracesToSave) == 0 {
			var slice []TraceDetail
			if err := json.Unmarshal(bodyBytes, &slice); err == nil && len(slice) > 0 {
				tracesToSave = slice
			}
		}

		if len(tracesToSave) == 0 {
			api.RespondError(w, r, http.StatusBadRequest, "INVALID_PAYLOAD", "Payload could not be parsed as OTLP or SentriX trace JSON")
			return
		}

		// Save to PostgreSQL using a single transactional pgx.Batch & update in-memory buffer
		ctx := r.Context()
		if pool != nil {
			tx, err := pool.Begin(ctx)
			if err != nil {
				slog.Error("failed to begin trace ingest tx", "error", err)
				api.RespondError(w, r, http.StatusInternalServerError, "DB_ERROR", "Failed to start trace transaction")
				return
			}
			defer tx.Rollback(ctx)

			batch := &pgx.Batch{}
			for _, t := range tracesToSave {
				batch.Queue(`
					INSERT INTO traces (trace_id, root_service, root_operation, duration_ms, start_time, status, span_count)
					VALUES ($1, $2, $3, $4, $5, $6, $7)
					ON CONFLICT (trace_id) DO UPDATE SET
						duration_ms = EXCLUDED.duration_ms,
						status = EXCLUDED.status,
						span_count = EXCLUDED.span_count
				`, t.TraceID, t.RootService, t.RootOperation, t.DurationMs, t.StartTime, t.Status, t.SpanCount)

				for _, s := range t.Spans {
					attrJSON, err := json.Marshal(s.Attributes)
					if err != nil {
						attrJSON = []byte("{}")
					}
					batch.Queue(`
						INSERT INTO spans (span_id, trace_id, parent_span_id, service_name, operation, duration_ms, start_time, status, status_code, attributes)
						VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
						ON CONFLICT (span_id) DO UPDATE SET
							duration_ms = EXCLUDED.duration_ms,
							status = EXCLUDED.status,
							status_code = EXCLUDED.status_code,
							attributes = EXCLUDED.attributes
					`, s.SpanID, s.TraceID, s.ParentSpanID, s.ServiceName, s.Operation, s.DurationMs, s.StartTime, s.Status, s.StatusCode, attrJSON)
				}
			}

			br := tx.SendBatch(ctx, batch)
			for i := 0; i < batch.Len(); i++ {
				if _, execErr := br.Exec(); execErr != nil {
					br.Close()
					slog.Error("failed to persist trace batch", "error", execErr)
					api.RespondError(w, r, http.StatusInternalServerError, "DB_WRITE_FAILED", "Failed to persist trace spans")
					return
				}
			}
			if closeErr := br.Close(); closeErr != nil {
				slog.Error("failed to close trace batch", "error", closeErr)
				api.RespondError(w, r, http.StatusInternalServerError, "DB_WRITE_FAILED", "Failed to finalize trace batch")
				return
			}
			if commitErr := tx.Commit(ctx); commitErr != nil {
				slog.Error("failed to commit trace batch", "error", commitErr)
				api.RespondError(w, r, http.StatusInternalServerError, "DB_COMMIT_FAILED", "Failed to commit trace transaction")
				return
			}
		}

		traceStoreMu.Lock()
		for _, t := range tracesToSave {
			traceBuffer = append([]TraceDetail{t}, traceBuffer...)
		}
		if len(traceBuffer) > 500 {
			traceBuffer = traceBuffer[:500]
		}
		traceStoreMu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]any{
			"status":        "accepted",
			"traces_stored": len(tracesToSave),
		})
	}
}

// parseOTLPPayload transforms an OpenTelemetry JSON payload into SentriX TraceDetail models
func parseOTLPPayload(otlp otlpPayload) []TraceDetail {
	traceMap := make(map[string]*TraceDetail)

	for _, rs := range otlp.ResourceSpans {
		serviceName := "unknown-service"
		for _, attr := range rs.Resource.Attributes {
			if attr.Key == "service.name" {
				if s, ok := attr.Value.ToValue().(string); ok && s != "" {
					serviceName = s
				}
			}
		}

		for _, ss := range rs.ScopeSpans {
			for _, oSpan := range ss.Spans {
				traceID := oSpan.TraceID
				if traceID == "" {
					traceID = "trc_" + uuid.New().String()[:12]
				}

				startNano := parseNano(oSpan.StartTimeUnixNano)
				endNano := parseNano(oSpan.EndTimeUnixNano)

				startTime := time.Unix(0, startNano).UTC()
				durationMs := float64(endNano-startNano) / 1e6
				if durationMs < 0 {
					durationMs = 0
				}

				status := "OK"
				if oSpan.Status != nil && oSpan.Status.Code == 2 {
					status = "ERROR"
				}

				attrs := make(map[string]any)
				var statusCode int
				for _, a := range oSpan.Attributes {
					val := a.Value.ToValue()
					attrs[a.Key] = val
					if a.Key == "http.status_code" || a.Key == "http.response.status_code" {
						if code, ok := val.(int64); ok {
							statusCode = int(code)
							if statusCode >= 400 {
								status = "ERROR"
							}
						}
					}
				}

				var parentSpanID *string
				if oSpan.ParentSpanID != "" {
					pid := oSpan.ParentSpanID
					parentSpanID = &pid
				}

				span := Span{
					SpanID:       oSpan.SpanID,
					ParentSpanID: parentSpanID,
					TraceID:      traceID,
					ServiceName:  serviceName,
					Operation:    oSpan.Name,
					DurationMs:   durationMs,
					StartTime:    startTime,
					Status:       status,
					StatusCode:   statusCode,
					Attributes:   attrs,
				}

				detail, exists := traceMap[traceID]
				if !exists {
					detail = &TraceDetail{
						TraceID:       traceID,
						RootService:   serviceName,
						RootOperation: oSpan.Name,
						DurationMs:    durationMs,
						StartTime:     startTime,
						Status:        status,
						SpanCount:     0,
						Spans:         []Span{},
					}
					traceMap[traceID] = detail
				}

				detail.Spans = append(detail.Spans, span)
				detail.SpanCount++
				if span.ParentSpanID == nil {
					detail.RootService = span.ServiceName
					detail.RootOperation = span.Operation
					detail.DurationMs = span.DurationMs
					detail.StartTime = span.StartTime
				}
				if span.Status == "ERROR" {
					detail.Status = "ERROR"
				}
			}
		}
	}

	result := make([]TraceDetail, 0, len(traceMap))
	for _, t := range traceMap {
		result = append(result, *t)
	}
	return result
}

func parseNano(val any) int64 {
	switch v := val.(type) {
	case string:
		n, _ := strconv.ParseInt(v, 10, 64)
		return n
	case float64:
		return int64(v)
	case int64:
		return v
	case json.Number:
		n, _ := v.Int64()
		return n
	default:
		return time.Now().UnixNano()
	}
}
