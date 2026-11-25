-- Migration 00017: Backfill nonces for legacy agents and bind K8s clusters to owner agents

UPDATE agents
SET last_restart_nonce = gen_random_uuid()::text
WHERE last_restart_nonce IS NULL OR last_restart_nonce = '';

ALTER TABLE agents
    ALTER COLUMN last_restart_nonce SET DEFAULT gen_random_uuid()::text;

ALTER TABLE k8s_clusters
    ADD COLUMN IF NOT EXISTS owner_agent_id UUID;
