<div align="center">

# CloudCTF

**CTF 竞赛与漏洞训练靶场平台**

单一可执行文件 · 内置前端 · 开箱即用 · 支持动态靶机

[![Release](https://img.shields.io/github/v/release/hexbay/CloudCTF?style=flat-square)](https://github.com/hexbay/CloudCTF/releases)
[![Build](https://img.shields.io/github/actions/workflow/status/hexbay/CloudCTF/integration-tests.yml?branch=main&style=flat-square&label=build)](https://github.com/hexbay/CloudCTF/actions)
[![Docs](https://img.shields.io/badge/docs-在线文档-0f766e?style=flat-square)](https://hexbay.github.io/CloudCTF/)
[![License](https://img.shields.io/badge/license-MIT-blue?style=flat-square)](LICENSE)

[在线预览](http://cloud-ctf.lostpeach.cn/) · [在线文档](https://hexbay.github.io/CloudCTF/) · [下载](https://github.com/hexbay/CloudCTF/releases) · [快速开始](#快速开始)

</div>

---

## 简介

CloudCTF 是一个面向 **CTF 竞赛** 与 **漏洞训练靶场** 的一体化平台。**前端（选手端 + 管理后台）已内嵌进单一可执行文件**，无需 Nginx、无需单独部署前端，下载一个二进制即可运行。

- **选手端**：浏览器访问 `/`
- **管理后台**：浏览器访问 `/admin`

> 在线预览环境：**http://cloud-ctf.lostpeach.cn/**
>
> 完整使用与部署文档请见在线文档站：**https://hexbay.github.io/CloudCTF/**

## 功能特性

- **竞赛管理**：比赛、题目、提交、计分、排行榜、公告、审计日志
- **多题型**：Web / Pwn / Crypto / 动态靶机 / 理论题 / 考试
- **动态靶机**：基于 Docker 的容器化靶机，支持单容器与多容器模板编排
- **权限体系**：管理员 / 租户 / 选手多角色，资源级权限控制
- **一体化交付**：前端内嵌，单二进制 / 单镜像部署
- **灵活存储**：默认 SQLite 零依赖启动，生产可切换 MySQL

![比赛管理](docs/public/images/competition.png)

## 快速开始

### 方式一：单二进制（推荐）

从 [Releases](https://github.com/hexbay/CloudCTF/releases) 下载对应平台的二进制：

| 文件 | 说明 |
| --- | --- |
| `cloudctf-linux-amd64` | API 服务（已内嵌前端），x86_64 |
| `cloudctf-linux-arm64` | API 服务（已内嵌前端），ARM64 |
| `*.sha256` | 对应文件的校验和 |

```bash
# 1. 下载最新版本并校验
curl -LO https://github.com/hexbay/CloudCTF/releases/latest/download/cloudctf-linux-amd64
curl -LO https://github.com/hexbay/CloudCTF/releases/latest/download/cloudctf-linux-amd64.sha256
sha256sum -c cloudctf-linux-amd64.sha256

# 2. 赋予执行权限
chmod +x cloudctf-linux-amd64

# 3. 启动（默认 SQLite，监听 :8005）
./cloudctf-linux-amd64
```

启动后访问：

- 选手端：`http://<服务器IP>:8005/`
- 管理后台：`http://<服务器IP>:8005/admin`

### 方式二：Docker

```bash
docker run -d --name cloudctf \
  -p 8005:8005 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v $PWD/data:/app/data \
  -e JWT_SECRET=请改成随机字符串 \
  <your-dockerhub>/ctf-server:latest
```

> 挂载 `/var/run/docker.sock` 是为了让平台能创建动态靶机容器。

### 从源码运行

开发环境需要 Go 1.25、Node.js 20+ 和 npm。涉及动态靶机、镜像构建的功能还需要可访问的 Docker 服务。

```bash
# 构建前端
cd frontend
npm ci
npm run build

# 启动后端；默认监听 8005 端口
cd ../backend
go run ./cmd/server
```

开发前端时可单独运行：

```bash
cd frontend
npm install
npm run dev
```

提交改动前建议执行：

```bash
cd backend && go test ./...
cd ../frontend && npm run build
```

## 配置

所有配置通过环境变量提供（也可放入 `.env`）。常用项如下：

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `HOST` | `0.0.0.0` | 监听地址 |
| `PORT` | `8005` | 监听端口 |
| `APP_ENV` | `development` | 运行环境：`development` / `production` |
| `DATABASE_URI` | `sqlite://data/cloudctf.db` | 数据库连接，支持 SQLite / MySQL |
| `JWT_SECRET` | `change-me` | **务必在生产环境修改为随机值** |
| `FIRST_SUPERUSER_USERNAME` | `admin` | 首次初始化数据库时创建的管理员用户名 |
| `FIRST_SUPERUSER_PASSWORD` | 无 | 首次管理员密码；生产环境必填且至少 12 位 |
| `REDIS_ADDR` | `localhost:6379` | Redis 地址（缓存等功能使用） |
| `ALLOWED_ORIGINS` | `*` | CORS 允许的来源，逗号分隔；生产环境不能使用 `*` |
| `STATIC_DIR` | `static` | 静态资源目录 |
| `UPLOAD_DIR` | `static/uploads` | 上传文件目录 |
| `LOG_LEVEL` | `info` | 日志级别：`debug` / `info` / `warn` / `error` |
| `LOG_FILE` | `logs/backend.log` | 日志文件路径 |

### 数据库示例

```bash
# SQLite（默认，零依赖）
DATABASE_URI="sqlite://data/cloudctf.db" ./cloudctf-...

# MySQL（生产推荐）
DATABASE_URI="mysql://user:pass@tcp(127.0.0.1:3306)/cloudctf?charset=utf8mb4&parseTime=True&loc=Local" ./cloudctf-...
```

## 确认运行

```bash
curl http://127.0.0.1:8005/healthz
# {"status":"ok","time":"..."}
```

## 生产部署建议

- 设置强随机 `JWT_SECRET`,并将 `APP_ENV=production`
- 设置 `FIRST_SUPERUSER_PASSWORD`；生产模式不会创建演示选手账号
- 显式设置 `ALLOWED_ORIGINS` 为实际访问源，例如 `https://ctf.example.com` 或 `http://服务器IP:8005`；生产环境使用默认 `*` 会导致服务拒绝启动
- 使用 MySQL 作为业务数据库，定期备份
- 可选通过反向代理（Nginx / Caddy）启用 80/443 与 HTTPS；反代时建议 `HOST=127.0.0.1`，并将 `ALLOWED_ORIGINS` 收紧到外部访问域名
- 动态靶机所在主机需可访问 Docker（`/var/run/docker.sock` 或远程 Docker API）
- 持久化 `data/`、`static/uploads/` 等目录

更多细节（架构、虚拟化、运维）见 **https://hexbay.github.io/CloudCTF/** 。

## 文档

完整文档：**https://hexbay.github.io/CloudCTF/**

## 项目结构

```text
.
├── backend/          # Go API、业务服务、数据库模型与嵌入式前端
├── frontend/         # React + TypeScript，包含选手端和管理后台
├── docs/             # VitePress 使用与部署文档
├── tests/integration # HTTP 端到端测试
├── deploy/           # 发布和部署相关文件
└── .github/workflows # 构建、测试、文档与 Release 工作流
```

架构说明参见 [docs/guide/architecture.md](docs/guide/architecture.md)，管理员操作参见 [docs/guide/admin-guide.md](docs/guide/admin-guide.md)。

## 参与贡献

欢迎提交 Issue 和 Pull Request。开始开发前请先阅读 [CONTRIBUTING.md](CONTRIBUTING.md)：

1. Fork 仓库并从 `main` 创建功能分支。
2. 一个提交只处理一个明确问题，并为行为变更补充测试。
3. 提交前运行后端测试和前端构建。
4. Pull Request 中说明问题、实现方案、验证方式和兼容性影响。

## 安全

CloudCTF 会运行用户可访问的动态靶机，并可能连接 Docker 或虚拟化宿主机。生产环境应将靶机运行节点与平台控制面隔离，不要在包含敏感业务的宿主机上直接挂载 Docker Socket。

如果发现安全漏洞，请不要创建公开 Issue。请按照 [SECURITY.md](SECURITY.md) 中的方式私下报告。

## 致谢

感谢所有提交代码、报告问题、维护题库和完善文档的贡献者，以及 Docker、React、Ant Design、GORM 等开源项目。

## 许可证

CloudCTF 使用 [MIT License](LICENSE) 发布。你可以自由使用、修改和分发，但需保留版权与许可证声明。
