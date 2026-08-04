-- 月球场景新增 8 项（含历史任务）：鹊桥号/CAPSTONE/月船2号/Danuri/辉夜姬/月船1号/嫦娥一号/GRAIL 双星
-- 视觉轨道参数为场景单位静态示意（与 LRO/鹊桥二号同模式）；地月 L2 晕轨/近直线晕轨用定点或高偏椭圆近似。
INSERT INTO moon_spacecraft(
    id, name_zh, name_en, type, operator_name, description,
    launch_date, launch_site, launch_vehicle, source_name,
    display_inclination, display_eccentricity, display_period,
    kind, orbit_a, orbit_e, inclination_deg, raan_deg, arg_periapsis_deg, period_seconds,
    stationary_offset_x, stationary_offset_y, stationary_offset_z, sort_order
) VALUES
(
    'queqiao', '鹊桥号中继星', 'QUEQIAO', 'CNSA RELAY SATELLITE', 'CNSA（中国国家航天局）',
    '嫦娥四号月背探测任务的地月拉格朗日 L2 中继星，人类首个地月 L2 晕轨道通信卫星；本页示意为定点。',
    '2018-05-21', '西昌卫星发射中心（中国）', '长征四号丙', 'CNSA 公开轨道信息（示意定点）',
    '55', '≈ 0.99', '约 14 天',
    'stationary', 0, 0, 0, 0, 0, 0,
    3.0, -1.2, 0.8, 3
),
(
    'capstone', 'CAPSTONE', 'CAPSTONE', 'NASA CUBESAT', 'NASA / Advanced Space',
    '验证未来 Gateway 空间站的近直线晕轨道（NRHO）的小型立方星，2022 年进入月球 NRHO。',
    '2022-06-28', '新西兰马希亚半岛（新西兰）', 'Electron', 'NASA 公开轨道信息（视觉近似）',
    '26', '≈ 0.65', '约 6.5 天',
    'orbital', 3.4, 0.55, 28, 120, 0, 900,
    0, 0, 0, 4
),
(
    'chandrayaan-2', '月船2号轨道器', 'CHANDRAYAAN-2', 'ISRO LUNAR ORBITER', 'ISRO（印度空间研究组织）',
    '印度重要月球遥感轨道器，与维克拉姆着陆器一同发射，至今持续执行月球测绘与科学观测。',
    '2019-07-22', '萨蒂什·达万航天中心（印度）', 'GSLV Mk III', 'ISRO 公开轨道信息（标称轨道）',
    '90', '≈ 0.01', '约 2 小时',
    'orbital', 2.6, 0.01, 90, 20, 0, 480,
    0, 0, 0, 5
),
(
    'danuri', 'Danuri（KPLO）', 'DANURI', 'KARI LUNAR ORBITER', 'KARI（韩国航空宇宙研究院）',
    '韩国首个月球轨道器，执行月球测绘、水冰探测与未来着陆点选址任务。',
    '2022-08-04', '卡纳维拉尔角 SLC-40（美国）', 'Falcon 9', 'KARI 公开轨道信息（标称轨道）',
    '90', '≈ 0', '约 2 小时',
    'orbital', 2.55, 0.01, 90, 110, 0, 360,
    0, 0, 0, 6
),
(
    'kaguya', '辉夜姬号', 'KAGUYA (SELENE)', 'JAXA LUNAR ORBITER', 'JAXA（日本宇宙航空研究开发机构）',
    '日本旗舰月球探测器，主卫星配两枚子卫星，完成高精度全球测绘，2009 年任务结束撞月。',
    '2007-09-14', '种子岛宇宙中心（日本）', 'H-IIA 2024', 'JAXA 公开轨道信息（任务期标称轨道）',
    '90', '≈ 0.01', '约 2 小时',
    'orbital', 2.7, 0.01, 90, 60, 0, 420,
    0, 0, 0, 7
),
(
    'chandrayaan-1', '月船1号', 'CHANDRAYAAN-1', 'ISRO LUNAR ORBITER', 'ISRO（印度空间研究组织）',
    '印度首颗月球探测器，携 NASA 雷达等载荷确认月表水冰与羟基信号，2009 年失联。',
    '2008-10-22', '萨蒂什·达万航天中心（印度）', 'PSLV-XL', 'ISRO 公开轨道信息（任务期标称轨道）',
    '90', '≈ 0.01', '约 2 小时',
    'orbital', 2.6, 0.01, 90, 150, 0, 450,
    0, 0, 0, 8
),
(
    'change-1', '嫦娥一号', 'CHANGE-1', 'CNSA LUNAR ORBITER', 'CNSA（中国国家航天局）',
    '中国首颗月球探测卫星，完成全月面立体成像与元素分布探测，2009 年受控撞月。',
    '2007-10-24', '西昌卫星发射中心（中国）', '长征三号甲', 'CNSA 公开轨道信息（任务期标称轨道）',
    '90', '≈ 0', '约 2 小时',
    'orbital', 2.65, 0.01, 90, 200, 0, 400,
    0, 0, 0, 9
),
(
    'grail-a', 'GRAIL-A（埃布）', 'GRAIL-A (EBB)', 'NASA LUNAR ORBITER', 'NASA（美国国家航空航天局）',
    'GRAIL 双星编队之一，与 GRAIL-B 前后同轨精密测量月球重力场，2012 年任务结束撞月。',
    '2011-09-10', '卡纳维拉尔角（美国）', 'Delta II Heavy', 'NASA 公开轨道信息（任务期标称轨道）',
    '90', '≈ 0', '约 2 小时',
    'orbital', 2.5, 0.01, 90, 250, 0, 360,
    0, 0, 0, 10
),
(
    'grail-b', 'GRAIL-B（福楼）', 'GRAIL-B (FLOW)', 'NASA LUNAR ORBITER', 'NASA（美国国家航空航天局）',
    'GRAIL 双星编队之一，与 GRAIL-A 保持前后编队测月，共同完成高分辨率月球重力场测绘。',
    '2011-09-10', '卡纳维拉尔角（美国）', 'Delta II Heavy', 'NASA 公开轨道信息（任务期标称轨道）',
    '90', '≈ 0', '约 2 小时',
    'orbital', 2.48, 0.01, 90, 255, 0, 355,
    0, 0, 0, 11
)
ON CONFLICT (id) DO NOTHING;
