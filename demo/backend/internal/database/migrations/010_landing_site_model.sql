-- 着陆点数据模型升级：地点（site）与设施（hardware）分开保存
ALTER TABLE moon_landing_sites
    ADD COLUMN site_name TEXT NOT NULL DEFAULT '',
    ADD COLUMN official_name TEXT NOT NULL DEFAULT '',
    ADD COLUMN mission_name TEXT NOT NULL DEFAULT '',
    ADD COLUMN hardware JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN side TEXT NOT NULL DEFAULT 'NEAR_SIDE',
    ADD COLUMN category TEXT NOT NULL DEFAULT 'ROBOTIC_LANDER';

UPDATE moon_landing_sites SET
  site_name = '广寒宫', mission_name = '嫦娥三号', official_name = 'Statio Guang Han Gong',
  side = 'NEAR_SIDE', category = 'ROBOTIC_LANDER',
  hardware = '["嫦娥三号着陆器", "玉兔号月球车"]'::jsonb
WHERE id = 'change3';

UPDATE moon_landing_sites SET
  site_name = '天河基地', mission_name = '嫦娥四号', official_name = 'Statio Tianhe',
  side = 'FAR_SIDE', category = 'ROBOTIC_LANDER',
  hardware = '["嫦娥四号着陆器", "玉兔二号月球车（仍在月面工作）"]'::jsonb
WHERE id = 'change4';

UPDATE moon_landing_sites SET
  site_name = '天船基地', mission_name = '嫦娥五号', official_name = 'Statio Tianchuan',
  side = 'NEAR_SIDE', category = 'SAMPLE_RETURN',
  hardware = '["嫦娥五号着陆上升组合体", "表取采样机械臂", "钻取采样钻机", "上升器（采样返回）"]'::jsonb
WHERE id = 'change5';

UPDATE moon_landing_sites SET
  site_name = '天江基地', mission_name = '嫦娥六号', official_name = 'Statio Tianjiang',
  side = 'FAR_SIDE', category = 'SAMPLE_RETURN',
  hardware = '["嫦娥六号着陆上升组合体", "表取/钻取采样系统", "上升器（采样返回）"]'::jsonb
WHERE id = 'change6';

UPDATE moon_landing_sites SET
  site_name = '静海基地', mission_name = 'Apollo 11', official_name = 'Statio Tranquillitatis',
  side = 'NEAR_SIDE', category = 'CREWED_LANDING',
  hardware = '["鹰号登月舱下降级（Eagle Descent Stage）", "激光测距后向反射器（LRRR）", "被动月震实验包（PSEP）", "太阳风收集器（SWC）", "阿波罗电视摄像机"]'::jsonb
WHERE id = 'apollo11';

UPDATE moon_landing_sites SET
  site_name = '风暴洋基地', mission_name = 'Apollo 12', official_name = 'Statio Cognitum',
  side = 'NEAR_SIDE', category = 'CREWED_LANDING',
  hardware = '["无畏号登月舱下降级（Intrepid Descent Stage）", "阿波罗月面实验包（ALSEP）", "勘测者 3 号回收部件（摄像机/铲斗）"]'::jsonb
WHERE id = 'apollo12';

UPDATE moon_landing_sites SET
  site_name = '弗拉·毛罗基地', mission_name = 'Apollo 14', official_name = 'Fra Mauro',
  side = 'NEAR_SIDE', category = 'CREWED_LANDING',
  hardware = '["安塔瑞斯登月舱下降级（Antares Descent Stage）", "阿波罗月面实验包（ALSEP）", "模块化设备运输车（MET）"]'::jsonb
WHERE id = 'apollo14';

UPDATE moon_landing_sites SET
  site_name = '哈德利-亚平宁基地', mission_name = 'Apollo 15', official_name = 'Hadley–Apennine',
  side = 'NEAR_SIDE', category = 'CREWED_LANDING',
  hardware = '["猎鹰登月舱下降级（Falcon Descent Stage）", "阿波罗月面实验包（ALSEP）", "月面车 1 号（LRV-1）"]'::jsonb
WHERE id = 'apollo15';

UPDATE moon_landing_sites SET
  site_name = '笛卡尔高地基地', mission_name = 'Apollo 16', official_name = 'Descartes Highlands',
  side = 'NEAR_SIDE', category = 'CREWED_LANDING',
  hardware = '["猎户座登月舱下降级（Orion Descent Stage）", "阿波罗月面实验包（ALSEP）", "月面车 2 号（LRV-2）"]'::jsonb
WHERE id = 'apollo16';

