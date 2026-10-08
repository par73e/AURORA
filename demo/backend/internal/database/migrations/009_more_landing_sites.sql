-- 补充着陆点（Luna 16、Surveyor 3、Chandrayaan-3、SLIM、Blue Ghost 1）
-- 并记录中国任务官方着陆点名（广寒宫/天河基地/天船基地/天江基地）

-- 1) 官方着陆点名写入描述
UPDATE moon_landing_sites SET description = '中国首次月面软着陆，携带玉兔号月球车；着陆点命名「广寒宫」' WHERE id = 'change3';
UPDATE moon_landing_sites SET description = '人类首次月球背面软着陆，携带玉兔二号；着陆点命名「天河基地」' WHERE id = 'change4';
UPDATE moon_landing_sites SET description = '中国首次月球采样返回任务；着陆点命名「天船基地」' WHERE id = 'change5';
UPDATE moon_landing_sites SET description = '人类首次月球背面采样返回；着陆点命名「天江基地」' WHERE id = 'change6';

-- 2) 按着陆时间重排 sort_order（含新增）
UPDATE moon_landing_sites SET sort_order = 1  WHERE id = 'luna9';
UPDATE moon_landing_sites SET sort_order = 2  WHERE id = 'surveyor1';
UPDATE moon_landing_sites SET sort_order = 4  WHERE id = 'surveyor7';
UPDATE moon_landing_sites SET sort_order = 5  WHERE id = 'apollo11';
UPDATE moon_landing_sites SET sort_order = 6  WHERE id = 'apollo12';
UPDATE moon_landing_sites SET sort_order = 9  WHERE id = 'apollo14';
UPDATE moon_landing_sites SET sort_order = 10 WHERE id = 'apollo15';
UPDATE moon_landing_sites SET sort_order = 11 WHERE id = 'apollo16';
UPDATE moon_landing_sites SET sort_order = 12 WHERE id = 'apollo17';
UPDATE moon_landing_sites SET sort_order = 13 WHERE id = 'luna21';
UPDATE moon_landing_sites SET sort_order = 14 WHERE id = 'luna24';
UPDATE moon_landing_sites SET sort_order = 15 WHERE id = 'change3';
UPDATE moon_landing_sites SET sort_order = 16 WHERE id = 'change4';
UPDATE moon_landing_sites SET sort_order = 17 WHERE id = 'change5';
UPDATE moon_landing_sites SET sort_order = 20 WHERE id = 'change6';

-- 3) 新增 5 个着陆点
INSERT INTO moon_landing_sites (id, name_zh, name_en, program, operator_name, landing_date, latitude, longitude, region, description, sort_order) VALUES
('luna16',       '月球 16 号',        'LUNA 16',            'LUNA PROGRAM',      '苏联',                     '1970-09-20', -0.6800,  56.300, '丰富海',        '苏联首次月球自动采样返回', 7),
('surveyor3',    '勘测者 3 号',       'SURVEYOR 3',         'SURVEYOR PROGRAM',  'NASA（美国国家航空航天局）', '1967-04-20', -2.9400, 336.660, '风暴洋',        '阿波罗 12 号回收其摄像机与铲斗部件', 3),
('chandrayaan3', '月船三号',          'CHANDRAYAAN-3',      'CHANDRAYAAN PROGRAM', 'ISRO（印度空间研究组织）', '2023-08-23', -69.3700, 32.320, '南极附近',      '印度首次月面软着陆；着陆点命名「Statio Shiv Shakti」', 18),
('slim',         'SLIM 月球精准着陆探测器', 'SLIM',          'SLIM PROGRAM',      'JAXA（日本宇宙航空研究开发机构）', '2024-01-19', -13.3200, 25.250, '酒海（Shioli 陨坑附近）', '日本首次月面软着陆，实现 100m 级精准着陆（Moon Sniper）', 19),
('blueghost1',   '萤火虫 Blue Ghost 1', 'BLUE GHOST M1',    'COMMERCIAL LUNAR',  'Firefly Aerospace（美国）', '2025-03-02', 17.7100, 61.550, '危海',          '商业月球着陆服务（CLPS）首次成功任务', 21);
