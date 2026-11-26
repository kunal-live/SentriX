package servers

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sentrix/server/internal/realtime"
)

const (
	StatusOnline  = "ONLINE"
	StatusSuspect = "SUSPECT"
	StatusOffline = "OFFLINE"
)

// StartStatusMonitor periodically reconciles server.status in PostgreSQL based on
// last_seen_at heartbeat thresholds and publishes realtime state transition events.
func StartStatusMonitor(ctx context.Context, pool *pgxpool.Pool) {
	if pool == nil {
		return
	}
	slog.Info("server status monitor started")

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	// Run an initial reconciliation pass at startup
	reconcileServerStatuses(ctx, pool)

	for {
		select {
		case <-ctx.Done():
			slog.Info("server status monitor stopped")
			return
		case <-ticker.C:
			reconcileServerStatuses(ctx, pool)
		}
	}
}

func reconcileServerStatuses(ctx context.Context, pool *pgxpool.Pool) {
	rows, err := pool.Query(ctx, `
		WITH computed AS (
			SELECT
				id,
				name,
				status AS old_status,
				CASE
					WHEN last_seen_at > NOW() - INTERVAL '30 seconds' THEN 'ONLINE'
					WHEN last_seen_at > NOW() - INTERVAL '60 seconds' THEN 'SUSPECT'
					ELSE 'OFFLINE'
				END AS new_status
			FROM servers
		)
		UPDATE servers s
		SET status = c.new_status, updated_at = NOW()
		FROM computed c
		WHERE s.id = c.id
		  AND s.status IS DISTINCT FROM c.new_status
		RETURNING c.id, c.name, c.old_status, c.new_status
	`)
	if err != nil {
		slog.Error("server status monitor failed to reconcile", "error", err)
		return
	}
	defer rows.Close()

	changed := false
	for rows.Next() {
		var id, name, oldStatus, newStatus string
		if scanErr := rows.Scan(&id, &name, &oldStatus, &newStatus); scanErr != nil {
			continue
		}
		changed = true
		slog.Info("server status transitioned", "server_id", id, "name", name, "from", oldStatus, "to", newStatus)
		realtime.PublishServerStatusChanged(id, name, oldStatus, newStatus)
	}

	rows.Close()

	if changed {
		realtime.PublishDashboardUpdated()
	}

	syncOfflineIncidents(ctx, pool)
}
