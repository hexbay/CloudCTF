# MCP 接入

CloudCTF 提供基于 Streamable HTTP 的 MCP 服务，可让支持 MCP 的客户端检索漏洞、启动和销毁个人漏洞环境。管理员还可以通过 MCP 提交漏洞环境模板编译任务。

## 获取 API Key

1. 登录用户端，进入 **个人中心**。
2. 在 **MCP API Key** 区域点击 **随机生成 API Key**。
3. 平台为每个用户只维护一个明文 API Key，可随时在个人中心查看和复制。
4. 密钥泄露时点击 **重置 API Key**，旧密钥会立即失效。

API Key 等同于账户凭据，请勿提交到代码仓库、日志或公开聊天中。

## 导入客户端

登录后打开用户首页 `/dashboard`，在 **MCP 导入配置** 卡片中复制配置或下载 `cloudctf-mcp.json`。已生成 API Key 时，导出的配置会自动带入当前密钥。

```json
{
  "mcpServers": {
    "cloudctf": {
      "type": "http",
      "url": "https://your-domain/mcp",
      "headers": {
        "Authorization": "Bearer ctf_mcp_your_key"
      }
    }
  }
}
```

服务地址为 `POST /mcp`，当前协议版本为 `2025-06-18`。

## 工具与权限

| 工具 | 普通用户 | 管理员 | 说明 |
| --- | --- | --- | --- |
| `search_vulnerabilities` | ✓ | ✓ | 按关键词、CVE、严重级别检索已公开且启用的漏洞环境 |
| `start_vulnerability_environment` | ✓ | ✓ | 为当前 API Key 所属用户启动环境 |
| `destroy_vulnerability_environment` | ✓ | ✓ | 仅销毁当前用户自己的环境 |
| `compile_vulnerability_environment` | — | ✓ | 提交漏洞模板编译任务，仅接受 `vul` 类型模板 |

管理员工具不会出现在普通用户的 `tools/list` 中；即使手工构造调用，服务端仍会再次校验管理员身份。环境启动和销毁沿用平台现有的配额、模板状态及所有权校验。

## HTTP API

以下接口使用用户登录令牌认证：

| 方法 | 地址 | 说明 |
| --- | --- | --- |
| `GET` | `/api/user/api-key` | 查询当前用户唯一的明文密钥 |
| `POST` | `/api/user/api-key` | 首次随机生成密钥 |
| `POST` | `/api/user/api-key/reset` | 重置密钥并使旧密钥失效 |
| `GET` | `/api/mcp/config` | 获取适配当前域名的 MCP 导入配置 |

生成、查询和重置响应均通过 `api_key` 返回当前密钥；重置会覆盖原记录。
