-- 鹊桥二号位置再修正：应位于月球正后方（地月 L2 方向）
-- 真实 L2 距月心约 64500km ≈ 37 倍月半径；场景可观察范围内取正后方 6.2 倍月半径（16 单位）示意
-- 上一版 [-3.2, 0.8, -0.6] 的 y/z 偏移使中继星从月背边缘"探出"，近地面视角可见——方向不纯
UPDATE moon_spacecraft
SET stationary_offset_x = -16, stationary_offset_y = 0, stationary_offset_z = 0
WHERE id = 'queqiao2';
