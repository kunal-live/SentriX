package servers

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sentrix/server/internal/alerts"
	"github.com/sentrix/server/internal/incidents"
	"github.com/sentrix/server/internal/notifications"
	"github.com/sentrix/server/internal/realtime"
)

const (
	incidentSourceOffline = "SERVER_OFFLINE"
	offlineSeverity       = "CRITICAL"

	// Servers silent for longer than this (a Postgres interval) are treated as
	// decommissioned, so a SentriX restart or upgrade doesn't page for long-dead
	// servers.
	offlineAlertMaxAge = "24 hours"
)

type offlineServer struct {
	ID       string
	Name     string
	LastSeen time.Time
}

// syncOfflineIncidents opens an incident (and notifies) for each server that is
// OFFLINE, and resolves it (and notifies) once the server is ONLINE again.
//
// It is level-based rather than tied to the status transition, so an outage is
// still caught after a SentriX restart or after a maintenance window ends. Each
// outage gets at most one incident: if a user resolves it by hand while the
// server is still down, it isn't reopened until the server has reported again.
// The partial unique index makes this safe with several server replicas.
func syncOfflineIncidents(ctx context.Context, pool *pgxpool.Pool) {
	openOfflineIncidents(ctx, pool)
	resolveOfflineIncidents(ctx, pool)
}

func openOfflineIncidents(ctx context.Context, pool *pgxpool.Pool) {
	rows, err := pool.Query(ctx, `
		SELECT s.id, s.name, s.last_seen_at
		FROM servers s
		WHERE s.status = 'OFFLINE'
		  AND s.last_seen_at IS NOT NULL
		  AND s.last_seen_at > NOW() - $1::interval
		  AND NOT EXISTS (
			SELECT 1 FROM incidents i
			WHERE i.server_id = s.id
			  AND i.source = $2
			  AND i.started_at >= s.last_seen_at
		  )
	`, offlineAlertMaxAge, incidentSourceOffline)
	if err != nil {
		slog.Error("offline monitor failed to list offline servers", "error", err)
		return
	}
	candidates, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (offlineServer, error) {
		var s offlineServer
		err := row.Scan(&s.ID, &s.Name, &s.LastSeen)
		return s, err
	})
	if err != nil {
		slog.Error("offline monitor failed to read offline servers", "error", err)
		return
	}

	for _, srv := range candidates {
		if alerts.IsServerSilenced(ctx, pool, srv.ID) {
			continue
		}

		title := fmt.Sprintf("Server offline: %s", srv.Name)
		summary := fmt.Sprintf("No heartbeat from %s since %s.", srv.Name, srv.LastSeen.UTC().Format(time.RFC3339))

		var incidentID string
		err := pool.QueryRow(ctx, `
			INSERT INTO incidents (id, server_id, title, severity, status, started_at, summary, source)
			VALUES ($1, $2, $3, $4, 'OPEN', NOW(), $5, $6)
			ON CONFLICT DO NOTHING
			RETURNING id
		`, uuid.New(), srv.ID, title, offlineSeverity, summary, incidentSourceOffline).Scan(&incidentID)
		if err == pgx.ErrNoRows {
			continue // another replica opened it first
		}
		if err != nil {
			slog.Error("offline monitor failed to open incident", "server_id", srv.ID, "error", err)
			continue
		}

		_ = incidents.WriteTimelinePool(ctx, pool, incidentID, nil, "OPENED",
			"Server stopped sending heartbeats. Opened automatically by the server monitor.")
		realtime.PublishIncidentUpdated(incidentID, srv.ID)
		slog.Warn("server offline incident opened", "server_id", srv.ID, "name", srv.Name, "incident_id", incidentID)

		notifications.EnqueueServerNotification(ctx, pool, notifications.ServerEventParams{
			EventType:  "server.offline",
			ServerID:   srv.ID,
			ServerName: srv.Name,
			IncidentID: incidentID,
			Severity:   offlineSeverity,
			Title:      title,
			Message:    summary,
			LastSeenAt: srv.LastSeen,
		})
	}
}

func resolveOfflineIncidents(ctx context.Context, pool *pgxpool.Pool) {
	rows, err := pool.Query(ctx, `
		UPDATE incidents i
		SET status = 'RESOLVED', resolved_at = NOW(), updated_at = NOW()
		FROM servers s
		WHERE i.server_id = s.id
		  AND s.status = 'ONLINE'
		  AND i.source = $1
		  AND i.status IN ('OPEN', 'ACKNOWLEDGED', 'INVESTIGATING')
		RETURNING i.id, s.id, s.name, s.last_seen_at, i.started_at, i.severity
	`, incidentSourceOffline)
	if err != nil {
		slog.Error("offline monitor failed to resolve incidents", "error", err)
		return
	}

	type resolved struct {
		IncidentID string
		Server     offlineServer
		StartedAt  time.Time
		Severity   string
	}
	done, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (resolved, error) {
		var r resolved
		err := row.Scan(&r.IncidentID, &r.Server.ID, &r.Server.Name, &r.Server.LastSeen, &r.StartedAt, &r.Severity)
		return r, err
	})
	if err != nil {
		slog.Error("offline monitor failed to read resolved incidents", "error", err)
		return
	}

	for _, r := range done {
		downtime := time.Since(r.StartedAt).Round(time.Second)
		message := fmt.Sprintf("%s is sending heartbeats again (incident open for %s).", r.Server.Name, downtime)

		_ = incidents.WriteTimelinePool(ctx, pool, r.IncidentID, nil, "RESOLVED",
			"Server is back online. Resolved automatically by the server monitor.")
		realtime.PublishIncidentUpdated(r.IncidentID, r.Server.ID)
		slog.Info("server offline incident resolved", "server_id", r.Server.ID, "name", r.Server.Name, "incident_id", r.IncidentID)

		notifications.EnqueueServerNotification(ctx, pool, notifications.ServerEventParams{
			EventType:  "server.online",
			ServerID:   r.Server.ID,
			ServerName: r.Server.Name,
			IncidentID: r.IncidentID,
			Severity:   r.Severity,
			Title:      fmt.Sprintf("Server back online: %s", r.Server.Name),
			Message:    message,
			LastSeenAt: r.Server.LastSeen,
		})
	}
}
