-- Migration 00018: Bind docker_containers rows to owning agent ID

ALTER TABLE docker_containers
    ADD COLUMN IF NOT EXISTS owner_agent_id UUID;
