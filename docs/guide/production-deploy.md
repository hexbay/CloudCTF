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
JWT_SECRET=replace-with-a-long-random-string
LOG_LEVEL=info
LOG_FILE=./cloudctf.log

# 生产环境必须配置为实际访问源，不能使用 *
# 有域名时推荐填写 https://ctf.example.com
ALLOWED_ORIGINS=http://your-server-ip:8005

# 数据库：默认 SQLite
DATABASE_URI=sqlite://./cloudctf.db

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

如果服务启动失败并看到：

```text
invalid configuration: ALLOWED_ORIGINS must not contain * in production
```

说明当前 `APP_ENV=production`，但 `ALLOWED_ORIGINS` 未设置或仍为 `*`。请在 `cloudctf.env` 中把它改为实际访问源，例如：

```bash
ALLOWED_ORIGINS=https://ctf.example.com
```

或没有域名时临时使用：

```bash
ALLOWED_ORIGINS=http://服务器IP:8005
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
如果是从 SQLite 迁移到 MySQL，建议先做好数据导出 / 导入与备份验证。

### 页面可以打开，但动态靶机无法启动？

请确认：

1. Docker 已安装并正在运行
2. CloudCTF 所在宿主机有权限访问 Docker
3. 题目模板、主机节点、出口地址已正确配置
