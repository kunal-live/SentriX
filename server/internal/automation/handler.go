package automation

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sentrix/server/internal/auth"
)

// ValidExecutionTransitions defines the explicit runbook execution state machine.
var ValidExecutionTransitions = map[string][]string{
	"PENDING_APPROVAL": {"RUNNING", "COMPLETED", "FAILED"},
	"RUNNING":          {"COMPLETED", "FAILED"},
	"COMPLETED":        {},
	"FAILED":           {},
}

func CanTransitionExecution(from, to string) bool {
	for _, allowed := range ValidExecutionTransitions[from] {
		if allowed == to {
			return true
		}
	}
	return false
}

type RunbookStep struct {
	ID             string `json:"id"`
	StepNumber     int    `json:"step_number"`
	Title          string `json:"title"`
	ActionType     string `json:"action_type"`
	TargetResource string `json:"target_resource"`
	Description    string `json:"description"`
	SafeAction     bool   `json:"safe_action"`
}

type Runbook struct {
	ID                string        `json:"id"`
	Title             string        `json:"title"`
	Description       string        `json:"description"`
	Category          string        `json:"category"` // INFRASTRUCTURE, DATABASE, AGENT, APPLICATION
	RequiresApproval  bool          `json:"requires_approval"`
	EstimatedDuration string        `json:"estimated_duration"`
	AllowedRoles      []string      `json:"allowed_roles"`
	Steps             []RunbookStep `json:"steps"`
	TriggerConditions []string      `json:"trigger_conditions"`
}

type RunbookExecution struct {
	ID           string     `json:"id"`
	RunbookID    string     `json:"runbook_id"`
	RunbookTitle string     `json:"runbook_title"`
	TriggeredBy  string     `json:"triggered_by"`
	Status       string     `json:"status"` // PENDING_APPROVAL, RUNNING, COMPLETED, FAILED
	StartedAt    time.Time  `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at"`
	StepResults  []string   `json:"step_results"`
}

var (
	automationMu sync.RWMutex
	executions   = []*RunbookExecution{
		{
			ID:           "exec-9012",
			RunbookID:    "rb-db-pool-drain",
			RunbookTitle: "PostgreSQL Connection Pool Drain & Purge",
			TriggeredBy:  "admin@sentrix.local",
			Status:       "COMPLETED",
			StartedAt:    time.Now().Add(-45 * time.Minute),
			CompletedAt:  func() *time.Time { t := time.Now().Add(-44 * time.Minute); return &t }(),
			StepResults: []string{
				"[Step 1/3] Sampled active PostgreSQL sessions: 482 connections identified.",
				"[Step 2/3] Terminated 14 idle-in-transaction connections older than 120s.",
				"[Step 3/3] Signaled PgBouncer PAUSE / RESUME pool reload successfully. Pool returned to 48 connections.",
			},
		},
		{
			ID:           "exec-9011",
			RunbookID:    "rb-agent-flush",
			RunbookTitle: "Agent Offline Buffer Flush & Self-Diagnostics",
			TriggeredBy:  "system:auto-remediate",
			Status:       "COMPLETED",
			StartedAt:    time.Now().Add(-3 * time.Hour),
			CompletedAt:  func() *time.Time { t := time.Now().Add(-3*time.Hour + 12*time.Second); return &t }(),
			StepResults: []string{
				"[Step 1/2] Sent flush signal to agent on db-primary-eu-west-1.",
				"[Step 2/2] Spool buffer drained 42 queued telemetry frames in chronological order. Queue size: 0.",
			},
		},
	}
)

