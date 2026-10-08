-- 深空探测器（VOYAGE 太阳系标注）：JPL Horizons 日同步位置采样
-- 静态目录（curation，精度等级 A：JPL 官方星历）+ 位置采样（真实日心黄道坐标 km）
CREATE TABLE IF NOT EXISTS deep_space_probes (
    id TEXT PRIMARY KEY,                  -- slug，如 'parker'
    name_zh TEXT NOT NULL,
    name_en TEXT NOT NULL,
    operator_name TEXT NOT NULL,
    launch_date TEXT NOT NULL DEFAULT '',
    mission_type TEXT NOT NULL DEFAULT '',   -- 太阳探测 / 行星任务 / 小行星任务 / 星际
    target TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    naif_id TEXT NOT NULL,                    -- JPL Horizons NAIF ID（负值），如 '-96'
    precision_grade TEXT NOT NULL DEFAULT 'A', -- A/B/C/D（构想文档精度分级）
    color TEXT NOT NULL DEFAULT '#7fd7ff',    -- 标记/轨迹线颜色（hex）
    sort_order INT NOT NULL DEFAULT 0
);

-- 位置采样：每个探测器的日心黄道状态矢量（km），每次同步整窗替换
CREATE TABLE IF NOT EXISTS probe_position_samples (
    id BIGSERIAL PRIMARY KEY,
    probe_id TEXT NOT NULL REFERENCES deep_space_probes(id) ON DELETE CASCADE,
    epoch TIMESTAMPTZ NOT NULL,
    x_km DOUBLE PRECISION NOT NULL,
    y_km DOUBLE PRECISION NOT NULL,
    z_km DOUBLE PRECISION NOT NULL,
    synced_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(probe_id, epoch)
);
CREATE INDEX probe_position_samples_probe_epoch_idx ON probe_position_samples(probe_id, epoch ASC);

INSERT INTO deep_space_probes
    (id, name_zh, name_en, operator_name, launch_date, mission_type, target, description, naif_id, precision_grade, color, sort_order)
VALUES
    ('parker', '帕克太阳探测器', 'PARKER SOLAR PROBE', 'NASA', '2018-08-12', '太阳探测', '太阳日冕',
     '人类首个飞入太阳日冕的探测器，依靠金星借力不断压低近日点，数次刷新距太阳最近纪录。', '-96', 'A', '#ff9e3d', 1),
    ('solar-orbiter', '太阳轨道器', 'SOLAR ORBITER', 'ESA / NASA', '2020-02-10', '太阳探测', '太阳极区',
     'ESA 主导的太阳观测任务，借助金星、地球借力抬高轨道倾角，首次对太阳极区进行近距离成像。', '-144', 'A', '#ffc94d', 2),
    ('bepicolombo', '贝皮科伦坡', 'BEPICOLOMBO', 'ESA / JAXA', '2018-10-20', '行星任务', '水星',
     'ESA 与 JAXA 联合的水星探测任务（MPO + Mio 双航天器），2025–2026 年进入水星轨道。', '-121', 'A', '#c98dff', 3),
    ('juno', '朱诺号', 'JUNO', 'NASA', '2011-08-05', '行星任务', '木星',
     'NASA 木星极轨探测器，研究木星大气、磁场与内部结构，扩展任务持续运行中。', '-61', 'A', '#ff8f7a', 4),
    ('psyche', '灵神星', 'PSYCHE', 'NASA', '2023-10-13', '小行星任务', '小行星 16 Psyche',
     'NASA 金属小行星探测任务，计划 2029 年进入 16 Psyche 轨道，研究行星核心形成过程。', '-255', 'A', '#b8c7d6', 5),
    ('osiris-apex', '奥西里斯-阿派克斯', 'OSIRIS-APEX', 'NASA', '2016-09-08', '小行星任务', '小行星 99942 Apophis',
     '原 OSIRIS-REx，完成贝努样本返回后更名 OSIRIS-APEX，2029 年前往近地小行星 Apophis。', '-64', 'A', '#9fd9de', 6),
    ('new-horizons', '新视野号', 'NEW HORIZONS', 'NASA', '2006-01-19', '星际', '冥王星 · 柯伊伯带',
     '首个飞越冥王星与柯伊伯带天体 Arrokoth 的探测器，正持续深入柯伊伯带。', '-98', 'A', '#7fa8ff', 7),
    ('voyager-1', '旅行者 1 号', 'VOYAGER 1', 'NASA', '1977-09-05', '星际', '星际空间',
     '已越过日球层顶进入星际空间、距离最远的人造物体，携带金唱片。', '-31', 'A', '#8fe3c0', 8),
    ('voyager-2', '旅行者 2 号', 'VOYAGER 2', 'NASA', '1977-08-20', '星际', '星际空间',
     '唯一造访过木星、土星、天王星、海王星四大巨行星的探测器，2018 年进入星际空间。', '-32', 'A', '#8fe3c0', 9);
