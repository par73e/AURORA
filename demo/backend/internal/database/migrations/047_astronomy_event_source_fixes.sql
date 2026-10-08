-- 047 补充 IAU MDC 数据源。它由天象资料同步任务写入 sync_runs、快照和事件，
-- 因此必须先登记到 data_sources，满足三个表的外键约束。
INSERT INTO data_sources (code, name, base_url)
VALUES
    ('iau_mdc', 'International Astronomical Union · Meteor Data Center', 'https://www.ta3.sk/IAUC22DB/MDC2022/'),
    ('jpl_small_bodies', 'NASA/JPL CNEOS Close-Approach Data', 'https://ssd-api.jpl.nasa.gov/cad.api')
ON CONFLICT (code) DO UPDATE SET base_url = EXCLUDED.base_url;
