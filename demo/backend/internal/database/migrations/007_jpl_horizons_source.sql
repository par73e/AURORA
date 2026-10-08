-- 注册 JPL Horizons 数据源（sync_runs 外键需要）
INSERT INTO data_sources (code, name, base_url)
VALUES ('jpl_horizons', 'JPL Horizons', 'https://ssd.jpl.nasa.gov/')
ON CONFLICT (code) DO NOTHING;
