-- 深空探测器补发射信息（与地球/月球 spacecraft 字段对齐：launch_date/launch_site/launch_vehicle）
ALTER TABLE deep_space_probes
    ADD COLUMN launch_site TEXT NOT NULL DEFAULT '',
    ADD COLUMN launch_vehicle TEXT NOT NULL DEFAULT '';

UPDATE deep_space_probes SET
    launch_site = '卡纳维拉尔角 SLC-37', launch_vehicle = 'Delta IV Heavy'
    WHERE id = 'parker';
UPDATE deep_space_probes SET
    launch_site = '卡纳维拉尔角 SLC-41', launch_vehicle = 'Atlas V 411'
    WHERE id = 'solar-orbiter';
UPDATE deep_space_probes SET
    launch_site = '卡纳维拉尔角 SLC-41', launch_vehicle = 'Atlas V 551'
    WHERE id = 'new-horizons';
UPDATE deep_space_probes SET
    launch_site = '卡纳维拉尔角 LC-41', launch_vehicle = 'Titan IIIE Centaur'
    WHERE id IN ('voyager-1', 'voyager-2');
