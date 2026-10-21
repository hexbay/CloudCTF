# CloudCTF 新旧项目迁移实施文档（AI 执行版）

生成时间：2026-06-02

## 0. 当前迁移进度

### 0.1 编码状态

- 当前仓库核心源码文件已统一按 UTF-8 处理。
- 先前出现的中文乱码，主要来自 PowerShell 按本地代码页读取 UTF-8 文件时的显示问题，不是主源码文件仍在混用多种编码。
- 后续迁移过程中，新增和修改文件统一保持为 UTF-8。

### 0.2 已完成迁移

#### 已提交

1. `3c57776` `feat: wire player flows to real backend data`
   - 玩家端仪表盘、练习题、比赛、排行榜、靶场、队伍页面切到真实后端数据。
   - 新增玩家端队伍列表、加入队伍接口。
   - 练习题排行榜支持全站聚合，不再依赖静态假数据。

2. `924d813` `feat: implement admin settings and stats compat handlers`
   - 管理员修改密码、批量重置用户密码、用户标签更新。
   - 队伍成员历史、用户队伍历史。
   - 比赛参赛者审批、拒绝、批量添加、导入。
   - 设置备份、恢复、上传、下载、删除、缓存清理、系统日志。
   - 通知统计、来源统计、访客统计、审计汇总。

3. `f066ca9` `feat: implement virtualization template and group handlers`
   - 虚拟化模板：列表、详情、创建、更新、删除。
   - Compose 文件读取与保存。
   - 模板构建、构建日志、同步日志。
   - 模板文件上传、列表、读取、删除。
   - 容器组：列表、详情、创建、删除、启动、停止。
   - challenge virtualization：题目容器列表、创建、删除、容器模板读取。

#### 已验证

- `cd backend && go test ./...`
- `cd backend && go build ./cmd/server`
- 玩家端第一轮迁移完成时，已验证：
  - `cd frontend && npm run typecheck`
  - `cd frontend && npm run build`

### 0.3 当前仍未完成

#### 管理端后端占位点

- `PUT /api-admin/virtualization/containers/{id}` 仍是静态成功返回。
- 路由兜底 `notImplemented("admin" | "player" | "public")` 仍保留，虽然多数核心业务接口已显式注册，但最终验收前需要继续收敛。

#### 更大范围待迁移项

- 管理端大量页面虽然已有入口，但仍需继续把剩余业务闭环全部换成真实后端。
- 前端仍残留占位页面，例如：
  - `frontend/src/apps/admin/pages/AdminPlaceholderPage.tsx`
  - `frontend/src/apps/player/pages/PlayerPlaceholderPage.tsx`
- 全仓 `successHandler(...)` 与 `notImplemented(...)` 仍未清零，不满足最终验收标准。

### 0.3.1 当前工作区已完成但尚未提交

- 比赛管理员绑定与解绑已切到真实后端存储。
- Writeup 下载与删除已切到真实文件和记录。
- 提交记录删除已切到真实删除。
- 理论比赛统计已切到真实统计返回。
- 考试提交评分已切到真实记录更新。
- 队伍加入申请、比赛绑定与解绑已切到真实后端存储。
- 题目文件删除、题目提示新增与删除已切到真实后端逻辑。
- 玩家 Writeup 上传与读取已切到真实文件落盘与记录存储。

### 0.4 下一轮建议执行顺序

1. 完成虚拟化容器更新接口的真实实现。
2. 回扫前后端占位页面与未接通的管理页交互。
3. 最后统一回扫：
   - `successHandler`
   - `notImplemented`
   - 占位页面
   - 静态业务假数据

## 1. 迁移目标

旧项目：

- 管理端前端：`front-admin`
- 用户端前端：`front-user`
- 后端：`backend_back`

新项目：

- 统一前端：`frontend`
- Go 后端：`backend`

迁移目标是：新项目必须完整承接旧项目全部接口和全部业务功能。页面交互和视觉样式可以重新设计，但任何功能不能消失、降级、伪造或只做静态展示。

## 2. 强制原则

