-- 发射场补全：第一层 11 个（常显）+ 第二层 5 个（展开后显示）
-- 增加 tier（1=第一层常显，2=第二层展开显示）与 sort_order（按用户清单顺序）

ALTER TABLE launch_sites ADD COLUMN tier INTEGER NOT NULL DEFAULT 1;
ALTER TABLE launch_sites ADD COLUMN sort_order INTEGER NOT NULL DEFAULT 0;

-- 新增第一层（tier=1）
INSERT INTO launch_sites(id, name_zh, name_en, country_code, country_name_zh, latitude, longitude, description, source_url, tier, sort_order) VALUES
    ('cape-canaveral', '卡纳维拉尔角太空军基地', 'Cape Canaveral Space Force Station', 'US', '美国', 28.4880, -80.5770, '美国东海岸主要发射场，承担商业与军用卫星任务。', 'https://www.spaceforce.mil/', 1, 2),
    ('baikonur', '拜科努尔航天发射场', 'Baikonur Cosmodrome', 'RU', '俄罗斯', 45.9650, 63.3050, '世界最大的航天发射场之一，俄罗斯主要载人发射场。', 'https://www.roscosmos.ru/', 1, 4),
    ('taiyuan', '太原卫星发射中心', 'Taiyuan Satellite Launch Center', 'CN', '中国', 38.8490, 111.6080, '位于山西，主要承担极地轨道与太阳同步轨道发射。', 'https://www.cnsa.gov.cn/', 1, 7),
    ('xichang', '西昌卫星发射中心', 'Xichang Satellite Launch Center', 'CN', '中国', 28.2460, 102.0270, '位于四川，主要承担地球同步轨道卫星发射。', 'https://www.cnsa.gov.cn/', 1, 8),
    ('guiana', '圭亚那航天中心', 'Guiana Space Centre', 'EU', '欧洲', 5.2360, -52.7670, '欧洲空间局位于法属圭亚那库鲁的主要发射场，使用阿丽亚娜/织女星/联盟号火箭。', 'https://www.esa.int/', 1, 11)
ON CONFLICT (id) DO NOTHING;

-- 新增第二层（tier=2，展开后显示）
INSERT INTO launch_sites(id, name_zh, name_en, country_code, country_name_zh, latitude, longitude, description, source_url, tier, sort_order) VALUES
    ('naro', '罗老航天中心', 'Naro Space Center', 'KR', '韩国', 34.4310, 127.5350, '韩国首座轨道发射场，位于高兴郡外罗老岛。', 'https://www.kari.re.kr/', 2, 12),
    ('rocket-lab', '火箭实验室发射场 1 号', 'Rocket Lab Launch Complex 1', 'NZ', '新西兰', -39.2600, 177.8660, '商业小卫星发射场，位于新西兰玛希亚半岛。', 'https://www.rocketlabusa.com/', 2, 13),
    ('alcantara', '阿尔坎塔拉航天发射场', 'Alcântara Space Center', 'BR', '巴西', -2.3730, -44.3960, '巴西位于赤道附近的发射场，低纬度发射优势明显。', 'https://www.gov.br/aeb/', 2, 14),
    ('woomera', '伍默拉试验场', 'Woomera Test Range', 'AU', '澳大利亚', -31.1230, 136.8570, '澳大利亚内陆大型试验场，曾用于英国早期太空计划。', 'https://www.airforce.gov.au/', 2, 15),
    ('ariane', '阿丽亚娜发射基地', 'Ariane Launch Complex', 'FR', '法属圭亚那', 5.2960, -52.7580, '阿丽亚娜火箭（ELA 发射工位）在库鲁的发射设施。', 'https://www.arianespace.com/', 2, 16)
ON CONFLICT (id) DO NOTHING;

-- 现有 6 个发射场：设置层级与顺序（按用户清单第一层顺序）
UPDATE launch_sites SET tier = 1, sort_order = 1  WHERE id = 'kennedy';
UPDATE launch_sites SET tier = 1, sort_order = 3  WHERE id = 'vandenberg';
UPDATE launch_sites SET tier = 1, sort_order = 5  WHERE id = 'jiuquan';
UPDATE launch_sites SET tier = 1, sort_order = 6  WHERE id = 'wenchang';
UPDATE launch_sites SET tier = 1, sort_order = 9  WHERE id = 'tanegashima';
UPDATE launch_sites SET tier = 1, sort_order = 10 WHERE id = 'satish-dhawan';
