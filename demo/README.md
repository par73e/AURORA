# AURORA 独立展示版

访问入口：https://par73e.cn/aurora/ 。仅支持电脑浏览器，与正式版保持一致。

本目录是正式版的独立副本，不改变上级 `frontend/`、`backend/` 和本地数据库。保留 Vue/Three.js 界面、原来的 `/aurora/` 路径和 hash 路由；Go 后端读取本地 JSON、年度光污染栅格并计算月相/基础天象。不连接数据库、不启动外部天文同步任务；地点查询实时调用高德，密钥仅放在服务器环境文件。

## 功能与数据边界

- 太阳系、行星、月球、火星、地球与镜头操作：沿用原界面，贴图全部由本服务器提供。
- 卫星与深空探测器：使用导出时的轨道/位置数据，动画只是基于旧数据的示意，不能作为实时追踪结果。
- 发射计划、任务资料、着陆点：本地快照，原始更新时间保留。
- 星图、月相：按选择的地点和时间本地计算。
- 天象日历：月球周期与季节节点独立计算，另外加载已有天象快照；不保证覆盖全部日食、流星雨和行星事件。
- 天气与空气质量：服务器按所选地点在线请求 Open-Meteo，提供当前天气、今天与明天的逐小时预报、PM2.5/PM10 和气溶胶数据。观测评分基于预报、月光等计算，按地点缓存一小时。天气失败时显示不可用，不回退上海静态数据；空气质量独立失败时相应字段暂缺。超出预报窗口的评分为 null。
- 地点搜索：高德实时地名查询。进入天文观测页或地球页时申请浏览器定位，浏览器提供真实经纬度，后端经高德坐标转换和逆地理编码获取地名。定位失败或拒绝授权时不使用上海示例冒充真实位置，仍可手动搜索或输入经纬度。旧示例地点缓存会清除。
- 光污染：2025 年 VIIRS 中国范围年度数据，按位置读取栅格并估算；不是当日实测。
- 图片墙：18 张已下载、解码验证和压缩的历史天文图片，保存原始发布日期、署名、来源和使用说明。不宣称 NASA 当日更新。来源链接仍跳转外部网站。
- 字体使用系统字体，地球夜景贴图改为本地资源，不请求 Google Fonts/unpkg。

轨道与任务快照的采集时间和数据数量见 `data/manifest.json`；天气采集时间见在线 conditions API 的 `retrievedAt`。页面上的悬浮数据说明已移除，数据边界保留在本文件中。

## 目录

- `frontend/`：独立前端源码及资源。
- `backend/cmd/demo/`：服务器入口，读取本地天文资料并实时请求地点和天气。
- `backend/cmd/export/`：电脑端只读数据导出工具；保留的 repository/database 源码只为导出使用。
- `data/`：可上传的公开 JSON 数据和 VIIRS 栅格，无账号或密钥。
- `scripts/prepare-assets.py`：在 Mac 下载并验证图片，需要 curl、sips。
- `scripts/build.sh`：前端构建、Linux amd64 后端交叉编译、打包。
- `scripts/deploy.sh`：首次迁移部署，移除 CSTS 容器与已识别的旧服务。
- `scripts/update-data.sh`：只上传数据和图片，不处理 CSTS、不修改 Nginx。
- `deploy/`：Nginx 配置、systemd 单元、首次安装脚本。

## 本地运行

```sh
cd demo/backend
go run ./cmd/demo -addr 127.0.0.1:18082 -data ../data
```

另开终端：

```sh
cd demo/frontend
pnpm install
pnpm dev
```

访问 `http://localhost:5174/aurora/`。刷新仍保留相应 hash 页面。

## 从电脑更新数据，再上传

1. 若要更新卫星、任务、发射计划和外部天象，先让本地正式版完成数据同步。这些数据从本地 `aurora` PostgreSQL 数据库读取，导出工具自身不会更新数据库，也不会调用 Horizons/CelesTrak。
2. 在项目根目录执行：

```sh
demo/scripts/refresh-local.sh
```

导出工具只读取本地数据库；图片脚本下载本地缓存对应的图片。天气由服务器自行在线请求，不再需要电脑下载或上传。图片无法下载时剔除该项，完全失败时保留上一份图片墙。图片不会重新标为今天发布。

默认数据库为 `postgres://localhost:5432/aurora?sslmode=disable`；需要自定义时可在本机通过 `DATABASE_URL` 环境变量指定。不要把包含密码的连接字符串上传。

3. 上传数据和图片：

```sh
demo/scripts/update-data.sh
```

服务重新读取数据，无需重新构建前端。更新脚本创建新版本目录并切换；健康检查失败会自动回退旧版本。

4. 若改了源码，执行 `demo/scripts/build.sh`，再运行 `demo/scripts/deploy.sh`。构建机需要 Go 1.26、Node.js、pnpm；当前包适用于 Linux amd64。

## 服务器

