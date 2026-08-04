-- CAPSTONE NRHO 示意轨道修正：偏心 0.55 → 0.3
-- 原 e=0.55 时近月点 a(1−e)=4.0×0.45=1.8 < 月球半径 2.6，圆点每圈穿入月球内部
-- 被遮挡（与"只有标签没有点"同类问题）；e=0.3 近月点 2.8 保持在月面之外。
UPDATE moon_spacecraft SET orbit_e = 0.3 WHERE id = 'capstone';
