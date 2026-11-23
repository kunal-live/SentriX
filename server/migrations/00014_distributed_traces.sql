-- Migration 00014: Distributed Tracing & OpenTelemetry Spans

CREATE TABLE IF NOT EXISTS traces (
    trace_id VARCHAR(64) PRIMARY KEY,
    root_service VARCHAR(255) NOT NULL,
    root_operation VARCHAR(255) NOT NULL,
    duration_ms DOUBLE PRECISION NOT NULL,
    start_time TIMESTAMPTZ NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'OK',
    span_count INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS spans (
    span_id VARCHAR(64) PRIMARY KEY,
    trace_id VARCHAR(64) REFERENCES traces(trace_id) ON DELETE CASCADE,
    parent_span_id VARCHAR(64),
    service_name VARCHAR(255) NOT NULL,
    operation VARCHAR(255) NOT NULL,
    duration_ms DOUBLE PRECISION NOT NULL,
    start_time TIMESTAMPTZ NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'OK',
    status_code INT,
    attributes JSONB DEFAULT '{}',
    events JSONB DEFAULT '[]',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_traces_start_time ON traces(start_time DESC);
CREATE INDEX IF NOT EXISTS idx_traces_service ON traces(root_service);
CREATE INDEX IF NOT EXISTS idx_traces_status ON traces(status);
CREATE INDEX IF NOT EXISTS idx_spans_trace_id ON spans(trace_id);
CREATE INDEX IF NOT EXISTS idx_spans_service ON spans(service_name);
