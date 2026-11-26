-- Migration 00019: make the incident lifecycle writable end-to-end.
--
-- 1. Incident handlers set updated_at on every change, but the column never
--    existed, so every PATCH / investigate / postmortem failed.
-- 2. The lifecycle allows RESOLVED -> OPEN (reopen) and RESOLVED -> CLOSED, but
--    the timeline CHECK rejected OPEN and CLOSED events, aborting those writes.
-- 3. Re-assert the 5-state status CHECK. Migration 00011 could silently skip
--    adding CLOSED on databases where 00005's 4-state constraint already existed.

ALTER TABLE incidents
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

ALTER TABLE incident_timeline_events
    DROP CONSTRAINT IF EXISTS incident_timeline_events_event_type_check;
ALTER TABLE incident_timeline_events
    ADD CONSTRAINT incident_timeline_events_event_type_check CHECK (
        event_type IN (
            'OPENED',
            'OPEN',
            'ACKNOWLEDGED',
            'INVESTIGATING',
            'RESOLVED',
            'CLOSED',
            'COMMENT',
            'SYSTEM'
        )
    );

ALTER TABLE incidents DROP CONSTRAINT IF EXISTS incidents_status_check;
ALTER TABLE incidents
    ADD CONSTRAINT incidents_status_check
    CHECK (status IN ('OPEN', 'ACKNOWLEDGED', 'INVESTIGATING', 'RESOLVED', 'CLOSED'));
