package notifications

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type CheckEventParams struct {
	EventType string
	CheckID   string
	ServerID  string
	Severity  string
	Title     string
	Message   string
	Status    string
	LatencyMs int
}

func EnqueueCheckNotification(
	ctx context.Context,
	pool *pgxpool.Pool,
	params CheckEventParams,
) {
	payload := map[string]any{
		"event_type":  params.EventType,
		"source_type": "check",
		"check_id":    params.CheckID,
		"server_id":   params.ServerID,
		"server_name": serverName(ctx, pool, params.ServerID),
		"severity":    params.Severity,
		"title":       params.Title,
		"message":     params.Message,
		"status":      params.Status,
		"latency_ms":  params.LatencyMs,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
	}

	enqueue(ctx, pool, params.EventType, nil, params.ServerID, params.Severity, payload)
}
