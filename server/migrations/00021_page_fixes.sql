-- 00021_page_fixes.sql
-- alert_rules never had updated_at, but the Alert Rules API selects and sets
-- it, so the page failed with "database query failed".
ALTER TABLE alert_rules ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Latest top-process snapshot per server, sent by the agent every 30s and
-- shown on the server page's Processes tab. One row per server, overwritten.
CREATE TABLE IF NOT EXISTS server_process_snapshots (
    server_id    UUID PRIMARY KEY REFERENCES servers(id) ON DELETE CASCADE,
    collected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    total        INT NOT NULL DEFAULT 0,
    processes    JSONB NOT NULL DEFAULT '[]'::jsonb
);
