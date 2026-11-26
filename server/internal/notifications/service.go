package notifications

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AlertEventParams struct {
	EventType    string
	RuleID       string
	ServerID     string
	Severity     string
	Title        string
	Message      string
	CurrentValue float64
	Threshold    float64
}

// EnqueueAlertNotification queues an alert-rule event (alert.firing /
// alert.resolved) for every enabled notification channel.
func EnqueueAlertNotification(
	ctx context.Context,
	pool *pgxpool.Pool,
	params AlertEventParams,
) {
	payload := map[string]any{
		"event_type":    params.EventType,
		"source_type":   "alert",
		"rule_id":       params.RuleID,
		"server_id":     params.ServerID,
		"server_name":   serverName(ctx, pool, params.ServerID),
		"severity":      params.Severity,
		"title":         params.Title,
		"message":       params.Message,
		"current_value": params.CurrentValue,
		"threshold":     params.Threshold,
		"timestamp":     time.Now().UTC().Format(time.RFC3339),
	}

	ruleID := params.RuleID
	enqueue(ctx, pool, params.EventType, &ruleID, params.ServerID, params.Severity, payload)
}

type ServerEventParams struct {
	EventType  string // server.offline / server.online
	ServerID   string
	ServerName string
	IncidentID string
	Severity   string
	Title      string
	Message    string
	LastSeenAt time.Time
}

// EnqueueServerNotification queues a server heartbeat event for every enabled
// notification channel.
func EnqueueServerNotification(
	ctx context.Context,
	pool *pgxpool.Pool,
	params ServerEventParams,
) {
	payload := map[string]any{
		"event_type":   params.EventType,
		"source_type":  "server",
		"server_id":    params.ServerID,
		"server_name":  params.ServerName,
		"incident_id":  params.IncidentID,
		"severity":     params.Severity,
		"title":        params.Title,
		"message":      params.Message,
		"last_seen_at": params.LastSeenAt.UTC().Format(time.RFC3339),
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
	}

	enqueue(ctx, pool, params.EventType, nil, params.ServerID, params.Severity, payload)
}

func serverName(ctx context.Context, pool *pgxpool.Pool, serverID string) string {
	var name string
	if err := pool.QueryRow(ctx, `SELECT name FROM servers WHERE id = $1`, serverID).Scan(&name); err != nil {
		return serverID
	}
	return name
}

// enqueue inserts one notification job per enabled channel. Channel IDs are read
// fully before inserting, so no result set is held open across the inserts.
func enqueue(
	ctx context.Context,
	pool *pgxpool.Pool,
	eventType string,
	ruleID *string,
	serverID string,
	severity string,
	payload map[string]any,
) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		slog.Error("failed to marshal notification payload", "event_type", eventType, "error", err)
		return
	}

	rows, err := pool.Query(ctx, `SELECT id FROM notification_channels WHERE enabled = true`)
	if err != nil {
		slog.Error("failed to load notification channels", "event_type", eventType, "error", err)
		return
	}
	var channelIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			slog.Error("failed to scan notification channel", "error", err)
			continue
		}
		channelIDs = append(channelIDs, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		slog.Error("failed to read notification channels", "event_type", eventType, "error", err)
		return
	}

	count := 0
	for _, channelID := range channelIDs {
		_, err := pool.Exec(ctx, `
			INSERT INTO notification_jobs (
				channel_id, event_type, alert_rule_id, server_id, severity, payload
			) VALUES ($1, $2, $3, $4, $5, $6)
		`, channelID, eventType, ruleID, serverID, severity, payloadJSON)
		if err != nil {
			slog.Error("failed to enqueue notification job", "event_type", eventType, "error", err)
			continue
		}
		count++
	}

	if count > 0 {
		slog.Info("notification jobs enqueued",
			"event_type", eventType,
			"server_id", serverID,
			"channels", count,
		)
	}
}
