-- 新增已完成的绕月/飞掠任务、在途火星双星，以及地球轨道太阳观测站。
-- 历史任务与在途任务使用独立 kind，避免被 Horizons 当成当前绕目标天体运行的航天器。
INSERT INTO moon_spacecraft (
  id, name_zh, name_en, type, operator_name, description,
  launch_date, launch_site, launch_vehicle, source_name,
  display_inclination, display_eccentricity, display_period,
  kind, orbit_a, orbit_e, inclination_deg, raan_deg, arg_periapsis_deg, period_seconds,
  stationary_offset_x, stationary_offset_y, stationary_offset_z, sort_order
) VALUES
('luna-10', '月球 10 号', 'Luna 10', 'HISTORIC LUNAR ORBITER', '苏联',
 '1966 年成为首个人造月球卫星，完成月球引力场与辐射环境观测。场景显示任务期轨道示意，并非当前星历。',
 '1966-03-31', '拜科努尔航天发射场', 'Molniya-M', 'NASA Moon Missions · 任务期轨道示意',
 '约 72', '任务期示意', '约 3 小时', 'historical_orbit', 3.05, 0.09, 72, 300, 20, 10800, 0, 0, 0, 12),
('apollo-8', '阿波罗 8 号', 'Apollo 8', 'CREWED LUNAR ORBITER', 'NASA',
 '1968 年首次载人绕月飞行，完成 10 圈月球轨道后返回地球。场景显示历史绕月轨道示意。',
 '1968-12-21', '肯尼迪航天中心 LC-39A', 'Saturn V', 'NASA Apollo 8 Mission Details · 任务期轨道示意',
 '任务期示意', '任务期示意', '约 2 小时', 'historical_orbit', 3.32, 0.07, 35, 235, 12, 7200, 0, 0, 0, 13),
('artemis-ii', '阿耳忒弥斯二号', 'Artemis II Orion', 'CREWED LUNAR FLYBY', 'NASA / CSA',
 '2026-04-01 发射，4 月 6 日完成载人绕月飞掠，4 月 10 日返回地球；未进入月球轨道。场景弧线为历史飞掠示意。',
 '2026-04-01', '肯尼迪航天中心 LC-39B', 'SLS Block 1', 'NASA Artemis II · 历史飞掠示意',
 '不适用', '不适用', '飞掠，不适用', 'flyby', 3.95, 0, 32, 155, 0, 0, 0, 0, 0, 14)
ON CONFLICT (id) DO NOTHING;

INSERT INTO mars_spacecraft (
  id, name_zh, name_en, type, operator_name, description,
  launch_date, launch_site, launch_vehicle, source_name,
  display_inclination, display_eccentricity, display_period,
  kind, catalog_group, orbit_a, orbit_e, inclination_deg, raan_deg, arg_periapsis_deg, period_seconds,
  stationary_offset_x, stationary_offset_y, stationary_offset_z, sort_order
) VALUES
('escapade-blue', 'ESCAPADE 蓝号', 'ESCAPADE Blue', 'MARS-BOUND SPACECRAFT', 'NASA / UC Berkeley',
 '双星任务之一。2025-11-13 发射，2026 年仍处于地球附近巡航阶段；计划 2027-09 抵达火星。场景仅示意前往火星的方向，不表示当前位置或已入火星轨道。',
 '2025-11-13', '卡纳维拉尔角 SLC-36', 'New Glenn', 'NASA ESCAPADE · 在途方向示意',
 '不适用', '不适用', '尚未入火星轨道', 'approach', 'orbit', 7.7, 0, 18, 24, 0, 0, 0, 0, 0, 8),
('escapade-gold', 'ESCAPADE 金号', 'ESCAPADE Gold', 'MARS-BOUND SPACECRAFT', 'NASA / UC Berkeley',
 '双星任务之二，与蓝号协同研究火星磁层及太阳风；计划 2027-09 抵达火星。场景仅示意前往火星的方向，不表示当前位置或已入火星轨道。',
 '2025-11-13', '卡纳维拉尔角 SLC-36', 'New Glenn', 'NASA ESCAPADE · 在途方向示意',
 '不适用', '不适用', '尚未入火星轨道', 'approach', 'orbit', 8.1, 0, -12, 32, 0, 0, 0, 0, 0, 9)
ON CONFLICT (id) DO NOTHING;

-- 两颗太阳观测站都绕地球运行，交由 ORBIT 的 CelesTrak/SGP4 链路显示真实轨道。
INSERT INTO spacecraft(id, name_zh, name_en, norad_catalog_id, category, operator_name, description, source_code,
                       launch_date, launch_site, launch_vehicle) VALUES
('sdo', '太阳动力学观测站', 'SDO', 36395, 'GEO · 太阳观测', 'NASA',
 '在地球倾斜地球同步轨道连续观测太阳活动；研究太阳，不绕太阳运行。', 'celestrak',
 '2010-02-11', '卡纳维拉尔角 SLC-41', 'Atlas V 401'),
('hinode', '日出号', 'HINODE (SOLAR-B)', 29479, 'SSO · 太阳观测', 'JAXA / NASA / ESA',
 '在地球太阳同步轨道观测太阳磁场与日冕；研究太阳，不绕太阳运行。', 'celestrak',
 '2006-09-23', '内之浦宇宙空间观测所', 'M-V')
ON CONFLICT (id) DO NOTHING;
