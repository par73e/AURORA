-- 修正月球新增飞行器的视觉轨道半径（bug 修复）
-- 此前 orbit_a 取值 2.48–2.7，落在月球半径（2.6）附近；经
-- exaggeratedA = 2.6 + max(0, a − 2.6) × 3 钳制后全部回到月面 → 标记点被月球网格
-- 遮挡不可见（只有屏幕空间标签在绕月球转）。
-- 改为按真实轨道半径 × MOON_SCENE_SCALE（2.6/1737.4 km/单位）取值，夸张后均清晰
-- 高于月面：极轨器视半径 ≈ 3.0（与 LRO 一致）、CAPSTONE 取 NRHO 示意（视半径 ≈ 4.0）。
UPDATE moon_spacecraft SET orbit_a = 2.75 WHERE id IN ('chandrayaan-2', 'danuri', 'kaguya', 'chandrayaan-1'); -- ~100km 极轨（≈ LRO 视半径）
UPDATE moon_spacecraft SET orbit_a = 2.90 WHERE id = 'change-1'; -- 嫦娥一号 ~200km 环月
UPDATE moon_spacecraft SET orbit_a = 2.68 WHERE id = 'grail-a';
UPDATE moon_spacecraft SET orbit_a = 2.67 WHERE id = 'grail-b'; -- GRAIL ~50km 极轨
UPDATE moon_spacecraft SET orbit_a = 3.07 WHERE id = 'capstone'; -- NRHO 示意（明显高于极轨器）
