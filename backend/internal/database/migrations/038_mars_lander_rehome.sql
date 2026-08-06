-- 着陆器归位：地表探测器/失败着陆器从航天器名录移到着陆点板块（着陆器=着陆点，航天器只留飞行器）
-- 1) 9 个着陆器（真实经纬度）进入 mars_landing_sites（排在既有 4 个着陆点之后）
INSERT INTO mars_landing_sites(
    id, name_zh, name_en, program, operator_name, landing_date,
    latitude, longitude, region, description, sort_order,
    site_name, official_name, mission_name, hardware, side, category, icon
) VALUES
('viking1', '海盗 1 号', 'Viking 1', 'VIKING', 'NASA（美国国家航空航天局）', '1976-07-20',
  22.48, 310.03, '克里斯平原', '美国首次成功在火星表面软着陆并长期工作的探测器，携带早期生命探测实验（生物学箱）与高分辨率相机，工作至 1982 年。', 5,
  '克里斯平原着陆点', 'Chryse Planitia', '海盗 1 号', '["海盗 1 号着陆器", "生物学实验箱", "地震仪"]', 'NEAR_SIDE', 'STATIC_LANDER', 'lander'),
('viking2', '海盗 2 号', 'Viking 2', 'VIKING', 'NASA（美国国家航空航天局）', '1976-09-03',
  47.64, 225.71, '乌托邦平原', '与海盗 1 号构成双着陆点，开展地貌、地震与气象长期观测，工作至 1980 年。', 6,
  '乌托邦平原着陆点', 'Utopia Planitia', '海盗 2 号', '["海盗 2 号着陆器", "机械臂采样器", "气象站"]', 'NEAR_SIDE', 'STATIC_LANDER', 'lander'),
('pathfinder', '火星探路者', 'Mars Pathfinder', 'MARS PATHFINDER', 'NASA（美国国家航空航天局）', '1997-07-04',
  19.13, 326.75, '阿瑞斯谷', '固定着陆平台 + 释放索杰纳号火星车，验证"着陆器 + 巡视器"低成本任务模式，传回 1.6 万张影像。', 7,
  '阿瑞斯谷着陆点', 'Ares Vallis', '火星探路者', '["探路者着陆平台", "索杰纳号火星车"]', 'NEAR_SIDE', 'STATIC_LANDER', 'lander'),
('sojourner', '索杰纳号', 'Sojourner', 'MARS PATHFINDER', 'NASA（美国国家航空航天局）', '1997-07-04',
  19.13, 326.75, '阿瑞斯谷（探路者着陆点旁）', '第一辆在火星表面行驶的火星车，以 10cm/s 的速度在着陆点周边巡视并分析岩石成分，验证了地外巡视探测的可行性；与探路者着陆器分开记录。', 8,
  '索杰纳号巡视区', 'Sojourner Traverse', '索杰纳号', '["索杰纳号火星车"]', 'NEAR_SIDE', 'ROVER_LANDING', 'rover'),
('spirit', '勇气号', 'Spirit', 'MER-A', 'NASA（美国国家航空航天局）', '2004-01-04',
 -14.57, 175.47, '古瑟夫撞击坑', '火星双胞胎漫游车之一，原计划 90 天，超额服役至 2010 年；发现碳酸盐沉积等水蚀痕迹。', 9,
  '古瑟夫着陆点', 'Gusev Crater', '勇气号', '["勇气号火星车"]', 'NEAR_SIDE', 'ROVER_LANDING', 'rover'),
('opportunity', '机遇号', 'Opportunity', 'MER-B', 'NASA（美国国家航空航天局）', '2004-01-25',
  -1.95, 354.47, '子午线高原', '火星探测史上最传奇的漫游车之一，原计划 90 天实际工作近 15 年、行驶 45 公里，发现火星曾存在液态水的关键证据。', 10,
  '子午线高原着陆点', 'Meridiani Planum', '机遇号', '["机遇号火星车"]', 'NEAR_SIDE', 'ROVER_LANDING', 'rover'),
('phoenix', '凤凰号', 'Phoenix', 'PHOENIX', 'NASA（美国国家航空航天局）', '2008-05-25',
  68.22, 234.25, '北极区（瓦斯塔蒂斯-博雷利斯平原）', '固定着陆器，机械臂挖掘确认火星地表下水冰的存在，并观测极地降雪。', 11,
  '凤凰号着陆区', 'Vastitas Borealis', '凤凰号', '["凤凰号着陆器", "机械臂", "热与电导率探测器"]', 'NEAR_SIDE', 'STATIC_LANDER', 'lander'),
('beagle2', '猎兔犬 2 号', 'Beagle 2', 'BEAGLE 2', '英国 / ESA（欧洲空间局）', '2003-12-25',
  11.53,  90.43, '伊西迪斯平原', '随火星快车发射的着陆器，着陆后失联；2015 年 HiRISE 影像确认其以太阳能板未完全展开的状态降落（任务失败）。', 12,
  '伊西迪斯平原着陆点', 'Isidis Planitia', '猎兔犬 2 号', '["猎兔犬 2 号着陆器"]', 'NEAR_SIDE', 'STATIC_LANDER', 'lander'),
('mars3-lander', '火星 3 号着陆器', 'Mars 3 Lander', 'MARS 3', '苏联', '1971-12-02',
 -45.00, 202.00, '阿尔西达平原（约）', '人类首个在火星表面完成软着陆的探测器，但着陆后约 14.5 秒即与地面失联，仅传回部分画面信号。', 13,
  '火星 3 号着陆区（约）', 'Aridal Planitia (approx.)', '火星 3 号', '["火星 3 号着陆器", "PROP-M 微型火星车（未部署）"]', 'NEAR_SIDE', 'STATIC_LANDER', 'lander')
ON CONFLICT (id) DO NOTHING;

-- 2) 从 mars_spacecraft 删除已移入着陆点的条目（航天器名录只保留 16 个飞行器：
--    3 实时轨道器 + 13 历史轨道器；火星极地着陆器着陆失败无坐标，不再展示）
DELETE FROM mars_spacecraft WHERE id IN (
  'viking1', 'viking2', 'pathfinder', 'sojourner', 'spirit', 'opportunity', 'phoenix',
  'curiosity', 'insight', 'perseverance', 'ingenuity',
  'mars-polar-lander', 'mars3-lander', 'beagle2'
);