1. 禁止写 `TODO`、`FIXME`、`placeholder`、`mock`、`fake`、`demo data`、`notImplemented` 作为未完成功能。
2. 禁止使用假数据、硬编码列表、静态成功返回、空数组冒充查询结果。
3. 禁止只做前端页面，不接真实接口。
4. 禁止只保留接口路径，不实现旧后端等价业务逻辑。
5. 禁止删除旧功能。旧功能如果在新 UI 中改名或换入口，必须在迁移对账表中标明新入口。
6. 接口响应结构必须兼容旧前端服务层期望，除非同步修改新前端调用方并补充兼容说明。
7. 文件上传、导入、导出、下载、SSE、容器启动/停止/续期等非普通 JSON 能力必须真实可用。
8. 所有权限控制必须迁移：管理员、超级管理员、普通选手、游客公开访问、比赛内权限、队伍权限。
9. 所有数据必须来自数据库、文件系统、Docker/PVE 等真实后端来源。
10. 迁移完成后，新项目中不得存在兜底返回“成功”的空实现。

## 3. 技术迁移边界

旧前端是 Umi / Ant Design Pro 双项目：

- `front-admin` 管理端，端口脚本为 `4001`。
- `front-user` 用户端。

新前端是 Vite / React / React Router：

- `frontend`
- 已有分层：`src/apps/admin`、`src/apps/player`、`src/shared/api`、`src/shared/auth`。

旧后端是 FastAPI：

- 根入口：`backend_back/main.py`
- 管理端前缀：`/api-admin`
- 用户端前缀：`/api`
- 公共前缀：`/api/public`
- 静态资源：`/static/uploads`、`/static`、`/templates`

新后端是 Go：

- 服务入口：`backend/cmd/server`
- HTTP 层：`backend/internal/httpserver`
- 当前已有大量兼容路由，但存在 `successHandler`、`notImplemented`、简单列表处理器等风险点，必须逐个替换为真实业务实现。

## 4. 功能域迁移清单

### 4.1 管理端必须覆盖

1. 登录、刷新令牌、退出、当前管理员、修改密码、首次密码状态。
2. 仪表盘：总体统计、近期比赛、热门题目、活跃用户、提交趋势、用户增长。
3. 普通用户管理：列表、搜索、创建、详情、编辑、删除、批量删除、封禁、解封、批量封禁、批量解封、重置密码、批量重置密码、标签管理、导入、导出模板。
4. 管理员管理：管理员列表、创建、详情、编辑、删除、重置密码。
5. 角色和权限：角色列表、创建、详情、编辑、删除、角色权限、全部权限。
6. 队伍管理：列表、创建、详情、编辑、删除、成员添加/移除、申请加入、比赛绑定/解绑、成员历史、用户队伍历史、黑名单、导入、导出、模板下载。
7. 比赛管理：列表、创建、详情、编辑、删除、状态切换、管理员绑定、参赛者列表、审批、拒绝、添加、批量添加、导入、移除。
8. 比赛公告：列表、创建、编辑、删除、发布、取消发布、置顶、取消置顶。
9. 题目管理：题目列表、创建、详情、编辑、删除、提交测试、文件上传/删除、提示新增/删除、导入、导出、比赛绑定/批量绑定/解绑。
10. 动态 Flag：生成、刷新、管理员查看、校验。
11. 容器和虚拟化：宿主管理、连通性检查、镜像列表、拉取、拉取流、构建、导入、容器列表、创建、详情、更新、删除、动作、日志、状态、健康检查、批量动作。
12. 模板管理：模板列表、选项、创建、详情、编辑、删除、Compose 文件读取/保存、构建、构建日志、构建历史、模板文件上传/列表/读取/删除、同步、同步日志。
13. 容器组：列表、详情、创建、删除、启动、停止。
14. 漏洞靶场：列表、详情、创建、编辑、删除、实例管理。
15. 理论题：题库列表、创建、详情、编辑、删除、导入、导出、测试提交、题目统计、比赛题目列表、绑定、批量绑定、解绑。
16. 理论考试：提交列表、提交详情、批改/更新、比赛统计、导出、比赛试卷读取、快速创建、绑定创建、更新、删除。
17. CTF 提交：比赛提交列表、排行榜、删除提交、导出分数、导出提交。
18. Writeup：比赛 Writeup 列表、详情、下载、删除。
19. 通知消息：发送、批量发送、列表、未读数、详情、单条已读、全部已读、删除、发送日志、统计。
20. 系统公告：列表、详情、发布、取消发布、置顶、取消置顶、删除。
21. 审计：日志列表、导出、敏感操作列表、创建、审批、拒绝。
22. 系统设置：公开设置读取、后台设置读取/更新、系统信息、备份、备份列表、恢复、下载、上传、删除、清缓存、系统日志。
23. 资源仓库：列表、创建、编辑、删除、同步。
24. 资源权限：创建、列表、编辑、删除、按用户查资源、按资源查用户。
25. 访客统计：今日、历史、汇总、指定日期、同步、图表。
26. 上传：通用上传接口、用户头像、题目附件、模板文件、备份上传。

