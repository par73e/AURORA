-- 上游目录永久下架的目标（如 CelesTrak 对已离目录的 NORAD 编号返回 404）记为
-- unpublished：它不是同步故障，单目标下架不应把整个数据源判失败。
ALTER TABLE spacecraft_sync_status DROP CONSTRAINT spacecraft_sync_status_status_check;
ALTER TABLE spacecraft_sync_status ADD CONSTRAINT spacecraft_sync_status_status_check
    CHECK (status IN ('success', 'failed', 'unsupported', 'canceled', 'unpublished'));
