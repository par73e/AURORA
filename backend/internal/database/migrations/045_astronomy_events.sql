-- 天象事件是独立于用户所在地的全球记录。当地可见性必须在读取时按经纬度计算，
-- 因而不能直接写进本表的事实字段；presentation 只保存编辑提示和迁移期展示文案。
INSERT INTO data_sources (code, name, base_url) VALUES
    ('aurora_astronomy_model', 'AURORA 天文计算模型', 'https://aurora.local/astronomy-model'),
    ('nasa_gsfc_eclipse', 'NASA/GSFC Eclipse Catalog', 'https://eclipse.gsfc.nasa.gov/'),
    ('imo_meteor_calendar', 'International Meteor Organization · Meteor Shower Calendar', 'https://imo.net/files/meteor-shower/'),
    ('usno_astronomy', 'USNO Astronomical Applications', 'https://aa.usno.navy.mil/data/api'),
    ('jpl_horizons_events', 'NASA/JPL Horizons · 天象星历', 'https://ssd.jpl.nasa.gov/horizons/')
ON CONFLICT (code) DO NOTHING;

CREATE TABLE astronomy_events (
    id TEXT PRIMARY KEY,
    kind TEXT NOT NULL,
    title_zh TEXT NOT NULL,
    title_en TEXT NOT NULL,
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ,
    date_label TEXT NOT NULL,
    summary TEXT NOT NULL,
    origin TEXT NOT NULL CHECK (origin IN ('computed', 'external_forecast', 'curated')),
    source_code TEXT NOT NULL REFERENCES data_sources(code),
    source_url TEXT NOT NULL,
    verified_at DATE NOT NULL,
    geometry JSONB NOT NULL DEFAULT '{}'::jsonb,
    presentation JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX astronomy_events_starts_at_idx ON astronomy_events(starts_at);
CREATE INDEX astronomy_events_kind_starts_at_idx ON astronomy_events(kind, starts_at);

-- 外部来源每天刷新一次时保存原始快照和校验摘要；请求失败时继续使用最后一份有效数据。
CREATE TABLE astronomy_event_source_snapshots (
    id BIGSERIAL PRIMARY KEY,
    source_code TEXT NOT NULL REFERENCES data_sources(code),
    coverage_start DATE,
    coverage_end DATE,
    source_url TEXT NOT NULL,
    payload_hash TEXT NOT NULL,
    raw_payload JSONB,
    fetched_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    records_written INTEGER NOT NULL DEFAULT 0,
    UNIQUE(source_code, payload_hash)
);
CREATE INDEX astronomy_event_source_snapshots_source_fetched_idx
    ON astronomy_event_source_snapshots(source_code, fetched_at DESC);

CREATE TABLE astronomy_event_reviews (
    id BIGSERIAL PRIMARY KEY,
    event_id TEXT NOT NULL REFERENCES astronomy_events(id) ON DELETE CASCADE,
    source_code TEXT NOT NULL REFERENCES data_sources(code),
    status TEXT NOT NULL CHECK (status IN ('verified', 'warning', 'rejected')),
    note TEXT NOT NULL DEFAULT '',
    reviewed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX astronomy_event_reviews_event_reviewed_idx
    ON astronomy_event_reviews(event_id, reviewed_at DESC);

-- 将原先前端内置、已人工核验的 2026 事件迁移到统一事件库。
INSERT INTO astronomy_events
    (id, kind, title_zh, title_en, starts_at, date_label, summary, origin, source_code, source_url, verified_at, presentation)
VALUES
    ('perseids-2026', 'meteor_shower', '英仙座流星雨极大', 'PERSEIDS', '2026-08-12T16:00:00Z', '8月12–13日夜间', '北半球夏季最稳定、最适合入门守候的流星雨之一。', 'curated', 'imo_meteor_calendar', 'https://imo.net/files/meteor-shower/cal2026.pdf', '2026-08-11', '{"localStatus":"北半球优先 · 需避开城市灯光","localStatusKind":"ready","observingWindow":"当地午夜后至黎明前","direction":"东北方；辐射点升高后更有利","moonlight":"峰值夜月面亮度约 1%，月光干扰很低","advice":"不必直盯辐射点；选择开阔天空，给眼睛至少 20 分钟适应黑暗。天气只在活动前 24 小时再复核。","focusMinute":90,"focusAzimuth":45}'::jsonb),
    ('total-solar-eclipse-2026', 'solar_eclipse', '日全食', 'TOTAL SOLAR ECLIPSE', '2026-08-12T17:13:00Z', '2026年8月12日', '一次沿特定地理路径可见的全球性天象，不应默认等同于当地可观测。', 'curated', 'nasa_gsfc_eclipse', 'https://eclipse.gsfc.nasa.gov/SEpath/SEpath.html', '2026-08-11', '{"localStatus":"全球发生 · 当地可见性待按地点判定","localStatusKind":"pending","observingWindow":"请以 NASA 路径图与当地接触时刻为准","direction":"不同地点差异很大","moonlight":"不适用","advice":"仅在路径覆盖区才可见；任何阶段都必须使用合格的太阳观测滤镜，不能裸眼或以普通墨镜替代。"}'::jsonb),
    ('venus-greatest-eastern-elongation-2026', 'planetary_elongation', '金星东大距', 'VENUS EAST ELONGATION', '2026-08-15T05:59:00Z', '8月15日', '金星与太阳的角距达到近期最大，是安排傍晚低空观测的好时机。', 'curated', 'nasa_gsfc_eclipse', 'https://eclipse.gsfc.nasa.gov/SKYCAL/SKYCAL.html?cal=2026', '2026-08-11', '{"localStatus":"傍晚低空 · 需有开阔西方地平线","localStatusKind":"caution","observingWindow":"日落后不久；具体高度随地点和日期变化","direction":"西方至西南方低空","moonlight":"通常不是主要限制因素","advice":"不要在太阳仍在地平线上方时寻找金星。等待日落后再观察，并避开建筑物与山体遮挡。"}'::jsonb),
    ('kappa-cygnids-2026', 'meteor_shower', '天鹅座 κ 流星雨极大', 'KAPPA CYGNIDS', '2026-08-17T00:00:00Z', '8月17日', '北半球可整夜守候的小型流星雨，但 2026 年不预期有明显爆发。', 'curated', 'imo_meteor_calendar', 'https://imo.net/files/meteor-shower/cal2026.pdf', '2026-08-11', '{"localStatus":"北半球可见 · 强度低","localStatusKind":"caution","observingWindow":"入夜后至黎明前","direction":"北方天空；辐射点位于天鹅座一带","moonlight":"请结合当日月相判断","advice":"适合作为夏夜附加目标，不建议为它单独远行。记录明亮火流星仍有观测价值。"}'::jsonb),
    ('partial-lunar-eclipse-2026', 'lunar_eclipse', '月偏食', 'PARTIAL LUNAR ECLIPSE', '2026-08-28T04:14:00Z', '8月28日', '月球部分进入地球本影的全球性天象；不同地区可见程度与月亮高度不同。', 'curated', 'nasa_gsfc_eclipse', 'https://eclipse.gsfc.nasa.gov/LEdecade/LEdecade2021.html', '2026-08-11', '{"localStatus":"部分地区可见 · 须按地点核对","localStatusKind":"pending","observingWindow":"最大食分约在 04:14 UTC；请查阅当地可见时间","direction":"以当地月球位置为准","moonlight":"不适用","advice":"月食可裸眼安全观看；但月亮在地平线下或白昼时，当地不会有可观测画面。"}'::jsonb),
    ('aurigids-2026', 'meteor_shower', '御夫座流星雨极大', 'AURIGIDS', '2026-09-01T00:00:00Z', '9月1日', '规模较小、峰值短促的北半球流星雨，适合作为早秋的附加观察目标。', 'curated', 'imo_meteor_calendar', 'https://imo.net/files/meteor-shower/cal2026.pdf', '2026-08-11', '{"localStatus":"北半球优先 · 常规强度较低","localStatusKind":"caution","observingWindow":"后半夜至黎明前","direction":"东北方至北方天空","moonlight":"请结合当日月相判断","advice":"优先选择无直射灯光的开阔场地；若云量较高，不必为低 ZHR 强行守候。"}'::jsonb),
    ('september-epsilon-perseids-2026', 'meteor_shower', '九月 ε 英仙座流星雨极大', 'SEPTEMBER EPSILON PERSEIDS', '2026-09-09T18:00:00Z', '9月9–10日', '可能出现明亮流星的小型流星雨；2026 年增强活动仍存在不确定性。', 'curated', 'imo_meteor_calendar', 'https://imo.net/files/meteor-shower/cal2026.pdf', '2026-08-11', '{"localStatus":"北半球可见 · 活动强度待观测确认","localStatusKind":"pending","observingWindow":"峰值约 9月9日 18:00 UTC 后的当地夜间","direction":"东北方天空","moonlight":"请结合当日月相判断","advice":"把它视为有潜力的观察夜，而非保证的高峰；若看到异常频率，可记录时间、方向与亮度。"}'::jsonb)
ON CONFLICT (id) DO NOTHING;

INSERT INTO astronomy_event_reviews (event_id, source_code, status, note, reviewed_at)
SELECT id, source_code, 'verified', '从前端内置人工校订清单迁移。', '2026-08-11T00:00:00Z'
FROM astronomy_events
WHERE id IN ('perseids-2026', 'total-solar-eclipse-2026', 'venus-greatest-eastern-elongation-2026', 'kappa-cygnids-2026', 'partial-lunar-eclipse-2026', 'aurigids-2026', 'september-epsilon-perseids-2026');
