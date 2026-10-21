# 快速开始

这页只保留 **最短可运行路径**：下载、执行、访问。  
如果你要看 `systemd`、配置文件、数据库切换等生产部署细节，请跳转到 [生产部署](/guide/production-deploy)。

如果你想先体验已部署好的环境，可以访问在线预览：

- [http://cloud-ctf.lostpeach.cn/](http://cloud-ctf.lostpeach.cn/)

## 前置要求

- Linux x86_64 或 ARM64
- 可执行文件下载权限

> 快速开始默认**不要求任何额外配置**。  
> 未显式配置时，服务会使用内置默认值启动，并默认使用 SQLite。

## 1. 下载二进制

```bash
curl -LO https://github.com/hexbay/CloudCTF/releases/latest/download/cloudctf-linux-amd64
curl -LO https://github.com/hexbay/CloudCTF/releases/latest/download/cloudctf-linux-amd64.sha256
sha256sum -c cloudctf-linux-amd64.sha256
chmod +x cloudctf-linux-amd64
```

如果你是 ARM64 机器，请将文件名替换为：

```bash
cloudctf-linux-arm64
```

## 2. 直接启动

```bash
./cloudctf-linux-amd64
```

默认情况下：

- 监听地址：`0.0.0.0:8005`
- 数据库：`sqlite://data/cloudctf.db`
- 选手端：`/`
- 管理后台：`/admin`

## 3. 访问平台

启动后访问：

- 选手端：`http://<server>:8005/`
- 管理后台：`http://<server>:8005/admin`
- 健康检查：`http://<server>:8005/healthz`

## 默认账号

首次启动后，系统会自动创建默认账号：

| 角色 | 用户名 | 密码 |
|------|--------|------|
| 管理员 | `admin` | `admin123456` |
| 选手 | `player1` | `password123` |

> 快速体验可以直接使用默认账号。  
> 正式环境请在首次登录后立即修改密码。

## 常见操作

停止服务：

```bash
Ctrl+C
```

重新启动：

```bash
./cloudctf-linux-amd64
```

## 遇到问题？

如果你只是想确认服务能不能跑起来，先检查：

1. 文件是否有执行权限
2. `8005` 端口是否被占用
3. 当前目录是否可写（SQLite 默认会在本地创建数据库文件）

## 下一步

- [生产部署](/guide/production-deploy)
- [技术架构](/guide/architecture)
- [管理员指南](/guide/admin-guide)