### 4.2 用户端必须覆盖

1. 注册、登录、刷新令牌、退出、当前用户。
2. 用户资料：资料读取/更新、头像上传、统计、活动、徽章、成就、修改密码。
3. 首页/仪表盘：平台概览、比赛入口、练习入口、用户状态。
4. 比赛：公开列表、详情、报名、取消报名、公告、规则、比赛内布局。
5. 比赛赛场：概览、题目、排行榜、公告、规则、问答/理论题、提交考试、Writeup 提交。
6. 题目练习：题目列表、详情、Flag 提交、启动容器、续期、停止、购买提示。
7. 比赛题目：比赛内题目列表、详情、Flag 提交、容器生命周期、提示购买。
8. 排行榜：练习榜、题目榜、我的排名、团队榜、用户榜、排名趋势、比赛统计、题目统计、一血榜。
9. 队伍：我的队伍、创建队伍、加入队伍、退出队伍、队伍详情。
10. 理论考试：读取试卷、提交答案、考试排行榜。
11. 漏洞靶场：列表、详情、启动、续期、停止。
12. 通知：列表、未读数、详情、单条已读、全部已读、删除。
13. 系统公告：列表、置顶公告、详情。
14. SSE：实时通知连接、在线连接状态。
15. Writeup：读取、保存、下载。
16. 公共能力：公开系统设置、来源追踪。

## 5. 页面迁移对账表

### 5.1 管理端页面

