-- JPL Horizons 的数据用途彼此独立：月球、火星和日心深空探测器不能共用一次同步状态。
-- 保留旧的 jpl_horizons 记录用于历史追溯；新同步从以下三个明确范围开始记录。
INSERT INTO data_sources (code, name, base_url) VALUES
    ('jpl_horizons_moon', 'JPL Horizons · 月球轨道', 'https://ssd.jpl.nasa.gov/horizons/'),
    ('jpl_horizons_mars', 'JPL Horizons · 火星轨道', 'https://ssd.jpl.nasa.gov/horizons/'),
    ('jpl_horizons_voyage', 'JPL Horizons · 深空探测器', 'https://ssd.jpl.nasa.gov/horizons/')
ON CONFLICT (code) DO NOTHING;