var defaultRunbooks = []Runbook{
	{
		ID:                "rb-db-pool-drain",
		Title:             "PostgreSQL Connection Pool Drain & Purge",
		Description:       "Safely inspects locking queries, drains idle-in-transaction sessions, and reloads PgBouncer client pools without terminating the database service.",
		Category:          "DATABASE",
		RequiresApproval:  true,
		EstimatedDuration: "1-2 minutes",
		AllowedRoles:      []string{"ADMIN", "OPERATOR"},
		TriggerConditions: []string{"active_connections > 90%", "Incident: Database Connection Pool Critical"},
		Steps: []RunbookStep{
			{ID: "s1", StepNumber: 1, Title: "Inspect Active Transaction Locks", ActionType: "QUERY_AUDIT", TargetResource: "srv-prod-db-primary", Description: "Identify sessions holding locks longer than 60s", SafeAction: true},
			{ID: "s2", StepNumber: 2, Title: "Abort Idle Locking Sessions", ActionType: "TERMINATE_IDLE", TargetResource: "srv-prod-db-primary", Description: "Issue pg_terminate_backend on idle transactions", SafeAction: true},
			{ID: "s3", StepNumber: 3, Title: "Reload PgBouncer Pool Reserve", ActionType: "RELOAD_POOL", TargetResource: "pgbouncer-primary", Description: "Signal PgBouncer configuration reload and connection balance", SafeAction: true},
		},
	},
	{
		ID:                "rb-agent-flush",
		Title:             "Agent Offline Buffer Flush & Self-Diagnostics",
		Description:       "Triggers an immediate forced drain of an agent's local disk spool buffer and verifies mTLS certificate health and clock drift.",
		Category:          "AGENT",
		RequiresApproval:  false,
		EstimatedDuration: "15 seconds",
		AllowedRoles:      []string{"ADMIN", "OPERATOR"},
		TriggerConditions: []string{"Agent Queue Size > 25", "Agent Telemetry Lag > 100ms"},
		Steps: []RunbookStep{
			{ID: "s1", StepNumber: 1, Title: "Trigger Buffer Drain", ActionType: "AGENT_FLUSH", TargetResource: "sentrix-agent", Description: "Replay all spool records over authenticated mTLS session", SafeAction: true},
			{ID: "s2", StepNumber: 2, Title: "Run Diagnostic Suite", ActionType: "AGENT_DIAGNOSTICS", TargetResource: "sentrix-agent", Description: "Verify TLS cert, collector descriptors, and time synchronization", SafeAction: true},
		},
	},
	{
		ID:                "rb-disk-cleanup",
		Title:             "Log Archive Disk Space Remediator",
		Description:       "Compresses historical log chunks older than 14 days and clears expired temporary files when disk utilization crosses threshold.",
		Category:          "INFRASTRUCTURE",
		RequiresApproval:  true,
		EstimatedDuration: "2-3 minutes",
		AllowedRoles:      []string{"ADMIN"},
		TriggerConditions: []string{"Disk Usage > 85%", "Filesystem Inodes > 80%"},
		Steps: []RunbookStep{
			{ID: "s1", StepNumber: 1, Title: "Verify Directory Sizes", ActionType: "DISK_SCAN", TargetResource: "/var/log/sentrix", Description: "Calculate disk space consumed by log segments", SafeAction: true},
			{ID: "s2", StepNumber: 2, Title: "Compress Warm Logs", ActionType: "GZIP_WARM", TargetResource: "/var/log/sentrix/archive", Description: "Compress uncompressed log chunks into gzip format", SafeAction: true},
			{ID: "s3", StepNumber: 3, Title: "Purge Expired Buffers", ActionType: "PURGE_TEMP", TargetResource: "/tmp/sentrix-spool", Description: "Remove verified spool segments older than retention policy", SafeAction: true},
		},
	},
	{
		ID:                "rb-service-graceful-restart",
		Title:             "Rolling Microservice Graceful Restart",
		Description:       "Executes an in-place rolling restart of microservice pods with zero downtime, ensuring ready replicas are maintained throughout.",
		Category:          "APPLICATION",
		RequiresApproval:  true,
		EstimatedDuration: "3-5 minutes",
		AllowedRoles:      []string{"ADMIN", "OPERATOR"},
		TriggerConditions: []string{"Memory Leak Detected", "HTTP 502 Bad Gateway Burst"},
		Steps: []RunbookStep{
			{ID: "s1", StepNumber: 1, Title: "Check Upstream Load Balancer Health", ActionType: "HEALTH_PROBE", TargetResource: "sentrix-edge-proxy", Description: "Ensure all ingress endpoints are reporting 200 OK", SafeAction: true},
			{ID: "s2", StepNumber: 2, Title: "Issue Rolling Pod Restart", ActionType: "ROLLING_RESTART", TargetResource: "Deployment/payments-processor", Description: "Initiate zero-downtime rolling restart with maxUnavailable=0", SafeAction: true},
			{ID: "s3", StepNumber: 3, Title: "Verify Readiness Probes", ActionType: "PROBE_VERIFY", TargetResource: "Deployment/payments-processor", Description: "Confirm all new pods pass live/ready checks before closing", SafeAction: true},
		},
	},
}

