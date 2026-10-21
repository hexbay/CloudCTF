# Docker 部署

CloudCTF 官方镜像发布在 Docker Hub：

- [`peach0214/cloud_ctf`](https://hub.docker.com/r/peach0214/cloud_ctf)
- `latest` 指向最近一次构建
- 版本 Tag 与 GitHub Release 一致，例如 `v1.0.0`

## 拉取镜像

```bash
docker pull peach0214/cloud_ctf:latest
```

生产环境建议固定版本，确认新版本后再主动升级：

```bash
docker pull peach0214/cloud_ctf:v1.0.0
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
  -e APP_ENV=production \
  -e JWT_SECRET='replace-with-a-long-random-string' \
  -e FIRST_SUPERUSER_PASSWORD='replace-with-a-strong-12+-character-password' \
  -e ALLOWED_ORIGINS='http://your-server-ip:8005' \
  peach0214/cloud_ctf:latest
```

启动后访问：

- 选手端：`http://<服务器 IP>:8005/`
- 管理后台：`http://<服务器 IP>:8005/admin`
- 健康检查：`http://<服务器 IP>:8005/healthz`

## 升级镜像

容器内的在线更新不适合镜像部署。请拉取新镜像并重建容器：

```bash
docker pull peach0214/cloud_ctf:latest
docker stop cloudctf
docker rm cloudctf
# 使用上一节的 docker run 命令重新创建容器
```

`data/` 和 `static/` 已挂载到宿主机时，重建容器不会删除其中的数据。升级前仍建议在管理后台创建备份。

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

配置完成并推送工作流改动后，可在 GitHub Actions 中手动运行 `Build and Release` 发布 `latest`；推送 `v*` Tag 时会同时发布对应版本 Tag 和 `latest`。

## 安全说明

挂载 `/var/run/docker.sock` 后，CloudCTF 可以管理宿主机上的 Docker 资源，这通常等同于较高的宿主机权限。生产环境应隔离靶场节点，并限制容器挂载、出口网络和镜像来源。如果不使用动态靶机，可以不挂载 Docker Socket。

单二进制与 systemd 部署方式参见 [生产部署](/guide/production-deploy)。
