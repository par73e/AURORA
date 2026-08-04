-- 深空探测器目录收窄：只保留"太阳系尺度巡航"的探测器
-- 删除绑定行星/绕行星运行的轨道器（贝皮科伦坡绕水星、朱诺绕木星）与
-- 目标为行星/小行星的近期任务（灵神星、OSIRIS-APEX——当前虽是日心巡航，但
-- 轨道范围小、最终环绕目标天体，不符合"太阳系层面大范围飞行"的展示意图）
DELETE FROM deep_space_probes
WHERE id IN ('bepicolombo', 'juno', 'psyche', 'osiris-apex');
-- 剩余目标重排 sort_order：帕克 / 太阳轨道器 / 新视野 / 旅行者 1 / 旅行者 2
UPDATE deep_space_probes SET sort_order = CASE id
    WHEN 'parker'        THEN 1
    WHEN 'solar-orbiter' THEN 2
    WHEN 'new-horizons'  THEN 3
    WHEN 'voyager-1'     THEN 4
    WHEN 'voyager-2'     THEN 5
    ELSE sort_order
END;
