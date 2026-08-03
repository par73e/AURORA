-- 着陆点图标类型 + 月球车行驶轨迹（示意折线，后续可替换为遥测真实轨迹）
ALTER TABLE moon_landing_sites
    ADD COLUMN icon TEXT NOT NULL DEFAULT 'lander',
    ADD COLUMN track JSONB NOT NULL DEFAULT '[]'::jsonb;

-- 图标归类：astronaut=载人 / rover=月球车任务 / sample=采样返回 / lander=其余着陆器
UPDATE moon_landing_sites SET icon = 'astronaut' WHERE id IN ('apollo11','apollo12','apollo14','apollo15','apollo16','apollo17');
UPDATE moon_landing_sites SET icon = 'rover'     WHERE id IN ('change3','change4','luna17','luna21','chandrayaan3');
UPDATE moon_landing_sites SET icon = 'sample'    WHERE id IN ('change5','change6','luna16','luna24','luna20');
UPDATE moon_landing_sites SET icon = 'lander'    WHERE id IN ('luna9','surveyor1','surveyor3','surveyor7','slim','blueghost1');

-- 补充 Luna 20（用户清单：样本任务）
INSERT INTO moon_landing_sites (id, name_zh, name_en, program, operator_name, landing_date, latitude, longitude, region, description, sort_order,
                                site_name, official_name, mission_name, hardware, side, category, icon)
VALUES ('luna20', '月球 20 号', 'LUNA 20', 'LUNA PROGRAM', '苏联', '1972-02-21', 3.5700, 56.500, '丰富海与危海之间高地', '苏联第二次自动采样返回（高地采样）', 11,
        '丰富海高地着陆点', '', 'Luna 20', '["月球 20 号下降级（采样钻机）", "上升级返回舱"]', 'NEAR_SIDE', 'SAMPLE_RETURN', 'sample');
-- 重排其后站点的 sort_order（luna20 插在 luna17 后、apollo14 前）
UPDATE moon_landing_sites SET sort_order = 12 WHERE id = 'apollo14';
UPDATE moon_landing_sites SET sort_order = 13 WHERE id = 'apollo15';
UPDATE moon_landing_sites SET sort_order = 14 WHERE id = 'apollo16';
UPDATE moon_landing_sites SET sort_order = 15 WHERE id = 'apollo17';
UPDATE moon_landing_sites SET sort_order = 16 WHERE id = 'luna21';
UPDATE moon_landing_sites SET sort_order = 17 WHERE id = 'luna24';
UPDATE moon_landing_sites SET sort_order = 18 WHERE id = 'change3';
UPDATE moon_landing_sites SET sort_order = 19 WHERE id = 'change4';
UPDATE moon_landing_sites SET sort_order = 20 WHERE id = 'change5';
UPDATE moon_landing_sites SET sort_order = 21 WHERE id = 'chandrayaan3';
UPDATE moon_landing_sites SET sort_order = 22 WHERE id = 'slim';
UPDATE moon_landing_sites SET sort_order = 23 WHERE id = 'change6';
UPDATE moon_landing_sites SET sort_order = 24 WHERE id = 'blueghost1';

-- 月球车行驶轨迹（示意折线 [[lat,lon],...]，按公开报道方向概绘）
UPDATE moon_landing_sites SET track = '[[44.120,344.050],[44.123,344.058],[44.124,344.066],[44.122,344.073],[44.119,344.079]]'::jsonb WHERE id = 'change3';
UPDATE moon_landing_sites SET track = '[[-45.440,177.600],[-45.444,177.592],[-45.448,177.585],[-45.452,177.577],[-45.456,177.570],[-45.460,177.562],[-45.464,177.555],[-45.468,177.548]]'::jsonb WHERE id = 'change4';
UPDATE moon_landing_sites SET track = '[[38.240,325.000],[38.233,325.006],[38.225,325.010],[38.217,325.013],[38.209,325.015],[38.201,325.017],[38.193,325.019]]'::jsonb WHERE id = 'luna17';
UPDATE moon_landing_sites SET track = '[[25.990,30.410],[25.978,30.418],[25.965,30.425],[25.952,30.432],[25.940,30.438],[25.928,30.443],[25.915,30.447],[25.902,30.450]]'::jsonb WHERE id = 'luna21';
UPDATE moon_landing_sites SET track = '[[-69.370,32.320],[-69.373,32.323],[-69.376,32.326],[-69.379,32.329]]'::jsonb WHERE id = 'chandrayaan3';
UPDATE moon_landing_sites SET track = '[[26.132,3.633],[26.136,3.630],[26.139,3.628],[26.141,3.633],[26.137,3.637],[26.133,3.639]]'::jsonb WHERE id = 'apollo15';
UPDATE moon_landing_sites SET track = '[[-8.973,15.501],[-8.977,15.498],[-8.980,15.496],[-8.982,15.500],[-8.978,15.504],[-8.974,15.506]]'::jsonb WHERE id = 'apollo16';
UPDATE moon_landing_sites SET track = '[[20.188,30.775],[20.191,30.771],[20.194,30.769],[20.196,30.774],[20.192,30.779],[20.189,30.781]]'::jsonb WHERE id = 'apollo17';
