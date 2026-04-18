# KTAUTH (簡単auth)

[English](./README.md) | [简体中文](./README_zh-CN.md)

**KTAUTH** acts as a robust and lightweight authentication and authorization service built with Go. It provides a secure foundation for managing user identities, controlling access based on IP addresses, and enforcing rate limits to protect your APIs.

The name "KTAUTH" is derived from "Kantan Auth" (Japanese: 簡単), meaning "Simple Auth".

## 🚀 Tech Stack

- **Language:** [Go 1.25+](https://go.dev/)
- **Web Framework:** [Gin](https://github.com/gin-gonic/gin)
- **Database:** [PostgreSQL](https://www.postgresql.org/)
- **Cache & Rate Limiting:** [Redis](https://redis.io/)
- **Authentication:** [JWT (JSON Web Tokens)](https://jwt.io/)
- **Email Service:** [Resend](https://resend.com/)
- **Containerization:** Docker & Docker Compose
- **Test:** Github Action + Go Testing

## ✨ Key Features

### ⚡ Simple & Fast
- **Caddy Ready:** Returns `204` on success, compatible with Caddy `forward_auth`.
- **Flexible Endpoints:**
  - `GET /kt/0`: Rate limits blacklist/greylist, allows whitelist.
  - `GET /kt/1`: Whitelist access only.
- **One-Command Deployment:** Support Docker Compose one-click deployment.(Please configure `resend.env`)
- `docker compose up -d`
- **ktauth image:** [stellashiina/ktauth](https://hub.docker.com/r/stellashiina/ktauth)

### 🔐 Secure Authentication
- **JWT Implementation:** Stateless authentication using JSON Web Tokens for secure API access.
- **Session Management:** Robust session handling backed by Redis.
- **Email Verification:** Integrated email verification flow using Resend for user registration.
- **Password Security:** Secure password hashing using `bcrypt`.

### 🛡️ Access Control & Security
- **IP Access Management:**
  - **Whitelist/Blacklist:** Flexible IP rule management to allow or deny traffic from specific sources.
  - **Matching Rules:** Based on single IPv4 and IPv6/64 subnets. Designed for high performance, keeping complex firewall-level filtering separate.
- **Advanced Rate Limiting:**
  - Implements a **Millisecond-level Sliding Window Algorithm** using Redis Lua scripts and Sorted Sets (ZSET).
  - Provides precise traffic control (default: 60 requests/minute) to prevent abuse and DDoS attacks.

### 🚀 Performance Optimization
- **Redis Caching:** IP rules are cached with adjustable TTL (Default: Blacklist 1h, Whitelist 30min, Greylist 5min).

### 🏗️ Clean Architecture
- Follows a structured **Layered Architecture** (Handler -> Service -> Repository -> DB).
- Separation of concerns ensures maintainability and testability.

## 🛠️ Getting Started

> [!IMPORTANT]
> Gin is configured with `TrustedProxies` set to trust all internal networks. Please adjust this setting if necessary.
>
> It is recommended to deploy with TLS, preferably using Caddy.


### Docker Compose Quick Start

**Prerequisites**
- [Docker](https://www.docker.com/) & [Docker Compose](https://docs.docker.com/compose/)

One-click installation script
```bash
bash <(curl -fsSL https://ktauth.kaju.win) install
```

Start after downloading and extracting.
```bash
cp .env.example .env && docker compose up -d
```

## 📖 Usage

### API Endpoints

The following table lists all API endpoints provided by the KTAUTH service.

> **Permission Note:** Endpoints marked with `*` are protected by dual mechanisms: the requester must possess **Administrator Privileges** and the IP address must be on the **Whitelist**.

#### 🔑 Core Authentication Endpoints

These endpoints are designed to integrate with the `forward_auth` directive in reverse proxies like Caddy to enforce access control at the gateway level.

| Method | Path | Description | Permission Control |
| :--- | :--- | :--- | :--- |
| `GET` | `/kt/0` | **Comprehensive Authentication Endpoint**<br>Rejects Blacklist entries, Rate-limits non-Whitelist entries. | Public |
| `GET` | `/kt/1` | **Strict Authentication Endpoint**<br>Access restricted to Whitelisted IPs only. | Public |

- **Caddy Configuration Example**
  ```Caddyfile
  example.com {
          # Replace with the actual deployment port
          forward_auth localhost:10000 {
                  uri /kt/0
          }
          # Replace with your backend, file_server, etc.
          reverse_proxy localhost:8080
  }
  ```
- **Nginx Example Configuration**
  ```conf
  server {
      listen 443 ssl;
      server_name example.com;
      ssl_certificate /path/to/certificate.crt;
      ssl_certificate_key /path/to/private.key;

      location / {
          # Replace with a non-conflicting path
          auth_request /auth;

          # Replace with your backend, file_server, etc.
          proxy_pass http://localhost:8080;
      }

      # Authentication sub-request
      location = /auth {
          internal;

          # Replace with the actual deployment port
          proxy_pass http://localhost:10000/kt/0;

          proxy_set_header X-Original-URI  $ request_uri;
          proxy_set_header X-Original-Method  $ request_method;
          proxy_set_header X-Forwarded-For  $ remote_addr;

          proxy_pass_request_body off;
          proxy_set_header Content-Length "";
      }
  }
  ```

#### 👤 User Management
Handles user lifecycle and authentication.

| Method | Path | Description | Access Control |
| :--- | :--- | :--- | :--- |
| `POST` | `/api/users/register` | User Registration | Non-Blacklist + Rate Limiting |
| `POST` | `/api/users/login` | User Login | Non-Blacklist + Rate Limiting |
| `GET` | `/api/users/auth` | Verify Login Status | Non-Blacklist + Rate Limiting + User |
| `GET` | `/api/users/logout` | Logout of Current Session | Non-Blacklist + Rate Limiting + User |
| `GET` | `/api/users` | Retrieve User List | `*` Administrator + Whitelist |

#### 🎫 Token Management
Used to generate and manage registration invitation codes.

| Method | Path | Description | Access Control |
| :--- | :--- | :--- | :--- |
| `GET` | `/api/tokens/restock` | Batch generate usable tokens | `*` Administrator + Whitelist |
| `DELETE` | `/api/tokens/flush` | Clear all usable tokens | `*` Administrator + Whitelist |
| `GET` | `/api/tokens` | Get a usable token | `*` Administrator + Whitelist |
| `GET` | `/api/tokens/all` | Get all usable tokens | `*` Administrator + Whitelist |

#### 🛡️ IP Access Control
Manage IP blacklist/whitelist rules.

| Method | Path | Description | Access Control |
| :--- | :--- | :--- | :--- |
| `GET` | `/api/ips` | List current IP rule table | `*` Administrator + Whitelist |
| `POST` | `/api/ips/new` | Create new IP rule | `*` Administrator + Whitelist |
| `DELETE` | `/api/ips` | Delete specified IP rule | `*` Administrator + Whitelist |

## 📂 Project Structure

```
ktauth/
├── cmd/                # Application entry points
├── init/               # Database initialization scripts
├── internal/
│   ├── auth/           # Authentication logic (JWT)
│   ├── db/             # Database connections (MySQL, Redis)
│   ├── handler/        # HTTP Handlers (Controllers)
│   ├── middleware/     # Gin Middlewares (Auth, RateLimit, IP Check)
│   ├── model/          # Data models
│   ├── repository/     # Data access layer
│   ├── router/         # Route definitions
│   └── service/        # Business logic
└── pkg/                # Utility packages
```
