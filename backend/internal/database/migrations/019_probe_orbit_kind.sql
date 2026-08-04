-- 探测器轨道绘制方式：绕日探测任务画"太阳在焦点"的拟合椭圆（一个完整轨道圈），
-- 其余（新视野/旅行者等逃逸、巡航轨迹）画真实采样折线。
-- 背景：帕克周期约 89 天，±90 天采样窗口含两圈轨道，折线绕两圈视觉上弯弯绕绕；
-- 拟合椭圆取真实采样近日/远日距离与近日点方向，太阳位于椭圆焦点，视觉自然。
ALTER TABLE deep_space_probes ADD COLUMN orbit_kind TEXT NOT NULL DEFAULT 'track';
UPDATE deep_space_probes SET orbit_kind = 'ellipse' WHERE id IN ('parker', 'solar-orbiter');
