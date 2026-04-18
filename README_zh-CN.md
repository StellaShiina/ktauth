# KTAUTH (简单认证)

[English](./README.md) | [简体中文](./README_zh-CN.md)

**KTAUTH** 是一个基于 Go 语言构建的稳健且轻量级的认证与授权服务。它为管理用户身份、基于 IP 地址的访问控制以及强制执行速率限制（Rate Limiting）以保护您的 API 提供了安全的基础。

项目名称 "KTAUTH" 源自日语 "Kantan Auth" (簡単)，意为 "简单认证"。

## 🚀 技术栈 (Tech Stack)

- **开发语言:** [Go 1.25+](https://go.dev/)
- **Web 框架:** [Gin](https://github.com/gin-gonic/gin)
- **数据库:** [PostgreSQL](https://www.postgresql.org/)
- **缓存 & 限流:** [Redis](https://redis.io/)
- **认证方式:** [JWT (JSON Web Tokens)](https://jwt.io/)
- **邮件服务:** [Resend](https://resend.com/)
- **容器化:** Docker & Docker Compose
- **测试:** Github action + Go Testing

## ✨ 核心特性 (Key Features)

### ⚡ 简单快速
认证通过返回204，caddy可以直接利用forward_auth对接
- `GET /kt/0` 对黑名单限速，灰名单限流，白名单放行
- `GET /kt/1` 仅限白名单

支持容器一键部署，安装好docker compose后一行指令完成部署（请配置`resend.env`）
- `docker compose up -d`
- **ktauth image:** [stellashiina/ktauth](https://hub.docker.com/r/stellashiina/ktauth)

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
- **Redis缓存:** 缓存IP规则，可以按照需要调整黑白灰名单的缓存时间，默认黑名单1h缓存，白名单30min，灰名单5min。

### 🏗️ 清晰架构
- 遵循结构化的 **分层架构** (Handler -> Service -> Repository -> DB)。
- 关注点分离，确保了代码的可维护性和可测试性。

## 🛠️ 快速开始 (Getting Started)

> [!IMPORTANT]
> Gin内设置了TrustedProxies为全内网信任，有需要请更改相关设置
>
> 请结合前置TLS部署，推荐结合caddy部署


### docker compose快速启动

前置要求
- [Docker](https://www.docker.com/) & [Docker Compose](https://docs.docker.com/compose/)

一键安装脚本
```bash
bash <(curl -fsSL https://ktauth.kaju.win/install.sh) install
```

下载解压后启动
```bash
cp .env.example .env && docker compose up -d
```

## 📖 使用

### API 端点说明

以下列出了 KTAUTH 服务提供的所有 API 端点。

> **权限说明：** 标有 `*` 的端点受双重保护：请求者必须拥有 **管理员权限** 且 IP 必须在 **白名单** 中。

#### 🔑 核心认证端点

用于对接 Caddy 等 reverse proxy 的 `forward_auth` 指令，实现网关层面的访问控制。

| 方法 | 路径 | 描述 | 权限控制 |
| :--- | :--- | :--- | :--- |
| `GET` | `/kt/0` | **综合认证端点**<br>黑名单拒绝，非白名单限速 | 公开 |
| `GET` | `/kt/1` | **严格认证端点**<br>仅限白名单 IP 访问。 | 公开 |

- **Caddy 配置示例**
  ```Caddyfile
  example.com {
          # 替换为实际部署的端口
          forward_auth localhost:10000 {
                  uri /kt/0
          }
          # 替换为你的后端、file_server等
          reverse_proxy localhost:8080
  }
  ```
- **Nginx 示例配置**
  ```conf
  server {
      listen 443 ssl;
      server_name example.com;
      ssl_certificate /path/to/certificate.crt;
      ssl_certificate_key /path/to/private.key;

      location / {
          # 替换为非冲突路径
          auth_request /auth;

         # 替换为你的后端、file_server等
          proxy_pass http://localhost:8080;
      }

      # 鉴权子请求
      location = /auth {
          internal;

          # 替换为实际部署的端口
          proxy_pass http://localhost:10000/kt/0;

          proxy_set_header X-Original-URI $request_uri;
          proxy_set_header X-Original-Method $request_method;
          proxy_set_header X-Forwarded-For $remote_addr;

          proxy_pass_request_body off;
          proxy_set_header Content-Length "";
      }
  }
  ```

#### 👤 用户管理
处理用户生命周期与身份验证。

| 方法 | 路径 | 描述 | 权限控制 |
| :--- | :--- | :--- | :--- |
| `POST` | `/api/users/register` | 用户注册 | 非黑名单 + 限速 |
| `POST` | `/api/users/login` | 用户登录 | 非黑名单 + 限速 |
| `GET` | `/api/users/auth` | 验证登录状态 | 非黑名单 + 限速 + 用户 |
| `GET` | `/api/users/logout` | 登出当前会话 | 非黑名单 + 限速 + 用户 |
| `GET` | `/api/users` | 获取用户列表 | `*` 管理员 + 白名单 |

#### 🎫 令牌管理
用于生成和管理注册邀请码。

| 方法 | 路径 | 描述 | 权限控制 |
| :--- | :--- | :--- | :--- |
| `GET` | `/api/tokens/restock` | 批量生成可用 Token | `*` 管理员 + 白名单 |
| `DELETE`| `/api/tokens/flush` | 清空所有可用 Token | `*` 管理员 + 白名单 |
| `GET` | `/api/tokens` | 获取一个可用 Token | `*` 管理员 + 白名单 |
| `GET` | `/api/tokens/all` | 获取所有可用 Token | `*` 管理员 + 白名单 |

#### 🛡️ IP 访问控制
管理 IP 黑白名单规则。

| 方法 | 路径 | 描述 | 权限控制 |
| :--- | :--- | :--- | :--- |
| `GET` | `/api/ips` | 列出当前 IP 规则表 | `*` 管理员 + 白名单 |
| `POST` | `/api/ips/new` | 新建 IP 规则 | `*` 管理员 + 白名单 |
| `DELETE`| `/api/ips` | 删除指定 IP 规则 | `*` 管理员 + 白名单 |

## 📂 项目结构

```
ktauth/
├── cmd/                # 应用程序入口
├── init/               # 数据库初始化脚本
├── internal/
│   ├── auth/           # 认证逻辑 (JWT)
│   ├── db/             # 数据库连接 (PostgreSQL, Redis)
│   ├── handler/        # HTTP 处理层 (Controllers)
│   ├── middleware/     # Gin 中间件 (Auth, RateLimit, IP Check)
│   ├── model/          # 数据模型
│   ├── repository/     # 数据访问层
│   ├── router/         # 路由定义
│   └── service/        # 业务逻辑层
└── pkg/                # 工具包
```
