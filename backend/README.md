# AURORA 后端入门说明

后端现在只承担三件事：

1. 从公开数据源获取轨道和发射事件。
2. 把获取到的原始数据与整理后的字段存进 PostgreSQL。
3. 通过 HTTP 接口把一份适合前端直接使用的 ORBIT 总览返回出去。

这是一套“小而完整”的初版结构。它没有为了显得专业而拆出很多空目录，但数据源、业务数据、数据库和接口已经彼此分开，后续增加用户登录或新的太阳系模块时无需推翻重写。

## 目录怎么读

```text
backend/
├── cmd/api/main.go                 程序入口：组装各部分并启动服务
├── internal/config/config.go       读取端口、数据库地址等运行配置
├── internal/database/              连接 PostgreSQL、执行建表脚本
├── internal/orbit/                 ORBIT 的数据结构和数据库读写
├── internal/syncer/                对接 CelesTrak、Launch Library 2
└── internal/httpapi/router.go       定义浏览器可以访问的 API 地址
```

如果用熟悉的 Java / Spring Boot 来类比：

- `cmd/api/main.go` 类似启动类与一小部分 Bean 装配。
- `internal/httpapi` 类似 Controller。
- `internal/orbit/repository.go` 类似 Repository / Mapper。
- `internal/orbit/model.go` 类似 DTO 与实体定义。
- `internal/syncer` 类似定时任务加第三方 API Client。
- `internal/database/migrations` 类似 Flyway 的 SQL 迁移文件。

Go 的 `internal` 有实际限制含义：它里面的包只允许被这个后端项目内部使用，可以避免未来代码互相随意依赖。

## 数据流

```text
CelesTrak ───────────────┐
                        ├─ syncer ─ PostgreSQL ─ repository ─ HTTP API ─ 前端
Launch Library 2 ────────┘
```

- 服务启动时先同步一次。
- 轨道数据之后每 2 小时刷新。
- 发射事件之后每 30 分钟刷新。
- 外部服务临时不可用时，已经写进 PostgreSQL 的缓存仍可供前端使用。
- 发射事件同时保存第三方原文、原始 JSON 和中文整理字段；界面默认使用中文，只有外文来源才提供“查看原文”。
- 浏览器根据后端提供的 OMM 轨道根数，用 SGP4 计算此刻位置；后端不需要每秒计算并写数据库。

## 本地启动

第一次创建数据库：

```bash
createdb aurora
```

启动服务：

```bash
go run ./cmd/api
```

默认连接：

- API：`http://localhost:8080`
- 数据库：`postgres://当前系统用户名@localhost:5432/aurora`

如需修改，可通过环境变量传入：

```bash
PORT=8081 DATABASE_URL='postgres://user:password@localhost:5432/aurora?sslmode=disable' go run ./cmd/api
```

建表不需要手工执行。服务启动时会自动运行尚未执行过的 `migrations/*.sql`，并通过 `schema_migrations` 记录版本。

## 当前接口

- `GET /api/health`：检查服务和数据库是否可用。
- `GET /api/v1/orbit/overview`：返回航天器、最新轨道数据、发射场、未来 30 天事件及数据新鲜度。

当前暂时没有写入类接口，也没有登录。未来增加账号时，可以新增独立的 `internal/identity` 模块和用户表，不需要让轨道数据依附于用户表。
