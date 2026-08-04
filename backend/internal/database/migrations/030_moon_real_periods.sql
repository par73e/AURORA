-- 月球静态轨道公转周期改为真实值
-- 背景：此前无 JPL 快照的静态轨道用视觉压缩周期（6-8 分钟），比真实周期（约 2 小时）
--       快 10-20 倍；CAPSTONE 用 900s（15 分钟）而真实 NRHO 周期约 6.5 天。
--       面板 display_period 文字一直是真实值（"约 2 小时"/"约 6.5 天"），本次让
--       period_seconds 与文字一致。LRO/月船2 有 JPL 快照（真实周期优先），此处仅改回退值。
-- 原则：公转周期必须真实（位置允许偏差，速度/周期不允许）。

UPDATE moon_spacecraft SET period_seconds = 6780   WHERE id = 'lro';           -- 113 分钟（快照回退值，与面板"约 113 分钟"一致）
UPDATE moon_spacecraft SET period_seconds = 7080   WHERE id = 'chandrayaan-2'; -- 118 分钟（快照回退值，与面板"约 2 小时"一致）
UPDATE moon_spacecraft SET period_seconds = 6840   WHERE id = 'chandrayaan-1'; -- 114 分钟
UPDATE moon_spacecraft SET period_seconds = 7620   WHERE id = 'change-1';      -- 127 分钟
UPDATE moon_spacecraft SET period_seconds = 7080   WHERE id = 'danuri';        -- 118 分钟
UPDATE moon_spacecraft SET period_seconds = 7080   WHERE id = 'kaguya';        -- 118 分钟
UPDATE moon_spacecraft SET period_seconds = 6810   WHERE id = 'grail-a';       -- 113.5 分钟
UPDATE moon_spacecraft SET period_seconds = 6810   WHERE id = 'grail-b';       -- 113.5 分钟
UPDATE moon_spacecraft SET period_seconds = 561600 WHERE id = 'capstone';      -- 约 6.5 天（NRHO）
