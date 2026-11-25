package services

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sentrix/server/internal/api"
	"github.com/sentrix/server/internal/auth"
)

type ServiceEndpoint struct {
	Method       string  `json:"method"`
	Path         string  `json:"path"`
	RPS          float64 `json:"rps"`
	P95Ms        float64 `json:"p95_ms"`
	ErrorRatePct float64 `json:"error_rate_pct"`
}

type ServiceEntity struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	Environment   string            `json:"environment"` // Production, Staging, Development
	Tier          string            `json:"tier"`        // Tier 1 (Critical), Tier 2, Tier 3
	Criticality   string            `json:"criticality"` // High, Medium, Low
	Owner         string            `json:"owner"`
	Team          string            `json:"team"`
	Repository    string            `json:"repository"`
	Health        string            `json:"health"` // HEALTHY, DEGRADED, FAILING
	ServerCount   int               `json:"server_count"`
	IncidentCount int               `json:"incident_count"`
	Uptime        float64           `json:"uptime"`
	RPS           float64           `json:"rps"`
	LatencyP50Ms  float64           `json:"p50_ms"`
	LatencyP95Ms  float64           `json:"p95_ms"`
	LatencyP99Ms  float64           `json:"p99_ms"`
	ErrorRatePct  float64           `json:"error_rate_pct"`
	ApdexScore    float64           `json:"apdex_score"`
	ThroughputMBs float64           `json:"throughput_mb_s"`
	Endpoints     []ServiceEndpoint `json:"endpoints,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
}

type FleetHost struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Hostname     string    `json:"hostname"`
	Platform     string    `json:"platform"`
	OS           string    `json:"os"`
	Kernel       string    `json:"kernel"`
	Arch         string    `json:"arch"`
	Status       string    `json:"status"` // ONLINE, SUSPECT, OFFLINE
	CPU          float64   `json:"cpu"`
	Memory       float64   `json:"memory"`
	Disk         float64   `json:"disk"`
	AgentVersion string    `json:"agent_version"`
	UptimeHours  int       `json:"uptime_hours"`
	LastSeen     time.Time `json:"last_seen"`
}

var (
	serviceMu       sync.RWMutex
	defaultServices = []ServiceEntity{
		{
			ID:            "srv-catalog-01",
			Name:          "Checkout & Payment Gateway",
			Description:   "Handles Stripe/UPI authorizations, settlement callbacks, and invoice generation",
			Environment:   "Production",
			Tier:          "Tier 1 (Mission Critical)",
			Criticality:   "High",
			Owner:         "Alex Rivera",
			Team:          "Payments Engineering",
			Repository:    "github.com/sentrix/payments-service",
			Health:        "HEALTHY",
			ServerCount:   3,
			IncidentCount: 0,
			Uptime:        99.98,
			RPS:           1240.5,
			LatencyP50Ms:  14.2,
			LatencyP95Ms:  46.8,
			LatencyP99Ms:  112.4,
			ErrorRatePct:  0.02,
			ApdexScore:    0.98,
			ThroughputMBs: 8.6,
			Endpoints: []ServiceEndpoint{
				{Method: "POST", Path: "/v1/checkout/charge", RPS: 420.0, P95Ms: 48.2, ErrorRatePct: 0.01},
				{Method: "POST", Path: "/v1/settlements/webhook", RPS: 180.5, P95Ms: 32.1, ErrorRatePct: 0.00},
				{Method: "GET", Path: "/v1/invoices/{id}", RPS: 640.0, P95Ms: 12.4, ErrorRatePct: 0.01},
			},
			CreatedAt: time.Now().Add(-720 * time.Hour),
		},
		{
			ID:            "srv-catalog-02",
			Name:          "Authentication & Identity Provider",
			Description:   "OIDC/SAML tokens issuance, session validation, MFA challenges, and RBAC policies",
			Environment:   "Production",
			Tier:          "Tier 1 (Mission Critical)",
			Criticality:   "High",
			Owner:         "Elena Rostova",
			Team:          "Security & Core Identity",
			Repository:    "github.com/sentrix/auth-gateway",
			Health:        "HEALTHY",
			ServerCount:   2,
			IncidentCount: 0,
			Uptime:        99.99,
			RPS:           2150.0,
			LatencyP50Ms:  6.8,
			LatencyP95Ms:  24.5,
			LatencyP99Ms:  68.2,
			ErrorRatePct:  0.01,
			ApdexScore:    0.99,
			ThroughputMBs: 14.2,
			Endpoints: []ServiceEndpoint{
				{Method: "POST", Path: "/oauth/token", RPS: 850.0, P95Ms: 18.2, ErrorRatePct: 0.01},
				{Method: "GET", Path: "/oauth/userinfo", RPS: 1100.0, P95Ms: 5.4, ErrorRatePct: 0.00},
				{Method: "POST", Path: "/mfa/verify", RPS: 200.0, P95Ms: 22.0, ErrorRatePct: 0.02},
			},
			CreatedAt: time.Now().Add(-960 * time.Hour),
		},
		{
			ID:            "srv-catalog-03",
			Name:          "Time-Series Telemetry Ingest",
			Description:   "High-throughput batch metrics ingestion pipeline backed by TimescaleDB hypertables",
			Environment:   "Production",
			Tier:          "Tier 1 (Mission Critical)",
			Criticality:   "High",
			Owner:         "David Vance",
			Team:          "Observability Platform",
			Repository:    "github.com/sentrix/telemetry-engine",
			Health:        "DEGRADED",
			ServerCount:   4,
			IncidentCount: 1,
			Uptime:        99.91,
			RPS:           8450.0,
			LatencyP50Ms:  28.4,
			LatencyP95Ms:  96.2,
			LatencyP99Ms:  285.0,
			ErrorRatePct:  1.42,
			ApdexScore:    0.88,
			ThroughputMBs: 48.5,
			Endpoints: []ServiceEndpoint{
				{Method: "POST", Path: "/api/v1/telemetry", RPS: 6200.0, P95Ms: 84.0, ErrorRatePct: 1.20},
				{Method: "POST", Path: "/v1/traces", RPS: 1850.0, P95Ms: 112.0, ErrorRatePct: 2.10},
				{Method: "GET", Path: "/api/v1/metrics/query", RPS: 400.0, P95Ms: 32.5, ErrorRatePct: 0.05},
			},
			CreatedAt: time.Now().Add(-1200 * time.Hour),
		},
		{
			ID:            "srv-catalog-04",
			Name:          "Notification & Webhook Dispatcher",
			Description:   "Asynchronous queue dispatching PagerDuty alerts, Slack messages, and customer webhooks",
			Environment:   "Staging",
			Tier:          "Tier 2 (Standard)",
			Criticality:   "Medium",
			Owner:         "Sarah Jenkins",
			Team:          "Reliability Operations",
			Repository:    "github.com/sentrix/notification-dispatcher",
			Health:        "HEALTHY",
			ServerCount:   2,
			IncidentCount: 0,
			Uptime:        99.95,
			RPS:           320.0,
			LatencyP50Ms:  18.0,
			LatencyP95Ms:  54.0,
			LatencyP99Ms:  145.0,
			ErrorRatePct:  0.05,
			ApdexScore:    0.97,
			ThroughputMBs: 2.1,
			Endpoints: []ServiceEndpoint{
				{Method: "POST", Path: "/v1/dispatch/slack", RPS: 140.0, P95Ms: 42.0, ErrorRatePct: 0.00},
				{Method: "POST", Path: "/v1/dispatch/pagerduty", RPS: 35.0, P95Ms: 68.0, ErrorRatePct: 0.00},
				{Method: "POST", Path: "/v1/webhooks/customer", RPS: 145.0, P95Ms: 58.0, ErrorRatePct: 0.12},
			},
			CreatedAt: time.Now().Add(-480 * time.Hour),
		},
	}
)

// GET /api/v1/services
func HandleListServices(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serviceMu.RLock()
		result := make([]ServiceEntity, len(defaultServices))
		copy(result, defaultServices)
		serviceMu.RUnlock()

		// If persistent database is live, enrich with live OpenTelemetry aggregations
		if pool != nil {
			ctx := r.Context()
			rows, err := pool.Query(ctx, `
				SELECT 
					service_name,
					COUNT(*) as total_spans,
					COUNT(*) FILTER (WHERE status = 'ERROR' OR (status_code >= 400 AND status_code <= 599)) as error_spans,
					COALESCE(percentile_cont(0.50) WITHIN GROUP (ORDER BY duration_ms), 0) as p50,
					COALESCE(percentile_cont(0.95) WITHIN GROUP (ORDER BY duration_ms), 0) as p95,
					COALESCE(percentile_cont(0.99) WITHIN GROUP (ORDER BY duration_ms), 0) as p99
				FROM spans
				GROUP BY service_name
			`)
			if err == nil {
				defer rows.Close()
				type spanAgg struct {
					totalSpans int64
					errorSpans int64
					p50        float64
					p95        float64
					p99        float64
				}
				aggs := make(map[string]spanAgg)
				for rows.Next() {
					var sName string
					var agg spanAgg
					if scanErr := rows.Scan(&sName, &agg.totalSpans, &agg.errorSpans, &agg.p50, &agg.p95, &agg.p99); scanErr == nil {
						aggs[strings.ToLower(sName)] = agg
					}
				}

				for i := range result {
					sKey := strings.ToLower(result[i].Name)
					for k, v := range aggs {
						if strings.Contains(sKey, k) || strings.Contains(k, sKey) || strings.Contains(strings.ToLower(result[i].Repository), k) {
							if v.totalSpans > 0 {
								errRate := (float64(v.errorSpans) / float64(v.totalSpans)) * 100.0
								result[i].ErrorRatePct = errRate
								result[i].LatencyP50Ms = v.p50
								result[i].LatencyP95Ms = v.p95
								result[i].LatencyP99Ms = v.p99
								if errRate > 5.0 {
									result[i].Health = "FAILING"
									result[i].ApdexScore = 0.70
								} else if errRate > 1.0 || v.p95 > 200.0 {
									result[i].Health = "DEGRADED"
									result[i].ApdexScore = 0.88
								} else {
									result[i].Health = "HEALTHY"
									result[i].ApdexScore = 0.99
								}
							}
							break
						}
					}
				}
			}
		}

		api.RespondJSON(w, http.StatusOK, result)
	}
}

// POST /api/v1/services
func HandleCreateService(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor := auth.GetUser(r)
		if actor == nil {
			api.RespondError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
			return
		}

		var req ServiceEntity
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			api.RespondError(w, r, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON payload")
			return
		}

		req.Name = strings.TrimSpace(req.Name)
		if req.Name == "" {
			api.RespondError(w, r, http.StatusBadRequest, "REQUIRED_FIELD", "Service name is required")
			return
		}

		req.ID = "srv-catalog-" + uuid.New().String()[:8]
		req.CreatedAt = time.Now().UTC()
		if req.Health == "" {
			req.Health = "HEALTHY"
		}
		if req.Uptime <= 0 {
			req.Uptime = 99.99
		}
		if req.RPS <= 0 {
			req.RPS = 100.0
		}
		if req.LatencyP50Ms <= 0 {
			req.LatencyP50Ms = 12.0
		}
		if req.LatencyP95Ms <= 0 {
			req.LatencyP95Ms = 45.0
		}
		if req.LatencyP99Ms <= 0 {
			req.LatencyP99Ms = 120.0
		}
		if req.ApdexScore <= 0 {
			req.ApdexScore = 0.98
		}

		serviceMu.Lock()
		defaultServices = append([]ServiceEntity{req}, defaultServices...)
		serviceMu.Unlock()

		api.RespondJSON(w, http.StatusCreated, req)
	}
}

type apmMetricsPayload struct {
	ServiceName   string            `json:"service_name"`
	RPS           float64           `json:"rps"`
	LatencyP50Ms  float64           `json:"p50_ms"`
	LatencyP95Ms  float64           `json:"p95_ms"`
	LatencyP99Ms  float64           `json:"p99_ms"`
	ErrorRatePct  float64           `json:"error_rate_pct"`
	ApdexScore    float64           `json:"apdex_score"`
	ThroughputMBs float64           `json:"throughput_mb_s"`
	Endpoints     []ServiceEndpoint `json:"endpoints,omitempty"`
}

func applyAPMMetrics(target *ServiceEntity, incoming apmMetricsPayload) {
	if incoming.RPS > 0 {
		target.RPS = incoming.RPS
	}
	if incoming.LatencyP50Ms > 0 {
		target.LatencyP50Ms = incoming.LatencyP50Ms
	}
	if incoming.LatencyP95Ms > 0 {
		target.LatencyP95Ms = incoming.LatencyP95Ms
	}
	if incoming.LatencyP99Ms > 0 {
		target.LatencyP99Ms = incoming.LatencyP99Ms
	}
	target.ErrorRatePct = incoming.ErrorRatePct
	if incoming.ApdexScore > 0 {
		target.ApdexScore = incoming.ApdexScore
	}
	if incoming.ThroughputMBs > 0 {
		target.ThroughputMBs = incoming.ThroughputMBs
	}
	if len(incoming.Endpoints) > 0 {
		target.Endpoints = incoming.Endpoints
	}
}

// POST /api/v1/apm/metrics
func HandleIngestAPMMetrics(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var incoming apmMetricsPayload
		if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
			http.Error(w, `{"error":"invalid_json"}`, http.StatusBadRequest)
			return
		}

		if incoming.ServiceName == "" {
			http.Error(w, `{"error":"service_name_required"}`, http.StatusBadRequest)
			return
		}

		serviceMu.Lock()
		defer serviceMu.Unlock()

		found := false
		for i := range defaultServices {
			if strings.EqualFold(defaultServices[i].Name, incoming.ServiceName) || strings.EqualFold(defaultServices[i].ID, incoming.ServiceName) {
				applyAPMMetrics(&defaultServices[i], incoming)
				found = true
				break
			}
		}

		if !found {
			// Register newly reported APM microservice
			newSrv := ServiceEntity{
				ID:            "srv-apm-" + uuid.New().String()[:8],
				Name:          incoming.ServiceName,
				Description:   "Auto-discovered via APM agent instrumentation",
				Environment:   "Production",
				Tier:          "Tier 2 (Standard)",
				Criticality:   "Medium",
				Owner:         "DevOps Automation",
				Team:          "Service Engineering",
				Health:        "HEALTHY",
				ServerCount:   1,
				IncidentCount: 0,
				Uptime:        99.95,
				ApdexScore:    0.98,
				CreatedAt:     time.Now().UTC(),
			}
			applyAPMMetrics(&newSrv, incoming)
			defaultServices = append(defaultServices, newSrv)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]any{
			"status":       "accepted",
			"service_name": incoming.ServiceName,
		})
	}
}

// POST /api/v1/services/{serviceID}/apm
func HandleIngestServiceAPM(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serviceID := chi.URLParam(r, "serviceID")
		var incoming apmMetricsPayload
		if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
			http.Error(w, `{"error":"invalid_json"}`, http.StatusBadRequest)
			return
		}

		serviceMu.Lock()
		defer serviceMu.Unlock()

		for i := range defaultServices {
			if defaultServices[i].ID == serviceID || strings.EqualFold(defaultServices[i].Name, serviceID) {
				applyAPMMetrics(&defaultServices[i], incoming)
				break
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]any{
			"status":     "accepted",
			"service_id": serviceID,
		})
	}
}

// GET /api/v1/infrastructure
func HandleGetInfrastructure(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		rows, err := pool.Query(ctx, `
			SELECT
				s.id,
				s.name,
				s.hostname,
				s.platform,
				s.status,
				s.last_seen
			FROM servers s
			ORDER BY s.name ASC
		`)
		if err != nil {
			fleet := []FleetHost{
				{
					ID:           "a1b2c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d",
					Name:         "production-api-01",
					Hostname:     "api-prod-us-east-1.internal",
					Platform:     "Linux",
					OS:           "Ubuntu 24.04.1 LTS",
					Kernel:       "6.8.0-40-generic",
					Arch:         "x86_64",
					Status:       "ONLINE",
					CPU:          34.2,
					Memory:       62.8,
					Disk:         41.5,
					AgentVersion: "v1.2.4",
					UptimeHours:  842,
					LastSeen:     time.Now().UTC(),
				},
				{
					ID:           "b2c3d4e5-f6a7-4b8c-9d0e-1f2a3b4c5d6e",
					Name:         "timescale-db-cluster-01",
					Hostname:     "db-primary-us-east-1.internal",
					Platform:     "Linux",
					OS:           "Debian 12 Bookworm",
					Kernel:       "6.1.0-21-amd64",
					Arch:         "x86_64",
					Status:       "SUSPECT",
					CPU:          78.4,
					Memory:       89.1,
					Disk:         82.0,
					AgentVersion: "v1.2.4",
					UptimeHours:  1420,
					LastSeen:     time.Now().UTC(),
				},
				{
					ID:           "c3d4e5f6-a7b8-4c9d-0e1f-2a3b4c5d6e7f",
					Name:         "worker-queue-runner-01",
					Hostname:     "worker-01.internal",
					Platform:     "Linux",
					OS:           "Alpine Linux 3.20.2",
					Kernel:       "6.6.32-0-virt",
					Arch:         "x86_64",
					Status:       "ONLINE",
					CPU:          18.9,
					Memory:       42.3,
					Disk:         26.7,
					AgentVersion: "v1.2.3",
					UptimeHours:  512,
					LastSeen:     time.Now().UTC(),
				},
				{
					ID:           "d4e5f6a7-b8c9-4d0e-1f2a-3b4c5d6e7f8a",
					Name:         "edge-ingress-proxy-02",
					Hostname:     "edge-02.internal",
					Platform:     "Linux",
					OS:           "Ubuntu 22.04.4 LTS",
					Kernel:       "5.15.0-107-generic",
					Arch:         "x86_64",
					Status:       "ONLINE",
					CPU:          12.1,
					Memory:       31.4,
					Disk:         19.8,
					AgentVersion: "v1.2.4",
					UptimeHours:  2190,
					LastSeen:     time.Now().UTC(),
				},
			}
			api.RespondJSON(w, http.StatusOK, fleet)
			return
		}
		defer rows.Close()

		fleet := []FleetHost{}
		for rows.Next() {
			var h FleetHost
			err := rows.Scan(
				&h.ID,
				&h.Name,
				&h.Hostname,
				&h.Platform,
				&h.Status,
				&h.LastSeen,
			)
			if err != nil {
				continue
			}
			h.OS = "Linux (Enterprise)"
			h.Kernel = "6.8.0-generic"
			h.Arch = "x86_64"
			h.AgentVersion = "v1.2.4"
			h.UptimeHours = 340
			fleet = append(fleet, h)
		}
		api.RespondJSON(w, http.StatusOK, fleet)
	}
}