| 旧页面 | 旧路径 | 新项目建议入口 | 必须保留能力 |
| --- | --- | --- | --- |
| 登录 | `/user/login` | `/admin/login` | 管理员登录、Token 保存、错误提示 |
| 仪表盘 | `/dashboard` | `/admin/dashboard` | 全部统计卡片和趋势图来自真实接口 |
| 用户列表 | `/users/list` | `/admin/users` | CRUD、搜索、批量、封禁、标签 |
| 队伍管理 | `/users/teams` | `/admin/teams` | 队伍 CRUD、成员、导入导出、黑名单 |
| 队伍详情 | `/users/teams/detail/:id` | `/admin/teams/:id` | 队伍成员、历史、比赛绑定 |
| 比赛列表 | `/competition/list` | `/admin/competitions` | 列表、筛选、创建、编辑、删除、状态 |
| 比赛创建 | `/competition/create` | `/admin/competitions/new` | 完整比赛表单 |
| 比赛详情 | `/competition/detail/:id` | `/admin/competitions/:id` | 参赛者、公告、题目、理论题、榜单、提交、Writeup |
| 比赛编辑 | `/competition/edit/:id` | `/admin/competitions/:id/edit` | 编辑所有比赛字段 |
| 理论题库 | `/theory/list` | `/admin/theory/questions` | CRUD、导入、导出、测试提交 |
| 理论提交 | `/theory/submissions` | `/admin/theory/submissions` | 提交列表、筛选、导出 |
| 理论提交详情 | `/theory/submission/:id` | `/admin/theory/submissions/:id` | 详情、评分/更新 |
| 理论统计 | `/theory/statistics/:id` | `/admin/theory/statistics/:id` | 比赛维度统计 |
| 题目列表 | `/challenge/list` | `/admin/challenges` | CRUD、文件、提示、测试提交、导入导出 |
| 题目详情 | `/challenge/detail/:id` | `/admin/challenges/:id` | 题目详情、容器、动态 Flag |
| 容器组 | `/challenge/containers` | `/admin/container-groups` | 列表、详情、启动、停止、删除 |
| 漏洞靶场 | `/vuln-lab/list` | `/admin/vuln-labs` | CRUD、实例 |
| 靶场实例 | `/vuln-lab/instances` | `/admin/vuln-labs/instances` | 实例生命周期 |
| 虚拟化宿主 | `/virtualization/hosts` | `/admin/virtualization/hosts` | 宿主 CRUD、检查、镜像、容器 |
| 宿主详情 | `/virtualization/host/:id` | `/admin/virtualization/hosts/:id` | 镜像、容器、日志、状态 |
| 模板管理 | `/virtualization/templates` | `/admin/virtualization/templates` | 模板 CRUD、Compose、构建、文件、同步 |
| 资源仓库 | `/virtualization/resource-repositories` | `/admin/resource-repositories` | CRUD、同步 |
| 管理员 | `/admin/users` | `/admin/admins` | 管理员 CRUD |
| 角色 | `/admin/roles` | `/admin/roles` | 角色 CRUD、权限 |
| 设置 | `/settings` | `/admin/settings` | 设置、备份、恢复、日志、缓存 |
| 审计 | `/audit` | `/admin/audit` | 日志、导出、敏感操作审批 |
| 通知 | `/notifications` | `/admin/notifications` | 消息列表、已读、发送、统计 |
| 来源统计 | `/referral-stats` | `/admin/referral-stats` | 来源历史统计 |

### 5.2 用户端页面

| 旧页面 | 旧路径 | 新项目建议入口 | 必须保留能力 |
| --- | --- | --- | --- |
| 首页 | `/dashboard` | `/` 或 `/player/dashboard` | 公开首页、平台状态、比赛入口 |
| 通知 | `/notifications` | `/player/notifications` | 消息列表、未读、已读、删除 |
| 比赛列表 | `/competition` | `/competitions` | 公开列表、筛选、报名入口 |
| 比赛详情 | `/competition/detail/:id` | `/competitions/:id` | 详情、报名、公告、规则 |
| 比赛赛场 | `/competition-arena/*` | `/arena/:competitionId/*` | 概览、题目、榜单、公告、规则、理论题、Writeup |
| 题目列表 | `/challenges` | `/challenges` | 登录后练习题列表 |
| 题目详情 | `/challenges/:id` | `/challenges/:id` | Flag、容器、提示 |
| 漏洞靶场 | `/vuln-labs` | `/vuln-labs` | 列表、详情、启动、续期、停止 |
| 漏洞靶场详情 | `/vuln-labs/:id` | `/vuln-labs/:id` | 实例和访问信息 |
| 队伍 | `/team` | `/team` | 我的队伍、创建、加入、退出 |
| 队伍详情 | `/team/detail/:id` | `/teams/:id` | 队伍信息 |
| 排行榜 | `/scoreboard` | `/scoreboard` | 多维排行榜、趋势、统计 |
| 账号设置 | `/account/settings` | `/account/settings` | 资料、安全设置 |
| 个人主页 | `/account/profile` | `/account/profile` | 用户资料、统计、活动、徽章 |

## 6. 接口迁移对账要求

### 6.1 路由前缀必须兼容

新后端必须继续支持以下前缀：

- `/api-admin/*`：管理端接口。
- `/api/*`：用户端接口。
- `/api/public/*`：公开接口。
- `/static/uploads/*`：上传文件访问。
- `/static/*`：静态资源访问。
- `/templates/*`：模板文件访问。

### 6.2 管理端接口族

必须从 `backend_back/app/api/admin` 迁移并对齐以下接口族：

