-- 人类月球着陆点（真实经纬度，历史测量值；镜像月球飞行器数据链路）
CREATE TABLE IF NOT EXISTS moon_landing_sites (
    id TEXT PRIMARY KEY,
    name_zh TEXT NOT NULL,
    name_en TEXT NOT NULL,
    program TEXT NOT NULL,
    operator_name TEXT NOT NULL,
    landing_date TEXT NOT NULL,
    latitude DOUBLE PRECISION NOT NULL,
    longitude DOUBLE PRECISION NOT NULL,
    region TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    sort_order INT NOT NULL DEFAULT 0
);

INSERT INTO moon_landing_sites (id, name_zh, name_en, program, operator_name, landing_date, latitude, longitude, region, description, sort_order) VALUES
('apollo11',  '阿波罗 11 号', 'APOLLO 11',  'APOLLO PROGRAM', 'NASA（美国国家航空航天局）', '1969-07-20',   0.6741,  23.473, '静海',            '人类首次载人登月，阿姆斯特朗与奥尔德林着陆', 1),
('apollo12',  '阿波罗 12 号', 'APOLLO 12',  'APOLLO PROGRAM', 'NASA（美国国家航空航天局）', '1969-11-19',  -3.0124, 336.578, '风暴洋',          '第二次载人登月，回收勘测者 3 号部件', 2),
('apollo14',  '阿波罗 14 号', 'APOLLO 14',  'APOLLO PROGRAM', 'NASA（美国国家航空航天局）', '1971-02-05',  -3.6453, 342.532, '弗拉·毛罗高地',  '谢泼德在月面击出高尔夫球', 3),
('apollo15',  '阿波罗 15 号', 'APOLLO 15',  'APOLLO PROGRAM', 'NASA（美国国家航空航天局）', '1971-07-30',  26.1322,   3.633, '哈德利-亚平宁',  '首次使用月球车（LRV）', 4),
('apollo16',  '阿波罗 16 号', 'APOLLO 16',  'APOLLO PROGRAM', 'NASA（美国国家航空航天局）', '1972-04-21',  -8.9733,  15.501, '笛卡尔高地',     '月球高地采样', 5),
('apollo17',  '阿波罗 17 号', 'APOLLO 17',  'APOLLO PROGRAM', 'NASA（美国国家航空航天局）', '1972-12-11',  20.1880,  30.775, '陶拉斯-利特罗',  '最后一次载人登月，地质学家施密特登月', 6),
('luna9',     '月球 9 号',    'LUNA 9',    'LUNA PROGRAM',   '苏联',                     '1966-02-03',   7.1300, 295.630, '风暴洋',          '人类首个在月面软着陆的探测器', 7),
('luna17',    '月球 17 号',   'LUNA 17',   'LUNA PROGRAM',   '苏联',                     '1970-11-17',  38.2400, 325.000, '雨海',            '运载月面车 1 号（Lunokhod 1）', 8),
('luna21',    '月球 21 号',   'LUNA 21',   'LUNA PROGRAM',   '苏联',                     '1973-01-15',  25.9900,  30.410, '澄海',            '运载月面车 2 号（Lunokhod 2）', 9),
('luna24',    '月球 24 号',   'LUNA 24',   'LUNA PROGRAM',   '苏联',                     '1976-08-18',  12.7100,  62.210, '危海',            '月球采样返回，苏联最后一次月球任务', 10),
('change3',   '嫦娥三号',     'CHANG''E-3', 'CHANG''E PROGRAM', 'CNSA（中国国家航天局）', '2013-12-14',  44.1200, 344.050, '雨海',            '中国首次月面软着陆，携带玉兔号月球车', 11),
('change4',   '嫦娥四号',     'CHANG''E-4', 'CHANG''E PROGRAM', 'CNSA（中国国家航天局）', '2019-01-03', -45.4400, 177.600, '冯·卡门撞击坑（月背）', '人类首次月球背面软着陆，携带玉兔二号', 12),
('change5',   '嫦娥五号',     'CHANG''E-5', 'CHANG''E PROGRAM', 'CNSA（中国国家航天局）', '2020-12-01',  43.0600, 308.080, '吕姆克山（风暴洋北缘）', '中国首次月球采样返回任务', 13),
('change6',   '嫦娥六号',     'CHANG''E-6', 'CHANG''E PROGRAM', 'CNSA（中国国家航天局）', '2024-06-01', -41.6400, 206.000, '南极-艾特肯盆地（月背）', '人类首次月球背面采样返回', 14),
('surveyor1', '勘测者 1 号',  'SURVEYOR 1', 'SURVEYOR PROGRAM', 'NASA（美国国家航空航天局）', '1966-06-02', -2.4500, 316.600, '风暴洋',          '美国首个月面软着陆探测器', 15),
('surveyor7', '勘测者 7 号',  'SURVEYOR 7', 'SURVEYOR PROGRAM', 'NASA（美国国家航空航天局）', '1968-01-10',-40.8800, 348.600, '第谷坑北缘',      '勘测者系列最后一次任务', 16);
