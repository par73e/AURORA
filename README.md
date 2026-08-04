# AURORA

AURORA 是一个把太阳系探索、地球轨道活动与天文事件放进统一时空界面的数字宇宙平台。

当前只实现第一个纵向闭环：**ORBIT 地球局部系统**。

## 当前初版

- Three.js 三维地球，可拖动、缩放，具备昼夜光照和大气边缘。
- ISS、中国空间站（天和）和哈勃望远镜的公开 OMM 轨道数据。
- 浏览器使用 satellite.js + SGP4 计算当前位置和轨道线。
- 少量代表性发射场。
- Launch Library 2 的近期真实发射事件。
- Go API、PostgreSQL 缓存、来源与同步时间说明。
- 只适配电脑端；窄屏显示桌面访问提示。

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

当前先实现 `EarthSystem`。未来拉远时切换到太阳系压缩尺度，接近地球时进入地心千米尺度，二者共享同一时间状态。

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

浏览器访问 `http://localhost:5173/AURORA/`（地址栏路径为 `/AURORA/`）。开发环境通过 Vite 把 `/api` 转发到 Go 的 `http://localhost:8080`。

后端各目录的初学者说明见 [`backend/README.md`](backend/README.md)。数据库表会在后端启动时自动创建，不需要逐条执行 SQL。