| 旧模块 | 新后端目标 | 覆盖要求 |
| --- | --- | --- |
| `auth.py` | `admin_auth.go` | 登录、刷新、当前用户、密码、退出 |
| `dashboard.py` | admin dashboard handlers | 统计全部真实计算 |
| `admin.py` | admin user handlers/store | 管理员 CRUD |
| `users.py`、`user_management.py` | user handlers/store | 用户 CRUD、批量、导入导出、标签、封禁 |
| `roles.py`、`resource_permissions.py` | role/permission handlers/store | RBAC 权限完整迁移 |
| `teams.py` | team handlers/store | 队伍全流程 |
| `competitions.py`、`participants.py` | competition handlers/store | 比赛、参赛者、公告、管理员、题目绑定 |
| `challenges.py`、`challenges_import.py`、`challenge_export.py` | challenge handlers/store | 题目、文件、提示、导入导出、测试提交 |
| `challenge_virtualization.py` | challenge virtualization handlers | 题目容器绑定 |
| `flags.py` | flag handlers/store | 动态 Flag 生成/查看/校验 |
| `virtualization.py`、`template.py`、`template_files.py` | virtualization handlers/runtime | Docker/PVE、模板、文件、构建、同步 |
| `container_group.py`、`container_cleanup.py` | container group handlers | 容器组生命周期 |
| `vuln_labs.py` | vuln lab handlers/store | 靶场 CRUD 和实例 |
| `theory.py`、`exam.py` | theory/exam handlers/store | 理论题、考试、提交、统计、导出 |
| `submission.py`、`scoreboard.py` | submission/scoreboard handlers | CTF 提交、排行榜、冻结 |
| `messages.py`、`announcements.py` | notification handlers/store | 通知、公告 |
| `audit.py` | audit handlers/store | 审计日志、敏感操作 |
| `settings.py` | settings handlers/store | 设置、备份、恢复、日志、缓存 |
| `resource_repositories.py` | resource repository handlers/store | 仓库同步 |
| `visitor_stats.py` | visitor handlers/store | 访客统计 |
| `upload.py` | upload handlers | 通用上传 |
| `writeup.py` | writeup handlers/store | Writeup 管理 |
| `referral.py` | referral handlers/store | 来源统计 |

### 6.3 用户端接口族

必须从 `backend_back/app/api/player` 迁移并对齐以下接口族：

| 旧模块 | 新后端目标 | 覆盖要求 |
| --- | --- | --- |
| `auth.py` | player auth handlers | 登录、注册、刷新、退出 |
| `users.py` | player user handlers/store | 当前用户、资料、头像、统计、活动、徽章 |
| `competitions.py`、`registration.py` | player competition handlers | 比赛列表、详情、报名、取消报名 |
| `challenge.py`、`practice.py` | player challenge handlers | 题目、Flag、容器、提示 |
| `leaderboard.py`、`competition_scoreboard.py` | player scoreboard handlers | 全部排行榜和统计 |
| `teams.py` | player team handlers | 我的队伍、创建、加入、退出、详情 |
| `theory.py` | player theory handlers | 试卷、提交、考试榜 |
| `vuln_lab.py` | player vuln lab handlers | 靶场列表、详情、启动、续期、停止 |
| `messages.py`、`system_announcements.py` | player notification handlers | 通知、系统公告 |
| `sse.py` | SSE handlers | 实时通知 |
| `writeup.py` | player writeup handlers | 读取、保存、下载 |

### 6.4 公共接口

必须从 `backend_back/app/api/public` 迁移：

- `GET /api/public/settings`
- `POST /api/public/referral/track`

## 7. 数据模型迁移要求

新 Go 后端必须等价承接旧 Python 模型。至少覆盖以下领域：

- 用户、管理员、角色、权限、Token、登录安全。
- 比赛、参赛者、比赛管理员、比赛公告、报名状态。
- 队伍、成员、邀请、黑名单、历史记录。
- CTF 题目、分类、难度、附件、提示、Flag、动态 Flag、提交记录。
- 理论题、选项、试卷、答题提交、评分记录、统计。
- 容器、容器组、虚拟化宿主、模板、模板文件、构建日志、同步日志。
- 漏洞靶场、靶场实例。
- 通知、系统公告、SSE 连接。
- Writeup。
- 审计日志、敏感操作审批。
- 系统设置、备份、上传文件、资源仓库、资源权限、访客统计、来源统计。

