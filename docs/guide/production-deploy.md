# 生产部署

本页用于说明 **systemd + 单二进制** 的正式部署方式。  
如果你只想先跑起来，请先看 [快速开始](/guide/quick-start)。

## 部署目标

生产部署完成后，你将得到：

- 常驻运行的 `cloudctf` 服务
- 开机自启
- 可控的配置文件路径
- 可切换的数据库配置

## 前置要求

- Linux，推荐 Ubuntu 22.04+ / Debian 12+ / CentOS Stream 9+
- `systemd`
- 至少 4GB 内存，生产建议 8GB+
- 至少 20GB 磁盘
- 如需动态靶机 / 漏洞环境，请确保宿主机可访问 Docker

## 1. 下载发布文件

```bash
curl -LO https://github.com/hexbay/CloudCTF/releases/latest/download/cloudctf-linux-amd64
curl -LO https://github.com/hexbay/CloudCTF/releases/latest/download/cloudctf-linux-amd64.sha256
sha256sum -c cloudctf-linux-amd64.sha256
chmod +x cloudctf-linux-amd64
```

## 2. 准备配置文件

在你的部署目录中创建配置文件，例如 `cloudctf.env`：

```bash
cat > cloudctf.env <<'EOF'
APP_ENV=production
HOST=0.0.0.0
PORT=8005
LOG_LEVEL=info
LOG_FILE=./cloudctf.log

# 同源访问无需设置；仅跨域部署时填写允许的来源
# ALLOWED_ORIGINS=https://ctf.example.com

# 在线更新：从 v* Tag 对应的 GitHub Release 获取新版本
UPDATE_ENABLED=true
UPDATE_REPOSITORY=hexbay/CloudCTF

# 数据库：默认 SQLite
DATABASE_URI=sqlite://./cloudctf.db

# Redis URI：redis://[:密码]@主机:端口/库编号
REDIS_ADDR=redis://:strong-password@127.0.0.1:6379/0

# 上传与备份
UPLOAD_DIR=./uploads
BACKUP_DIR=./backups
EOF
```

### 数据库切换示例

SQLite：

```bash
DATABASE_URI=sqlite://./cloudctf.db
```

MySQL：

```bash
DATABASE_URI=mysql://cloudctf:strong-password@tcp(127.0.0.1:3306)/cloudctf?charset=utf8mb4&parseTime=True&loc=Local
```

PostgreSQL：

```bash
DATABASE_URI=postgres://cloudctf:strong-password@127.0.0.1:5432/cloudctf?sslmode=disable
```

MySQL 或 PostgreSQL 数据库需要提前创建，连接账号需要建表、修改表、创建索引和读写数据的权限。远程 PostgreSQL 生产环境建议按数据库服务要求启用 TLS，例如使用 `sslmode=require`。

## 3. 创建 systemd 服务

创建主服务 `/etc/systemd/system/cloudctf.service`：

```ini
[Unit]
Description=CloudCTF Server
After=network.target

[Service]
Type=simple
WorkingDirectory=/srv/cloudctf
EnvironmentFile=/srv/cloudctf/cloudctf.env
ExecStart=/srv/cloudctf/cloudctf-linux-amd64
Restart=always
RestartSec=5
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
```

其中 `/srv/cloudctf` 只是示例路径，你可以替换成自己的实际部署目录。

## 4. 启动服务

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now cloudctf
```

首次启动会在工作目录生成 `data/.env`，其中包含随机 JWT 密钥和随机管理员密码。管理员用户名默认为 `admin`，登录前查看密码：

```bash
sudo grep '^FIRST_SUPERUSER_PASSWORD=' /srv/cloudctf/data/.env
```

请限制该文件的访问权限并纳入安全备份；不要提交到 Git。后续重启不会重新生成密码，修改该字段也不会重置数据库中已经存在的管理员账号。

## 5. 验证运行

```bash
systemctl status cloudctf
curl http://127.0.0.1:8005/healthz
```

正常情况下会返回：

```json
{"status":"ok"}
```

浏览器访问：

- `http://<server>:8005/`
- `http://<server>:8005/admin`

同源访问不需要设置 `ALLOWED_ORIGINS`。只有前端与 API 位于不同域名或端口时，才在 `cloudctf.env` 中填写允许的前端来源，例如：

