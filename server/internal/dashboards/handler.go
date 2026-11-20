package dashboards

import (
	"context"
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

type DashboardWidget struct {
	ID      string         `json:"id"`
	Type    string         `json:"type"` // METRIC_LINE, METRIC_GAUGE, LOG_STREAM, ALERT_LIST, PROCESS_LIST
	Title   string         `json:"title"`
	Metric  string         `json:"metric,omitempty"`
	ColSpan int            `json:"col_span"` // 1, 2, 3
	Config  map[string]any `json:"config,omitempty"`
}

type Dashboard struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	IsDefault   bool              `json:"is_default"`
	Widgets     []DashboardWidget `json:"widgets"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

var (
	dashMu     sync.RWMutex
	dashboards []Dashboard
)

func init() {
	now := time.Now().UTC()

	d1 := Dashboard{
		ID:          "dash-primary",
		Title:       "Production Mission Control",
		Description: "Unified cross-stack telemetry view for core API gateways, microservices, and databases",
		IsDefault:   true,
		CreatedAt:   now.Add(-720 * time.Hour),
		UpdatedAt:   now,
		Widgets: []DashboardWidget{
			{
				ID:      "w-01",
				Type:    "METRIC_LINE",
				Title:   "Fleet CPU Saturation",
				Metric:  "system.cpu.utilization",
				ColSpan: 2,
			},
			{
				ID:      "w-02",
				Type:    "METRIC_GAUGE",
				Title:   "Host Memory Saturation",
				Metric:  "system.memory.utilization",
				ColSpan: 1,
			},
			{
				ID:      "w-03",
				Type:    "LOG_STREAM",
				Title:   "Realtime Error & Warning Log Stream",
				ColSpan: 2,
			},
			{
				ID:      "w-04",
				Type:    "ALERT_LIST",
				Title:   "Active Incidents & Triages",
				ColSpan: 1,
			},
		},
	}

	d2 := Dashboard{
		ID:          "dash-database",
		Title:       "TimescaleDB & Storage Performance",
		Description: "Hypertable chunk ingestion rates, buffer cache pressure, and disk IOPS latency",
		IsDefault:   false,
		CreatedAt:   now.Add(-480 * time.Hour),
		UpdatedAt:   now,
		Widgets: []DashboardWidget{
			{
				ID:      "w-11",
				Type:    "METRIC_LINE",
				Title:   "Timescale Chunk Write IOPS",
				Metric:  "system.disk.utilization",
				ColSpan: 2,
			},
			{
				ID:      "w-12",
				Type:    "METRIC_GAUGE",
				Title:   "Storage Utilization",
				Metric:  "system.disk.utilization",
				ColSpan: 1,
			},
		},
	}

	dashboards = []Dashboard{d1, d2}
}

// ensureDBInitialDashboards ensures default dashboards exist in PostgreSQL
func ensureDBInitialDashboards(ctx context.Context, pool *pgxpool.Pool) {
	if pool == nil {
		return
	}

	var count int
	err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM custom_dashboards").Scan(&count)
	if err != nil || count > 0 {
		return
	}

	dashMu.RLock()
	seed := make([]Dashboard, len(dashboards))
	copy(seed, dashboards)
	dashMu.RUnlock()

	for _, d := range seed {
		wBytes, _ := json.Marshal(d.Widgets)
		_, _ = pool.Exec(ctx, `
			INSERT INTO custom_dashboards (id, title, description, is_default, widgets, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (id) DO NOTHING
		`, d.ID, d.Title, d.Description, d.IsDefault, wBytes, d.CreatedAt, d.UpdatedAt)
	}
}

// GET /api/v1/dashboards
func HandleListDashboards(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if pool != nil {
			ctx := r.Context()
			ensureDBInitialDashboards(ctx, pool)

			rows, err := pool.Query(ctx, `
				SELECT id, title, description, is_default, widgets, created_at, updated_at
				FROM custom_dashboards
				ORDER BY is_default DESC, created_at ASC
			`)
			if err == nil {
				defer rows.Close()
				var list []Dashboard
				for rows.Next() {
					var d Dashboard
					var wBytes []byte
					if scanErr := rows.Scan(&d.ID, &d.Title, &d.Description, &d.IsDefault, &wBytes, &d.CreatedAt, &d.UpdatedAt); scanErr == nil {
						_ = json.Unmarshal(wBytes, &d.Widgets)
						if d.Widgets == nil {
							d.Widgets = []DashboardWidget{}
						}
						list = append(list, d)
					}
				}
				if list != nil {
					api.RespondJSON(w, http.StatusOK, list)
					return
				}
			}
		}

		dashMu.RLock()
		defer dashMu.RUnlock()
		api.RespondJSON(w, http.StatusOK, dashboards)
	}
}

// GET /api/v1/dashboards/{id}
func HandleGetDashboard(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		if pool != nil {
			ctx := r.Context()
			var d Dashboard
			var wBytes []byte
			err := pool.QueryRow(ctx, `
				SELECT id, title, description, is_default, widgets, created_at, updated_at
				FROM custom_dashboards
				WHERE id = $1
			`, id).Scan(&d.ID, &d.Title, &d.Description, &d.IsDefault, &wBytes, &d.CreatedAt, &d.UpdatedAt)
			if err == nil {
				_ = json.Unmarshal(wBytes, &d.Widgets)
				if d.Widgets == nil {
					d.Widgets = []DashboardWidget{}
				}
				api.RespondJSON(w, http.StatusOK, d)
				return
			}
		}

		dashMu.RLock()
		defer dashMu.RUnlock()

		for _, d := range dashboards {
			if d.ID == id {
				api.RespondJSON(w, http.StatusOK, d)
				return
			}
		}

		api.RespondError(w, r, http.StatusNotFound, "NOT_FOUND", "Dashboard not found")
	}
}

// POST /api/v1/dashboards
func HandleCreateDashboard(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor := auth.GetUser(r)
		if actor == nil {
			api.RespondError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
			return
		}

		var req Dashboard
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			api.RespondError(w, r, http.StatusBadRequest, "INVALID_PAYLOAD", "Invalid dashboard payload")
			return
		}

		req.Title = strings.TrimSpace(req.Title)
		if req.Title == "" {
			api.RespondError(w, r, http.StatusBadRequest, "REQUIRED_FIELD", "Dashboard title is required")
			return
		}

		now := time.Now().UTC()
		req.ID = "dash-" + uuid.New().String()[:8]
		req.CreatedAt = now
		req.UpdatedAt = now
		if len(req.Widgets) == 0 {
			req.Widgets = []DashboardWidget{
				{
					ID:      "w-" + uuid.New().String()[:8],
					Type:    "METRIC_LINE",
					Title:   "Fleet CPU Saturation",
					Metric:  "system.cpu.utilization",
					ColSpan: 2,
				},
				{
					ID:      "w-" + uuid.New().String()[:8],
					Type:    "METRIC_GAUGE",
					Title:   "Host Memory Saturation",
					Metric:  "system.memory.utilization",
					ColSpan: 1,
				},
			}
		}

		if pool != nil {
			ctx := r.Context()
			ensureDBInitialDashboards(ctx, pool)

			wBytes, _ := json.Marshal(req.Widgets)
			_, err := pool.Exec(ctx, `
				INSERT INTO custom_dashboards (id, title, description, is_default, widgets, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7)
			`, req.ID, req.Title, req.Description, req.IsDefault, wBytes, req.CreatedAt, req.UpdatedAt)
			if err != nil {
				api.RespondError(w, r, http.StatusInternalServerError, "DB_ERROR", "Failed to create dashboard in database")
				return
			}
		}

		dashMu.Lock()
		dashboards = append([]Dashboard{req}, dashboards...)
		dashMu.Unlock()

		api.RespondJSON(w, http.StatusCreated, req)
	}
}

// PUT /api/v1/dashboards/{id}
func HandleUpdateDashboard(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		var req Dashboard
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			api.RespondError(w, r, http.StatusBadRequest, "INVALID_PAYLOAD", "Invalid dashboard payload")
			return
		}

		now := time.Now().UTC()
		req.UpdatedAt = now

		if pool != nil {
			ctx := r.Context()
			wBytes, _ := json.Marshal(req.Widgets)
			_, err := pool.Exec(ctx, `
				UPDATE custom_dashboards
				SET title = COALESCE(NULLIF($2, ''), title),
				    description = $3,
				    widgets = $4,
				    updated_at = $5
				WHERE id = $1
			`, id, req.Title, req.Description, wBytes, now)
			if err != nil {
				api.RespondError(w, r, http.StatusInternalServerError, "DB_ERROR", "Failed to update dashboard in database")
				return
			}
		}

		dashMu.Lock()
		for i, d := range dashboards {
			if d.ID == id {
				if req.Title != "" {
					dashboards[i].Title = req.Title
				}
				dashboards[i].Description = req.Description
				dashboards[i].Widgets = req.Widgets
				dashboards[i].UpdatedAt = now
				req = dashboards[i]
				break
			}
		}
		dashMu.Unlock()

		api.RespondJSON(w, http.StatusOK, req)
	}
}

// POST /api/v1/dashboards/{id}/widgets
func HandleAddWidget(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		var widget DashboardWidget
		if err := json.NewDecoder(r.Body).Decode(&widget); err != nil {
			api.RespondError(w, r, http.StatusBadRequest, "INVALID_PAYLOAD", "Invalid widget payload")
			return
		}

		widget.Title = strings.TrimSpace(widget.Title)
		if widget.Title == "" {
			widget.Title = "Custom Widget"
		}
		if widget.ID == "" {
			widget.ID = "w-" + uuid.New().String()[:8]
		}
		if widget.ColSpan <= 0 || widget.ColSpan > 3 {
			widget.ColSpan = 2
		}
		if widget.Type == "" {
			widget.Type = "METRIC_LINE"
		}

		now := time.Now().UTC()

		if pool != nil {
			ctx := r.Context()
			ensureDBInitialDashboards(ctx, pool)

			var d Dashboard
			var wBytes []byte
			err := pool.QueryRow(ctx, `
				SELECT id, title, description, is_default, widgets, created_at, updated_at
				FROM custom_dashboards
				WHERE id = $1
			`, id).Scan(&d.ID, &d.Title, &d.Description, &d.IsDefault, &wBytes, &d.CreatedAt, &d.UpdatedAt)
			if err != nil {
				api.RespondError(w, r, http.StatusNotFound, "NOT_FOUND", "Dashboard not found")
				return
			}

			_ = json.Unmarshal(wBytes, &d.Widgets)
			d.Widgets = append(d.Widgets, widget)
			d.UpdatedAt = now

			newWBytes, _ := json.Marshal(d.Widgets)
			_, err = pool.Exec(ctx, `
				UPDATE custom_dashboards
				SET widgets = $2, updated_at = $3
				WHERE id = $1
			`, id, newWBytes, now)
			if err != nil {
				api.RespondError(w, r, http.StatusInternalServerError, "DB_ERROR", "Failed to add widget to database")
				return
			}

			api.RespondJSON(w, http.StatusCreated, d)
			return
		}

		dashMu.Lock()
		defer dashMu.Unlock()

		for i, d := range dashboards {
			if d.ID == id {
				dashboards[i].Widgets = append(dashboards[i].Widgets, widget)
				dashboards[i].UpdatedAt = now
				api.RespondJSON(w, http.StatusCreated, dashboards[i])
				return
			}
		}

		api.RespondError(w, r, http.StatusNotFound, "NOT_FOUND", "Dashboard not found")
	}
}

// DELETE /api/v1/dashboards/{id}/widgets/{widgetId}
func HandleDeleteWidget(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		widgetID := chi.URLParam(r, "widgetId")

		now := time.Now().UTC()

		if pool != nil {
			ctx := r.Context()
			var d Dashboard
			var wBytes []byte
			err := pool.QueryRow(ctx, `
				SELECT id, title, description, is_default, widgets, created_at, updated_at
				FROM custom_dashboards
				WHERE id = $1
			`, id).Scan(&d.ID, &d.Title, &d.Description, &d.IsDefault, &wBytes, &d.CreatedAt, &d.UpdatedAt)
			if err != nil {
				api.RespondError(w, r, http.StatusNotFound, "NOT_FOUND", "Dashboard not found")
				return
			}

			_ = json.Unmarshal(wBytes, &d.Widgets)
			filtered := make([]DashboardWidget, 0, len(d.Widgets))
			for _, wItem := range d.Widgets {
				if wItem.ID != widgetID {
					filtered = append(filtered, wItem)
				}
			}
			d.Widgets = filtered
			d.UpdatedAt = now

			newWBytes, _ := json.Marshal(d.Widgets)
			_, err = pool.Exec(ctx, `
				UPDATE custom_dashboards
				SET widgets = $2, updated_at = $3
				WHERE id = $1
			`, id, newWBytes, now)
			if err != nil {
				api.RespondError(w, r, http.StatusInternalServerError, "DB_ERROR", "Failed to delete widget in database")
				return
			}

			api.RespondJSON(w, http.StatusOK, d)
			return
		}

		dashMu.Lock()
		defer dashMu.Unlock()

		for i, d := range dashboards {
			if d.ID == id {
				filtered := make([]DashboardWidget, 0, len(dashboards[i].Widgets))
				for _, wItem := range dashboards[i].Widgets {
					if wItem.ID != widgetID {
						filtered = append(filtered, wItem)
					}
				}
				dashboards[i].Widgets = filtered
				dashboards[i].UpdatedAt = now
				api.RespondJSON(w, http.StatusOK, dashboards[i])
				return
			}
		}

		api.RespondError(w, r, http.StatusNotFound, "NOT_FOUND", "Dashboard not found")
	}
}

// DELETE /api/v1/dashboards/{id}
func HandleDeleteDashboard(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		if pool != nil {
			ctx := r.Context()
			var isDefault bool
			err := pool.QueryRow(ctx, "SELECT is_default FROM custom_dashboards WHERE id = $1", id).Scan(&isDefault)
			if err != nil {
				api.RespondError(w, r, http.StatusNotFound, "NOT_FOUND", "Dashboard not found")
				return
			}
			if isDefault {
				api.RespondError(w, r, http.StatusBadRequest, "PROTECTED_RESOURCE", "Cannot delete the default primary dashboard")
				return
			}

			_, _ = pool.Exec(ctx, "DELETE FROM custom_dashboards WHERE id = $1", id)
		}

		dashMu.Lock()
		filtered := make([]Dashboard, 0, len(dashboards))
		for _, d := range dashboards {
			if d.ID == id {
				if d.IsDefault {
					dashMu.Unlock()
					api.RespondError(w, r, http.StatusBadRequest, "PROTECTED_RESOURCE", "Cannot delete default dashboard")
					return
				}
				continue
			}
			filtered = append(filtered, d)
		}
		dashboards = filtered
		dashMu.Unlock()

		api.RespondJSON(w, http.StatusOK, map[string]any{
			"status": "deleted",
			"id":     id,
		})
	}
}
