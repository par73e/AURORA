-- 042：着陆点 mission_name 统一为中文主 + 英文括号备注（用户全局双语规则）
-- 背景：早期"外国=英文主"双语格式的残留（如 "Perseverance（毅力号）"、"Tianwen-1"），
-- 用户明确：默认中文，然后备注英文 → 中文（English）。
-- 火星（14）与月球（22）着陆点全部规范化；表为静态数据，幂等 UPDATE 安全。

UPDATE mars_landing_sites SET mission_name = '毅力号（Perseverance）' WHERE id = 'perseverance';
UPDATE mars_landing_sites SET mission_name = '火星科学实验室（Mars Science Laboratory）' WHERE id = 'curiosity';
UPDATE mars_landing_sites SET mission_name = '天问一号（Tianwen-1）' WHERE id = 'zhurong';
UPDATE mars_landing_sites SET mission_name = '洞察号（InSight）' WHERE id = 'insight';
UPDATE mars_landing_sites SET mission_name = '海盗 1 号（Viking 1）' WHERE id = 'viking1';
UPDATE mars_landing_sites SET mission_name = '海盗 2 号（Viking 2）' WHERE id = 'viking2';
UPDATE mars_landing_sites SET mission_name = '火星探路者（Mars Pathfinder）' WHERE id = 'pathfinder';
UPDATE mars_landing_sites SET mission_name = '索杰纳号（Sojourner）' WHERE id = 'sojourner';
UPDATE mars_landing_sites SET mission_name = '勇气号（Spirit）' WHERE id = 'spirit';
UPDATE mars_landing_sites SET mission_name = '机遇号（Opportunity）' WHERE id = 'opportunity';
UPDATE mars_landing_sites SET mission_name = '凤凰号（Phoenix）' WHERE id = 'phoenix';
UPDATE mars_landing_sites SET mission_name = '猎兔犬 2 号（Beagle 2）' WHERE id = 'beagle2';
UPDATE mars_landing_sites SET mission_name = '火星 3 号（Mars 3）' WHERE id = 'mars3-lander';
UPDATE mars_landing_sites SET mission_name = '机智号（Ingenuity）' WHERE id = 'ingenuity';

UPDATE moon_landing_sites SET mission_name = '月球 9 号（Luna 9）' WHERE id = 'luna9';
UPDATE moon_landing_sites SET mission_name = '勘测者 1 号（Surveyor 1）' WHERE id = 'surveyor1';
UPDATE moon_landing_sites SET mission_name = '勘测者 3 号（Surveyor 3）' WHERE id = 'surveyor3';
UPDATE moon_landing_sites SET mission_name = '勘测者 7 号（Surveyor 7）' WHERE id = 'surveyor7';
UPDATE moon_landing_sites SET mission_name = '阿波罗 11 号（Apollo 11）' WHERE id = 'apollo11';
UPDATE moon_landing_sites SET mission_name = '阿波罗 12 号（Apollo 12）' WHERE id = 'apollo12';
UPDATE moon_landing_sites SET mission_name = '月球 16 号（Luna 16）' WHERE id = 'luna16';
UPDATE moon_landing_sites SET mission_name = '月球 17 号（Luna 17）' WHERE id = 'luna17';
UPDATE moon_landing_sites SET mission_name = '月球 20 号（Luna 20）' WHERE id = 'luna20';
UPDATE moon_landing_sites SET mission_name = '阿波罗 14 号（Apollo 14）' WHERE id = 'apollo14';
UPDATE moon_landing_sites SET mission_name = '阿波罗 15 号（Apollo 15）' WHERE id = 'apollo15';
UPDATE moon_landing_sites SET mission_name = '阿波罗 16 号（Apollo 16）' WHERE id = 'apollo16';
UPDATE moon_landing_sites SET mission_name = '阿波罗 17 号（Apollo 17）' WHERE id = 'apollo17';
UPDATE moon_landing_sites SET mission_name = '月球 21 号（Luna 21）' WHERE id = 'luna21';
UPDATE moon_landing_sites SET mission_name = '月球 24 号（Luna 24）' WHERE id = 'luna24';
UPDATE moon_landing_sites SET mission_name = '嫦娥三号（Chang''e 3）' WHERE id = 'change3';
UPDATE moon_landing_sites SET mission_name = '嫦娥四号（Chang''e 4）' WHERE id = 'change4';
UPDATE moon_landing_sites SET mission_name = '嫦娥五号（Chang''e 5）' WHERE id = 'change5';
UPDATE moon_landing_sites SET mission_name = '月船三号（Chandrayaan-3）' WHERE id = 'chandrayaan3';
UPDATE moon_landing_sites SET mission_name = '智慧月球探测器（SLIM）' WHERE id = 'slim';
UPDATE moon_landing_sites SET mission_name = '嫦娥六号（Chang''e 6）' WHERE id = 'change6';
UPDATE moon_landing_sites SET mission_name = '蓝色幽灵 1 号（Blue Ghost M1）' WHERE id = 'blueghost1';