UPDATE moon_landing_sites SET
  site_name = '陶拉斯-利特罗基地', mission_name = 'Apollo 17', official_name = 'Taurus–Littrow',
  side = 'NEAR_SIDE', category = 'CREWED_LANDING',
  hardware = '["挑战者登月舱下降级（Challenger Descent Stage）", "阿波罗月面实验包（ALSEP）", "月面车 3 号（LRV-3）"]'::jsonb
WHERE id = 'apollo17';

UPDATE moon_landing_sites SET
  site_name = '风暴洋着陆点', mission_name = 'Luna 9', official_name = '',
  side = 'NEAR_SIDE', category = 'ROBOTIC_LANDER',
  hardware = '["月球 9 号着陆舱", "电视摄像系统", "辐射计"]'::jsonb
WHERE id = 'luna9';

UPDATE moon_landing_sites SET
  site_name = '丰富海着陆点', mission_name = 'Luna 16', official_name = '',
  side = 'NEAR_SIDE', category = 'SAMPLE_RETURN',
  hardware = '["月球 16 号下降级（采样钻机）", "上升级返回舱"]'::jsonb
WHERE id = 'luna16';

UPDATE moon_landing_sites SET
  site_name = '雨海着陆点（月面车 1 号）', mission_name = 'Luna 17', official_name = 'Lunokhod-1 着陆区',
  side = 'NEAR_SIDE', category = 'ROBOTIC_LANDER',
  hardware = '["月球 17 号下降级", "月面车 1 号（Lunokhod 1）"]'::jsonb
WHERE id = 'luna17';

UPDATE moon_landing_sites SET
  site_name = '澄海着陆点（月面车 2 号）', mission_name = 'Luna 21', official_name = 'Lunokhod-2 着陆区',
  side = 'NEAR_SIDE', category = 'ROBOTIC_LANDER',
  hardware = '["月球 21 号下降级", "月面车 2 号（Lunokhod 2）"]'::jsonb
WHERE id = 'luna21';

UPDATE moon_landing_sites SET
  site_name = '危海着陆点', mission_name = 'Luna 24', official_name = '',
  side = 'NEAR_SIDE', category = 'SAMPLE_RETURN',
  hardware = '["月球 24 号下降级（2 米深钻）", "上升级返回舱"]'::jsonb
WHERE id = 'luna24';

UPDATE moon_landing_sites SET
  site_name = '勘测者 1 号着陆点', mission_name = 'Surveyor 1', official_name = '',
  side = 'NEAR_SIDE', category = 'ROBOTIC_LANDER',
  hardware = '["勘测者 1 号着陆器", "电视摄像机", "应变计着陆腿"]'::jsonb
WHERE id = 'surveyor1';

UPDATE moon_landing_sites SET
  site_name = '勘测者 3 号着陆点', mission_name = 'Surveyor 3', official_name = '',
  side = 'NEAR_SIDE', category = 'ROBOTIC_LANDER',
  hardware = '["勘测者 3 号着陆器", "电视摄像机", "表面取样铲斗", "阿波罗 12 回收部件（摄像机/铲斗）"]'::jsonb
WHERE id = 'surveyor3';

UPDATE moon_landing_sites SET
  site_name = '第谷坑北缘着陆点', mission_name = 'Surveyor 7', official_name = '',
  side = 'NEAR_SIDE', category = 'ROBOTIC_LANDER',
  hardware = '["勘测者 7 号着陆器", "电视摄像机", "表面取样铲斗"]'::jsonb
WHERE id = 'surveyor7';

UPDATE moon_landing_sites SET
  site_name = 'Statio Shiv Shakti（湿婆之力站）', mission_name = 'Chandrayaan-3', official_name = 'Statio Shiv Shakti',
  side = 'NEAR_SIDE', category = 'ROBOTIC_LANDER',
  hardware = '["维克拉姆着陆器（Vikram Lander）", "普拉吉安月球车（Pragyan Rover）"]'::jsonb
WHERE id = 'chandrayaan3';

UPDATE moon_landing_sites SET
  site_name = 'SLIM 着陆点（酒海）', mission_name = 'SLIM（Moon Sniper）', official_name = '',
  side = 'NEAR_SIDE', category = 'ROBOTIC_LANDER',
  hardware = '["SLIM 着陆器", "LEV-1 跳跃式巡视器", "LEV-2 迷你巡视器"]'::jsonb
WHERE id = 'slim';

UPDATE moon_landing_sites SET
  site_name = 'Blue Ghost 着陆点', mission_name = 'Blue Ghost M1', official_name = '',
  side = 'NEAR_SIDE', category = 'COMMERCIAL_LANDER',
  hardware = '["Blue Ghost 着陆器", "NASA CLPS 载荷 10 项（LTD 激光反射器、LISTER 热流探针等）"]'::jsonb
WHERE id = 'blueghost1';
