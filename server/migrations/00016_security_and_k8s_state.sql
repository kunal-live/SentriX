-- Migration 00016: Security restart nonce & persistent K8s/container telemetry state

ALTER TABLE agents ADD COLUMN IF NOT EXISTS last_restart_nonce TEXT;

CREATE TABLE IF NOT EXISTS k8s_clusters (
    cluster_name VARCHAR(255) PRIMARY KEY,
    version VARCHAR(128) NOT NULL DEFAULT '',
    status VARCHAR(50) NOT NULL DEFAULT 'HEALTHY',
    total_nodes INT NOT NULL DEFAULT 0,
    ready_nodes INT NOT NULL DEFAULT 0,
    total_pods INT NOT NULL DEFAULT 0,
    running_pods INT NOT NULL DEFAULT 0,
    pending_pods INT NOT NULL DEFAULT 0,
    failed_pods INT NOT NULL DEFAULT 0,
    total_cpu_milli BIGINT NOT NULL DEFAULT 0,
    allocated_cpu_milli BIGINT NOT NULL DEFAULT 0,
    total_memory_bytes BIGINT NOT NULL DEFAULT 0,
    allocated_memory_bytes BIGINT NOT NULL DEFAULT 0,
    is_live_collector BOOLEAN NOT NULL DEFAULT TRUE,
    last_sync_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    nodes JSONB NOT NULL DEFAULT '[]',
    deployments JSONB NOT NULL DEFAULT '[]',
    pods JSONB NOT NULL DEFAULT '[]',
    events JSONB NOT NULL DEFAULT '[]'
);

CREATE TABLE IF NOT EXISTS docker_containers (
    id VARCHAR(128) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    image VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'RUNNING',
    server_id VARCHAR(128) NOT NULL DEFAULT '',
    server_name VARCHAR(255) NOT NULL DEFAULT '',
    cpu_usage_pct DOUBLE PRECISION NOT NULL DEFAULT 0,
    memory_usage_bytes BIGINT NOT NULL DEFAULT 0,
    memory_limit_bytes BIGINT NOT NULL DEFAULT 0,
    net_rx_bytes BIGINT NOT NULL DEFAULT 0,
    net_tx_bytes BIGINT NOT NULL DEFAULT 0,
    restart_count INT NOT NULL DEFAULT 0,
    oom_killed BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
