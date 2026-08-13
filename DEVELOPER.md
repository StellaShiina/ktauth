# KTAUTH 开发者文档

> **KTAUTH（簡単auth）** — 基于 Go 的轻量级认证与授权网关服务。
> 面向开发者的架构设计、模块说明、开发指南与扩展指引。

---

## 目录

1. [项目概述](#1-项目概述)
2. [技术栈与依赖](#2-技术栈与依赖)
3. [架构设计](#3-架构设计)
4. [项目结构详解](#4-项目结构详解)
5. [核心流程](#5-核心流程)
6. [配置与环境变量](#6-配置与环境变量)
7. [数据库设计](#7-数据库设计)
8. [API 接口规范](#8-api-接口规范)
9. [本地开发环境搭建](#9-本地开发环境搭建)
10. [测试指南](#10-测试指南)
11. [构建与部署](#11-构建与部署)
12. [CI/CD 流水线](#12-cicd-流水线)
13. [扩展指南](#13-扩展指南)
14. [代码规范与约定](#14-代码规范与约定)
15. [路线图](#15-路线图)

---

## 1. 项目概述

KTAUTH 是一个部署在反向代理（Caddy / Nginx）后方的认证网关服务。它的核心职责是：

- **IP 访问控制**：基于黑白灰名单的 IP 级别准入判断
- **用户认证**：JWT + Redis Session 的用户登录/登出/鉴权
- **速率限制**：毫秒级滑动窗口算法的请求频控
- **令牌管理**：基于 Redis Set 的注册邀请码体系

服务监听在 `:51214` 端口，设计上不直接面向公网，而是作为 Caddy `forward_auth` 或 Nginx `auth_request` 的后端子请求目标。

### 设计哲学

- **关注点分离**：严格分层 `Handler → Service → Repository → DB`
- **无状态 JWT**：结合 Redis 实现可控的会话失效
- **高性能缓存**：IP 规则带 TTL 的 Redis 缓存，减少数据库查询
- **原子化操作**：限流和 Abuse 检测通过 Redis Lua 脚本保证原子性

---

## 2. 技术栈与依赖

| 组件 | 技术选型 | 说明 |
|------|---------|------|
| 语言 | Go 1.26+ | 模块路径 `github.com/StellaShiina/ktauth` |
| Web 框架 | [Gin v1.12](https://github.com/gin-gonic/gin) | HTTP 路由、中间件、参数绑定 |
| 数据库 | PostgreSQL | 通过 [pgx v5](https://github.com/jackc/pgx) 连接池访问 |
| 缓存 | Redis | 通过 [go-redis v9](https://github.com/redis/go-redis) 访问 |
| JWT | [golang-jwt v5](https://github.com/golang-jwt/jwt) | HS256 签名，7 天过期 |
| 密码 | [bcrypt](https://pkg.go.dev/golang.org/x/crypto/bcrypt) | DefaultCost 哈希 |
| UUID | [google/uuid](https://github.com/google/uuid) | v4 随机 UUID |

### 完整依赖树（go.mod）

```
github.com/gin-gonic/gin          # Web 框架
github.com/golang-jwt/jwt/v5      # JWT 签名/验证
github.com/google/uuid            # UUID 生成
github.com/jackc/pgx/v5           # PostgreSQL 驱动 + 连接池
github.com/redis/go-redis/v9      # Redis 客户端
golang.org/x/crypto               # bcrypt 密码哈希
```

---

## 3. 架构设计

### 3.1 分层架构

```
┌─────────────────────────────────────────┐
│               HTTP Request               │
└─────────────────┬───────────────────────┘
                  │
    ┌─────────────▼─────────────┐
    │        Middleware          │  ← 请求拦截层
    │  (CheckIP / Auth / Rate)   │
    └─────────────┬─────────────┘
                  │
    ┌─────────────▼─────────────┐
    │         Handler            │  ← 请求处理层（Controller）
    │  参数绑定 / 响应构造        │
    └─────────────┬─────────────┘
                  │
    ┌─────────────▼─────────────┐
    │         Service            │  ← 业务逻辑层
    │  编排 Repository 调用       │
    └─────────────┬─────────────┘
                  │
    ┌─────────────▼─────────────┐
    │        Repository          │  ← 数据访问层
    │  SQL / Redis 命令封装       │
    └─────────────┬─────────────┘
                  │
    ┌─────────────▼─────────────┐
    │     PostgreSQL / Redis     │  ← 数据存储层
    └───────────────────────────┘
```

### 3.2 依赖注入

所有依赖在 `cmd/ktauth/main.go` 中手动组装（无框架 DI）：

```go
// 1. 初始化数据库连接
redis  := db.NewRedis()
postgres := connectPostgres(30 * time.Second)

// 2. 初始化 Repository
ipRepo    := repository.NewIPRepo(postgres)
userRepo  := repository.NewUserRepo(postgres)
tokenRepo := repository.NewTokenRepo(redis)
// ...

// 3. 初始化 Service（注入 Repository）
ipAccessService := access.NewIPAccessService(ipRepo, ipCache)
accountService  := identity.NewAccountService(userRepo)
// ...

// 4. 初始化 Middleware（注入 Service）
checkIPMiddleware := middleware.NewCheckIPMiddleware(ipAccessService)

// 5. 初始化 Handler（注入 Service）
userHandler := handler.NewUserHandler(sessionService, accountService, consumeTokenService)

// 6. 注册路由
router.RegisterUserRouter(r, userHandler, checkIPMiddleware, authMiddleWare, rateLimitMiddleware)
```

### 3.3 请求处理流水线

以 `POST /api/users/login` 为例：

```
Request
  │
  ▼
[CheckIP Middleware]  ─── 查询 IP 规则（带缓存），黑名单直接 403
  │
  ▼
[RateLimit Middleware] ─── 滑动窗口限流，白名单 IP 自动跳过
  │
  ▼
[UserHandler.LoginUser]
  │
  ├─► AccountService.GetUserByName()   ──► UserRepo.GetUserByName()  ──► PostgreSQL
  ├─► crypto.VerifyPassword()         （bcrypt 比对）
  ├─► auth.SignToken()                （JWT 签发）
  └─► SessionService.CreateSession()  ──► SessionRepo.CreateSession() ──► Redis
  │
  ▼
Response (200 + JWT token)
```

---

## 4. 项目结构详解

```
ktauth/
├── cmd/ktauth/main.go              # ★ 应用入口：依赖注入 + 路由注册
├── internal/
│   ├── auth/jwt.go                 # JWT 签名（HS256）与解析
│   ├── crypto/
│   │   ├── password.go             # bcrypt 密码哈希与验证
│   │   └── rand.go                 # 加密安全的随机数字串生成
│   ├── db/
│   │   ├── postgres.go             # PostgreSQL 连接池（pgxpool）
│   │   └── redis.go                # Redis 客户端初始化
│   ├── handler/
│   │   ├── admin_handler.go        # IP 规则管理 + 用户列表 Handler
│   │   ├── user_handler.go         # 注册 / 登录 / 登出 Handler
│   │   └── token_handler.go        # Token 管理 Handler
│   ├── middleware/
│   │   ├── auth.go                 # JWT Session 验证中间件
│   │   ├── checkip.go              # IP 黑白灰名单 ACL 中间件
│   │   └── ratelimit.go            # 速率限制 + Abuse 自动封禁中间件
│   ├── model/
│   │   ├── ip.go                   # IP 规则数据模型 + 类型常量
│   │   └── user.go                 # User 数据模型
│   ├── repository/
│   │   ├── countdown_repo.go       # 倒计时/冷却 Repository（Redis）
│   │   ├── ip_repo.go              # IP 规则 CRUD（PostgreSQL）
│   │   ├── iprule_cache.go         # IP 规则缓存（Redis，带差异化 TTL）
│   │   ├── ratelimit_repo.go       # 滑动窗口限流（Redis Lua）
│   │   ├── register_repo.go        # 注册验证码 Repository（Redis）
│   │   ├── session_repo.go         # JWT 会话 Repository（Redis）
│   │   ├── token_repo.go           # 邀请 Token Repository（Redis Set）
│   │   └── user_repo.go            # 用户 CRUD（PostgreSQL）
│   ├── router/
│   │   ├── admin_router.go         # /api/ips + /api/users（管理）路由
│   │   ├── token_router.go         # /api/tokens 路由
│   │   └── user_router.go          # /api/users 路由
│   └── service/
│       ├── access/
│       │   ├── cd.go               # 冷却服务（CountDown）
│       │   ├── ip.go               # IP 规则查询服务（缓存优先）
│       │   └── ratelimit.go        # 限流 + Abuse 检测服务
│       ├── admin/
│       │   ├── manage_iprule.go    # IP 规则管理服务
│       │   ├── manage_token.go     # Token 管理服务
│       │   ├── manage_user.go      # 用户管理服务
│       │   └── types.go            # API 响应类型定义
│       └── identity/
│           ├── account.go          # 账户服务（创建/查询/更新用户）
│           ├── consume_token.go    # Token 消费服务
│           └── session.go          # 会话服务（创建/删除/验证）
├── pkg/iputils/processip.go        # IP 地址解析 + CIDR 规范化工具
├── sql/
│   ├── 00-init.sql                 # 数据库建表 + 初始数据（admin 用户 + 内网白名单）
│   └── 10-ipdata.sql               # 额外的预置 IP 白名单数据
├── scripts/install.sh              # 一键部署脚本
├── docker-compose.yaml             # 生产部署（ktauth + postgres + redis）
├── docker-compose.db.yaml          # 仅数据库（本地开发用）
├── docker-compose.test.yaml        # 测试部署（使用本地构建镜像）
├── .env.example                    # 环境变量模板
└── .github/workflows/ci.yaml       # CI/CD：测试 + 发布 + Docker 构建
```

---

## 5. 核心流程

### 5.1 IP 访问控制流程

```
Client IP → IPAccessService.QueryRule()
              │
              ├─► iputils.ProcessIP(ipStr)
              │     单 IP → /32 (IPv4) 或 /64 (IPv6) 掩码
              │     CIDR → 保持原样
              │
              ├─► IPCache.Get(ipNet)          ← Redis 缓存查询
              │     Hit  → 直接返回规则类型
              │     Miss → 继续
              │
              ├─► IPRepo.QueryIP(version, ip)  ← PostgreSQL 查询
              │     Found    → 回写 Redis 缓存
              │     Not Found → 视为 greylist，写缓存（5min TTL）
              │
              └─► 返回 IPRuleType (whitelist/blacklist/greylist)
```

**缓存 TTL 策略：**

| 规则类型 | TTL | 原因 |
|---------|-----|------|
| Blacklist | 1 小时 | 需快速拒绝，变化少 |
| Whitelist | 30 分钟 | 需快速放行 |
| Greylist | 5 分钟 | 默认状态，变化可能性大 |

### 5.2 JWT 认证流程

```
登录请求
  │
  ├─► AccountService.GetUserByName()  查询用户
  ├─► crypto.VerifyPassword()         验证密码
  ├─► auth.SignToken(uuid, name, role)
  │     └─► 生成 JWT（HS256，7 天过期）
  │         Claims: { UUID, Name, Role, jti, exp, iat, iss }
  │
  └─► SessionService.CreateSession(uuid, jti)
        └─► Redis SET "jwt:active:{uuid}:{jti}" = uuid (144h TTL)

后续请求
  │
  ├─► AuthMiddleWare.VerifySession()
  │     ├─► 提取 Authorization: Bearer <token>
  │     ├─► auth.ParseToken() 解析 JWT
  │     └─► SessionService.GetSession(uuid, jti)  Redis 验证
  │          存在 → 认证通过，设置 ctx uuid/jti
  │          不存在 → 401（会话已失效/登出）
```

### 5.3 滑动窗口限流算法

实现在 `ratelimit_repo.go` 的 Lua 脚本中：

```lua
-- 1. 清理窗口外的旧记录
ZREMRANGEBYSCORE key '-inf' (now - window)

-- 2. 统计窗口内请求数
count = ZCARD key

-- 3. 判断
if count < limit then
    ZADD key now member    -- 记录本次请求（score=时间戳, member=UUID）
    PEXPIRE key window     -- 刷新 Key 过期时间
    return 1               -- 允许
else
    return 0               -- 拒绝
end
```

**特点：**
- 使用 Redis Sorted Set（ZSET），score 为毫秒时间戳
- member 用随机 UUID 防止同一毫秒内重复请求互相覆盖
- 原子化操作，无需分布式锁
- 默认 60 次/分钟，可在 `.env` 中配置

### 5.4 Abuse 自动封禁机制

当请求被限流返回 429 后，中间件会额外执行 abuse 检测：

```go
// ratelimit.go 中间件
if !allow {
    c.String(http.StatusTooManyRequests, "Rate limit exceed!")
    // 检测 abuse
    if abuse, err := m.rateLimitService.Abuse(ctx, ip); err == nil && abuse {
        // 自动加入黑名单
        m.adminIPRuleService.AddRule(ctx, ip, false, &note)
    }
}
```

Abuse 检测使用 Redis INCR + EXPIRE 模式：
- Key: `abuse:429:{cidr}`
- 默认：5 分钟内收到 100 次 429 → 触发自动封禁
- 触发后自动清除计数 Key，下次重新计数

### 5.5 注册凭据体系

```
注册流程：
  POST /api/users/register { token, user, password }
    │
    ├─ token 非空 ► ConsumeTokenService.Consume(token)
    │     └─► Redis SREM "admin:tokens" token
    │           成功（n > 0）→ Token 有效，已消费
    │           失败（n = 0）→ Token 无效或已被使用
    │
    ├─ 无 token ► EmailService.VerifyCode(email, code)
    │     └─► Redis GETDEL "register:{email}:{code}"
    │           成功 → 验证码有效，已消费
    │           失败 → 验证码无效或已被使用
    │
    └─► AccountService.NewUser() → PostgreSQL INSERT

同时提供 token 和邮箱验证码时，优先验证 token。

管理员操作：
  GET  /api/tokens/restock  → 批量生成 10 个 UUID Token 到 Redis Set
  GET  /api/tokens          → 随机获取一个未使用的 Token
  GET  /api/tokens/all      → 列出所有可用 Token
  DELETE /api/tokens/flush   → 清空所有 Token
```

---

## 6. 配置与环境变量

### 6.1 环境变量一览

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `ADMIN_NAME` | `admin` | 管理员用户名 |
| `ADMIN_PASSWD` | `admin` | 管理员密码（bcrypt 加密后存储） |
| `JWT_SECRET` | `ktauthsecret` | JWT 签名密钥（HS256） |
| `RATELIMIT` | `60` | 每分钟允许的请求数 |
| `ENABLE_RATELIMIT` | 空（启用） | 设为 `NO` 禁用限流 |
| `ABUSELIMIT` | `100` | 触发自动封禁的 429 次数阈值 |
| `ABUSEWINDOW` | `5` | Abuse 检测时间窗口（分钟） |
| `LOGLEVEL` | `error` | 日志级别：`debug` / `info` / `warn` / `error` |
| `SMTP_HOST` | - | SMTP 服务器主机名 |
| `SMTP_PORT` | `587` | SMTP Submission 端口 |
| `SMTP_USERNAME` | - | SMTP 登录用户名；留空则不认证 |
| `SMTP_PASSWORD` | - | SMTP 登录密码 |
| `SMTP_FROM` | - | 发件人地址，可使用 `KTAUTH <ktauth@example.com>` 格式 |
| `REDIS_HOST` | `127.0.0.1` | Redis 连接地址 |
| `POSTGRES_HOST` | `127.0.0.1` | PostgreSQL 连接地址 |
| `POSTGRES_PORT` | `5432` | PostgreSQL 端口 |
| `GIN_MODE` | - | Gin 运行模式（设为 `release` 启用生产模式） |

### 6.2 硬编码常量

以下值目前硬编码在源码中，修改需要改代码：

| 位置 | 常量 | 值 | 说明 |
|------|------|----|------|
| `auth/jwt.go` | Token 过期时间 | 168 小时（7 天） | JWT exp |
| `cmd/ktauth/main.go` | 监听端口 | `:51214` | Gin Run 地址 |
| `db/postgres.go` | 数据库凭据 | `ktauth:ktauth` | PostgreSQL 用户/密码/库名 |
| `session_repo.go` | Session TTL | 144 小时（6 天） | Redis Key 过期 |
| `ratelimit_repo.go` | 滑动窗口 | 1 分钟 | Rate Limit 窗口大小 |

---

## 7. 数据库设计

### 7.1 PostgreSQL 表结构

#### `users` 表

```sql
CREATE TABLE users (
    uuid          UUID PRIMARY KEY,           -- 用户唯一标识
    name          VARCHAR(64) NOT NULL UNIQUE, -- 用户名
    password_hash CHAR(60) NOT NULL,           -- bcrypt 哈希（固定 60 字符）
    email         VARCHAR(255) UNIQUE,         -- 邮箱（可选）
    role          VARCHAR(32) NOT NULL DEFAULT 'user' -- 角色：user / admin
);
```

**内置管理员：**
- UUID: `00000000-0000-0000-0000-000000000000`
- 默认密码: `admin`（bcrypt hash）

#### `ip` 表

```sql
CREATE TABLE ip (
    id           BIGSERIAL PRIMARY KEY,       -- 自增主键
    version      SMALLINT NOT NULL,            -- IP 版本：4 或 6
    ip_range     CIDR NOT NULL UNIQUE,         -- IP/CIDR 范围（PostgreSQL CIDR 类型）
    is_whitelist BOOLEAN NOT NULL,             -- true=白名单, false=黑名单
    create_at    TIMESTAMPTZ DEFAULT NOW(),    -- 创建时间
    update_at    TIMESTAMPTZ DEFAULT NOW(),    -- 更新时间（触发器自动维护）
    note         TEXT                           -- 备注
);
```

**IP 匹配使用 PostgreSQL CIDR 包含运算符 `<<=`：**
```sql
SELECT is_whitelist FROM ip
WHERE version = $1 AND $2::inet <<= ip_range
```

**内置规则（00-init.sql）：**
- `127.0.0.0/8` — localhost 白名单
- `10.0.0.0/8` — A 类私有地址白名单
- `192.168.0.0/16` — C 类私有地址白名单
- `172.16.0.0/12` — B 类私有地址白名单

### 7.2 Redis 数据结构

| Key Pattern | 类型 | 值 | TTL | 说明 |
|-------------|------|----|----|------|
| `jwt:active:{uuid}:{jti}` | String | uuid | 144h | JWT 会话 |
| `rule:ip:{cidr}` | String | "whitelist" / "blacklist" / "greylist" | 30min / 1h / 5min | IP 规则缓存 |
| `ratelimit:ip:{cidr}` | ZSET | member(UUID) → score(ms) | 窗口大小 | 滑动窗口计数 |
| `abuse:429:{cidr}` | String (counter) | 计数值 | ABUSEWINDOW | Abuse 检测 |
| `admin:tokens` | Set | UUID strings | 永久 | 注册邀请码池 |
| `register:{email}:{code}` | String | "" | 15min | 一次性邮箱验证码 |
| `{email}` | String | "" | 1min | 邮箱验证码发送冷却 |

---

## 8. API 接口规范

### 8.1 核心认证端点（反向代理用）

| 方法 | 路径 | 说明 | 返回 |
|------|------|------|------|
| `GET` | `/kt/0` | 综合认证：黑名单拒，非白名单限速 | `204 No Content` / `403 Forbidden` / `429 Too Many Requests` |
| `GET` | `/kt/1` | 严格认证：仅白名单放行 | `204 No Content` / `403 Forbidden` |

### 8.2 用户接口

| 方法 | 路径 | 鉴权 | 说明 |
|------|------|------|------|
| `POST` | `/api/users/register` | Token 或邮箱验证码 | 用户注册；同时提供时优先 Token |
| `POST` | `/api/users/send-code` | 无 | 发送 6 位邮箱验证码 |
| `POST` | `/api/users/verify-code` | 无 | 验证并消费邮箱验证码 |
| `POST` | `/api/users/login` | 无 | 用户登录，返回 JWT |
| `GET` | `/api/users/auth` | Bearer JWT | 验证登录状态，`204` 表示有效 |
| `GET` | `/api/users/logout` | Bearer JWT | 登出当前会话 |
| `GET` | `/api/users` | Admin + 白名单 | 列出所有用户 |

**注册请求体：**
```json
{
  "user": "username",
  "password": "password",
  "email": "user@example.com",
  "code": "123456"
}
```

也可使用邀请码，将 `email`、`code` 替换为 `"token": "uuid-token"`；如果同时提供，优先验证 `token`。验证码有效期为 15 分钟且只能使用一次，独立验证接口成功后也会消费验证码。

**发送验证码请求体：**
```json
{ "email": "user@example.com" }
```

**独立验证请求体：**
```json
{ "email": "user@example.com", "code": "123456" }
```

**登录请求体：**
```json
{
  "user": "username",
  "password": "password"
}
```

**登录响应：**
```json
{ "token": "eyJhbGciOi..." }
```
或 `?format=string` → 纯文本 Token

### 8.3 Token 管理接口（管理员）

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/tokens/restock` | 批量生成 10 个 Token |
| `DELETE` | `/api/tokens/flush` | 清空所有 Token |
| `GET` | `/api/tokens` | 随机获取一个 Token |
| `GET` | `/api/tokens/all` | 列出所有可用 Token |

### 8.4 IP 规则管理接口（管理员）

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/ips` | 列出 IP 规则（支持 `?version=4/6&type=white/black` 过滤） |
| `POST` | `/api/ips/new` | 添加 IP 规则 |
| `DELETE` | `/api/ips` | 删除 IP 规则 |

**添加规则请求体：**
```json
{
  "ip": "192.168.1.0/24",
  "isWhiteList": true,
  "note": "office network"
}
```

---

## 9. 本地开发环境搭建

### 9.1 前置要求

- Go 1.26+
- Docker & Docker Compose（用于启动 PostgreSQL 和 Redis）

### 9.2 快速启动

```bash
# 1. 克隆仓库
git clone https://github.com/StellaShiina/ktauth.git
cd ktauth

# 2. 仅启动数据库（PostgreSQL + Redis）
docker compose -f docker-compose.db.yaml up -d

# 3. 复制环境变量配置
cp .env.example .env
# 按需编辑 .env

# 4. 运行应用
go run ./cmd/ktauth

# 或者构建后运行
go build -o ktauth.exe ./cmd/ktauth
./ktauth.exe
```

### 9.3 开发工作流建议

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./internal/repository/ -v
go test ./pkg/iputils/ -v

# 带竞态检测
go test -race ./...

# 代码格式化
go fmt ./...

# 依赖管理
go mod tidy
go mod verify
```

---

## 10. 测试指南

### 10.1 测试文件组织

```
internal/db/postgres_test.go                # DB 连接测试
internal/repository/ip_repo_test.go         # IP Repository 测试
internal/repository/user_repo_test.go       # User Repository 测试
internal/service/admin/manage_iprule_test.go # IP 管理服务测试
pkg/iputils/processip_test.go               # IP 工具函数测试
```

### 10.2 运行测试

测试依赖真实的 PostgreSQL 和 Redis 实例：

```bash
# 启动测试数据库
docker compose -f docker-compose.db.yaml up -d

# 运行全部测试
go test ./... -v

# CI 中也用同样方式：
# docker compose -f ./docker-compose.db.yaml up -d
# go test ./...
```

### 10.3 测试覆盖范围

| 测试文件 | 覆盖内容 |
|---------|---------|
| `processip_test.go` | IPv4/IPv6 解析、CIDR 规范化、非法输入 |
| `postgres_test.go` | 数据库连接验证 |
| `ip_repo_test.go` | IP 规则增删改查、重复检测、不存在检测 |
| `user_repo_test.go` | 用户增删改查、重复检测、不存在检测 |
| `manage_iprule_test.go` | IP 管理服务（Add/Del/List）+ 缓存联动 |

### 10.4 编写新测试

约定：
- 测试包名使用 `{package}_test`（黑盒测试）
- 测试函数名：`Test{FunctionName}`
- 使用标准库 `testing`，不使用第三方断言库
- 测试后清理数据（参见 `user_repo_test.go` 的 `DelUser` 清理模式）

```go
func TestNewFeature(t *testing.T) {
    // Setup
    db, err := db.NewPostgres()
    if err != nil {
        t.Fatal(err)
    }
    ctx := context.Background()

    // Test
    result, err := DoSomething(ctx)
    if err != nil {
        t.Errorf("unexpected error: %v", err)
    }

    // Cleanup
    // ...
}
```

---

## 11. 构建与部署

### 11.1 本地构建

```bash
# 开发构建
go build -o ktauth.exe ./cmd/ktauth

# 生产构建（Linux 交叉编译）
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ktauth ./cmd/ktauth
```

### 11.2 Docker 部署

**生产部署（从 Docker Hub 拉取镜像）：**

```bash
cp .env.example .env
# 编辑 .env 配置
docker compose up -d
```

**测试部署（使用本地构建镜像）：**

```bash
# 先构建本地镜像
docker build -t ktauth:test .

# 使用测试 compose 文件
docker compose -f docker-compose.test.yaml up -d
```

### 11.3 Caddy 集成示例

```caddyfile
example.com {
    forward_auth localhost:51214 {
        uri /kt/0
    }
    reverse_proxy localhost:8080
}
```

### 11.4 Nginx 集成示例

```nginx
server {
    listen 443 ssl;
    server_name example.com;

    location / {
        auth_request /auth;
        proxy_pass http://localhost:8080;
    }

    location = /auth {
        internal;
        proxy_pass http://localhost:51214/kt/0;
        proxy_set_header X-Forwarded-For $remote_addr;
        proxy_pass_request_body off;
        proxy_set_header Content-Length "";
    }
}
```

---

## 12. CI/CD 流水线

定义在 `.github/workflows/ci.yaml`，触发条件：

- **Push 到 `main` 分支** → 运行测试
- **推送 `v*` 标签** → 测试 → 创建 Release + 构建 Docker 镜像

### 流水线步骤

```
┌──────────────────┐
│   go-test Job    │
│  1. 启动 DB 容器  │
│  2. 安装 Go 1.26 │
│  3. go test ./... │
└────────┬─────────┘
         │ (仅 tag 触发)
    ┌────┴────┐
    │         │
    ▼         ▼
┌────────┐ ┌──────────┐
│ Release│ │  Docker  │
│ 打包zip│ │ 构建推送  │
└────────┘ └──────────┘
```

**Release 产物：**
- `ktauth_{version}.zip`（包含 `docker-compose.yaml`、`00-init.sql`、`.env.example`）
- `install.sh` 安装脚本

**Docker 镜像标签：**
- `stellashiina/ktauth:{version}`
- `stellashiina/ktauth:latest`

---

## 13. 扩展指南

### 13.1 添加新的 API 端点

1. **定义 Handler** → `internal/handler/`
2. **定义 Service（如需业务逻辑）** → `internal/service/`
3. **定义 Repository（如需数据访问）** → `internal/repository/`
4. **注册路由** → `internal/router/`
5. **在 `main.go` 中组装依赖**

**示例：添加 "重置密码" 端点**

```go
// internal/handler/user_handler.go 添加方法
func (h *UserHandler) ResetPassword(c *gin.Context) {
    // 获取当前用户 UUID（中间件注入）
    uuid := c.GetString("uuid")
    // 解析请求体...
    // 调用 service...
}

// internal/router/user_router.go 注册路由
user.POST("/reset-password", h.ResetPassword)
```

### 13.2 添加新的中间件

中间件遵循 Gin 的 `gin.HandlerFunc` 签名：

```go
func NewMyMiddleware(dep *SomeDependency) *MyMiddleware {
    return &MyMiddleware{dep}
}

func (m *MyMiddleware) Handle() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 前置处理
        // ...

        c.Next()

        // 后置处理（可选）
        // ...
    }
}
```

在 `main.go` 中实例化并应用到路由：

```go
myMiddleware := middleware.NewMyMiddleware(dep)
r.Use(myMiddleware.Handle())  // 全局应用
// 或
g := r.Group("/api/xxx", myMiddleware.Handle())  // 分组应用
```

### 13.3 添加新的配置项

1. 在 `.env.example` 添加变量
2. 在 `cmd/ktauth/main.go` 的 `main()` 中读取（`os.Getenv()`）
3. 传递给需要的 Service/Middleware
4. 更新本文档的「配置」章节

### 13.4 邮箱验证实现

`EmailService` 使用 Go 标准库 `net/smtp` 发送 HTML 邮件。验证码由 `crypto/rand` 生成，存入 Redis 15 分钟，并通过 `CountDownRepo` 限制同一邮箱每分钟发送一次。验证码使用 `GETDEL` 原子验证并消费。

---

## 14. 代码规范与约定

### 14.1 命名规范

| 类型 | 约定 | 示例 |
|------|------|------|
| 包名 | 小写，简短 | `iputils`, `access`, `identity` |
| 文件名 | snake_case | `user_repo.go`, `manage_iprule.go` |
| 结构体 | PascalCase | `IPAccessService`, `RateLimitRepo` |
| 方法 | PascalCase（导出）/ camelCase（私有） | `QueryRule()`, `connectPostgres()` |
| 变量 | camelCase | `ipRepo`, `isWhitelist` |
| 常量 | PascalCase 或 UPPER_SNAKE | `IPWhiteList`, `ErrIPNotFound` |
| 错误变量 | `Err` 前缀 | `ErrUserNotFound`, `ErrIPExist` |

### 14.2 项目规范

- **错误处理**：Repository 层定义哨兵错误（`var ErrXxx = errors.New(...)`），Service 层传递，Handler 层用 `errors.As()` 区分处理
- **日志**：使用 `log/slog` 标准库，不用 `fmt.Println`
- **Context 传递**：所有数据访问方法第一个参数为 `context.Context`
- **SQL 安全**：使用参数化查询（`$1`, `$2`），绝不拼接 SQL
- **Redis Key 命名**：`{domain}:{sub}:{identifier}` 格式（如 `jwt:active:{uuid}:{jti}`）

### 14.3 导入顺序

```go
import (
    // 1. 标准库
    "context"
    "fmt"

    // 2. 本项目包
    "github.com/StellaShiina/ktauth/internal/model"
    "github.com/StellaShiina/ktauth/internal/repository"

    // 3. 第三方库
    "github.com/gin-gonic/gin"
    "github.com/redis/go-redis/v9"
)
```

## 15. 路线图

- [x] 实现SMTP邮箱验证码发送与注册验证
- [ ] 优化session管理
- [ ] 管理员web面板

---

## 附录：快速参考卡片

### 常用命令

```bash
# 开发
go run ./cmd/ktauth                          # 启动服务
go test ./... -v                             # 运行所有测试
docker compose -f docker-compose.db.yaml up -d  # 启动开发数据库

# 构建
go build -o ktauth ./cmd/ktauth              # 本地构建
GOOS=linux GOARCH=amd64 go build -o ktauth ./cmd/ktauth  # 交叉编译

# 部署
docker compose up -d                         # 启动全栈
docker compose logs -f ktauth                # 查看日志
docker compose restart ktauth                # 重启服务
```

### 默认端口

| 服务 | 端口 |
|------|------|
| KTAUTH API | 51214 |
| PostgreSQL | 5432 |
| Redis | 6379 |

### 仓库信息

- **模块路径**: `github.com/StellaShiina/ktauth`
- **Docker 镜像**: `stellashiina/ktauth`
- **安装脚本**: `https://ktauth.kaju.win`
