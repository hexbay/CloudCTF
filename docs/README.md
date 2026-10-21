# CloudCTF 文档

CloudCTF 平台的完整文档，使用 VitePress 构建。

## 本地开发

### 安装依赖

```bash
pnpm install
```

### 启动开发服务器

```bash
pnpm docs:dev
```

启动后访问本地开发服务器（默认端口 5173）

### 构建文档

```bash
pnpm docs:build
```

构建产物在 `.vitepress/dist` 目录。

### 预览构建结果

```bash
pnpm docs:preview
```

## 文档结构

```
docs/
├── .vitepress/
│   └── config.ts          # VitePress 配置
├── guide/                 # 使用指南
│   ├── introduction.md    # 项目介绍
│   ├── features.md        # 功能特性
│   ├── quick-start.md     # 快速开始
│   └── faq.md            # 常见问题
├── deployment/            # 部署说明
│   └── docker.md         # Docker 镜像部署
├── api/                   # API 文档
│   └── overview.md       # API 概览
└── index.md              # 首页
```

## 贡献指南

欢迎贡献文档！请遵循以下规范：

1. 使用 Markdown 格式
2. 保持文档简洁清晰
3. 添加代码示例
4. 提交 PR 前先本地预览

## 部署

### GitHub Pages

文档会自动部署到 GitHub Pages。

### 自定义服务器

```bash
# 构建
npm run docs:build

# 将 .vitepress/dist 目录部署到服务器
scp -r .vitepress/dist/* user@server:/var/www/docs/
```

## License

MIT
