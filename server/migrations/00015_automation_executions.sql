-- Migration 00015: Persistent Automation Runbook Executions

CREATE TABLE IF NOT EXISTS automation_executions (
    id VARCHAR(64) PRIMARY KEY,
    runbook_id VARCHAR(128) NOT NULL,
    runbook_title VARCHAR(255) NOT NULL,
    triggered_by VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL CHECK (status IN ('PENDING_APPROVAL', 'RUNNING', 'COMPLETED', 'FAILED')),
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    step_results JSONB NOT NULL DEFAULT '[]'
);

CREATE INDEX IF NOT EXISTS idx_automation_executions_started ON automation_executions(started_at DESC);
CREATE INDEX IF NOT EXISTS idx_automation_executions_status ON automation_executions(status);
