-- 鹊桥二号距离再微调：10 → 7（2.7 倍月半径）——方位保持月球远地端（L2 方向）
UPDATE moon_spacecraft
SET stationary_offset_x = -7, stationary_offset_y = 0, stationary_offset_z = 0
WHERE id = 'queqiao2';
