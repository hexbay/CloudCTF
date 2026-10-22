# Docker 部署

CloudCTF 官方镜像发布在 Docker Hub：

- [`peach0214/cloud_ctf`](https://hub.docker.com/r/peach0214/cloud_ctf)
- `latest` 指向最近一次构建
- 版本 Tag 对应 GitHub Release，例如 GitHub Tag `v1.0.0` 对应镜像 Tag `1.0.0`

## 拉取镜像

```bash
docker pull peach0214/cloud_ctf:latest
```

生产环境建议固定版本，确认新版本后再主动升级：

```bash
docker pull peach0214/cloud_ctf:1.0.0
```

## 启动服务

```bash
docker run -d \
  --name cloudctf \
  --restart unless-stopped \
  -p 8005:8005 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v "$PWD/data:/app/data" \
  -v "$PWD/static:/app/static" \
  peach0214/cloud_ctf:latest
```

官方镜像默认使用 `APP_ENV=production`，不需要额外传入。首次启动发现 `data/.env` 不存在时，CloudCTF 会复制镜像内的 `.env.example` 创建配置文件，写入生产环境模式，并将 JWT 密钥和初始管理员密码替换为安全随机值（文件权限为仅当前用户可读写）。管理员用户名默认为 `admin`。第一次登录前查看生成的密码：

```bash
grep '^FIRST_SUPERUSER_PASSWORD=' data/.env
```

请妥善保管该文件，不要提交到 Git。后续重建容器时继续挂载同一个 `data/` 目录，密钥和密码就会保持不变。若数据库中已经存在管理员账号，生成的初始密码不会重置该账号密码。

直接通过该服务自身地址访问时无需配置 `ALLOWED_ORIGINS`。只有当前端部署在另一个域名/端口、需要跨域调用 API 时，才添加 `-e ALLOWED_ORIGINS='https://your-frontend.example.com'`。

## 配置自定义数据库

默认使用 SQLite，数据库文件保存在 `data/cloudctf.db`。要改用 MySQL 或 PostgreSQL，先创建数据库和专用账号，再在 `docker run` 命令中添加 `DATABASE_URI`。数据库需要提前存在；CloudCTF 首次连接后会自动创建和迁移数据表。

MySQL：

```bash
-e DATABASE_URI='mysql://cloudctf:strong-password@tcp(mysql:3306)/cloudctf?charset=utf8mb4&parseTime=True&loc=Local'
```

PostgreSQL：

```bash
-e DATABASE_URI='postgres://cloudctf:strong-password@postgres:5432/cloudctf?sslmode=require'
```

将对应选项加到上面的完整 `docker run` 命令中即可。将 `mysql` / `postgres` 替换成容器可访问的数据库主机名或 IP；若数据库也运行在 Docker 中，需要让两个容器加入同一个 Docker 网络。不要在应用容器里使用 `localhost` 连接另一个容器。数据库账号需要创建/修改表、创建索引以及读写数据的权限。远程 PostgreSQL 请按服务端要求配置 TLS。切换数据库不会自动搬迁原数据库中的数据，迁移前请先备份并单独完成数据导入验证。

## 配置 Redis

Redis 使用一个 URI 同时配置地址、密码和逻辑数据库编号：

```bash
-e REDIS_ADDR='redis://:strong-password@redis:6379/0'
```

格式为 `redis://[:密码]@主机:端口/库编号`。无密码时可使用 `redis://redis:6379/0`。如果 Redis 运行在另一个容器中，请让两个容器加入同一个 Docker 网络，并将 `redis` 替换成 Redis 容器名或网络别名；不要填写 `localhost`。

启动后访问：

- 选手端：`http://<服务器 IP>:8005/`
- 管理后台：`http://<服务器 IP>:8005/admin`
- 健康检查：`http://<服务器 IP>:8005/healthz`

## 在线更新与镜像升级

容器部署支持在管理后台手动确认在线更新。程序会在更新前自动创建数据备份，校验 GitHub Release 的 SHA256 后，将新版本保存到持久化目录 `data/updates/`。因此，只要重建容器时继续挂载同一个 `data/` 目录，即使使用原镜像，新程序版本也不会丢失。

新版本第一次启动成功后才会被标记为可用；如果它在完成启动前退出，`--restart unless-stopped` 会让容器再次启动并自动回退到更新前保留的版本。镜像内置二进制始终作为最终兜底。管理后台只会提示版本，必须由超级管理员手动确认才会安装。

在线更新只替换 CloudCTF 单二进制。需要升级 Alpine 基础环境或镜像系统依赖时，仍应拉取新镜像并重建容器：

```bash
docker pull peach0214/cloud_ctf:latest
docker stop cloudctf
docker rm cloudctf
# 使用上一节的 docker run 命令重新创建容器
```

`data/` 和 `static/` 已挂载到宿主机时，重建容器不会删除数据库、配置、在线更新版本和静态文件。建议使用固定的宿主机绝对路径，避免在不同工作目录执行命令时误挂载新的空目录。

## CI 镜像发布配置

仓库的 `Build and Release` 工作流会将镜像发布到 `peach0214/cloud_ctf`。Docker Hub 用户名和镜像名已经写入工作流，只需在 GitHub 仓库中添加一个 Secret：

| 类型 | 名称 | 值 |
| --- | --- | --- |
| Repository secret | `DOCKERHUB_TOKEN` | Docker Hub 中具有 Read & Write 权限的 Personal Access Token |

配置位置：`Settings → Secrets and variables → Actions → Secrets → New repository secret`。

也可以使用 GitHub CLI 设置，命令会提示你粘贴 Token：

```bash
gh secret set DOCKERHUB_TOKEN --repo hexbay/CloudCTF
```

配置完成并推送工作流改动后，可在 GitHub Actions 中手动运行 `Build and Release`，填写源码 `source_ref` 与镜像 `image_tag`；推送 `v*` Tag 时会自动发布去掉 `v` 前缀的版本 Tag 和 `latest`。

## 安全说明

挂载 `/var/run/docker.sock` 后，CloudCTF 可以管理宿主机上的 Docker 资源，这通常等同于较高的宿主机权限。生产环境应隔离靶场节点，并限制容器挂载、出口网络和镜像来源。如果不使用动态靶机，可以不挂载 Docker Socket。

单二进制与 systemd 部署方式参见 [生产部署](/guide/production-deploy)。