```bash
ALLOWED_ORIGINS=https://ctf.example.com
```

## 6. 可选：通过 Nginx 反向代理

CloudCTF 可以直接暴露 `:8005` 端口访问，也可以放在 Nginx 后面统一使用 80/443 端口和 HTTPS。

使用 Nginx 反代时，建议让 CloudCTF 只监听本机地址：

```bash
HOST=127.0.0.1
PORT=8005
ALLOWED_ORIGINS=https://ctf.example.com
```

示例 Nginx 配置：

```nginx
server {
    listen 80;
    server_name ctf.example.com;

    client_max_body_size 200m;

    location / {
        proxy_pass http://127.0.0.1:8005;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

保存为 `/etc/nginx/sites-available/cloudctf` 后启用：

```bash
sudo ln -s /etc/nginx/sites-available/cloudctf /etc/nginx/sites-enabled/cloudctf
sudo nginx -t
sudo systemctl reload nginx
```

如果使用 HTTPS，推荐通过 Certbot 或 Caddy 等工具签发证书，并把 `ALLOWED_ORIGINS` 改成最终访问地址，例如：

```bash
ALLOWED_ORIGINS=https://ctf.example.com
```

如果暂时只通过 HTTP 域名访问，则填写：

```bash
ALLOWED_ORIGINS=http://ctf.example.com
```

## 日常运维

### 在线更新

超级管理员进入管理端时，系统会检查一次新版本；发现新版本后只显示通知，不会自动安装。管理员可以通过通知进入“系统设置 → 系统信息”，查看发布说明并手动确认更新，也可以随时点击“检查更新”。

在线更新流程如下：

1. 读取 `UPDATE_REPOSITORY` 的最新 GitHub Release（由 `v*` Tag 发布）
2. 按当前架构下载 `cloudctf-linux-amd64` 或 `cloudctf-linux-arm64`
3. 下载并验证同名 `.sha256` 文件
4. 自动创建数据备份
5. 单二进制部署将当前程序保留为 `.previous` 后原子替换；容器部署将版本保存到 `data/updates/`
6. 自动重启；容器中新版本首次启动失败时自动回退到上一版本

只有超级管理员可以执行更新。公共仓库无需 GitHub Token；如果遇到 API 限额，可以在环境文件中设置 `UPDATE_GITHUB_TOKEN`，然后重启服务。

运行服务的用户必须对二进制所在目录具有写权限，否则检查功能仍可使用，但安装会被拒绝。

如需关闭入口：

```bash
UPDATE_ENABLED=false
```

容器化部署会将在线更新持久化到 `data/updates/`，重建容器时继续挂载同一个 `data/` 目录即可保留版本。基础镜像或系统依赖变化时仍需拉取新镜像。完整说明参见 [Docker 部署](/deployment/docker)。

查看状态：

```bash
systemctl status cloudctf
```

查看日志：

```bash
journalctl -u cloudctf -f
```

重启服务：

```bash
sudo systemctl restart cloudctf
```

重置管理员密码：

```bash
cd /srv/cloudctf
sudo systemctl stop cloudctf
./cloudctf-linux-amd64 reset-admin-password --username admin
sudo systemctl start cloudctf
```

密码会以隐藏方式交互输入。命令使用当前环境或 `CONFIG_FILE` 指定配置中的 `DATABASE_URI`，同时支持 SQLite、MySQL 和 PostgreSQL。修改 `FIRST_SUPERUSER_PASSWORD` 只影响首次创建账号，不能重置已有管理员。

## 常见问题

### 必须使用固定目录结构吗？

不需要。  
文档里的路径只是示例，你只要保证：

1. `ExecStart` 指向真实二进制
2. `EnvironmentFile` 指向真实配置文件
3. 服务运行用户对数据库文件、上传目录、备份目录有读写权限

就可以按自己的目录规范部署。

### 想切换数据库怎么办？

直接修改 `DATABASE_URI`，然后重启服务即可。  
如果是在 SQLite、MySQL 和 PostgreSQL 之间切换，修改连接串本身不会迁移已有数据，请先做好数据导出、导入与备份验证。

### 页面可以打开，但动态靶机无法启动？

请确认：

1. Docker 已安装并正在运行
2. CloudCTF 所在宿主机有权限访问 Docker
3. 题目模板、主机节点、出口地址已正确配置
