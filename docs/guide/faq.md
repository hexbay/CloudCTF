# 常见问题

## 部署相关

### Q: 当前推荐的生产部署方式是什么？

A: 当前主推方案是：

- `systemd + 单二进制`
- 前端已内嵌进服务二进制
- 数据库按环境选择 SQLite、MySQL 或 PostgreSQL

这也是当前文档默认描述的生产部署口径。

### Q: 支持哪些操作系统？

A: 生产环境推荐 Linux，并使用 `systemd` 托管服务：

- ✅ Ubuntu 22.04+
- ✅ Debian 12+
- ✅ CentOS Stream 9+

理论上开发环境也可在 macOS / Windows 运行，但生产环境不作为主推方案。

### Q: 为什么不再推荐 Docker Compose 一键部署？

A: 当前交付方式已经改为 **单二进制 + 内嵌前端**，部署链路更短：

- 不需要单独部署前端
- 不强制绑定容器编排
- 更适合直接接入现有 Linux 生产主机和 `systemd`

Docker 仍可用于题目运行时和动态靶机场景，但不再作为平台主服务的默认生产部署方式。

### Q: 前端现在怎么部署？

A: 不需要单独部署。  
前端静态资源已经打包进服务二进制，访问：

- `/`
- `/admin`

都会由同一个后端服务直接返回。

### Q: 数据库必须用 MySQL 吗？

A: 不是。

当前支持通过 `DATABASE_URI` 配置三种数据库：

- `SQLite`：单机、轻量、开箱即用
- `MySQL`：更适合正式生产环境
- `PostgreSQL`：适合已有 PostgreSQL 基础设施的正式生产环境

### Q: 如何更新到新版本？

A:

```bash
# 1. 停服务
sudo systemctl stop cloudctf

# 2. 替换二进制
cp new-cloudctf /opt/cloudctf/bin/cloudctf
chmod +x /opt/cloudctf/bin/cloudctf

# 3. 启动服务
sudo systemctl start cloudctf
```

更新前建议：

1. 备份数据库
2. 备份上传目录与备份目录
3. 记录当前配置文件

## 功能相关

### Q: 支持哪些题目类型？

A: 支持：

- Web
- Pwn
- Reverse
- Crypto
- Misc
- 理论题 / 考试
- 动态靶机题
- 漏洞复现环境

### Q: 动态环境为什么还依赖 Docker？

A: 平台主服务已经不依赖 Docker 部署，但 **动态靶机 / 漏洞环境运行时** 仍依赖 Docker 或对应虚拟化资源。这是“平台部署方式”和“题目运行环境”两个不同层面。

### Q: 如何导入 Vulhub / CTFDB 模板？

A: 通过管理后台的资源仓库与模板同步能力完成。  
建议部署完成后先配置好虚拟化主机、出口地址、资源仓库，再执行同步。

## 故障排查

### Q: 服务启动失败怎么办？

A:

1. 查看 `systemd` 状态：

```bash
systemctl status cloudctf
```

2. 查看实时日志：

```bash
journalctl -u cloudctf -f
```

3. 检查配置文件：

```bash
cat /opt/cloudctf/config/cloudctf.env
```

4. 检查端口占用：

```bash
ss -lntp | grep 8005
```

如果日志中出现：

```text
invalid configuration: ALLOWED_ORIGINS must not contain * in production
```

说明当前配置了 `APP_ENV=production`，但 `ALLOWED_ORIGINS` 为空、未配置或仍为默认 `*`。生产环境必须改成实际访问源，例如：

```bash
ALLOWED_ORIGINS=https://ctf.example.com
```

没有域名时可先使用：

```bash
ALLOWED_ORIGINS=http://服务器IP:8005
```

### Q: 页面能打开，但后台静态资源异常？

A: 先确认当前运行的二进制是否为最新版本。因为前端资源已经内嵌进二进制，页面资源是否更新取决于你实际启动的是不是新产物，而不是单独的前端目录。

### Q: API 返回 500 怎么办？

A:

1. 查看 `journalctl -u cloudctf -f`
2. 检查 `DATABASE_URI`
3. 检查上传目录、备份目录是否有写权限
4. 如涉及动态靶机，再检查 Docker 与虚拟化配置

### Q: 动态靶机启动失败怎么办？

A:

1. 检查 Docker 是否正常运行
2. 检查主机节点配置是否正确
3. 检查模板端口、镜像、Dockerfile、Compose 文件是否完整
4. 检查出口地址是否可达
5. 查看容器日志与平台日志

## 其他

### Q: 如何获取技术支持？

A:

- 文档：站内文档
- GitHub Issues：提交问题
- Email：`5829662@qq.com`

### Q: 是否支持定制部署？

A: 支持，但当前公开文档默认只维护主线生产方案，也就是 `systemd + 单二进制`。如果你们内部有额外的发布链路，建议在此基础上做局部扩展，而不是继续维护完全独立的旧部署体系。
