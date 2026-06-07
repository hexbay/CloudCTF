<div align="center">

# CloudCTF

**CTF 竞赛与漏洞训练靶场平台**

单一可执行文件 · 内置前端 · 开箱即用 · 支持动态靶机

[![Release](https://img.shields.io/github/v/release/hexbay/CloudCTF?style=flat-square)](https://github.com/hexbay/CloudCTF/releases)
[![Docs](https://img.shields.io/badge/docs-在线文档-0f766e?style=flat-square)](http://ctf-docs.lostpeach.cn/)

[在线文档](http://ctf-docs.lostpeach.cn/) · [下载](https://github.com/hexbay/CloudCTF/releases) · [快速开始](#快速开始)

</div>

---

## 简介

CloudCTF 是一个面向 **CTF 竞赛** 与 **漏洞训练靶场** 的一体化平台。**前端（选手端 + 管理后台）已内嵌进单一可执行文件**，无需 Nginx、无需单独部署前端，下载一个二进制即可运行。

- **选手端**：浏览器访问 `/`
- **管理后台**：浏览器访问 `/admin`

> 本仓库以预编译产物（二进制 / 镜像）形式发布，不包含完整源码。
>
> 完整使用与部署文档请见在线文档站：**http://ctf-docs.lostpeach.cn/**

## 功能特性

- **竞赛管理**：比赛、题目、提交、计分、排行榜、公告、审计日志
- **多题型**：Web / Pwn / Crypto / 动态靶机 / 理论题 / 考试
- **动态靶机**：基于 Docker 的容器化靶机，支持单容器与 docker-compose 多容器编排
- **权限体系**：管理员 / 租户 / 选手多角色，资源级权限控制
- **一体化交付**：前端内嵌，单二进制 / 单镜像部署
- **灵活存储**：默认 SQLite 零依赖启动，生产可切换 MySQL

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

## 配置

所有配置通过环境变量提供（也可放入 `.env`）。常用项如下：

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `HOST` | `0.0.0.0` | 监听地址 |
| `PORT` | `8005` | 监听端口 |
| `APP_ENV` | `development` | 运行环境：`development` / `production` |
| `DATABASE_URI` | `sqlite://data/cloudctf.db` | 数据库连接，支持 SQLite / MySQL |
| `JWT_SECRET` | `change-me` | **务必在生产环境修改为随机值** |
| `REDIS_ADDR` | `localhost:6379` | Redis 地址（缓存等功能使用） |
| `ALLOWED_ORIGINS` | `*` | CORS 允许的来源，逗号分隔 |
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
- 使用 MySQL 作为业务数据库，定期备份
- 通过反向代理（Nginx / Caddy）启用 HTTPS,并将 `ALLOWED_ORIGINS` 收紧到实际域名
- 动态靶机所在主机需可访问 Docker（`/var/run/docker.sock` 或远程 Docker API）
- 持久化 `data/`、`static/uploads/` 等目录

更多细节（架构、虚拟化、运维）见 **http://ctf-docs.lostpeach.cn/** 。

## 文档

完整文档：**http://ctf-docs.lostpeach.cn/**

## 说明

本仓库仅用于分发 CloudCTF 的预编译产物（二进制与容器镜像），不开放完整源码。产物可免费下载用于学习、教学与竞赛环境搭建。
