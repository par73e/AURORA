-- 月球飞行器轨道快照（JPL Horizons 日同步，镜像 orbit_snapshots）
-- 存储每次同步的瞬时（osculating）轨道根数，前端按此渲染真实轨道
CREATE TABLE IF NOT EXISTS moon_orbit_snapshots (
    id BIGSERIAL PRIMARY KEY,
    spacecraft_id TEXT NOT NULL REFERENCES moon_spacecraft(id) ON DELETE CASCADE,
    epoch TIMESTAMPTZ NOT NULL,
    a_km DOUBLE PRECISION NOT NULL,
    eccentricity DOUBLE PRECISION NOT NULL,
    inclination_deg DOUBLE PRECISION NOT NULL,
    raan_deg DOUBLE PRECISION NOT NULL,
    arg_periapsis_deg DOUBLE PRECISION NOT NULL,
    mean_anomaly_deg DOUBLE PRECISION NOT NULL,
    period_seconds DOUBLE PRECISION NOT NULL,
    raw TEXT NOT NULL DEFAULT '',
    synced_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(spacecraft_id, epoch)
);
CREATE INDEX moon_orbit_snapshots_spacecraft_epoch_idx ON moon_orbit_snapshots(spacecraft_id, epoch DESC);
