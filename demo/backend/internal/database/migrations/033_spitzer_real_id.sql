-- 修正斯皮策（spitzer）深空探测器数据：
-- 1) NAIF ID：-9 在 JPL Horizons 实为 EscaPADE-Blue（2025-11 发射、2026 停日地 L2 的火星任务），
--    真斯皮策（Spitzer Space Telescope，地球尾随轨道）是 -79。
-- 2) orbit_kind：track → ellipse——启用太阳系视图的拟合椭圆轨道线（像帕克那样画出轨道）。

UPDATE deep_space_probes
SET naif_id = -79, orbit_kind = 'ellipse'
WHERE id = 'spitzer';
