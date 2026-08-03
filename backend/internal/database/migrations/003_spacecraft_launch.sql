-- 航天器发射信息：CelesTrak GP 不含发射场，补充手工维护的发射数据（curation）
ALTER TABLE spacecraft
    ADD COLUMN launch_date TEXT,
    ADD COLUMN launch_site TEXT,
    ADD COLUMN launch_vehicle TEXT;

UPDATE spacecraft SET
    launch_date = '1998-11-20',
    launch_site = '拜科努尔航天发射场（哈萨克斯坦）',
    launch_vehicle = '质子-K 运载火箭'
WHERE id = 'iss';

UPDATE spacecraft SET
    launch_date = '2021-04-29',
    launch_site = '文昌航天发射场（中国）',
    launch_vehicle = '长征五号 B 运载火箭'
WHERE id = 'tianhe';

UPDATE spacecraft SET
    launch_date = '1990-04-24',
    launch_site = '肯尼迪航天中心 LC-39B（美国）',
    launch_vehicle = '发现号航天飞机 STS-31'
WHERE id = 'hubble';
