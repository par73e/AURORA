-- Each target keeps its own successful acquisition time when a later attempt fails.
CREATE TABLE spacecraft_sync_status (
    source_code TEXT NOT NULL REFERENCES data_sources(code),
    target_id TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('success', 'failed', 'unsupported', 'canceled')),
    last_attempt_at TIMESTAMPTZ NOT NULL,
    last_success_at TIMESTAMPTZ,
    error_message TEXT,
    PRIMARY KEY (source_code, target_id)
);

-- Existing data has an acquisition timestamp; never infer success from a failed run.
INSERT INTO spacecraft_sync_status (source_code, target_id, status, last_attempt_at, last_success_at)
SELECT 'celestrak', spacecraft_id, 'success', max(synced_at), max(synced_at)
FROM orbit_snapshots GROUP BY spacecraft_id;
INSERT INTO spacecraft_sync_status (source_code, target_id, status, last_attempt_at, last_success_at)
SELECT 'jpl_horizons_moon', spacecraft_id, 'success', max(synced_at), max(synced_at)
FROM moon_orbit_snapshots GROUP BY spacecraft_id;
INSERT INTO spacecraft_sync_status (source_code, target_id, status, last_attempt_at, last_success_at)
SELECT 'jpl_horizons_mars', spacecraft_id, 'success', max(synced_at), max(synced_at)
FROM mars_orbit_snapshots GROUP BY spacecraft_id;
INSERT INTO spacecraft_sync_status (source_code, target_id, status, last_attempt_at, last_success_at)
SELECT 'jpl_horizons_voyage', probe_id, 'success', max(synced_at), max(synced_at)
FROM probe_position_samples GROUP BY probe_id;
