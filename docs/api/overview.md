# API 概览

CloudCTF 提供完整的 RESTful API，支持所有平台功能的编程访问。

## API 基础信息

### Base URL

```
http://your-domain/api
```

### 认证方式

API 使用 JWT (JSON Web Token) 进行认证。

**获取 Token**：

```bash
POST /api/auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "password"
}
```

**响应**：

```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "bearer"
}
```

**使用 Token**：

```bash
GET /api/users/me
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

## 响应格式

### 成功响应

```json
{
  "data": {
    "id": 1,
    "name": "示例数据"
  },
  "total": 1
}
```

### 错误响应

```json
{
  "detail": "错误信息"
}
```

### HTTP 状态码

- `200 OK` - 请求成功
- `201 Created` - 创建成功
- `204 No Content` - 删除成功
- `400 Bad Request` - 请求参数错误
- `401 Unauthorized` - 未认证
- `403 Forbidden` - 无权限
- `404 Not Found` - 资源不存在
- `422 Unprocessable Entity` - 验证失败
- `500 Internal Server Error` - 服务器错误

## API 分类

### 认证 API

- `POST /api/auth/login` - 用户登录
- `POST /api/auth/register` - 用户注册
- `POST /api/auth/logout` - 用户登出
- `POST /api/auth/refresh` - 刷新 Token

### 用户 API

- `GET /api/users/me` - 获取当前用户信息
- `PUT /api/users/me` - 更新当前用户信息
- `GET /api/users/{id}` - 获取用户信息
- `GET /api/users` - 获取用户列表（管理员）

### 题目 API

- `GET /api/challenges` - 获取题目列表
- `GET /api/challenges/{id}` - 获取题目详情
- `POST /api/challenges` - 创建题目（管理员）
- `PUT /api/challenges/{id}` - 更新题目（管理员）
- `DELETE /api/challenges/{id}` - 删除题目（管理员）

### 提交 API

- `POST /api/submissions` - 提交 Flag
- `GET /api/submissions` - 获取提交记录
- `GET /api/submissions/{id}` - 获取提交详情

### 比赛 API

- `GET /api/competitions` - 获取比赛列表
- `GET /api/competitions/{id}` - 获取比赛详情
- `POST /api/competitions` - 创建比赛（管理员）
- `PUT /api/competitions/{id}` - 更新比赛（管理员）

### 排行榜 API

- `GET /api/scoreboard` - 获取排行榜
- `GET /api/scoreboard/{competition_id}` - 获取指定比赛排行榜

## 分页

列表接口支持分页：

```bash
GET /api/challenges?page=1&page_size=20
```

**参数**：
- `page` - 页码（从 1 开始）
- `page_size` - 每页数量（默认 20，最大 100）

**响应**：

```json
{
  "data": [...],
  "total": 100
}
```

## 过滤和排序

### 过滤

```bash
GET /api/challenges?category=web&difficulty=easy
```

### 排序

```bash
GET /api/challenges?sort=created_at&order=desc
```

**参数**：
- `sort` - 排序字段
- `order` - 排序方向（`asc` 或 `desc`）

## 速率限制

为防止滥用，API 实施了速率限制：

- **未认证用户**: 100 请求/小时
- **已认证用户**: 1000 请求/小时
- **管理员**: 无限制

超过限制时返回 `429 Too Many Requests`。

响应头包含限制信息：

```
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1640995200
```

## WebSocket API

实时功能使用 WebSocket：

```javascript
const ws = new WebSocket('ws://your-domain/api/ws');

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log(data);
};
```

### 事件类型

- `scoreboard_update` - 排行榜更新
- `challenge_update` - 题目更新
- `notification` - 系统通知

## API 文档

完整的交互式 API 文档：

- **Swagger UI**: http://your-domain/api/docs
- **ReDoc**: http://your-domain/api/redoc
- **OpenAPI JSON**: http://your-domain/api/openapi.json

## SDK 和示例

### Python 示例

```python
import requests

# 登录
response = requests.post('http://your-domain/api/auth/login', json={
    'email': 'user@example.com',
    'password': 'password'
})
token = response.json()['access_token']

# 获取题目列表
headers = {'Authorization': f'Bearer {token}'}
response = requests.get('http://your-domain/api/challenges', headers=headers)
challenges = response.json()['data']

# 提交 Flag
response = requests.post('http://your-domain/api/submissions',
    headers=headers,
    json={
        'challenge_id': 1,
        'flag': 'flag{test}'
    }
)
```

### JavaScript 示例

```javascript
// 登录
const response = await fetch('http://your-domain/api/auth/login', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    email: 'user@example.com',
    password: 'password'
  })
});
const { access_token } = await response.json();

// 获取题目列表
const challenges = await fetch('http://your-domain/api/challenges', {
  headers: { 'Authorization': `Bearer ${access_token}` }
}).then(r => r.json());

// 提交 Flag
await fetch('http://your-domain/api/submissions', {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${access_token}`,
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    challenge_id: 1,
    flag: 'flag{test}'
  })
});
```

## 下一步

- 认证接口 - 登录、注册、Token 刷新
- 用户接口 - 用户管理相关接口
- 题目接口 - 题目管理相关接口
