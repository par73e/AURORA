-- 鹊桥二号距离微调：16 → 10（3.8 倍月半径）——视觉更协调，方位保持月球远地端（L2 方向）
UPDATE moon_spacecraft
SET stationary_offset_x = -10, stationary_offset_y = 0, stationary_offset_z = 0
WHERE id = 'queqiao2';
