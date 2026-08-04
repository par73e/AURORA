-- 清理 CAPSTONE 残留的实时快照：其 NRHO 半长轴（~42300km）经月面尺度夸张后达
-- ~185 单位无法合理显示，已从 Horizons 支持映射移除并回退静态 NRHO 示意轨道；
-- 旧快照若不清除，前端仍会优先使用快照画出巨大轨道。
DELETE FROM moon_orbit_snapshots WHERE spacecraft_id = 'capstone';
