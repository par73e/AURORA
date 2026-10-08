-- 月球飞行器目录：静态 curation 数据（与地球 spacecraft 表同性质），
-- 包含面板展示字段与三维场景视觉轨道参数（数据驱动渲染）。
CREATE TABLE moon_spacecraft (
    id TEXT PRIMARY KEY,
    name_zh TEXT NOT NULL,
    name_en TEXT NOT NULL,
    type TEXT NOT NULL,               -- 面板 kicker（如 NASA LUNAR ORBITER）
    operator_name TEXT NOT NULL,      -- 运营方/机构
    description TEXT NOT NULL,        -- 任务说明
    launch_date TEXT NOT NULL,
    launch_site TEXT NOT NULL,
    launch_vehicle TEXT NOT NULL,
    source_name TEXT NOT NULL,
    display_inclination TEXT NOT NULL, -- 面板展示：轨道倾角（度）
    display_eccentricity TEXT NOT NULL, -- 面板展示：偏心率
    display_period TEXT NOT NULL,      -- 面板展示：轨道周期
    -- 三维场景视觉轨道参数
    kind TEXT NOT NULL,                -- 'orbital' 绕月轨道 / 'stationary' 定点（拉格朗日示意）
    orbit_a DOUBLE PRECISION NOT NULL DEFAULT 0,   -- 半长轴（场景单位）
    orbit_e DOUBLE PRECISION NOT NULL DEFAULT 0,   -- 离心率
    inclination_deg DOUBLE PRECISION NOT NULL DEFAULT 0, -- 轨道倾角
    raan_deg DOUBLE PRECISION NOT NULL DEFAULT 0,   -- 升交点经度
    arg_periapsis_deg DOUBLE PRECISION NOT NULL DEFAULT 0, -- 近心点幅角
    period_seconds DOUBLE PRECISION NOT NULL DEFAULT 0,    -- 视觉公转周期
    stationary_offset_x DOUBLE PRECISION NOT NULL DEFAULT 0,
    stationary_offset_y DOUBLE PRECISION NOT NULL DEFAULT 0,
    stationary_offset_z DOUBLE PRECISION NOT NULL DEFAULT 0,
    sort_order INTEGER NOT NULL DEFAULT 0
);

INSERT INTO moon_spacecraft(
    id, name_zh, name_en, type, operator_name, description,
    launch_date, launch_site, launch_vehicle, source_name,
    display_inclination, display_eccentricity, display_period,
    kind, orbit_a, orbit_e, inclination_deg, raan_deg, arg_periapsis_deg, period_seconds,
    stationary_offset_x, stationary_offset_y, stationary_offset_z, sort_order
) VALUES
(
    'lro', '月球勘测轨道飞行器', 'LRO', 'NASA LUNAR ORBITER', 'NASA（美国国家航空航天局）',
    '对月球表面进行高精度测绘（LOLA 激光测高、LROC 影像），探测极区水冰与辐射环境；2009 年与 LCROSS 协同完成月背撞击探测，为后续载人登月选址提供数据。',
    '2009-06-18', '卡纳维拉尔角（美国）', 'Atlas V 401', 'NASA LRO 任务公开数据（标称轨道）',
    '90', '≈ 0.001', '约 113 分钟',
    'orbital', 2.675, 0, 90, 40, 0, 300,
    0, 0, 0, 1
),
(
    'queqiao2', '鹊桥二号中继星', 'QUEQIAO-2', 'CNSA RELAY SATELLITE', 'CNSA（中国国家航天局）',
    '运行于地月拉格朗日 L2 点附近的晕轨道，为嫦娥六号等月背采样任务提供地月中继通信，并携带极紫外相机等科学载荷；本页示意为定点。',
    '2024-03-20', '文昌航天发射场（中国）', '长征八号', 'CNSA 公开轨道信息（示意定点）',
    '57', '≈ 0.93', '约 24 小时',
    'stationary', 0, 0, 0, 0, 0, 0,
    2.9, 1.0, -1.6, 2
)
ON CONFLICT (id) DO NOTHING;
