-- 月球绕月飞行器视觉轨道分层 + 倾角微调
-- 背景：此前 6 颗静态轨道（danuri/kaguya/chandrayaan-1/change-1/grail-a/b）全是
--       90° 极轨、半径 2.67–2.9（~100–300km 高度），视觉上密集在同一平面、同一高度束。
-- 说明：真实情况确实是几乎所有月球轨道器都走 ~90° 极轨 ~100km 高度（因此密集）；
--       这里按真实轨道高度把视觉半径分成几层、并给倾角 ±2° 微差，让每颗可区分，
--       仍然贴近期实（倾角都在 87.5–92° 的极轨带内，半径对应真实 100–300km 高度）。
-- 注意：lro / chandrayaan-2 使用 JPL 实时快照（快照优先），不在此调整；capstone 已分层（28°、3.07）。

-- 高度分层（orbit_a 越大 = 显示越高）
UPDATE moon_spacecraft SET orbit_a = 2.62, inclination_deg = 89.5 WHERE id = 'grail-a';
UPDATE moon_spacecraft SET orbit_a = 2.64, inclination_deg = 90.5 WHERE id = 'grail-b';
UPDATE moon_spacecraft SET orbit_a = 2.72, inclination_deg = 92.0 WHERE id = 'chandrayaan-1';
UPDATE moon_spacecraft SET orbit_a = 2.80, inclination_deg = 90.0 WHERE id = 'danuri';
UPDATE moon_spacecraft SET orbit_a = 2.85, inclination_deg = 87.5 WHERE id = 'kaguya';
UPDATE moon_spacecraft SET orbit_a = 2.90, inclination_deg = 88.0 WHERE id = 'change-1';

-- RAAN 再均匀错开（避免极轨在极点交会成束）
UPDATE moon_spacecraft SET raan_deg = 30  WHERE id = 'danuri';
UPDATE moon_spacecraft SET raan_deg = 90  WHERE id = 'kaguya';
UPDATE moon_spacecraft SET raan_deg = 150 WHERE id = 'chandrayaan-1';
UPDATE moon_spacecraft SET raan_deg = 210 WHERE id = 'change-1';
UPDATE moon_spacecraft SET raan_deg = 260 WHERE id = 'grail-a';
UPDATE moon_spacecraft SET raan_deg = 320 WHERE id = 'grail-b';
