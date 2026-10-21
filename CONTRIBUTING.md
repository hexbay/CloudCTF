# 参与贡献

感谢你为 CloudCTF 做出贡献。欢迎修复缺陷、改进文档、补充测试和提出经过验证的功能建议。

## 开发环境

- Go 1.25+
- Node.js 20+
- npm
- Docker（仅动态靶机、镜像构建和完整集成测试需要）

```bash
git clone <your-fork-url>
cd CloudCTF

cd frontend
npm ci
npm run build

cd ../backend
go test ./...
go run ./cmd/server
```

## 提交改动

1. 从 `main` 创建分支。
2. 一个提交只处理一个明确问题，避免混入格式化或无关重构。
3. 行为变更应补充测试；界面变更请提供截图。
4. 不要提交 `.env`、数据库、日志、上传文件、访问令牌或真实 Flag。
5. 提交前至少运行：

```bash
cd backend && go test ./...
cd ../frontend && npm run build
```

建议使用清晰的提交前缀，例如 `feat:`、`fix:`、`docs:`、`test:`、`refactor:` 和 `security:`。

## Pull Request

PR 描述应包含：

- 问题背景和关联 Issue
- 实现方案及重要取舍
- 验证命令和结果
- 数据库、配置或 API 的兼容性影响
- 界面改动前后截图（如适用）

维护者可能要求拆分过大的 PR。涉及认证、权限、Flag、文件上传、备份恢复、容器或虚拟化的改动需要额外安全审查。

## 安全问题

请勿通过公开 Issue 披露安全漏洞，改用 [SECURITY.md](SECURITY.md) 中的私下报告方式。
