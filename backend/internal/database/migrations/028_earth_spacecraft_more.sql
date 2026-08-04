-- 地球场景目录扩充：新增 16 颗知名在轨航天器（真实 TLE，CelesTrak 自动同步）
-- 覆盖空间科学（X射线/γ射线/UV）、对地观测、气象（GEO）与历史卫星。
INSERT INTO spacecraft(id, name_zh, name_en, norad_catalog_id, category, operator_name, description, source_code) VALUES
    ('cxo', '钱德拉X射线天文台', 'CHANDRA X-RAY OBSERVATORY (CXO)', 25867, 'HEO · 空间科学', 'NASA',
     '旗舰 X 射线天文台，在深长椭圆轨道上观测黑洞、超新星遗迹与星系团。', 'celestrak'),
    ('fermi', '费米伽马射线空间望远镜', 'FGRST (GLAST)', 33053, 'LEO · 空间科学', 'NASA / DOE',
     '伽马射线空间望远镜（原 GLAST），巡天观测伽马暴、脉冲星与活动星系核。', 'celestrak'),
    ('nustar', 'NuSTAR', 'NUSTAR', 38358, 'LEO · 空间科学', 'NASA / Caltech',
     '硬 X 射线聚焦望远镜，观测黑洞、超新星遗迹等高能天体。', 'celestrak'),
    ('swift', 'Swift 伽马暴望远镜', 'SWIFT', 28485, 'LEO · 空间科学', 'NASA',
     '伽马暴快速响应望远镜，多波段追踪瞬变天体。', 'celestrak'),
    ('xmm-newton', 'XMM-Newton', 'XMM-NEWTON', 25989, 'HEO · 空间科学', 'ESA',
     '欧洲 X 射线多镜面望远镜，深长椭圆轨道上的 X 射线观测主力。', 'celestrak'),
    ('integral', 'INTEGRAL', 'INTEGRAL', 27540, 'HEO · 空间科学', 'ESA / NASA / Roscosmos',
     '伽马射线与 X 射线天文台，高椭圆轨道，专注核天体物理。', 'celestrak'),
    ('terra', 'Terra 地球观测卫星', 'TERRA', 25994, 'SSO · 对地观测', 'NASA',
     '地球系统旗舰卫星（EOS-AM1），太阳同步极轨综合观测陆海大气。', 'celestrak'),
    ('landsat-9', '陆地卫星 9 号', 'LANDSAT 9', 49260, 'SSO · 对地观测', 'NASA / USGS',
     'Landsat 系列最新一代，全球陆地遥感与地表变化监测主力。', 'celestrak'),
    ('sentinel-1a', '哨兵 1A', 'SENTINEL-1A', 39634, 'SSO · 对地观测', 'ESA（哥白尼计划）',
     '哥白尼计划雷达成像卫星，全天候监测地表形变与海洋。', 'celestrak'),
    ('icesat-2', 'ICESat-2 激光测高卫星', 'ICESAT-2', 43613, 'SSO · 对地观测', 'NASA',
     '光子计数激光测高卫星，精密测量极地冰盖高程变化。', 'celestrak'),
    ('grace-fo-1', 'GRACE-FO 1', 'GRACE-FO 1', 43476, 'SSO · 对地观测', 'NASA / GFZ（德国）',
     '重力恢复与气候实验后继卫星（双星之一），与 GRACE-FO 2 编队测地球重力场。', 'celestrak'),
    ('grace-fo-2', 'GRACE-FO 2', 'GRACE-FO 2', 43477, 'SSO · 对地观测', 'NASA / GFZ（德国）',
     'GRACE-FO 双星编队之二，与 1 号保持前后编队，测量水循环引起的重力变化。', 'celestrak'),
    ('goes-16', 'GOES-16（GOES-East）', 'GOES 16', 41866, 'GEO · 气象', 'NOAA',
     '美国新一代静止气象卫星（GOES-East 位），高频云图与闪电成像。', 'celestrak'),
    ('fy-4a', '风云四号 A 星', 'FENGYUN 4A', 41882, 'GEO · 气象', '中国气象局',
     '中国新一代静止轨道气象卫星首发星，凝视式高频成像。', 'celestrak'),
    ('himawari-8', '向日葵 8 号', 'HIMAWARI-8', 40267, 'GEO · 气象', 'JMA（日本气象厅）',
     '日本静止气象卫星，十分钟级高频云图，东亚地区气象观测主力。', 'celestrak'),
    ('vanguard-1', '先锋 1 号', 'VANGUARD 1', 5, 'LEO · 历史', '美国海军研究实验室',
     '人类早期人造卫星（1958），至今仍在轨，是太空时代最古老的见证者。', 'celestrak')
ON CONFLICT (id) DO NOTHING;

-- 发射信息补充（curation，与 003/022 同模式）
UPDATE spacecraft SET launch_date='1999-07-23', launch_site='肯尼迪航天中心（美国）', launch_vehicle='航天飞机哥伦比亚号 STS-93' WHERE id='cxo';
UPDATE spacecraft SET launch_date='2008-06-11', launch_site='卡纳维拉尔角 SLC-17B（美国）', launch_vehicle='Delta II 7920H' WHERE id='fermi';
UPDATE spacecraft SET launch_date='2012-06-13', launch_site='夸贾林环礁（马绍尔群岛）', launch_vehicle='Pegasus XL' WHERE id='nustar';
UPDATE spacecraft SET launch_date='2004-11-20', launch_site='卡纳维拉尔角 SLC-17A（美国）', launch_vehicle='Delta II 7320' WHERE id='swift';
UPDATE spacecraft SET launch_date='1999-12-10', launch_site='法属圭亚那库鲁（法属圭亚那）', launch_vehicle='Ariane 5' WHERE id='xmm-newton';
UPDATE spacecraft SET launch_date='2002-10-17', launch_site='拜科努尔航天发射场（哈萨克斯坦）', launch_vehicle='质子-K' WHERE id='integral';
UPDATE spacecraft SET launch_date='1999-12-18', launch_site='范登堡太空军基地（美国）', launch_vehicle='Atlas II AS' WHERE id='terra';
UPDATE spacecraft SET launch_date='2021-09-27', launch_site='范登堡太空军基地（美国）', launch_vehicle='Atlas V 401' WHERE id='landsat-9';
UPDATE spacecraft SET launch_date='2014-04-03', launch_site='法属圭亚那库鲁（法属圭亚那）', launch_vehicle='Soyuz ST-A' WHERE id='sentinel-1a';
UPDATE spacecraft SET launch_date='2018-09-15', launch_site='范登堡太空军基地（美国）', launch_vehicle='Delta II 7420' WHERE id='icesat-2';
UPDATE spacecraft SET launch_date='2018-05-22', launch_site='范登堡太空军基地（美国）', launch_vehicle='Falcon 9' WHERE id IN ('grace-fo-1','grace-fo-2');
UPDATE spacecraft SET launch_date='2016-11-19', launch_site='卡纳维拉尔角 SLC-41（美国）', launch_vehicle='Atlas V 541' WHERE id='goes-16';
UPDATE spacecraft SET launch_date='2016-12-10', launch_site='西昌卫星发射中心（中国）', launch_vehicle='长征三号乙' WHERE id='fy-4a';
UPDATE spacecraft SET launch_date='2014-10-07', launch_site='种子岛宇宙中心（日本）', launch_vehicle='H-IIA 202' WHERE id='himawari-8';
UPDATE spacecraft SET launch_date='1958-03-17', launch_site='卡纳维拉尔角 LC-18A（美国）', launch_vehicle='Vanguard 火箭' WHERE id='vanguard-1';
