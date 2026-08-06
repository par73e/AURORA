-- 机智号：非轨道飞行器、非着陆器，但按用户要求必须单独展示——归入着陆点板块（在毅力号着陆区旁）
-- 类别 AERIAL=动力飞行（直升机），区别于 ROVER_LANDING 巡视探测 / STATIC_LANDER 静态着陆
INSERT INTO mars_landing_sites(
    id, name_zh, name_en, program, operator_name, landing_date,
    latitude, longitude, region, description, sort_order,
    site_name, official_name, mission_name, hardware, side, category, icon
) VALUES
('ingenuity', '机智号', 'Ingenuity', 'MARS 2020', 'NASA / JPL（美国喷气推进实验室）', '2021-04-19',
  18.38, 77.58, '杰泽罗撞击坑（毅力号旁）',
  '首个在地外星体实现受控动力飞行的人造飞行器，随毅力号抵达火星，2021-04-19 完成人类首次地外动力飞行；累计飞行 72 次、航程约 17 公里，2024 年因旋翼损伤结束任务。', 14,
  '机智号起降区', 'Jezero Airfield', '机智号', '["机智号火星直升机", "太阳能板", "双旋翼系统"]', 'NEAR_SIDE', 'AERIAL', 'rover')
ON CONFLICT (id) DO NOTHING;
