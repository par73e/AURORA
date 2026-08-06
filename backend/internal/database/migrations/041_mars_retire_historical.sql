-- 041：火星航天器只保留当前在役的 7 个轨道器（用户决定：移除全部历史/退役探测器）
--
-- 移除（已退役/失联，现实已不在火星轨道）：
--   mariner9（水手9号 1971-72）、viking1-orbiter/viking2-orbiter（海盗1/2轨道器 1976-80）、
--   mars-global-surveyor（火星全球勘测者 1997-2007）、mom（曼加里安号 2014-2022 失联）、
--   mars2-orbiter/mars3-orbiter（1971-74）、mars5（1974）、phobos2（火卫一2号 1988-89）
--
-- 保留（在役）：
--   mro（2006-）、maven（2014-）、tianwen1（2021-）——JPL 实时同步（kind=orbital）
--   2001-mars-odyssey（2001-）、mars-express（2003-）、exomars-tgo（2016-）、hope（2021-）——静态标称轨道
--
-- 同步器仅对 kind='orbital' 拉取 JPL，删除的历史行不影响同步。

DELETE FROM mars_spacecraft
 WHERE id IN (
   'mariner9', 'viking1-orbiter', 'viking2-orbiter', 'mars-global-surveyor',
   'mom', 'mars2-orbiter', 'mars3-orbiter', 'mars5', 'phobos2'
 );

-- 重排 sort_order 1-7
UPDATE mars_spacecraft SET sort_order = 1 WHERE id = 'mro';
UPDATE mars_spacecraft SET sort_order = 2 WHERE id = 'maven';
UPDATE mars_spacecraft SET sort_order = 3 WHERE id = 'tianwen1';
UPDATE mars_spacecraft SET sort_order = 4 WHERE id = '2001-mars-odyssey';
UPDATE mars_spacecraft SET sort_order = 5 WHERE id = 'mars-express';
UPDATE mars_spacecraft SET sort_order = 6 WHERE id = 'exomars-tgo';
UPDATE mars_spacecraft SET sort_order = 7 WHERE id = 'hope';
