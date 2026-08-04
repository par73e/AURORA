-- 地球场景新增：GNSS 星座代表星（GPS/伽利略/北斗，真实 TLE 实时位置）
-- 均为著名现代星座的代表卫星；CelesTrak 按 norad_catalog_id 自动同步，无需改代码。
INSERT INTO spacecraft(id, name_zh, name_en, norad_catalog_id, category, operator_name, description, source_code) VALUES
    ('gps-2f', 'GPS 2F-1（纳夫星66）', 'NAVSTAR 66 (USA 232)', 37753, 'MEO · GNSS 导航', '美国太空军（US Space Force）',
     'GPS Block IIF 首发卫星（SVN-62），代表全球定位系统 GPS 的中轨导航星座。', 'celestrak'),
    ('gps-3', 'GPS 3-1（韦斯普奇）', 'NAVSTAR 77 (USA 289)', 43873, 'MEO · GNSS 导航', '美国太空军（US Space Force）',
     'GPS Block III 首发卫星（SV-01）"Vespucci"，新一代 GPS 的精度与抗干扰升级。', 'celestrak'),
    ('galileo-iov1', '伽利略 IOV-1', 'GSAT0101 (GALILEO-PFM)', 37846, 'MEO · GNSS 导航', 'ESA / EUSPA（欧盟）',
     '伽利略导航系统首颗在轨验证卫星，代表欧洲全球导航星座。', 'celestrak'),
    ('galileo-foc15', '伽利略 15', 'GSAT0207 (GALILEO 15)', 41859, 'MEO · GNSS 导航', 'ESA / EUSPA（欧盟）',
     '伽利略系统全面运行能力（FOC）卫星之一，代表欧洲导航星座完整在轨部署。', 'celestrak'),
    ('beidou-3-m1', '北斗三号 M1', 'BEIDOU-3 M1', 43001, 'MEO · GNSS 导航', '中国卫星导航系统管理办公室',
     '北斗三号首颗中圆轨道（MEO）组网卫星，代表覆盖全球的中国北斗导航星座。', 'celestrak')
ON CONFLICT (id) DO NOTHING;

-- 发射信息补充（curation，与 003 同模式）
UPDATE spacecraft SET
    launch_date = '2010-05-28', launch_site = '卡纳维拉尔角 SLC-37B（美国）', launch_vehicle = 'Delta IV Medium'
    WHERE id = 'gps-2f';
UPDATE spacecraft SET
    launch_date = '2018-12-23', launch_site = '卡纳维拉尔角 SLC-40（美国）', launch_vehicle = 'Falcon 9'
    WHERE id = 'gps-3';
UPDATE spacecraft SET
    launch_date = '2011-10-21', launch_site = '法属圭亚那库鲁（法属圭亚那）', launch_vehicle = 'Soyuz ST-B'
    WHERE id = 'galileo-iov1';
UPDATE spacecraft SET
    launch_date = '2016-11-17', launch_site = '法属圭亚那库鲁（法属圭亚那）', launch_vehicle = 'Ariane 5 ES'
    WHERE id = 'galileo-foc15';
UPDATE spacecraft SET
    launch_date = '2017-11-05', launch_site = '西昌卫星发射中心（中国）', launch_vehicle = '长征三号乙'
    WHERE id = 'beidou-3-m1';
