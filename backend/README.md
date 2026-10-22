# CloudCTF Go Backend

This directory is the Go rewrite target for the previous `backend_back` FastAPI service.

The first migration step keeps the existing frontend contract:

- player API: `/api`
- admin API: `/api-admin`
- public API: `/api/public`
- static files: `/static`

## Development

```bash
go run ./cmd/server
```

必须运行整个 `cmd/server` 包；不要使用 `go run cmd/server/main.go`，因为单文件模式不会编译同目录下按平台拆分的重启实现。`go run` 属于开发模式，不提供在线安装更新；生产环境请使用 GitHub Release 构建出的单二进制。

## Default credentials

首次启动后，系统会自动创建两个默认账号：

| 角色   | 用户名  | 密码        |
|--------|---------|-------------|
| 管理员 | `admin` | `admin123456` |
| 选手   | `player1` | `password123` |

管理后台：`http://localhost:8005/admin`（或通过前端访问 `/admin`）。

> **⚠️ 安全提示**：生产环境部署后请立即修改默认密码。

## CLI 重置管理员密码

忘记管理员密码时，可以直接使用主程序的 CLI 子命令。命令只连接数据库，不会启动 HTTP 服务：

```bash
go run ./cmd/server reset-admin-password --username admin
```

使用 Release 二进制时：

```bash
./cloudctf-linux-amd64 reset-admin-password --username admin
```

程序会在终端中隐藏输入新密码并要求确认。它默认读取 `.env` 中的 `DATABASE_URI`，也可以显式指定：

```bash
./cloudctf-linux-amd64 reset-admin-password \
  --username admin \
  --database-uri 'sqlite://data/cloudctf.db'
```

脚本环境可以通过标准输入传入密码：

```bash
printf '%s\n' "$NEW_ADMIN_PASSWORD" | \
  ./cloudctf-linux-amd64 reset-admin-password --username admin --password-stdin
```

重置会恢复账号为启用状态、撤销已有刷新会话，并要求管理员下次登录后再次修改密码。修改 `.env` 中的 `FIRST_SUPERUSER_PASSWORD` **不会**重置已经存在的管理员。

## Database

The server accepts a single `DATABASE_URI` value and uses GORM for database access.

```bash
DATABASE_URI=sqlite://data/cloudctf.db go run ./cmd/server
```

```bash
DATABASE_URI="mysql://root:root@tcp(localhost:3306)/cloudctf?charset=utf8mb4&parseTime=True&loc=Local" go run ./cmd/server
```

```bash
DATABASE_URI="postgres://cloudctf:password@localhost:5432/cloudctf?sslmode=disable" go run ./cmd/server
```

Supported schemes are `sqlite://` (and `sqlite3://`), `mysql://`, `postgres://`, and `postgresql://`. If `DATABASE_URI` is not set, embedded SQLite is used by default at `data/cloudctf.db`. MySQL and PostgreSQL databases must exist before startup; CloudCTF automatically creates and migrates their tables.

## Configuration

后端支持两种配置方式，按优先级排序：

1. **真实环境变量**（`export KEY=value`）
2. **配置文件**（默认读取 `.env`，也可通过 `CONFIG_FILE` 指定）

复制示例文件后按需修改即可生效：

```bash
cp .env.example .env
```

### 环境变量清单

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `APP_NAME` | 服务名称 | `CloudCTF API` |
| `APP_ENV` | 运行环境 | `development` |
| `HOST` | 监听地址 | `0.0.0.0` |
| `PORT` | 监听端口 | `8005` |
| `LOG_LEVEL` | 日志级别（debug/info/warn/error） | `info` |
| `LOG_FILE` | 日志文件路径 | `logs/backend.log` |
| `DATABASE_URI` | 数据库连接（`sqlite://...`、`mysql://...`、`postgres://...` 或 `postgresql://...`） | `sqlite://data/cloudctf.db` |
| `REDIS_ADDR` | Redis 地址 | `localhost:6379` |
| `JWT_SECRET` | JWT 签名密钥 | `change-me` |
| `UPLOAD_DIR` | 上传文件目录 | `static/uploads` |
| `STATIC_DIR` | 静态资源目录 | `static` |
| `FRONTEND_DIR` | 前端构建产物目录 | `../frontend/dist` |
| `BACKUP_DIR` | 备份文件目录 | `data/backups` |
| `BACKUP_RETENTION` | 备份保留天数 | `7` |
| `ALLOWED_ORIGINS` | 允许的 CORS 来源（逗号分隔）；生产环境不能使用 `*` | `*` |
| `CONFIG_FILE` | 配置文件路径（默认 `.env`） | `.env` |

生产环境启用 `APP_ENV=production` 时，必须显式配置：

- `JWT_SECRET`：非默认的强随机字符串
- `ALLOWED_ORIGINS`：实际访问源，例如 `https://ctf.example.com` 或 `http://服务器IP:8005`

如果 `ALLOWED_ORIGINS` 为空或未配置，会回退到默认值 `*`，生产模式下服务会拒绝启动。

## API compatibility

The Go backend keeps the existing frontend contracts:

- admin API: `/api-admin`
- player API: `/api`
- public API: `/api/public`

During the migration, generic admin/player resource endpoints are backed by a GORM JSON resource table so the new frontend can be integrated before all domain tables are fully migrated.

## Structure

```text
cmd/
  server/   HTTP API process
internal/
  config/
  httpserver/
```

The backend intentionally starts without a global repository layer. Services can use the database directly, and complex modules can add local `query.go` or `store.go` files when the query logic grows.

