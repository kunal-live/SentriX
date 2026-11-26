package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type jobRow struct {
	ID          string
	ChannelID   string
	Attempts    int
	MaxAttempts int
	Payload     []byte
}

func StartWorker(ctx context.Context, pool *pgxpool.Pool) {
	slog.Info("notification worker started")

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("notification worker stopped")
			return
		case <-ticker.C:
			processDueJobs(ctx, pool)
		}
	}
}

// jobLease is how long a claimed job is hidden from other pollers. If this
// process dies mid-delivery, the job becomes due again after the lease.
const jobLease = 2 * time.Minute

func processDueJobs(ctx context.Context, pool *pgxpool.Pool) {
	// Jobs that ran out of attempts without being marked would otherwise sit in
	// PENDING/RETRYING forever, since the claim below never selects them.
	if _, err := pool.Exec(ctx, `
		UPDATE notification_jobs
		SET status = 'FAILED', last_error = COALESCE(last_error, 'max attempts exhausted'), updated_at = NOW()
		WHERE status IN ('PENDING', 'RETRYING') AND attempts >= max_attempts
	`); err != nil {
		slog.Error("failed to sweep exhausted notification jobs", "error", err)
	}

	jobs, err := claimDueJobs(ctx, pool)
	if err != nil {
		slog.Error("failed to claim notification jobs", "error", err)
		return
	}

	// Each job gets its own transaction: a failure on one job must not roll back
	// the SENT status of jobs whose webhooks were already delivered.
	for _, job := range jobs {
		tx, err := pool.Begin(ctx)
		if err != nil {
			slog.Error("failed to begin notification job tx", "job_id", job.ID, "error", err)
			continue
		}
		processJob(ctx, tx, job)
		if err := tx.Commit(ctx); err != nil {
			slog.Error("failed to commit notification job", "job_id", job.ID, "error", err)
		}
		_ = tx.Rollback(ctx)
	}
}

// claimDueJobs atomically leases up to 20 due jobs by pushing next_attempt_at
// forward, and returns them fully read so no result set stays open while the
// jobs are processed (pgx cannot run other statements on a busy connection).
func claimDueJobs(ctx context.Context, pool *pgxpool.Pool) ([]jobRow, error) {
	rows, err := pool.Query(ctx, `
		UPDATE notification_jobs
		SET next_attempt_at = NOW() + $1::interval, updated_at = NOW()
		WHERE id IN (
			SELECT id
			FROM notification_jobs
			WHERE status IN ('PENDING', 'RETRYING')
			  AND attempts < max_attempts
			  AND next_attempt_at <= NOW()
			ORDER BY next_attempt_at ASC
			LIMIT 20
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, channel_id, attempts, max_attempts, payload
	`, fmt.Sprintf("%d seconds", int(jobLease.Seconds())))
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (jobRow, error) {
		var job jobRow
		err := row.Scan(&job.ID, &job.ChannelID, &job.Attempts, &job.MaxAttempts, &job.Payload)
		return job, err
	})
}

func processJob(ctx context.Context, tx pgx.Tx, job jobRow) {
	var channelType string
	var enabled bool
	var config []byte

	err := tx.QueryRow(ctx, `
		SELECT type, enabled, config
		FROM notification_channels
		WHERE id = $1
	`, job.ChannelID).Scan(&channelType, &enabled, &config)

	if err != nil {
		markJobFailed(ctx, tx, job.ID, "notification channel not found", 0)
		return
	}

	if !enabled {
		markJobFailed(ctx, tx, job.ID, "notification channel disabled", 0)
		return
	}

	if channelType != "WEBHOOK" {
		markJobFailed(ctx, tx, job.ID, "unsupported notification channel type", 0)
		return
	}

	var cfg struct {
		URL string `json:"url"`
	}

	if err := json.Unmarshal(config, &cfg); err != nil || cfg.URL == "" {
		markJobFailed(ctx, tx, job.ID, "invalid webhook configuration", 0)
		return
	}

	statusCode, err := sendWebhook(ctx, cfg.URL, job.Payload)

	if err == nil {
		_, updateErr := tx.Exec(ctx, `
			UPDATE notification_jobs
			SET
				status = 'SENT',
				attempts = attempts + 1,
				last_error = NULL,
				updated_at = NOW()
			WHERE id = $1
		`, job.ID)

		if updateErr != nil {
			slog.Error("failed to mark notification job sent", "error", updateErr)
		}

		insertDelivery(ctx, tx, job.ID, "SUCCESS", statusCode, "")
		return
	}

	job.Attempts++

	newStatus := "RETRYING"
	if job.Attempts >= job.MaxAttempts {
		newStatus = "FAILED"
	}

	delaySeconds := 30 * (1 << (job.Attempts - 1))
	nextAttempt := time.Now().Add(time.Duration(delaySeconds) * time.Second)

	_, updateErr := tx.Exec(ctx, `
		UPDATE notification_jobs
		SET
			status = $2,
			attempts = $3,
			last_error = $4,
			next_attempt_at = $5,
			updated_at = NOW()
		WHERE id = $1
	`, job.ID, newStatus, job.Attempts, err.Error(), nextAttempt)

	if updateErr != nil {
		slog.Error("failed to update failed notification job", "error", updateErr)
	}

	insertDelivery(ctx, tx, job.ID, "FAILED", statusCode, err.Error())
}

func sendWebhook(ctx context.Context, url string, payload []byte) (int, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(
		reqCtx,
		http.MethodPost,
		url,
		bytes.NewReader(payload),
	)

	if err != nil {
		return 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "SentriX-Notifier")

	dialer := &net.Dialer{Timeout: 5 * time.Second}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
			if err != nil {
				return nil, err
			}
			for _, ip := range ips {
				if isDisallowedIP(ip) {
					return nil, fmt.Errorf("SSRF protection blocked dial to %s (%s)", host, ip)
				}
			}
			return dialer.DialContext(ctx, network, addr)
		},
	}

	client := &http.Client{Transport: transport}

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.StatusCode, fmt.Errorf("unexpected webhook status: %d", resp.StatusCode)
	}

	return resp.StatusCode, nil
}

func markJobFailed(ctx context.Context, tx pgx.Tx, jobID string, reason string, httpStatus int) {
	_, err := tx.Exec(ctx, `
		UPDATE notification_jobs
		SET
			status = 'FAILED',
			attempts = max_attempts,
			last_error = $2,
			updated_at = NOW()
		WHERE id = $1
	`, jobID, reason)

	if err != nil {
		slog.Error("failed to mark notification job failed", "error", err)
	}

	insertDelivery(ctx, tx, jobID, "FAILED", httpStatus, reason)
}

func insertDelivery(ctx context.Context, tx pgx.Tx, jobID string, status string, httpStatus int, errMsg string) {
	var httpStatusValue interface{}
	if httpStatus != 0 {
		httpStatusValue = httpStatus
	}

	_, err := tx.Exec(ctx, `
		INSERT INTO notification_deliveries (
			job_id,
			status,
			http_status,
			error
		) VALUES (
			$1, $2, $3, $4
		)
	`, jobID, status, httpStatusValue, errMsg)

	if err != nil {
		slog.Error("failed to insert notification delivery", "error", err)
	}
}
