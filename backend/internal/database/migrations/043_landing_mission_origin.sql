-- 043：着陆点 mission_name 按来源定显示（用户最终规则）
-- 规则：国外任务=英文主 + 中文备注（English（中文））；国内任务=纯中文（不带英文备注）。
-- 修正 042 的"全中文主"方案（用户：中文不需要备注，英文需要备注下中文）。

-- 火星（国外=英文主+中文备注）
UPDATE mars_landing_sites SET mission_name = 'Perseverance（毅力号）' WHERE id = 'perseverance';
UPDATE mars_landing_sites SET mission_name = 'Mars Science Laboratory（火星科学实验室）' WHERE id = 'curiosity';
UPDATE mars_landing_sites SET mission_name = 'InSight（洞察号）' WHERE id = 'insight';
UPDATE mars_landing_sites SET mission_name = 'Viking 1（海盗 1 号）' WHERE id = 'viking1';
UPDATE mars_landing_sites SET mission_name = 'Viking 2（海盗 2 号）' WHERE id = 'viking2';
UPDATE mars_landing_sites SET mission_name = 'Mars Pathfinder（火星探路者）' WHERE id = 'pathfinder';
UPDATE mars_landing_sites SET mission_name = 'Sojourner（索杰纳号）' WHERE id = 'sojourner';
UPDATE mars_landing_sites SET mission_name = 'Spirit（勇气号）' WHERE id = 'spirit';
UPDATE mars_landing_sites SET mission_name = 'Opportunity（机遇号）' WHERE id = 'opportunity';
UPDATE mars_landing_sites SET mission_name = 'Phoenix（凤凰号）' WHERE id = 'phoenix';
UPDATE mars_landing_sites SET mission_name = 'Beagle 2（猎兔犬 2 号）' WHERE id = 'beagle2';
UPDATE mars_landing_sites SET mission_name = 'Mars 3（火星 3 号）' WHERE id = 'mars3-lander';
UPDATE mars_landing_sites SET mission_name = 'Ingenuity（机智号）' WHERE id = 'ingenuity';
-- 火星（国内=纯中文，不带备注）
UPDATE mars_landing_sites SET mission_name = '天问一号' WHERE id = 'zhurong';

-- 月球（国外=英文主+中文备注）
UPDATE moon_landing_sites SET mission_name = 'Luna 9（月球 9 号）' WHERE id = 'luna9';
UPDATE moon_landing_sites SET mission_name = 'Surveyor 1（勘测者 1 号）' WHERE id = 'surveyor1';
UPDATE moon_landing_sites SET mission_name = 'Surveyor 3（勘测者 3 号）' WHERE id = 'surveyor3';
UPDATE moon_landing_sites SET mission_name = 'Surveyor 7（勘测者 7 号）' WHERE id = 'surveyor7';
UPDATE moon_landing_sites SET mission_name = 'Apollo 11（阿波罗 11 号）' WHERE id = 'apollo11';
UPDATE moon_landing_sites SET mission_name = 'Apollo 12（阿波罗 12 号）' WHERE id = 'apollo12';
UPDATE moon_landing_sites SET mission_name = 'Luna 16（月球 16 号）' WHERE id = 'luna16';
UPDATE moon_landing_sites SET mission_name = 'Luna 17（月球 17 号）' WHERE id = 'luna17';
UPDATE moon_landing_sites SET mission_name = 'Luna 20（月球 20 号）' WHERE id = 'luna20';
UPDATE moon_landing_sites SET mission_name = 'Apollo 14（阿波罗 14 号）' WHERE id = 'apollo14';
UPDATE moon_landing_sites SET mission_name = 'Apollo 15（阿波罗 15 号）' WHERE id = 'apollo15';
UPDATE moon_landing_sites SET mission_name = 'Apollo 16（阿波罗 16 号）' WHERE id = 'apollo16';
UPDATE moon_landing_sites SET mission_name = 'Apollo 17（阿波罗 17 号）' WHERE id = 'apollo17';
UPDATE moon_landing_sites SET mission_name = 'Luna 21（月球 21 号）' WHERE id = 'luna21';
UPDATE moon_landing_sites SET mission_name = 'Luna 24（月球 24 号）' WHERE id = 'luna24';
UPDATE moon_landing_sites SET mission_name = 'Chandrayaan-3（月船三号）' WHERE id = 'chandrayaan3';
UPDATE moon_landing_sites SET mission_name = 'SLIM（智慧月球探测器）' WHERE id = 'slim';
UPDATE moon_landing_sites SET mission_name = 'Blue Ghost M1（蓝色幽灵 1 号）' WHERE id = 'blueghost1';
-- 月球（国内=纯中文，不带备注）
UPDATE moon_landing_sites SET mission_name = '嫦娥三号' WHERE id = 'change3';
UPDATE moon_landing_sites SET mission_name = '嫦娥四号' WHERE id = 'change4';
UPDATE moon_landing_sites SET mission_name = '嫦娥五号' WHERE id = 'change5';
UPDATE moon_landing_sites SET mission_name = '嫦娥六号' WHERE id = 'change6';
