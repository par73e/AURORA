-- 火星：绕行器目录 + 轨道快照 + 着陆点（镜像月球数据链路；用户确认：MRO/MAVEN/天问一号 + 毅力/好奇/祝融/洞察）
-- 视觉轨道半径采用夸张比例（沿用月球既有做法），公转周期全部为真实值。

CREATE TABLE mars_spacecraft (
    id TEXT PRIMARY KEY,
    name_zh TEXT NOT NULL,
    name_en TEXT NOT NULL,
    type TEXT NOT NULL,               -- 面板 kicker
    operator_name TEXT NOT NULL,      -- 运营方/机构
    description TEXT NOT NULL,        -- 任务说明
    launch_date TEXT NOT NULL,
    launch_site TEXT NOT NULL,
    launch_vehicle TEXT NOT NULL,
    source_name TEXT NOT NULL,
    display_inclination TEXT NOT NULL,
    display_eccentricity TEXT NOT NULL,
    display_period TEXT NOT NULL,
    -- 三维场景视觉轨道参数
    kind TEXT NOT NULL,               -- 'orbital' 绕火轨道
    orbit_a DOUBLE PRECISION NOT NULL DEFAULT 0,   -- 半长轴（场景单位，夸张）
    orbit_e DOUBLE PRECISION NOT NULL DEFAULT 0,   -- 离心率（真实）
    inclination_deg DOUBLE PRECISION NOT NULL DEFAULT 0,
    raan_deg DOUBLE PRECISION NOT NULL DEFAULT 0,
    arg_periapsis_deg DOUBLE PRECISION NOT NULL DEFAULT 0,
    period_seconds DOUBLE PRECISION NOT NULL DEFAULT 0, -- 公转周期（真实秒）
    stationary_offset_x DOUBLE PRECISION NOT NULL DEFAULT 0,
    stationary_offset_y DOUBLE PRECISION NOT NULL DEFAULT 0,
    stationary_offset_z DOUBLE PRECISION NOT NULL DEFAULT 0,
    sort_order INTEGER NOT NULL DEFAULT 0
);

INSERT INTO mars_spacecraft(
    id, name_zh, name_en, type, operator_name, description,
    launch_date, launch_site, launch_vehicle, source_name,
    display_inclination, display_eccentricity, display_period,
    kind, orbit_a, orbit_e, inclination_deg, raan_deg, arg_periapsis_deg, period_seconds,
    stationary_offset_x, stationary_offset_y, stationary_offset_z, sort_order
) VALUES
(
    'mro', '火星勘测轨道飞行器', 'MRO', 'NASA MARS ORBITER', 'NASA（美国国家航空航天局）',
    '2005 年发射的火星高分辨率侦察轨道器，携带 HiRISE 相机持续拍摄火星表面，并为好奇号/毅力号等火星车提供 UHF 中继通信；2006 年入轨后工作至今。',
    '2005-08-12', '卡纳维拉尔角（美国）', 'Atlas V 401', 'NASA MRO 任务公开数据（标称轨道）',
    '93', '≈ 0.009', '约 112 分钟',
    'orbital', 3.35, 0.009, 93, 30, 0, 6720,
    0, 0, 0, 1
),
(
    'maven', 'MAVEN 火星大气探测器', 'MAVEN', 'NASA MARS ORBITER', 'NASA（美国国家航空航天局）',
    '火星大气与挥发物演化任务（Mars Atmosphere and Volatile EvolutioN），2013 年发射，研究火星大气逃逸过程；长期作为火星车的中继通信节点。',
    '2013-11-18', '卡纳维拉尔角（美国）', 'Atlas V 401', 'NASA MAVEN 任务公开数据（标称轨道）',
    '74', '≈ 0.46', '约 4.5 小时',
    'orbital', 4.10, 0.461, 74, 20, 180, 16148,
    0, 0, 0, 2
),
(
    'tianwen1', '天问一号轨道器', 'TIANWEN-1', 'CNSA MARS ORBITER', 'CNSA（中国国家航天局）',
    '中国首次火星探测任务轨道器，2021 年 2 月进入环火轨道，释放祝融号着陆巡视器后继续长期科学探测（中分辨率相机、矿物光谱仪等），并承担中继通信。',
    '2020-07-23', '文昌航天发射场（中国）', '长征五号', 'CNSA 公开轨道信息（标称轨道）',
    '86', '≈ 0.62', '约 7.8 小时',
    'orbital', 5.00, 0.616, 86, 350, 90, 28210,
    0, 0, 0, 3
)
ON CONFLICT (id) DO NOTHING;

