-- 火星静态轨道半径对齐前端夸张系数（MARS_ALTITUDE_EXAGGERATION=1.5，火面半径 3.0）
-- 与月球 025 同模式：存「夸大前」数值（aKm×MARS_SCENE_SCALE），前端 exaggeratedA() 统一施加夸张
-- （MRO 3649km→3.23 / MAVEN 6565km→5.81 / 天问一号 9522km→8.43；夸张后 3.35/7.22/11.14），
-- 静态回退与快照路径走同一函数，避免有/无快照时轨道突然跳变。
UPDATE mars_spacecraft SET orbit_a = 3.23 WHERE id = 'mro';
UPDATE mars_spacecraft SET orbit_a = 5.81 WHERE id = 'maven';
UPDATE mars_spacecraft SET orbit_a = 8.43 WHERE id = 'tianwen1';
