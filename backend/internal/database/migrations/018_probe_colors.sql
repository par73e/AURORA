-- 帕克（#ff9e3d 橙）与太阳轨道器（#ffc94d 金黄橙）颜色过近，界面上难以区分；
-- 太阳轨道器改为浅紫，与掠日橙拉开辨识度
UPDATE deep_space_probes SET color = '#c9a9ff' WHERE id = 'solar-orbiter';
