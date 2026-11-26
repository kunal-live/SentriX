package servers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sentrix/server/internal/auth"
)

type ServerListItem struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Hostname string    `json:"hostname"`
	Platform string    `json:"platform"`
	Status   string    `json:"status"` // Computed: ONLINE, SUSPECT, OFFLINE
	LastSeen time.Time `json:"last_seen"`
	CPU      *float64  `json:"cpu"`
	Memory   *float64  `json:"memory"`
	Disk     *float64  `json:"disk"`
}

// GET /api/v1/servers
func HandleListServers(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// We compute the status dynamically based on heartbeat timeout
		rows, err := pool.Query(ctx, `
			SELECT 
				s.id, s.name, s.hostname, s.platform, s.last_seen_at,
				CASE 
					WHEN s.last_seen_at > NOW() - INTERVAL '30 seconds' THEN 'ONLINE'
					WHEN s.last_seen_at > NOW() - INTERVAL '60 seconds' THEN 'SUSPECT'
					ELSE 'OFFLINE'
				END as status,
				c.value as cpu, m.value as memory, d.value as disk
			FROM servers s
			LEFT JOIN LATERAL (SELECT value FROM metric_cpu WHERE server_id = s.id ORDER BY time DESC LIMIT 1) c ON true
			LEFT JOIN LATERAL (SELECT value FROM metric_memory WHERE server_id = s.id ORDER BY time DESC LIMIT 1) m ON true
			LEFT JOIN LATERAL (SELECT value FROM metric_disk WHERE server_id = s.id ORDER BY time DESC LIMIT 1) d ON true
			ORDER BY s.name ASC
		`)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var servers []ServerListItem
		for rows.Next() {
			var srv ServerListItem
			// Scan handles NULLs for metrics if a server has no data yet
			err := rows.Scan(&srv.ID, &srv.Name, &srv.Hostname, &srv.Platform, &srv.LastSeen, &srv.Status, &srv.CPU, &srv.Memory, &srv.Disk)
			if err != nil {
				http.Error(w, "Scan error", http.StatusInternalServerError)
				return
			}
			servers = append(servers, srv)
		}

		if servers == nil {
			servers = []ServerListItem{} // Return empty array, not null
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(servers)
	}
}

// DELETE /api/v1/servers/{serverID}
func HandleDeleteServer(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor := auth.GetUser(r)
		if actor == nil || actor.Role != "ADMIN" {
			http.Error(w, "forbidden: only administrators can delete servers", http.StatusForbidden)
			return
		}

		serverID := chi.URLParam(r, "serverID")
		if _, err := uuid.Parse(serverID); err != nil {
			http.Error(w, "invalid server ID format", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		tx, err := pool.Begin(ctx)
		if err != nil {
			http.Error(w, "database transaction failed", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback(ctx)

		// Retrieve associated agent ID to revoke credential upon server deletion
		var agentID *string
		if err := tx.QueryRow(ctx, "SELECT agent_id FROM servers WHERE id = $1", serverID).Scan(&agentID); err != nil {
			http.Error(w, "server not found", http.StatusNotFound)
			return
		}

		// Delete dependents children-first. Order matters: alert/check states
		// point at incidents, and check incidents point at checks, without
		// ON DELETE CASCADE. Any failure aborts the transaction, so every
		// statement is checked and reported instead of failing later.
		steps := []struct{ what, sql string }{
			{"alert states", `DELETE FROM alert_states WHERE server_id = $1
				OR incident_id IN (SELECT id FROM incidents WHERE server_id = $1)`},
			{"check states", `DELETE FROM check_states WHERE server_id = $1
				OR incident_id IN (SELECT id FROM incidents WHERE server_id = $1)
				OR check_id IN (SELECT id FROM checks WHERE server_id = $1)`},
			{"incidents", `DELETE FROM incidents WHERE server_id = $1
				OR root_check_id IN (SELECT id FROM checks WHERE server_id = $1)`},
			{"check results", `DELETE FROM check_results WHERE server_id = $1`},
			{"checks", `DELETE FROM checks WHERE server_id = $1`},
			{"cpu metrics", `DELETE FROM metric_cpu WHERE server_id = $1`},
			{"memory metrics", `DELETE FROM metric_memory WHERE server_id = $1`},
			{"disk metrics", `DELETE FROM metric_disk WHERE server_id = $1`},
			{"network metrics", `DELETE FROM metric_network WHERE server_id = $1`},
		}
		for _, step := range steps {
			if _, err := tx.Exec(ctx, step.sql, serverID); err != nil {
				slog.Error("delete server: failed to remove dependent rows", "server_id", serverID, "step", step.what, "error", err)
				http.Error(w, "failed to delete server ("+step.what+")", http.StatusInternalServerError)
				return
			}
		}

		result, err := tx.Exec(ctx, "DELETE FROM servers WHERE id = $1", serverID)
		if err != nil {
			slog.Error("delete server failed", "server_id", serverID, "error", err)
			http.Error(w, "failed to delete server", http.StatusInternalServerError)
			return
		}

		if result.RowsAffected() == 0 {
			http.Error(w, "server not found", http.StatusNotFound)
			return
		}

		// Revoke the agent so it can no longer submit telemetry
		if agentID != nil {
			_, _ = tx.Exec(ctx, "UPDATE agents SET status = 'REVOKED' WHERE id = $1", *agentID)
		}

		if err := tx.Commit(ctx); err != nil {
			http.Error(w, "transaction commit failed", http.StatusInternalServerError)
			return
		}

		auth.WriteAudit(
			ctx,
			pool,
			&actor.ID,
			"server.delete",
			"server",
			serverID,
			auth.ClientIP(r),
			nil,
		)

		w.WriteHeader(http.StatusNoContent)
	}
}
