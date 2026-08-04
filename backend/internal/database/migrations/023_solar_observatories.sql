-- 太阳系场景新增：太空天文台 JWST（日地 L2）与斯皮策（日心轨道）
-- 它们不是地球轨道器，无法用 TLE 渲染，故放入太阳系深空探测器目录（JPL Horizons 真实位置）。
-- JWST NAIF -170、斯皮策 NAIF -9（均实测 2026 数据可用）；orbit_kind=track 不画轨迹线。
INSERT INTO deep_space_probes
    (id, name_zh, name_en, operator_name, launch_date, mission_type, target, description,
     naif_id, precision_grade, color, sort_order, launch_site, launch_vehicle, orbit_kind)
VALUES
    ('jwst', '詹姆斯·韦布空间望远镜', 'JAMES WEBB SPACE TELESCOPE', 'NASA / ESA / CSA', '2021-12-25', '空间天文台', '日地 L2 · 红外宇宙',
     '史上最强大的红外空间望远镜，运行于日地拉格朗日 L2 点晕轨道，观测早期星系、系外行星大气与恒星形成。',
     '-170', 'A', '#e8c36a', 12, '法属圭亚那库鲁（法属圭亚那）', 'Ariane 5', 'track'),
    ('spitzer', '斯皮策空间望远镜', 'SPITZER SPACE TELESCOPE', 'NASA', '2003-08-25', '空间天文台', '日心轨道 · 红外',
     '大型红外天文台，运行在地球后方的日心轨道（地球尾随轨道），2020 年任务结束，是斯皮策深空红外巡天的主力。',
     '-9', 'A', '#ff8c69', 13, '卡纳维拉尔角 SLC-17B（美国）', 'Delta II 7920H', 'track');