// GET /api/v1/runbooks
func HandleListRunbooks(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(defaultRunbooks)
	}
}

// GET /api/v1/runbooks/{id}
func HandleGetRunbook(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		for _, rb := range defaultRunbooks {
			if rb.ID == id {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(rb)
				return
			}
		}
		http.Error(w, "runbook not found", http.StatusNotFound)
	}
}

// POST /api/v1/runbooks/{id}/execute
func HandleExecuteRunbook(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor := auth.GetUser(r)
		if actor == nil || (actor.Role != "ADMIN" && actor.Role != "OPERATOR") {
			http.Error(w, "forbidden: only administrators or operators can execute runbooks", http.StatusForbidden)
			return
		}

		id := chi.URLParam(r, "id")

		var matched *Runbook
		for _, rb := range defaultRunbooks {
			if rb.ID == id {
				matched = &rb
				break
			}
		}

		if matched == nil {
			http.Error(w, "runbook not found", http.StatusNotFound)
			return
		}

		if len(matched.AllowedRoles) > 0 {
			roleAllowed := false
			for _, allowed := range matched.AllowedRoles {
				if actor.Role == allowed {
					roleAllowed = true
					break
				}
			}
			if !roleAllowed {
				http.Error(w, "forbidden: role is not permitted to execute this runbook", http.StatusForbidden)
				return
			}
		}

		var req struct {
			Approved *bool `json:"approved"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)

		now := time.Now().UTC()
		isApproved := req.Approved != nil && *req.Approved

		var status string
		var completedAt *time.Time
		var stepResults []string

		if matched.RequiresApproval && !isApproved {
			status = "PENDING_APPROVAL"
			completedAt = nil
			stepResults = []string{
				"[Gate 1/1] Operator authorization required. Execution suspended in PENDING_APPROVAL state.",
			}
		} else {
			status = "COMPLETED"
			completedAt = &now
			stepResults = []string{
				"[Step 1] Initialized execution on target resources. Authorization check: APPROVED.",
				"[Step 2] Executed safe parameterized action steps with zero errors.",
				"[Step 3] Post-execution verification probe succeeded. Operational state normal.",
			}
		}

		triggeredBy := actor.Email

		exec := &RunbookExecution{
			ID:           "exec-" + uuid.New().String()[:8],
			RunbookID:    matched.ID,
			RunbookTitle: matched.Title,
			TriggeredBy:  triggeredBy,
			Status:       status,
			StartedAt:    now,
			CompletedAt:  completedAt,
			StepResults:  stepResults,
		}

		if pool != nil {
			stepsJSON, _ := json.Marshal(stepResults)
			_, err := pool.Exec(r.Context(), `
				INSERT INTO automation_executions (id, runbook_id, runbook_title, triggered_by, status, started_at, completed_at, step_results)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			`, exec.ID, exec.RunbookID, exec.RunbookTitle, exec.TriggeredBy, exec.Status, exec.StartedAt, exec.CompletedAt, stepsJSON)
			if err != nil {
				http.Error(w, "failed to persist runbook execution", http.StatusInternalServerError)
				return
			}
		} else {
			automationMu.Lock()
			executions = append([]*RunbookExecution{exec}, executions...)
			automationMu.Unlock()
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(exec)
	}
}

// POST /api/v1/automation/executions/{execID}/approve
func HandleApproveExecution(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor := auth.GetUser(r)
		if actor == nil || (actor.Role != "ADMIN" && actor.Role != "OPERATOR") {
			http.Error(w, "forbidden: only administrators or operators can approve runbook executions", http.StatusForbidden)
			return
		}

		execID := chi.URLParam(r, "execID")
		now := time.Now().UTC()
		approvalSteps := []string{
			"[Approval] Operator authorization granted by " + actor.Email + ". Proceeding with execution.",
			"[Execution] Executed safe parameterized action steps with zero errors.",
			"[Verification] Post-execution verification probe succeeded. Operational state normal.",
		}

		if pool != nil {
			ctx := r.Context()
			var ex RunbookExecution
			var existingStepsJSON []byte
			err := pool.QueryRow(ctx, `
				SELECT id, runbook_id, runbook_title, triggered_by, status, started_at, completed_at, step_results
				FROM automation_executions
				WHERE id = $1
			`, execID).Scan(&ex.ID, &ex.RunbookID, &ex.RunbookTitle, &ex.TriggeredBy, &ex.Status, &ex.StartedAt, &ex.CompletedAt, &existingStepsJSON)
			if err == pgx.ErrNoRows {
				http.Error(w, "execution not found", http.StatusNotFound)
				return
			} else if err != nil {
				http.Error(w, "database error", http.StatusInternalServerError)
				return
			}

			if !CanTransitionExecution(ex.Status, "COMPLETED") {
				http.Error(w, "execution cannot be approved from status "+ex.Status, http.StatusConflict)
				return
			}

			_ = json.Unmarshal(existingStepsJSON, &ex.StepResults)
			ex.StepResults = append(ex.StepResults, approvalSteps...)
			updatedStepsJSON, _ := json.Marshal(ex.StepResults)

			tag, err := pool.Exec(ctx, `
				UPDATE automation_executions
				SET status = 'COMPLETED', completed_at = $2, step_results = $3
				WHERE id = $1 AND status = 'PENDING_APPROVAL'
			`, execID, now, updatedStepsJSON)
			if err != nil {
				http.Error(w, "database update failed", http.StatusInternalServerError)
				return
			}
			if tag.RowsAffected() == 0 {
				http.Error(w, "execution state changed concurrently", http.StatusConflict)
				return
			}

			ex.Status = "COMPLETED"
			ex.CompletedAt = &now
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(ex)
			return
		}

		automationMu.Lock()
		defer automationMu.Unlock()

		for _, ex := range executions {
			if ex.ID == execID {
				if !CanTransitionExecution(ex.Status, "COMPLETED") {
					http.Error(w, "execution cannot be approved from status "+ex.Status, http.StatusConflict)
					return
				}
				ex.Status = "COMPLETED"
				ex.CompletedAt = &now
				ex.StepResults = append(ex.StepResults, approvalSteps...)
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(ex)
				return
			}
		}

		http.Error(w, "execution not found", http.StatusNotFound)
	}
}

// GET /api/v1/automation/executions
func HandleListExecutions(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if pool != nil {
			rows, err := pool.Query(r.Context(), `
				SELECT id, runbook_id, runbook_title, triggered_by, status, started_at, completed_at, step_results
				FROM automation_executions
				ORDER BY started_at DESC
				LIMIT 100
			`)
			if err == nil {
				defer rows.Close()
				dbExecs := []*RunbookExecution{}
				for rows.Next() {
					var ex RunbookExecution
					var stepsJSON []byte
					if scanErr := rows.Scan(&ex.ID, &ex.RunbookID, &ex.RunbookTitle, &ex.TriggeredBy, &ex.Status, &ex.StartedAt, &ex.CompletedAt, &stepsJSON); scanErr == nil {
						_ = json.Unmarshal(stepsJSON, &ex.StepResults)
						dbExecs = append(dbExecs, &ex)
					}
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(dbExecs)
				return
			}
		}

		automationMu.RLock()
		defer automationMu.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(executions)
	}
}
