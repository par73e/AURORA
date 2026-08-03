-- LRO 起始位置修正：RAAN 40° 使初始点落在月球背面（被遮挡且转得慢）；
-- 改为 -50°，初始点位于相机一侧（z 正方向），进入页面即可见
UPDATE moon_spacecraft SET raan_deg = -50 WHERE id = 'lro';