-- 火星绕行器轨道快照（JPL Horizons 日同步，镜像 moon_orbit_snapshots）
CREATE TABLE mars_orbit_snapshots (
    id BIGSERIAL PRIMARY KEY,
    spacecraft_id TEXT NOT NULL REFERENCES mars_spacecraft(id) ON DELETE CASCADE,
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
CREATE INDEX mars_orbit_snapshots_spacecraft_epoch_idx ON mars_orbit_snapshots(spacecraft_id, epoch DESC);

-- 火星着陆点（真实经纬度；与月球着陆点同模型：地点/设施分离 + 图标 + 行驶轨迹）
CREATE TABLE mars_landing_sites (
    id TEXT PRIMARY KEY,
    name_zh TEXT NOT NULL,
    name_en TEXT NOT NULL,
    program TEXT NOT NULL,
    operator_name TEXT NOT NULL,
    landing_date TEXT NOT NULL,
    latitude DOUBLE PRECISION NOT NULL,
    longitude DOUBLE PRECISION NOT NULL,
    region TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    sort_order INT NOT NULL DEFAULT 0,
    site_name TEXT NOT NULL DEFAULT '',
    official_name TEXT NOT NULL DEFAULT '',
    mission_name TEXT NOT NULL DEFAULT '',
    hardware JSONB NOT NULL DEFAULT '[]'::jsonb,
    side TEXT NOT NULL DEFAULT 'NEAR_SIDE',
    category TEXT NOT NULL DEFAULT 'ROBOTIC_LANDER',
    icon TEXT NOT NULL DEFAULT 'lander',
    track JSONB NOT NULL DEFAULT '[]'::jsonb
);

INSERT INTO mars_landing_sites(
    id, name_zh, name_en, program, operator_name, landing_date,
    latitude, longitude, region, description, sort_order,
    site_name, official_name, mission_name, hardware, side, category, icon
) VALUES
('perseverance', '毅力号', 'PERSEVERANCE', 'MARS 2020', 'NASA（美国国家航空航天局）', '2021-02-18',
  18.38,  77.58, '杰泽罗撞击坑', 'NASA 最新一代火星车，在古河流三角洲遗迹中寻找古代生命痕迹并采集样本（与未来的火星采样返回任务衔接）；携带机智号直升机完成人类首次地外飞行。', 1,
  '杰泽罗陨击坑着陆点', 'Octavia E. Butler Landing', 'Mars 2020', '["毅力号火星车", "机智号直升机（已退役）", "MOXIE 制氧实验装置"]', 'NEAR_SIDE', 'ROVER_LANDING', 'rover'),
('curiosity',    '好奇号',    'CURIOSITY',    'MSL',          'NASA（美国国家航空航天局）', '2012-08-06',
  -4.59, 137.44, '盖尔撞击坑', '火星科学实验室（MSL）火星车，在夏普山（Aeolis Mons）山麓勘察含水矿物层序，验证火星曾具备宜居环境。', 2,
  '布拉德伯里着陆点', 'Bradbury Landing', 'Mars Science Laboratory', '["好奇号火星车", "火星样品分析设备（SAM）"]', 'NEAR_SIDE', 'ROVER_LANDING', 'rover'),
('zhurong',      '祝融号',    'ZHU RONG',     'TIANWEN-1',    'CNSA（中国国家航天局）', '2021-05-15',
  25.10, 109.70, '乌托邦平原', '天问一号着陆巡视器，中国首次火星软着陆；祝融号火星车在乌托邦平原开展巡视探测（2021—2022），之后天问一号轨道器继续中继与科学探测。', 3,
  '天问一号着陆区', 'Tianwen-1 Landing Area', 'Tianwen-1', '["天问一号着陆巡视器", "祝融号火星车"]', 'NEAR_SIDE', 'ROVER_LANDING', 'rover'),
('insight',      '洞察号',    'INSIGHT',      'INSIGHT',      'NASA（美国国家航空航天局）', '2018-11-26',
   4.50, 135.62, '埃律西昂平原', '洞察号（InSight）静默式着陆器，通过 SEIS 地震仪首次记录火星震与流星撞击信号，并测量火星内核结构（任务结束于 2022 年）。', 4,
  '霍姆斯特德洼地', 'Homestead Hollow', 'InSight', '["洞察号着陆器", "SEIS 地震仪", "HP³ 热流探头"]', 'NEAR_SIDE', 'STATIC_LANDER', 'lander')
ON CONFLICT (id) DO NOTHING;
