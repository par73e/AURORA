CREATE TABLE data_sources (
    code TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    base_url TEXT NOT NULL
);

CREATE TABLE spacecraft (
    id TEXT PRIMARY KEY,
    name_zh TEXT NOT NULL,
    name_en TEXT NOT NULL,
    norad_catalog_id BIGINT NOT NULL UNIQUE,
    category TEXT NOT NULL,
    operator_name TEXT NOT NULL,
    description TEXT NOT NULL,
    source_code TEXT NOT NULL REFERENCES data_sources(code)
);

CREATE TABLE orbit_snapshots (
    id BIGSERIAL PRIMARY KEY,
    spacecraft_id TEXT NOT NULL REFERENCES spacecraft(id) ON DELETE CASCADE,
    epoch TIMESTAMPTZ NOT NULL,
    raw_omm JSONB NOT NULL,
    synced_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(spacecraft_id, epoch)
);
CREATE INDEX orbit_snapshots_spacecraft_epoch_idx ON orbit_snapshots(spacecraft_id, epoch DESC);

CREATE TABLE launch_sites (
    id TEXT PRIMARY KEY,
    name_zh TEXT NOT NULL,
    name_en TEXT NOT NULL,
    country_code TEXT NOT NULL,
    country_name_zh TEXT NOT NULL,
    latitude DOUBLE PRECISION NOT NULL,
    longitude DOUBLE PRECISION NOT NULL,
    description TEXT NOT NULL,
    source_url TEXT NOT NULL
);

CREATE TABLE launch_events (
    external_id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    status_name TEXT NOT NULL,
    status_abbrev TEXT NOT NULL,
    net TIMESTAMPTZ NOT NULL,
    window_start TIMESTAMPTZ,
    window_end TIMESTAMPTZ,
    pad_name TEXT,
    location_name TEXT,
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    mission_name TEXT,
    mission_type TEXT,
    mission_description TEXT,
    provider_name TEXT,
    source_url TEXT NOT NULL,
    raw_payload JSONB NOT NULL,
    synced_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX launch_events_net_idx ON launch_events(net);

CREATE TABLE sync_runs (
    id BIGSERIAL PRIMARY KEY,
    source_code TEXT NOT NULL REFERENCES data_sources(code),
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at TIMESTAMPTZ,
    success BOOLEAN,
    records_written INTEGER NOT NULL DEFAULT 0,
    error_message TEXT
);

INSERT INTO data_sources(code, name, base_url) VALUES
    ('celestrak', 'CelesTrak GP Data', 'https://celestrak.org/'),
    ('launch_library_2', 'Launch Library 2', 'https://ll.thespacedevs.com/')
ON CONFLICT (code) DO NOTHING;

INSERT INTO spacecraft(id, name_zh, name_en, norad_catalog_id, category, operator_name, description, source_code) VALUES
    ('iss', '国际空间站', 'ISS (ZARYA)', 25544, 'LEO · 载人航天', '国际空间站合作伙伴', '持续有人驻留的近地轨道空间实验室。', 'celestrak'),
    ('tianhe', '中国空间站', 'TIANHE', 48274, 'LEO · 载人航天', '中国载人航天工程', '以天和核心舱公开轨道目标代表中国空间站位置。', 'celestrak'),
    ('hubble', '哈勃空间望远镜', 'HUBBLE SPACE TELESCOPE', 20580, 'LEO · 空间科学', 'NASA / ESA', '长期运行的空间光学望远镜。', 'celestrak')
ON CONFLICT (id) DO NOTHING;
INSERT INTO launch_sites(id, name_zh, name_en, country_code, country_name_zh, latitude, longitude, description, source_url) VALUES
    ('wenchang', '文昌航天发射场', 'Wenchang Space Launch Site', 'CN', '中国', 19.6145, 110.9510, '位于海南文昌，承担新一代大型运载火箭和空间站任务。', 'https://www.cnsa.gov.cn/'),
    ('jiuquan', '酒泉卫星发射中心', 'Jiuquan Satellite Launch Center', 'CN', '中国', 40.9606, 100.2983, '中国历史最悠久的航天发射中心之一，承担载人和卫星任务。', 'https://www.cnsa.gov.cn/'),
    ('kennedy', '肯尼迪航天中心', 'Kennedy Space Center', 'US', '美国', 28.5721, -80.6480, 'NASA 重要发射设施，包含 LC-39A 和 LC-39B。', 'https://www.nasa.gov/kennedy/'),
    ('vandenberg', '范登堡太空军基地', 'Vandenberg Space Force Base', 'US', '美国', 34.7420, -120.5724, '适合极地轨道和太阳同步轨道发射。', 'https://www.vandenberg.spaceforce.mil/'),
    ('tanegashima', '种子岛宇宙中心', 'Tanegashima Space Center', 'JP', '日本', 30.4009, 130.9775, 'JAXA 主要大型火箭发射设施。', 'https://global.jaxa.jp/about/centers/tnsc/'),
    ('satish-dhawan', '萨蒂什·达万航天中心', 'Satish Dhawan Space Centre', 'IN', '印度', 13.7199, 80.2304, 'ISRO 的主要轨道发射场。', 'https://www.isro.gov.in/')
ON CONFLICT (id) DO NOTHING;
