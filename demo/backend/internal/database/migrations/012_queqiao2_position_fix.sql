-- 鹊桥二号位置修正：应位于月球远地端（地月 L2 区域，服务月背任务的嫦娥四号/六号）
-- 原 offset [2.9, 1.0, -1.6] 的 x 为正（朝向地球/相机方向 = 月面近地面一侧）——方向错误
-- 修正为月背外侧：x 为负（远离地球），距月心约 3.4（晕轨道位置示意）
UPDATE moon_spacecraft
SET stationary_offset_x = -3.2, stationary_offset_y = 0.8, stationary_offset_z = -0.6
WHERE id = 'queqiao2';
