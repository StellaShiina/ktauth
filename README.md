# KTAUTH (簡単auth)

[English](./README.md) | [简体中文](./README_zh-CN.md)

**KTAUTH** acts as a robust and lightweight authentication and authorization service built with Go. It provides a secure foundation for managing user identities, controlling access based on IP addresses, and enforcing rate limits to protect your APIs.

The name "KTAUTH" is derived from "Kantan Auth" (Japanese: 簡単), meaning "Simple Auth".

## 🚀 Tech Stack

- **Language:** [Go 1.25+](https://go.dev/)
- **Web Framework:** [Gin](https://github.com/gin-gonic/gin)
- **Database:** [MySQL](https://www.mysql.com/)
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
- **MySQL Storage:** Unified `BINARY(16)` type with `version+IP` indexing.
- **Redis Caching:** IP rules are cached with adjustable TTL (Default: Blacklist 1h, Whitelist 30min, Greylist 5min).

### 🏗️ Clean Architecture
- Follows a structured **Layered Architecture** (Handler -> Service -> Repository -> DB).
- Separation of concerns ensures maintainability and testability.

## 🛠️ Getting Started

> [!IMPORTANT]
> Gin is configured with `TrustedProxies` set to trust all internal networks. Please adjust this setting if necessary.
>
> It is recommended to deploy with TLS, preferably using Caddy.

> [!TIP]
> Two deployment methods are supported:

### Method 1: Docker Compose (Recommended)

**Prerequisites**
- [Docker](https://www.docker.com/) & [Docker Compose](https://docs.docker.com/compose/)

**Start**
```bash
cp .env.example .env && docker compose up -d
```

### Method 2: Local Go + Docker Compose

**Prerequisites**
- [Go](https://go.dev/dl/) (version 1.25 or later)
- [Docker](https://www.docker.com/) & [Docker Compose](https://docs.docker.com/compose/)

**Installation**

1. **Clone the repository:**
   ```bash
   git clone https://github.com/StellaShiina/ktauth.git
   cd ktauth
   ```

2. **Setup Environment:**
   Ensure you have the necessary environment variables (e.g., `RESEND_API_TOKEN`, `SENDGRID_API_TOKEN`) if you plan to use email services.
   *(Note: Database credentials are currently configured in `internal/db/mysql.go`. For production, please externalize these configurations.)*

**Running the Application**

Start the MySQL and Redis dependencies:

```bash
docker compose -f ./docker-compose.db.yaml up -d
```

This will spin up:
- **MySQL** on port `3306` (Pre-configured with `ktauth` database and user)
- **Redis** on port `6379`

Once the dependencies are up, you can run the application:

```bash
go mod tidy
go run cmd/ktauth/main.go
```

The server will start on port `10000`.

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
