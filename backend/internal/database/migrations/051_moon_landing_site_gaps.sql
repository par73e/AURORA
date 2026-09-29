-- 补齐已确认着陆的月球任务。坐标采用 NASA 任务档案 / PDS 公布的实际落点，东经为 0..360°。
-- Luna 13: https://science.nasa.gov/wp-content/uploads/2023/09/DSC_monograph24.pdf
-- Surveyor 5/6: https://science.nasa.gov/mission/surveyor-5/ ; https://science.nasa.gov/mission/surveyor-6/
-- IM-1/IM-2 实际落点: NASA PDS CLPS TO2-IM 与 TO-PRIME1 Athena 档案。
INSERT INTO moon_landing_sites (
  id, name_zh, name_en, program, operator_name, landing_date, latitude, longitude,
  region, description, sort_order, site_name, official_name, mission_name,
  hardware, side, category, icon
) VALUES
  ('luna13', '月球 13 号', 'LUNA 13', 'LUNA PROGRAM', '苏联', '1966-12-24',
   18.87, 297.95, '风暴洋', '苏联第二个成功月面软着陆器，开展月壤力学与辐射测量。', 25,
   '风暴洋着陆点（月球 13 号）', '', '月球 13 号（Luna 13）',
   '["月球 13 号着陆器", "月壤密度测量杆", "辐射计"]'::jsonb,
   'NEAR_SIDE', 'ROBOTIC_LANDER', 'lander'),
  ('surveyor5', '勘测者 5 号', 'SURVEYOR 5', 'SURVEYOR PROGRAM', 'NASA（美国国家航空航天局）', '1967-09-11',
   1.42, 23.20, '静海', '克服推进系统氦泄漏后着陆，首次在月面进行土壤化学分析。', 26,
   '静海着陆点（勘测者 5 号）', '', '勘测者 5 号（Surveyor 5）',
   '["勘测者 5 号着陆器", "电视摄像机", "阿尔法散射分析仪"]'::jsonb,
   'NEAR_SIDE', 'ROBOTIC_LANDER', 'lander'),
  ('surveyor6', '勘测者 6 号', 'SURVEYOR 6', 'SURVEYOR PROGRAM', 'NASA（美国国家航空航天局）', '1967-11-10',
   0.473, 358.573, '中央湾', '在月面短暂点火升空并向西移动约 2.5 米后再次着陆。', 27,
   '中央湾着陆点（勘测者 6 号）', '', '勘测者 6 号（Surveyor 6）',
   '["勘测者 6 号着陆器", "电视摄像机", "阿尔法散射分析仪"]'::jsonb,
   'NEAR_SIDE', 'ROBOTIC_LANDER', 'lander'),
  ('im1-odysseus', '奥德修斯号', 'ODYSSEUS', 'CLPS / IM-1', 'Intuitive Machines / NASA', '2024-02-22',
   -80.13, 1.44, '马拉佩特 A 坑附近', 'IM-1 在月球南极地区着陆，成为首个成功着陆月球的商业月球着陆器。', 28,
   '奥德修斯着陆点', '', 'IM-1（Odysseus）',
   '["Nova-C 奥德修斯着陆器", "NASA CLPS 科学与技术载荷"]'::jsonb,
   'NEAR_SIDE', 'COMMERCIAL_LANDER', 'lander'),
  ('im2-athena', '雅典娜号', 'ATHENA', 'CLPS / IM-2', 'Intuitive Machines / NASA', '2025-03-06',
   -84.79, 29.20, '穆顿山附近', 'IM-2 在月球南极地区着陆后侧翻，任务提前结束，TRIDENT 钻机未能完成预定作业。', 29,
   '雅典娜着陆点', '', 'IM-2（Athena）',
   '["Nova-C 雅典娜着陆器", "PRIME-1 载荷", "TRIDENT 钻机"]'::jsonb,
   'NEAR_SIDE', 'COMMERCIAL_LANDER', 'lander');

-- 列表按实际着陆日期排列，包含此前的历史站点。
WITH ordered AS (
  SELECT id, row_number() OVER (ORDER BY landing_date, id) AS sequence
  FROM moon_landing_sites
)
UPDATE moon_landing_sites AS site
SET sort_order = ordered.sequence
FROM ordered
WHERE site.id = ordered.id;
