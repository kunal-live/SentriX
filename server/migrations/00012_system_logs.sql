-- 00012_system_logs.sql
-- Persistent real-time and historical structured system logs

CREATE TABLE IF NOT EXISTS system_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    level TEXT NOT NULL, -- 'ERROR', 'WARN', 'INFO', 'DEBUG'
    service TEXT NOT NULL, -- 'sentrix-server', 'sentrix-agent', 'timescale-db', 'telemetry-engine', 'auth-gateway', 'node-monitor', 'synthetic-checker', 'systemd'
    server TEXT NOT NULL, -- Server hostname or display name
    server_id UUID REFERENCES servers(id) ON DELETE SET NULL,
    message TEXT NOT NULL,
    trace_id TEXT,
    request_id TEXT,
    metadata JSONB DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_system_logs_timestamp ON system_logs (timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_system_logs_level ON system_logs (level);
CREATE INDEX IF NOT EXISTS idx_system_logs_service ON system_logs (service);
CREATE INDEX IF NOT EXISTS idx_system_logs_server_id ON system_logs (server_id);
