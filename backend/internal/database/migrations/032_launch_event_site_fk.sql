-- 发射日程坐标单源化：launch_events 通过 launch_site_id 关联 launch_sites。
-- 能匹配到库内发射场的，坐标一律以 launch_sites 为准（单源）；匹配不到的保留事件自带坐标（回退）。
-- 匹配顺序：工位名（pad_name，归属最精确）→ 位置名（location_name，区域级）。

ALTER TABLE launch_events ADD COLUMN launch_site_id TEXT REFERENCES launch_sites(id) ON DELETE SET NULL;

-- 0) 工位名（pad_name）别名回填：如 "LC-39A, Kennedy Space Center" → kennedy（先于区域匹配，避免误归到卡纳维拉尔角）
-- 站点名通常在 pad 名中部/末尾，用包含语义（词边界）
UPDATE launch_events e SET launch_site_id = m.id
FROM (VALUES
    ('kennedy space center', 'kennedy'),
    ('cape canaveral sfs', 'cape-canaveral'),
    ('vandenberg sfb', 'vandenberg'),
    ('vandenberg space force base', 'vandenberg'),
    ('rocket lab launch complex 1', 'rocket-lab'),
    ('mahia', 'rocket-lab'),
    ('guiana space centre', 'guiana'),
    ('kourou', 'guiana'),
    ('sriharikota', 'satish-dhawan'),
    ('satish dhawan space centre', 'satish-dhawan'),
    ('woomera', 'woomera'),
    ('alcantara', 'alcantara'),
    ('naro', 'naro'),
    ('baikonur', 'baikonur')
) AS m(key, id)
WHERE e.launch_site_id IS NULL
  AND (lower(regexp_replace(e.pad_name, '[^a-zA-Z0-9]', ' ', 'g')) = m.key
       OR lower(regexp_replace(e.pad_name, '[^a-zA-Z0-9]', ' ', 'g')) LIKE m.key || ' %'
       OR lower(regexp_replace(e.pad_name, '[^a-zA-Z0-9]', ' ', 'g')) LIKE '% ' || m.key);

-- 1) 位置名（location_name）别名回填（词边界保护）
UPDATE launch_events e SET launch_site_id = m.id
FROM (VALUES
    ('vandenberg sfb', 'vandenberg'),
    ('cape canaveral sfs', 'cape-canaveral'),
    ('cape canaveral', 'cape-canaveral'),
    ('sriharikota', 'satish-dhawan'),
    ('mahia', 'rocket-lab'),
    ('kourou', 'guiana'),
    ('woomera', 'woomera'),
    ('alcantara', 'alcantara'),
    ('naro', 'naro'),
    ('baikonur', 'baikonur')
) AS m(key, id)
WHERE e.launch_site_id IS NULL
  AND (lower(regexp_replace(e.location_name, '[^a-zA-Z0-9]', ' ', 'g')) = m.key
       OR lower(regexp_replace(e.location_name, '[^a-zA-Z0-9]', ' ', 'g')) LIKE m.key || ' %');

-- 2) 站点全名前缀回填（如 "Wenchang Space Launch Site, PRC" → wenchang）
UPDATE launch_events e SET launch_site_id = s.id
FROM launch_sites s
WHERE e.launch_site_id IS NULL
  AND length(lower(regexp_replace(s.name_en, '[^a-zA-Z0-9]', ' ', 'g'))) >= 8
  AND (lower(regexp_replace(e.location_name, '[^a-zA-Z0-9]', ' ', 'g')) = lower(regexp_replace(s.name_en, '[^a-zA-Z0-9]', ' ', 'g'))
       OR lower(regexp_replace(e.location_name, '[^a-zA-Z0-9]', ' ', 'g')) LIKE lower(regexp_replace(s.name_en, '[^a-zA-Z0-9]', ' ', 'g')) || ' %');
