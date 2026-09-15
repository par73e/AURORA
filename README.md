# AURORA

AURORA 是一个把太阳系探索、行星任务、地球轨道活动与本地天文观测放进统一时空界面的数字宇宙平台。

## 当前初版

- **太阳系 / 行星探索**：太阳系总览，以及地球、月球、火星和主要行星的 Three.js 场景；火星与月球包含飞行器、轨道和着陆点目录。
- **ORBIT 地球系统**：公开 OMM 轨道数据、satellite.js + SGP4 实时位置与轨道线、发射场和近期真实发射事件。
- **SKY 天文观测**：本地月相与星历、天气与逐小时观测评分、星图、天象日历、NASA 每日一图和专题图片墙。
- **本地光污染参考**：读取年度 VIIRS 静态栅格，按经纬度估算辐射值、SQM 与 Bortle 等级；运行时不依赖外部光污染 API。
- **数据后端**：Go API、PostgreSQL 缓存、自动迁移、来源与同步时间说明。

当前仍以桌面端体验为主；窄屏会显示桌面访问提示。

## 架构方向

地球不是孤立页面，而是未来太阳系场景中的局部模块：

```text
SolarSystemScene
└── EarthSystem
    ├── EarthSurface
    ├── OrbitLayer
    ├── SpacecraftLayer
    ├── LaunchSiteLayer
    └── LaunchEventLayer
```

当前已经具备太阳系总览与多个局部场景；后续仍需继续统一各场景的时间状态、搜索入口和移动端信息架构。

## 本地运行

需要 PostgreSQL 17、Go 和 pnpm。

第一次运行先创建本地数据库：

```bash
createdb aurora
```

后端：

```bash
cd backend
go run ./cmd/api
```

前端：

```bash
cd frontend
pnpm dev
```

浏览器访问 `http://localhost:5173/aurora/`（地址栏路径为小写 `/aurora/`，地球页为 `#earth`）。开发环境通过 Vite 把 `/api` 转发到 Go 的 `http://localhost:8080`。

光污染静态数据准备见 [`backend/data/light-pollution/README.md`](backend/data/light-pollution/README.md)，后端各目录说明见 [`backend/README.md`](backend/README.md)。数据库表会在后端启动时自动创建，不需要逐条执行 SQL。
