-- JPL Horizons 地心黄道坐标缓存：每天同步一次、由后端计算行星合/冲/大距，
-- 不在用户打开页面时向 JPL 发请求。
CREATE TABLE astronomy_ephemeris_samples (
    body TEXT NOT NULL,
    epoch TIMESTAMPTZ NOT NULL,
    x_au DOUBLE PRECISION NOT NULL,
    y_au DOUBLE PRECISION NOT NULL,
    z_au DOUBLE PRECISION NOT NULL,
    source_url TEXT NOT NULL,
    synced_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (body, epoch)
);
CREATE INDEX astronomy_ephemeris_samples_body_epoch_idx
    ON astronomy_ephemeris_samples(body, epoch);
