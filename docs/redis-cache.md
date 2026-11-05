# AURORA 页面缓存与 Mac 查看入口

部署日期：2026-11-05（Asia/Shanghai）。本次采用独立的轻量缓存网关，不替换正式 API 二进制、不重启其数据同步任务，也不重新发布前端。Redis 是可丢弃的响应缓存，原 PostgreSQL 数据和原有天气、月相缓存继续工作。

## 请求链路

Nginx `/api/` → `aurora-cache.service`（本机 18084）→ 正式 API（本机 18083）。缓存网关连接独立的 `aurora-cache-redis.service`（本机 6380）。Redis 不开放公网端口。网关停止时，Nginx 的备用上游是原 API；Redis 故障时，网关短暂跳过缓存并直接访问 API。

## 当前缓存范围

| GET 路径（省略 `/api/v1`） | 最长有效期 |
| --- | --- |
| `/orbit/overview` | 60 秒 |
| `/moon/spacecraft`、`/mars/spacecraft` | 120 秒 |
| `/moon/landing-sites`、`/mars/landing-sites` | 6 小时 |
| `/voyage/probes`、`/astronomy/image-wall` | 5 分钟 |
| `/astronomy/events` | 5 分钟 |
| `/astronomy/conditions`（不带 `time`） | 120 秒 |

有效期还会受当日 UTC 零点限制。采用按需生成和 TTL 到期重建，没有新增定时抓取、批量预热、后台轮询或同步后主动失效；新入库数据最多在上述窗口内更新到响应缓存。数据本身的来源时间、同步时间、预报范围保持原义，不将缓存生成时间当作上游更新时间。

只缓存成功的 JSON 响应，单项正文上限 1 MiB；超过上限转交原 API。认证、Cookie、Range、未知参数、重复参数、实时评分、带指定时间的天气请求以及其他路径均绕过缓存。坐标参数不进行粗略合并；参数排序后取 SHA-256 生成键后缀，地点数值不直接出现在键名中。响应缓存可见的 JSON 仍可能包含公共天气和地点数据；不记录用户身份。

键前缀为 `aurora:prod:response:v1:`。值是可读 JSON：`body` 是实际接口内容，`storedAt` 是本次缓存生成时间。HTTP 响应头 `X-Aurora-Cache` 显示 `HIT`、`MISS` 或 `BYPASS`。缓存命中前后 JSON 语义相同，序列化空白可能不同。

## 资源控制

- Redis：`maxmemory 64mb`、`allkeys-lfu`；不启用 AOF/RDB 持久化。进程 MemoryHigh 96 MiB、MemoryMax 128 MiB，CPU 上限为单核的 25%。数据上限不是进程 RSS 上限。
- 网关：Go 软内存目标 48 MiB，进程 MemoryHigh 64 MiB、MemoryMax 96 MiB，CPU 上限为单核的 30%；Redis 连接最多 8 条，上游连接最多 16 条。
- Redis 操作预算 60 ms，不重试命令；连接故障后旁路 10 秒。相同缓存键的并发首次请求合并为一次上游加载。请求取消不会中断其他等待者共享的加载。
- 首次访问和过期后的首次请求仍需等待原 API。不保证渲染帧率、纹理或海外图片下载提速。

## 浏览器内存缓存（2026-11-05 后续增补）

`frontend/src/dataCache.ts` 在统一 API 入口复用近期成功的 JSON 数据。只保存当前标签页内存，刷新、关闭后释放；不使用 localStorage、IndexedDB、Service Worker 或额外定时器。

- 轨道总览和目录分页有效期 30 秒；月火航天器、深空、天气、天象、图片墙有效期 60 秒；着陆点、光污染和逆地理编码 10 分钟，地点搜索 5 分钟。到 UTC 零点会提前失效。
- 缓存保留字符串预算为 5 MiB、最多 64 项，按最近使用顺序淘汰；这是缓存自身字符串的预算，不是整个网页或 JS 堆的内存上限。图片、纹理、模型和场景对象不进入此缓存。
- 缓存过期后按正常流程请求，失败不会写入缓存，不返回已过期缓存；已有组件继续使用各自的错误和资料状态展示。不新增后台刷新或轮询。
- 经纬度、海拔、时区和日期参数保持隔离；实时评分、指定时刻的月相/天气、正则目录查询绕过本层。来源时间、轨道历元保持不变。
- 相同请求合并；单个使用者取消不会影响其他等待者，全部取消才中止网络请求，迟到结果不能覆盖后续请求。缓存返回独立解析的数据，避免组件修改影响其他场景。
- `no-store`、`private` 及与身份/语言等相关的 `Vary` 响应不保留；同一标签页固定的 Origin 与 fetch 已解码的压缩格式不妨碍复用。

