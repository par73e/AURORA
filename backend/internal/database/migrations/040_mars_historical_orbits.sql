-- 历史轨道器补静态轨道参数：13 个 catalog 飞行器获得 3D 轨道（真实周期/倾角/偏心率 + 分层视觉半径）
-- 视觉半径沿用 MARS_SCENE_SCALE + ×1.5 夸张（存夸大前值，前端 exaggeratedA 统一处理）：
--   3.60→3.90 … 8.20→10.8，与 3 个实时轨道器分层错开；
--   高离心率轨道（如 MOM e=0.91）远心点可达 20+，超出默认取景属正常——轨道线环绕火星、相机在环内可见。
-- 周期为真实值（部分按公开资料取整）；raan/argp 为视觉排布值（真实值多已无法考证）。

UPDATE mars_spacecraft SET
  orbit_a = 5.00, orbit_e = 0.60, inclination_deg = 64,   raan_deg = 40,  arg_periapsis_deg = 120, period_seconds = 43500,
  display_inclination = '≈ 64', display_eccentricity = '≈ 0.60', display_period = '约 12 小时'
WHERE id = 'mariner9';

UPDATE mars_spacecraft SET
  orbit_a = 6.60, orbit_e = 0.76, inclination_deg = 39,   raan_deg = 70,  arg_periapsis_deg = 200, period_seconds = 88500,
  display_inclination = '≈ 39', display_eccentricity = '≈ 0.76', display_period = '约 24.6 小时'
WHERE id = 'viking1-orbiter';

UPDATE mars_spacecraft SET
  orbit_a = 7.00, orbit_e = 0.76, inclination_deg = 39,   raan_deg = 100, arg_periapsis_deg = 240, period_seconds = 88500,
  display_inclination = '≈ 39', display_eccentricity = '≈ 0.76', display_period = '约 24.6 小时'
WHERE id = 'viking2-orbiter';

UPDATE mars_spacecraft SET
  orbit_a = 3.60, orbit_e = 0.001, inclination_deg = 93,  raan_deg = 30,  arg_periapsis_deg = 0,   period_seconds = 7060,
  display_inclination = '≈ 93', display_eccentricity = '≈ 0.001', display_period = '约 118 分钟'
WHERE id = 'mars-global-surveyor';

UPDATE mars_spacecraft SET
  orbit_a = 3.90, orbit_e = 0.001, inclination_deg = 93,  raan_deg = 350, arg_periapsis_deg = 180, period_seconds = 7100,
  display_inclination = '≈ 93', display_eccentricity = '≈ 0.001', display_period = '约 2 小时'
WHERE id = '2001-mars-odyssey';

UPDATE mars_spacecraft SET
  orbit_a = 5.80, orbit_e = 0.57, inclination_deg = 86,   raan_deg = 250, arg_periapsis_deg = 60,  period_seconds = 27000,
  display_inclination = '≈ 86', display_eccentricity = '≈ 0.57', display_period = '约 7.5 小时'
WHERE id = 'mars-express';

UPDATE mars_spacecraft SET
  orbit_a = 4.20, orbit_e = 0.01, inclination_deg = 74,   raan_deg = 330, arg_periapsis_deg = 200, period_seconds = 7100,
  display_inclination = '≈ 74', display_eccentricity = '≈ 0.01', display_period = '约 2 小时'
WHERE id = 'exomars-tgo';

UPDATE mars_spacecraft SET
  orbit_a = 8.20, orbit_e = 0.91, inclination_deg = 151,  raan_deg = 90,  arg_periapsis_deg = 30,  period_seconds = 262656,
  display_inclination = '≈ 151', display_eccentricity = '≈ 0.91', display_period = '约 73 小时'
WHERE id = 'mom';

UPDATE mars_spacecraft SET
  orbit_a = 7.60, orbit_e = 0.33, inclination_deg = 25,   raan_deg = 180, arg_periapsis_deg = 90,  period_seconds = 197920,
  display_inclination = '≈ 25', display_eccentricity = '≈ 0.33', display_period = '约 55 小时'
WHERE id = 'hope';

UPDATE mars_spacecraft SET
  orbit_a = 6.20, orbit_e = 0.75, inclination_deg = 49,   raan_deg = 120, arg_periapsis_deg = 210, period_seconds = 64800,
  display_inclination = '≈ 49', display_eccentricity = '≈ 0.75', display_period = '约 18 小时'
WHERE id = 'mars2-orbiter';

UPDATE mars_spacecraft SET
  orbit_a = 6.40, orbit_e = 0.75, inclination_deg = 49,   raan_deg = 150, arg_periapsis_deg = 250, period_seconds = 64800,
  display_inclination = '≈ 49', display_eccentricity = '≈ 0.75', display_period = '约 18 小时'
WHERE id = 'mars3-orbiter';

UPDATE mars_spacecraft SET
  orbit_a = 7.40, orbit_e = 0.75, inclination_deg = 35,   raan_deg = 60,  arg_periapsis_deg = 160, period_seconds = 88200,
  display_inclination = '≈ 35', display_eccentricity = '≈ 0.75', display_period = '约 24.5 小时'
WHERE id = 'mars5';

UPDATE mars_spacecraft SET
  orbit_a = 5.40, orbit_e = 0.85, inclination_deg = 1.5,  raan_deg = 0,   arg_periapsis_deg = 0,   period_seconds = 28800,
  display_inclination = '≈ 1.5', display_eccentricity = '≈ 0.85', display_period = '约 8 小时'
WHERE id = 'phobos2';