- 地址：`tencent` SSH 别名。
- `/opt/aurora-demo/current`：当前版本符号链接。
- `/opt/aurora-demo/releases/`：历史版本。
- `/opt/aurora-demo/backups/`：迁移前配置和简历校验记录。
- `aurora-demo.service`：www-data 用户、开机启动、异常重启，监听 `127.0.0.1:18082`；内存上限 256 MB。
- Nginx：`/aurora/` 提供前端，`/api/` 转发后端，HTTPS 使用已有证书；`/resume/` 原配置与文件保持不变。
- CSTS 的容器、网络和四个旧 systemd 服务移除；数据库卷、镜像、旧文件与原快照保留。未执行全局 Docker 清理。

```sh
ssh tencent 'sudo systemctl status aurora-demo --no-pager'
ssh tencent 'sudo journalctl -u aurora-demo -n 30 --no-pager'
```

## 路由示例

- `/aurora/#home`
- `/aurora/#solar-system`
- `/aurora/#earth`、`#objects`、`#sites`、`#launches`
- `/aurora/#moon-scene`、`#moon-profile`、`#moon-objects`、`#moon-sites`
- `/aurora/#mars-scene`、`#mars-profile`、`#mars-objects`、`#mars-sites`
- 其他行星：`#mercury`、`#venus`、`#jupiter`、`#saturn`、`#uranus`、`#neptune`、`#sun`
- `/aurora/#astronomy-conditions`
- `/aurora/#astronomy-sky`
- `/aurora/#astronomy-events`
- `/aurora/#astronomy-daily-image`

完整映射沿用 `frontend/src/routes.ts`。

## 首版部署验证（2026-10-08，部分行为已被下方实时更新替代）

- 前端类型检查、生产构建通过；100 个前端测试通过。
- 后端 `go test ./...` 通过，新增离线日历和目录筛选检查；初版静态天气边界测试随在线天气恢复而移除。
- 本地 18 个 API 检查通过；服务器再次检查主要 API，18 张图片经 HTTPS 返回 JPEG 文件。
- 桌面浏览器检查全部 43 个主要页面/栏目路由，刷新保留 hash；另检查了 7 个行星入口别名。
- 历史方案：上海默认示例地点正常显示月相、天气快照、光污染与观测建议。其他城市不套用上海天气；未来超出天气预报窗口的评分返回 null。
- CSTS Compose 项目的 9 个容器、专属网络和四个旧 systemd 单元已移除，`/CSTSd/` 返回 410。
- `aurora-demo.service` 已启用且正常运行；验证时内存约 7 MB，无异常重启。
- 简历目录 6 个文件的 SHA-256 全部与迁移前一致，`https://par73e.cn/resume/` 返回 200。
- 从电脑上传数据的脚本已实际执行并验证服务正常；Nginx 配置和简历未受影响。
- 浏览器验证使用桌面尺寸；没有验证手机交互，本项目沿用正式版的电脑浏览器限制。

## 实时定位更新（2026-10-08）

高德密钥保存在 `/etc/aurora-demo.env`（root:root，0600），systemd 通过 EnvironmentFile 读取；不包含在前端、数据包或 Git 文件中。本地运行后端时需设置 `AMAP_WEB_KEY`。真实定位依赖 HTTPS 与浏览器定位授权，高德负责坐标转换、地名解析和地点搜索；系统定位失败时仍可手动设置地点。天气已在后续更新中恢复在线请求，见下文。

服务器实测：逆地理编码返回上海市黄浦区（adcode 310101，约 0.38 秒）；搜索上海市崇明区返回地名与有效经纬度（约 0.10 秒）。该结果来自实时高德请求，并非预置城市表。前端构建与 100 项测试通过，简历仍返回 200。

## 在线天气更新（2026-10-08）

服务器上的 conditions、score 接口已接回 Open-Meteo。逐小时天气表、今夜/明夜评分和推荐窗口使用在线预报；定位、天气和空气质量在线获取，其余轨道、任务和图片仍使用本地资料。预测是模型预报，不是未来实测值。单次上游请求超时 6 秒，地点缓存 1 小时；天气故障不展示虚构数据，空气质量独立失败时允许相应字段暂缺。

服务器实测上海和北京均返回 48 小时天气、48 小时空气质量和 48 条评分，采集时间为 2026-10-08T04:56:37Z（北京时间 12:56:37）；单次约 2.68 秒、0.45 秒。score 接口在预报窗口内正常返回分数。前端 100 项测试、后端测试和生产构建通过，简历仍返回 200。旧 weather-shanghai 文件不再由后端读取，也不再打入新部署包。

## Git 与资源准备

临时部署方案维护在独立分支，不替代正式版 main。Git 保存源码和公开 JSON；构建输出、下载图片、夜景贴图与 VIIRS 栅格均被忽略。新检出后需运行 `scripts/prepare-assets.py` 准备图像，并将正式版本地的 `backend/data/light-pollution/viirs-2025-cn.avnl` 复制到 `demo/data/`，再运行或打包。高德密钥需要单独配置。

`docs/deployed.jpg` 是首版部署的历史截图，其中的数据说明已按后续要求移除，不代表最终页面。