迁移时优先阅读旧模型目录：

- `backend_back/app/models`
- `backend_back/app/schemas`
- `backend_back/app/api/admin/schemas`
- `backend_back/app/api/player/schemas`

## 8. 新前端实现要求

1. `frontend/src/shared/api` 必须提供统一 HTTP 客户端，自动处理 Token、刷新、401 跳转、错误提示、文件下载。
2. 管理端和用户端 API 方法必须按领域拆分，不能在页面里散落字符串 URL。
3. 页面可以重新设计，但必须保留旧操作能力。
4. 表格必须支持旧页面已有的分页、搜索、筛选、排序、批量操作。
5. 表单必须提交真实接口，后端失败要显示真实错误。
6. 上传、导入、导出、下载必须走真实接口。
7. 容器启动、停止、续期、构建、拉取镜像等长操作必须有真实状态反馈。
8. SSE 通知必须接入真实事件流，不能轮询假消息冒充。
9. 前端不允许保留 `PlayerPlaceholderPage` 或 `AdminPlaceholderPage` 作为最终页面。
10. `frontend/src/apps/player/pages/playerData.ts` 如果包含静态数据，只能作为开发参考，最终页面不能依赖它展示业务数据。

## 9. 后端实现要求

1. 删除或替换所有 `notImplemented(...)` 兜底。
2. 替换所有只返回固定成功文案的 `successHandler(...)`，除非该接口在旧项目中确实只做无状态确认；否则必须接数据库或真实服务。
3. 替换所有固定空数组、固定统计值、固定 CSV 的实现。
4. 旧 FastAPI 的响应字段、分页字段、错误字段要兼容新前端调用。
5. 写入类接口必须做输入校验、权限校验、审计记录。
6. 删除类接口必须处理关联数据，不得静默失败。
7. Docker/PVE 操作必须调用真实 runtime，并正确处理超时、失败、日志、状态同步。
8. 导入导出必须解析/生成真实文件，不能返回模板空文件冒充业务导出。
9. 备份恢复必须真实操作数据库和上传目录。
10. 动态 Flag 必须按旧规则生成、绑定、校验，不能使用固定 Flag。

## 10. AI 执行顺序

### 阶段 1：生成接口清单

1. 扫描旧后端：
   - `backend_back/app/api/admin`
   - `backend_back/app/api/player`
   - `backend_back/app/api/public`
2. 扫描旧前端服务：
   - `front-admin/src/services/ctf-saas`
   - `front-user/src/services/ctf-saas`
   - `front-user/src/services`
3. 扫描新后端：
   - `backend/internal/httpserver/routes.go`
   - `backend/internal/httpserver/frontend_routes.go`
   - `backend/internal/httpserver/*handlers*.go`
4. 输出接口差异表：旧接口、旧方法、旧模块、新接口、新 handler、状态、备注。

### 阶段 2：补齐后端数据层

1. 对照旧 `models` 和 `schemas` 建立 Go 模型。
2. 对照旧 `crud` 和 `services` 迁移业务逻辑。
3. 先完成认证、用户、角色权限、比赛、题目、提交这些核心链路。
4. 再完成虚拟化、模板、靶场、导入导出、备份、SSE 等复杂链路。

### 阶段 3：补齐后端接口

按接口族逐个替换占位实现。每完成一个接口族，必须：

1. 新增或更新 handler。
2. 新增或更新 store/service。
3. 用真实数据库或真实外部服务验证。
4. 在接口差异表中标记为完成。

### 阶段 4：补齐前端 API

1. 在 `frontend/src/shared/api` 按领域补齐 admin/player API。
2. 从旧前端服务层迁移请求参数、响应类型、下载/上传处理。
3. 禁止页面直接写裸 `fetch('/api...')`，必须使用统一 API。

### 阶段 5：补齐页面

