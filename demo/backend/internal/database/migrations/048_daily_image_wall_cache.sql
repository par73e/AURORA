-- 每日一图页面的外部图源按天缓存。前端仍请求 AURORA 后端；
-- 后端每天最多刷新一次外部 APOD / NASA Image Library 聚合结果，避免每次打开页面都实时请求外部源。
CREATE TABLE daily_image_wall_cache (
    cache_date DATE PRIMARY KEY,
    recent JSONB NOT NULL,
    collection JSONB NOT NULL,
    generated_at TIMESTAMPTZ NOT NULL,
    refreshed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX daily_image_wall_cache_refreshed_idx
    ON daily_image_wall_cache(refreshed_at DESC);