部署从既有正式版源代码及其归档 patch 重建，保留当前线上图标；只增补客户端缓存及相关测试。工作区其他未发布改动未随本次发布。旧静态资源保留在新发布目录，兼容尚未刷新的旧页面。

前端发布目录为 `/opt/aurora-full/web/releases/20261105-client-cache/web/aurora`，发布根目录保存了增量补丁、原部署补丁和构建文件校验信息。Nginx 本轮回退配置为 `/etc/nginx/sites-available/aurora-resume-new.before-client-cache-20261105`；恢复它可撤回浏览器缓存前端，同时保留之前的服务器 Redis 缓存。

## Mac RedisInsight

本机 `127.0.0.1:16381` 经 SSH 别名 `tencent` 转发到服务器本机 6380。LaunchAgent：`~/Library/LaunchAgents/com.louis.aurora-redis-tunnel.plist`，登录后自动启动，SSH 退出后由 launchd 重试。用户休眠、离线或锁屏情况下的长时间恢复行为尚需实际使用验证。网站不依赖 Mac 或此通道在线。

RedisInsight 连接名称：**AURORA**。用户名为 `aurora-insight`，仅提供读取缓存及常用诊断的权限；写入缓存的 `aurora-app` 使用不同凭据。检查账号不能写入或清空缓存，部分管理功能可能因权限限制不可用。RedisInsight 只展示服务器 Redis，浏览器内存缓存不会出现在这里。

本机凭据和导入文件在 `~/Library/Application Support/AURORA/Redis/`，目录 0700、敏感文件 0600；不要提交、分享或粘贴内容。服务器凭据在 `/etc/aurora-cache/`，不进入仓库。

## 维护与回退

源码入口：`backend/cmd/cacheproxy`、`backend/internal/responsecache`。不含秘密的配置模板在 `backend/deploy/redis-cache/`。

服务器程序：`/opt/aurora-cache/releases/20261105/aurora-cache`。

只读检查示例：

```sh
ssh tencent 'systemctl is-active aurora-cache.service aurora-cache-redis.service'
ssh tencent 'curl --silent http://127.0.0.1:18084/internal/cache-stats'
```

统计接口只在本机网关提供，不经公开 `/api/` 路由开放。缓存不参与健康接口的结果。

Nginx 原配置保留于 `/etc/nginx/sites-available/aurora-resume-new.before-redis-20261105`。需要整体回退时，经授权恢复该配置、执行 `nginx -t` 并 reload Nginx，流量即可直接回到原正式 API；无需删除文件、改数据库或重建前端。随后是否停用新增服务另行决定。

## 已验证

- 后端 `go test ./...` 通过；缓存模块的 `go test -race` 及 `go vet` 通过。
- 真实 Redis 写入、读取、TTL、只读 SSH 连接通过；首页和健康、轨道、月球、火星、深空、图片墙接口经现有 Nginx/TLS 入口返回 200，缓存命中前后 JSON 内容一致。
- Mac RedisInsight 已导入并实际连接 **AURORA · 页面缓存**，已打开 `aurora:prod:response:v1:moon/landing-sites`，确认能看到 TTL 和月球着陆点 JSON。另验证了带地点的天气、天象请求：首轮 MISS、再次 HIT，JSON 内容一致。
- 分别临时停止新 Redis 和新网关验证降级，网站接口均继续返回 200；随后恢复服务，缓存重新命中。正式 API 和 AS Robot Redis 没有重启。
- 接口耗时只做了少量服务器内部对比，不代表真实访客网络延迟或页面整体性能基准；没有执行高压测试。
- 浏览器缓存增补：117 项前端测试及生产构建通过。在 ego lite 中验证了首次请求 1 次、相同 API 再次请求增加 0 次，以及有效期内断网仍能读取已缓存数据。线上实际执行月球→火星→月球切换，预热后没有新增接口请求；刷新网页后重新出现两条火星数据请求，确认缓存不会跨刷新保留。