1. 按第 5 节页面对账表逐页实现。
2. 先完成核心业务闭环：登录、比赛、题目、提交、排行榜。
3. 再完成管理闭环：用户、队伍、比赛、题库、理论题、虚拟化、设置。
4. 最后完成审计、统计、通知、导入导出、备份恢复等辅助能力。

### 阶段 6：验收

1. 全仓搜索禁止词。
2. 跑后端测试。
3. 跑前端类型检查和构建。
4. 用真实数据手动走完核心路径。
5. 输出最终迁移对账表。

## 11. 验收命令

在仓库根目录执行：

```powershell
rg -n "TODO|FIXME|placeholder|mock|fake|demo data|notImplemented|successHandler" backend frontend
```

除非命中第三方依赖或明确的测试断言，否则不得有结果。

后端：

```powershell
cd backend
go test ./...
go build ./cmd/server
```

前端：

```powershell
cd frontend
npm run typecheck
npm run build
```

旧项目接口参考：

```powershell
cd backend_back
rg -n "@router\\." app\api
```

新项目接口参考：

```powershell
cd backend
rg -n "mux.HandleFunc" internal\httpserver
```

旧前端 API 参考：

```powershell
cd front-admin
rg -n "request" src\services\ctf-saas

cd ..\front-user
rg -n "request" src\services src\services\ctf-saas
```

## 12. 最低通过标准

1. 新项目能启动：`backend` 服务启动，`frontend` 能访问。
2. 管理员能登录并进入管理端。
3. 选手能注册/登录并进入用户端。
4. 管理端能完成用户、队伍、比赛、题目、理论题的 CRUD。
5. 用户端能查看比赛、报名、进入赛场、查看题目、提交 Flag、查看排行榜。
6. 容器题能真实启动、续期、停止，并返回真实连接信息。
7. 通知、公告、Writeup、导入导出、上传下载、备份恢复都可用。
8. 审计、权限、访问控制正确生效。
9. 所有接口都返回真实数据或真实错误。
10. 全仓禁止词扫描通过。

## 13. 交付物

迁移完成后必须交付：

1. `backend` 完整实现。
2. `frontend` 完整实现。
3. 接口迁移对账表。
4. 页面迁移对账表。
5. 数据模型迁移说明。
6. 验收测试记录。
7. 已知差异说明。若无差异，明确写“无已知差异”。

## 14. AI 迁移提示词

可以把下面提示词交给后续 AI 继续执行迁移：

```text
你正在迁移 CloudCTF 项目。

旧项目：
- front-admin
- front-user
- backend_back

新项目：
- frontend
- backend

目标：
保证旧项目所有接口和所有功能在新项目中都有真实实现。页面交互和样式可以自由发挥，但业务能力、接口能力、权限、数据、文件、导入导出、容器生命周期、SSE、备份恢复等不能缺失。

强制约束：
- 禁止 TODO、FIXME、placeholder、mock、fake、demo data、notImplemented。
- 禁止假数据。
- 禁止固定成功返回冒充实现。
- 禁止静态空数组冒充查询结果。
- 禁止删除旧功能。

执行方式：
1. 阅读 docs/AI迁移实施文档.md。
2. 扫描旧后端 backend_back/app/api 下所有 @router 接口。
3. 扫描旧前端 front-admin/src/services 和 front-user/src/services 下所有 request 调用。
4. 扫描新后端 backend/internal/httpserver 下所有 mux.HandleFunc。
5. 生成旧接口 -> 新接口 -> 新 handler 的差异表。
6. 逐接口补齐 backend 的真实业务实现。
7. 逐页面补齐 frontend 的真实接口调用和真实交互。
8. 每完成一个功能域，运行相关测试和构建。
9. 最终运行禁止词扫描、go test、go build、npm run typecheck、npm run build。

优先级：
1. 认证和权限。
2. 用户、队伍、比赛、题目、提交、排行榜。
3. 理论题和考试。
4. 虚拟化、容器、模板、漏洞靶场。
5. 通知、公告、Writeup、导入导出、备份恢复、审计、统计。
```
