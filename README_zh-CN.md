# KTAUTH (简单认证)

[English](./README.md) | [简体中文](./README_zh-CN.md)

**KTAUTH** 是一个基于 Go 语言构建的稳健且轻量级的认证与授权服务。它为管理用户身份、基于 IP 地址的访问控制以及强制执行速率限制（Rate Limiting）以保护您的 API 提供了安全的基础。

项目名称 "KTAUTH" 源自日语 "Kantan Auth" (簡単)，意为 "简单认证"。

## 🚀 技术栈 (Tech Stack)

- **开发语言:** [Go 1.25+](https://go.dev/)
- **Web 框架:** [Gin](https://github.com/gin-gonic/gin)
- **数据库:** [MySQL](https://www.mysql.com/)
- **缓存 & 限流:** [Redis](https://redis.io/)
- **认证方式:** [JWT (JSON Web Tokens)](https://jwt.io/)
- **邮件服务:** [Resend](https://resend.com/)
- **容器化:** Docker & Docker Compose

## ✨ 核心特性 (Key Features)

### ⚡ 简单快速
认证通过返回204，caddy可以直接利用forward_auth对接
- `GET /kt/0` 对黑名单限速，灰名单限流，白名单放行
- `GET /kt/1` 仅限白名单

### 🔐 安全认证
- **JWT 实现:** 使用 JSON Web Tokens 进行无状态认证，确保 API 访问安全。
- **会话管理:** 基于 Redis 的稳健会话（Session）处理。
- **邮件验证:** 集成 Resend 邮件服务，实现用户注册时的邮箱验证流程。
- **密码安全:** 使用 `bcrypt` 进行安全的密码哈希存储。

### 🛡️ 访问控制与安全
- **IP 访问管理:**
  - **白名单/黑名单:** 灵活的 IP 规则管理，允许或拒绝来自特定来源的流量。
  - **匹配规则:** 基于单个IPv4和IPv6/64网段的判断。（简单起见，再以后加入ufw，firewall级别的过滤规则前不复杂化，力求高性能的数据库查询和缓存规则。
- **高级限流策略 (Rate Limiting):**
  - 使用 Redis Lua 脚本和有序集合 (ZSET) 实现了**毫秒级滑动窗口算法 (Sliding Window Algorithm)**。
  - 提供精准的流量控制（默认：60 请求/分钟），有效防止滥用和 DDoS 攻击。

### 🚀 性能优化
- **MySQL储存规则:** 统一类型BINARY(16)，version+IP索引规则
- **Redis缓存:** 缓存IP规则，可以按照需要调整黑白灰名单的缓存时间，默认黑名单1h缓存，白名单30min，灰名单5min。

### 🏗️ 清晰架构
- 遵循结构化的 **分层架构** (Handler -> Service -> Repository -> DB)。
- 关注点分离，确保了代码的可维护性和可测试性。

## 🛠️ 快速开始 (Getting Started)

### 前置要求

- [Go](https://go.dev/dl/) (版本 1.25 或更高)
- [Docker](https://www.docker.com/) & [Docker Compose](https://docs.docker.com/compose/)

### 安装步骤

1. **克隆仓库:**
   ```bash
   git clone https://github.com/StellaShiina/ktauth.git
   cd ktauth
   ```

2. **环境配置:**
   如果您计划使用邮件服务，请确保已配置必要的环境变量（如 `RESEND_API_TOKEN`, `SENDGRID_API_TOKEN`）。
   *(注意: 数据库凭据当前在 `intervnal/db/mysql.go` 中配置。在生产环境中，请务必将其外部化配置。)*

### 🚀 运行应用

#### 使用 Docker Compose (推荐)

启动 MySQL 和 Redis 依赖服务：

```bash
docker compose up -d
```

这将启动：
- **MySQL** 端口 `3306` (预配置了 `ktauth` 数据库和用户)
- **Redis** 端口 `6379`

#### 本地运行

依赖服务启动后，您可以运行应用程序：

```bash
go mod tidy
go run cmd/ktauth/main.go
```

服务器将在端口 `10000` 上启动。

## 📂 项目结构

```
ktauth/
├── cmd/                # 应用程序入口
├── init/               # 数据库初始化脚本
├── internal/
│   ├── auth/           # 认证逻辑 (JWT)
│   ├── db/             # 数据库连接 (MySQL, Redis)
│   ├── handler/        # HTTP 处理层 (Controllers)
│   ├── middleware/     # Gin 中间件 (Auth, RateLimit, IP Check)
│   ├── model/          # 数据模型
│   ├── repository/     # 数据访问层
│   ├── router/         # 路由定义
│   └── service/        # 业务逻辑层
└── pkg/                # 工具包
```
