-- 00020_server_offline_incidents.sql
-- Incidents opened by the server heartbeat monitor (server went OFFLINE).
-- `source` marks them so they can be found and auto-resolved when the server
-- comes back; NULL for incidents from alert rules, checks or manual creation.

ALTER TABLE incidents ADD COLUMN IF NOT EXISTS source TEXT;

-- At most one active offline incident per server.
CREATE UNIQUE INDEX IF NOT EXISTS uniq_open_incident_server_offline
ON incidents(server_id)
WHERE source = 'SERVER_OFFLINE'
  AND status IN ('OPEN', 'ACKNOWLEDGED', 'INVESTIGATING');
