-- 火星航天器名录分组：着陆器（地表探测器 + 失败着陆器） / 飞行器（实时 + 历史轨道器）
-- 前端航天器区块按此拆成两个标签页（着陆器标签 / 飞行器标签，样式不同）。
ALTER TABLE mars_spacecraft ADD COLUMN catalog_group TEXT NOT NULL DEFAULT 'orbit';

-- 地表探测器（kind=surface，11 个）
UPDATE mars_spacecraft SET catalog_group = 'surface' WHERE kind = 'surface';

-- 失败着陆器（kind=catalog 但属于地表组：火星极地着陆器 / 火星 3 号着陆器 / 猎兔犬 2 号）
UPDATE mars_spacecraft SET catalog_group = 'surface' WHERE id IN ('mars-polar-lander', 'mars3-lander', 'beagle2');

-- 其余（orbital 3 + 历史轨道器 13）保持默认 'orbit'
