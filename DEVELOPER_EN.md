# KTAUTH Developer Guide

> **KTAUTH（簡単auth）** — A lightweight Go-based authentication and authorization gateway.
> Architecture design, module reference, development guide, and extension walkthrough for developers.

---

## Table of Contents

1. [Project Overview](#1-project-overview)
2. [Tech Stack & Dependencies](#2-tech-stack--dependencies)
3. [Architecture Design](#3-architecture-design)
4. [Project Structure Reference](#4-project-structure-reference)
5. [Core Workflows](#5-core-workflows)
6. [Configuration & Environment Variables](#6-configuration--environment-variables)
7. [Database Design](#7-database-design)
8. [API Reference](#8-api-reference)
9. [Local Development Setup](#9-local-development-setup)
10. [Testing Guide](#10-testing-guide)
11. [Build & Deployment](#11-build--deployment)
12. [CI/CD Pipeline](#12-cicd-pipeline)
13. [Extension Guide](#13-extension-guide)
14. [Code Conventions](#14-code-conventions)

---

## 1. Project Overview

KTAUTH is an authentication gateway deployed behind reverse proxies (Caddy / Nginx). Its core responsibilities are:

- **IP Access Control**: Whitelist / blacklist / greylist-based IP-level admission
- **User Authentication**: JWT + Redis Session for login / logout / authorization
- **Rate Limiting**: Millisecond-precision sliding window algorithm
- **Token Management**: Redis Set-based registration invitation code system

The service listens on port `:51214` and is designed to sit behind a reverse proxy as the `forward_auth` (Caddy) or `auth_request` (Nginx) backend target — never directly exposed to the internet.

### Design Philosophy

- **Separation of Concerns**: Strict layering `Handler → Service → Repository → DB`
- **Stateless JWT**: Combined with Redis for controllable session invalidation
- **High-Performance Caching**: IP rules cached in Redis with differentiated TTLs
- **Atomic Operations**: Rate limiting and abuse detection via Redis Lua scripts

---

## 2. Tech Stack & Dependencies

| Component | Choice | Notes |
|-----------|--------|-------|
| Language | Go 1.26+ | Module path `github.com/StellaShiina/ktauth` |
| Web Framework | [Gin v1.12](https://github.com/gin-gonic/gin) | Routing, middleware, request binding |
| Database | PostgreSQL | Accessed via [pgx v5](https://github.com/jackc/pgx) connection pool |
| Cache | Redis | Accessed via [go-redis v9](https://github.com/redis/go-redis) |
| JWT | [golang-jwt v5](https://github.com/golang-jwt/jwt) | HS256 signing, 7-day expiry |
| Password | [bcrypt](https://pkg.go.dev/golang.org/x/crypto/bcrypt) | DefaultCost hashing |
| UUID | [google/uuid](https://github.com/google/uuid) | v4 random UUID |

### Full Dependency Tree (go.mod)

```
github.com/gin-gonic/gin          # Web framework
github.com/golang-jwt/jwt/v5      # JWT sign / verify
github.com/google/uuid            # UUID generation
github.com/jackc/pgx/v5           # PostgreSQL driver + pool
github.com/redis/go-redis/v9      # Redis client
golang.org/x/crypto               # bcrypt password hashing
```

---

## 3. Architecture Design

### 3.1 Layered Architecture

```
┌─────────────────────────────────────────┐
│               HTTP Request               │
└─────────────────┬───────────────────────┘
                  │
    ┌─────────────▼─────────────┐
    │        Middleware          │  ← Request interception
    │  (CheckIP / Auth / Rate)   │
    └─────────────┬─────────────┘
                  │
    ┌─────────────▼─────────────┐
    │         Handler            │  ← Controller layer
    │  Binding / Response        │
    └─────────────┬─────────────┘
                  │
    ┌─────────────▼─────────────┐
    │         Service            │  ← Business logic
    │  Orchestrates Repositories │
    └─────────────┬─────────────┘
                  │
    ┌─────────────▼─────────────┐
    │        Repository          │  ← Data access layer
    │  SQL / Redis commands      │
    └─────────────┬─────────────┘
                  │
    ┌─────────────▼─────────────┐
    │     PostgreSQL / Redis     │  ← Storage layer
    └───────────────────────────┘
```

### 3.2 Dependency Injection

All dependencies are wired manually in `cmd/ktauth/main.go` (no DI framework):

```go
// 1. Initialize connections
redis    := db.NewRedis()
postgres := connectPostgres(30 * time.Second)

// 2. Initialize Repositories
ipRepo    := repository.NewIPRepo(postgres)
userRepo  := repository.NewUserRepo(postgres)
tokenRepo := repository.NewTokenRepo(redis)
// ...

// 3. Initialize Services (inject Repositories)
ipAccessService := access.NewIPAccessService(ipRepo, ipCache)
accountService  := identity.NewAccountService(userRepo)
// ...

// 4. Initialize Middleware (inject Services)
checkIPMiddleware := middleware.NewCheckIPMiddleware(ipAccessService)

// 5. Initialize Handlers (inject Services)
userHandler := handler.NewUserHandler(sessionService, accountService, consumeTokenService)

// 6. Register routes
router.RegisterUserRouter(r, userHandler, checkIPMiddleware, authMiddleWare, rateLimitMiddleware)
```

### 3.3 Request Pipeline

Example: `POST /api/users/login`

```
Request
  │
  ▼
[CheckIP Middleware]  ─── Query IP rule (cache-first), blacklist → 403
  │
  ▼
[RateLimit Middleware] ─── Sliding window check, whitelist IPs auto-skip
  │
  ▼
[UserHandler.LoginUser]
  │
  ├─► AccountService.GetUserByName()   ──► UserRepo.GetUserByName()  ──► PostgreSQL
  ├─► crypto.VerifyPassword()         (bcrypt comparison)
  ├─► auth.SignToken()                (JWT issuance)
  └─► SessionService.CreateSession()  ──► SessionRepo.CreateSession() ──► Redis
  │
  ▼
Response (200 + JWT token)
```

---

## 4. Project Structure Reference

```
ktauth/
├── cmd/ktauth/main.go              # ★ Entry point: DI + route registration
├── internal/
│   ├── auth/jwt.go                 # JWT signing (HS256) & parsing
│   ├── crypto/
│   │   ├── password.go             # bcrypt hash & verify
│   │   └── rand.go                 # Crypto-safe random digit generation
│   ├── db/
│   │   ├── postgres.go             # PostgreSQL pool (pgxpool)
│   │   └── redis.go                # Redis client init
│   ├── handler/
│   │   ├── admin_handler.go        # IP rule management + user list handlers
│   │   ├── user_handler.go         # Register / login / logout handlers
│   │   └── token_handler.go        # Token management handlers
│   ├── middleware/
│   │   ├── auth.go                 # JWT session verification middleware
│   │   ├── checkip.go              # IP whitelist/blacklist/greylist ACL middleware
│   │   └── ratelimit.go            # Rate limiting + abuse auto-ban middleware
│   ├── model/
│   │   ├── ip.go                   # IP rule data model + type constants
│   │   └── user.go                 # User data model
│   ├── repository/
│   │   ├── countdown_repo.go       # Cooldown repository (Redis)
│   │   ├── ip_repo.go              # IP rule CRUD (PostgreSQL)
│   │   ├── iprule_cache.go         # IP rule cache (Redis, with differentiated TTLs)
│   │   ├── ratelimit_repo.go       # Sliding window rate limit (Redis Lua)
│   │   ├── register_repo.go        # Registration verification code repo (Redis)
│   │   ├── session_repo.go         # JWT session repository (Redis)
│   │   ├── token_repo.go           # Invitation token repository (Redis Set)
│   │   └── user_repo.go            # User CRUD (PostgreSQL)
│   ├── router/
│   │   ├── admin_router.go         # /api/ips + /api/users (admin) routes
│   │   ├── token_router.go         # /api/tokens routes
│   │   └── user_router.go          # /api/users routes
│   └── service/
│       ├── access/
│       │   ├── cd.go               # Cooldown service
│       │   ├── ip.go               # IP rule query service (cache-first)
│       │   └── ratelimit.go        # Rate limit & abuse detection service
│       ├── admin/
│       │   ├── manage_iprule.go    # IP rule admin service
│       │   ├── manage_token.go     # Token admin service
│       │   ├── manage_user.go      # User admin service
│       │   └── types.go            # API response type definitions
│       └── identity/
│           ├── account.go          # Account service (create/query/update users)
│           ├── consume_token.go    # Token consumption service
│           └── session.go          # Session service (create/delete/verify)
├── pkg/iputils/processip.go        # IP address parsing + CIDR normalization
├── sql/
│   ├── 00-init.sql                 # Schema + seed data (admin user + private IP whitelist)
│   └── 10-ipdata.sql               # Additional preset IP whitelist data
├── scripts/install.sh              # One-click deployment script
├── docker-compose.yaml             # Production stack (ktauth + postgres + redis)
├── docker-compose.db.yaml          # Database only (local development)
├── docker-compose.test.yaml        # Test stack (locally built image)
├── .env.example                    # Environment variable template
└── .github/workflows/ci.yaml       # CI/CD: test + release + Docker build
```

---

## 5. Core Workflows

### 5.1 IP Access Control

```
Client IP → IPAccessService.QueryRule()
              │
              ├─► iputils.ProcessIP(ipStr)
              │     Single IP → /32 (IPv4) or /64 (IPv6) mask
              │     CIDR → kept as-is
              │
              ├─► IPCache.Get(ipNet)            ← Redis cache lookup
              │     Hit  → return cached rule type
              │     Miss → continue
              │
              ├─► IPRepo.QueryIP(version, ip)    ← PostgreSQL lookup
              │     Found    → write back to Redis cache
              │     Not Found → treat as greylist, cache (5min TTL)
              │
              └─► Return IPRuleType (whitelist/blacklist/greylist)
```

**Cache TTL Strategy:**

| Rule Type | TTL | Rationale |
|-----------|-----|-----------|
| Blacklist | 1 hour | Needs fast rejection, rarely changes |
| Whitelist | 30 minutes | Needs fast allowance |
| Greylist | 5 minutes | Default state, likely to change |

### 5.2 JWT Authentication

```
Login Request
  │
  ├─► AccountService.GetUserByName()  Query user
  ├─► crypto.VerifyPassword()        Verify password
  ├─► auth.SignToken(uuid, name, role)
  │     └─► Generate JWT (HS256, 7-day expiry)
  │         Claims: { UUID, Name, Role, jti, exp, iat, iss }
  │
  └─► SessionService.CreateSession(uuid, jti)
        └─► Redis SET "jwt:active:{uuid}:{jti}" = uuid (144h TTL)

Subsequent Requests
  │
  ├─► AuthMiddleWare.VerifySession()
  │     ├─► Extract Authorization: Bearer <token>
  │     ├─► auth.ParseToken() parse JWT
  │     └─► SessionService.GetSession(uuid, jti)  Redis verification
  │          Exists → authenticated, set ctx uuid/jti
  │          Missing → 401 (session invalidated / logged out)
```

### 5.3 Sliding Window Rate Limiting

Implemented in `ratelimit_repo.go` via Lua script:

```lua
-- 1. Remove entries outside the sliding window
ZREMRANGEBYSCORE key '-inf' (now - window)

-- 2. Count requests in current window
count = ZCARD key

-- 3. Decision
if count < limit then
    ZADD key now member    -- Record this request (score=timestamp, member=UUID)
    PEXPIRE key window     -- Refresh key TTL
    return 1               -- Allowed
else
    return 0               -- Denied
end
```

**Characteristics:**
- Uses Redis Sorted Set (ZSET), score = millisecond timestamp
- Member is a random UUID to prevent overwrites from same-millisecond requests
- Atomic execution, no distributed lock required
- Default: 60 req/min, configurable via `.env`

### 5.4 Abuse Auto-Ban

When a request is rate-limited (429), the middleware additionally checks for abuse:

```go
// ratelimit.go middleware
if !allow {
    c.String(http.StatusTooManyRequests, "Rate limit exceed!")
    // Check abuse
    if abuse, err := m.rateLimitService.Abuse(ctx, ip); err == nil && abuse {
        // Auto-add to blacklist
        m.adminIPRuleService.AddRule(ctx, ip, false, &note)
    }
}
```

Abuse detection uses Redis INCR + EXPIRE:
- Key: `abuse:429:{cidr}`
- Default: 100 × 429 within 5 minutes → triggers auto-ban
- After trigger, the counter key is automatically deleted; next detection starts fresh

### 5.5 Token Invitation System

```
Registration Flow:
  POST /api/users/register { token, user, password }
    │
    ├─► ConsumeTokenService.Consume(token)
    │     └─► Redis SREM "admin:tokens" token
    │           Success (n > 0) → Token valid, consumed
    │           Failure (n = 0) → Token invalid or already used
    │
    └─► AccountService.NewUser() → PostgreSQL INSERT

Admin Operations:
  GET   /api/tokens/restock  → Bulk-generate 10 UUID tokens into Redis Set
  GET   /api/tokens          → Randomly fetch one unused token
  GET   /api/tokens/all      → List all available tokens
  DELETE /api/tokens/flush    → Clear all tokens
```

---

## 6. Configuration & Environment Variables

### 6.1 Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `ADMIN_NAME` | `admin` | Admin username |
| `ADMIN_PASSWD` | `admin` | Admin password (stored as bcrypt hash) |
| `JWT_SECRET` | `ktauthsecret` | JWT signing key (HS256) |
| `RATELIMIT` | `60` | Allowed requests per minute |
| `ENABLE_RATELIMIT` | (empty = enabled) | Set to `NO` to disable rate limiting |
| `ABUSELIMIT` | `100` | 429 count threshold for auto-ban |
| `ABUSEWINDOW` | `5` | Abuse detection window (minutes) |
| `LOGLEVEL` | `error` | Log level: `debug` / `info` / `warn` / `error` |
| `REDIS_HOST` | `127.0.0.1` | Redis host |
| `POSTGRES_HOST` | `127.0.0.1` | PostgreSQL host |
| `POSTGRES_PORT` | `5432` | PostgreSQL port |
| `GIN_MODE` | - | Gin mode (set `release` for production) |

### 6.2 Hardcoded Constants

The following are currently hardcoded; changes require modifying source:

| Location | Constant | Value | Description |
|----------|----------|-------|-------------|
| `auth/jwt.go` | Token expiry | 168 hours (7 days) | JWT exp |
| `cmd/ktauth/main.go` | Listen port | `:51214` | Gin Run address |
| `db/postgres.go` | DB credentials | `ktauth:ktauth` | PostgreSQL user/pass/dbname |
| `session_repo.go` | Session TTL | 144 hours (6 days) | Redis key expiry |
| `ratelimit_repo.go` | Sliding window | 1 minute | Rate limit window size |

---

## 7. Database Design

### 7.1 PostgreSQL Schema

#### `users` Table

```sql
CREATE TABLE users (
    uuid          UUID PRIMARY KEY,              -- Unique user identifier
    name          VARCHAR(64) NOT NULL UNIQUE,    -- Username
    password_hash CHAR(60) NOT NULL,              -- bcrypt hash (fixed 60 chars)
    email         VARCHAR(255) UNIQUE,            -- Email (optional)
    role          VARCHAR(32) NOT NULL DEFAULT 'user' -- Role: user / admin
);
```

**Built-in admin:**
- UUID: `00000000-0000-0000-0000-000000000000`
- Default password: `admin` (bcrypt hashed)

#### `ip` Table

```sql
CREATE TABLE ip (
    id           BIGSERIAL PRIMARY KEY,          -- Auto-increment ID
    version      SMALLINT NOT NULL,               -- IP version: 4 or 6
    ip_range     CIDR NOT NULL UNIQUE,            -- IP/CIDR range (PostgreSQL CIDR type)
    is_whitelist BOOLEAN NOT NULL,                -- true=whitelist, false=blacklist
    create_at    TIMESTAMPTZ DEFAULT NOW(),       -- Created at
    update_at    TIMESTAMPTZ DEFAULT NOW(),       -- Updated at (auto-maintained by trigger)
    note         TEXT                              -- Remark
);
```

**IP matching uses PostgreSQL CIDR containment operator `<<=`**:
```sql
SELECT is_whitelist FROM ip
WHERE version = $1 AND $2::inet <<= ip_range
```

**Built-in rules (00-init.sql):**
- `127.0.0.0/8` — localhost whitelist
- `10.0.0.0/8` — Class A private whitelist
- `192.168.0.0/16` — Class C private whitelist
- `172.16.0.0/12` — Class B private whitelist

### 7.2 Redis Data Structures

| Key Pattern | Type | Value | TTL | Purpose |
|-------------|------|-------|-----|---------|
| `jwt:active:{uuid}:{jti}` | String | uuid | 144h | JWT session |
| `rule:ip:{cidr}` | String | "whitelist" / "blacklist" / "greylist" | 30min / 1h / 5min | IP rule cache |
| `ratelimit:ip:{cidr}` | ZSET | member(UUID) → score(ms) | window size | Sliding window counter |
| `abuse:429:{cidr}` | String (counter) | count | ABUSEWINDOW | Abuse detection |
| `admin:tokens` | Set | UUID strings | ∞ | Registration invitation pool |
| `register:{email}:{code}` | String | "" | 15min | Email verification code (reserved) |

---

## 8. API Reference

### 8.1 Core Auth Endpoints (for Reverse Proxy)

| Method | Path | Description | Response |
|--------|------|-------------|----------|
| `GET` | `/kt/0` | Comprehensive auth: deny blacklist, rate-limit non-whitelist | `204 No Content` / `403 Forbidden` / `429 Too Many Requests` |
| `GET` | `/kt/1` | Strict auth: whitelist only | `204 No Content` / `403 Forbidden` |

### 8.2 User Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `POST` | `/api/users/register` | Token | Register (requires invitation token) |
| `POST` | `/api/users/login` | None | Login, returns JWT |
| `GET` | `/api/users/auth` | Bearer JWT | Verify session, `204` = valid |
| `GET` | `/api/users/logout` | Bearer JWT | Logout current session |
| `GET` | `/api/users` | Admin + whitelist | List all users |

**Register request body:**
```json
{
  "token": "uuid-token",
  "user": "username",
  "password": "password",
  "email": "optional@example.com"
}
```

**Login request body:**
```json
{
  "user": "username",
  "password": "password"
}
```

**Login response:**
```json
{ "token": "eyJhbGciOi..." }
```
Or `?format=string` → plain text token

### 8.3 Token Management Endpoints (Admin)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/tokens/restock` | Bulk-generate 10 tokens |
| `DELETE` | `/api/tokens/flush` | Clear all tokens |
| `GET` | `/api/tokens` | Randomly fetch one token |
| `GET` | `/api/tokens/all` | List all available tokens |

### 8.4 IP Rule Management Endpoints (Admin)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/ips` | List IP rules (supports `?version=4/6&type=white/black` filters) |
| `POST` | `/api/ips/new` | Add IP rule |
| `DELETE` | `/api/ips` | Delete IP rule |

**Add rule request body:**
```json
{
  "ip": "192.168.1.0/24",
  "isWhiteList": true,
  "note": "office network"
}
```

---

## 9. Local Development Setup

### 9.1 Prerequisites

- Go 1.26+
- Docker & Docker Compose (for PostgreSQL and Redis)

### 9.2 Quick Start

```bash
# 1. Clone the repo
git clone https://github.com/StellaShiina/ktauth.git
cd ktauth

# 2. Start databases only (PostgreSQL + Redis)
docker compose -f docker-compose.db.yaml up -d

# 3. Copy environment config
cp .env.example .env
# Edit .env as needed

# 4. Run the app
go run ./cmd/ktauth

# Or build then run
go build -o ktauth ./cmd/ktauth
./ktauth
```

### 9.3 Suggested Dev Workflow

```bash
# Run all tests
go test ./...

# Run specific package tests
go test ./internal/repository/ -v
go test ./pkg/iputils/ -v

# With race detector
go test -race ./...

# Format code
go fmt ./...

# Dependency management
go mod tidy
go mod verify
```

---

## 10. Testing Guide

### 10.1 Test File Organization

```
internal/db/postgres_test.go                # DB connection test
internal/repository/ip_repo_test.go         # IP Repository tests
internal/repository/user_repo_test.go       # User Repository tests
internal/service/admin/manage_iprule_test.go # IP admin service tests
pkg/iputils/processip_test.go               # IP utility tests
```

### 10.2 Running Tests

Tests require real PostgreSQL and Redis instances:

```bash
# Start test databases
docker compose -f docker-compose.db.yaml up -d

# Run all tests
go test ./... -v

# CI uses the same approach:
# docker compose -f ./docker-compose.db.yaml up -d
# go test ./...
```

### 10.3 Test Coverage

| Test File | Coverage |
|-----------|----------|
| `processip_test.go` | IPv4/IPv6 parsing, CIDR normalization, invalid input |
| `postgres_test.go` | Database connection verification |
| `ip_repo_test.go` | IP rule CRUD, duplicate detection, not-found detection |
| `user_repo_test.go` | User CRUD, duplicate detection, not-found detection |
| `manage_iprule_test.go` | IP admin service (Add/Del/List) + cache invalidation |

### 10.4 Writing New Tests

Conventions:
- Test package name: `{package}_test` (black-box testing)
- Test function name: `Test{FunctionName}`
- Use standard `testing` package; no third-party assertion libraries
- Clean up data after tests (see the `DelUser` cleanup pattern in `user_repo_test.go`)

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

## 11. Build & Deployment

### 11.1 Local Build

```bash
# Development build
go build -o ktauth ./cmd/ktauth

# Production build (Linux cross-compile)
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ktauth ./cmd/ktauth
```

### 11.2 Docker Deployment

**Production (pull image from Docker Hub):**

```bash
cp .env.example .env
# Edit .env configuration
docker compose up -d
```

**Testing (local image build):**

```bash
# Build local image first
docker build -t ktauth:test .

# Use test compose file
docker compose -f docker-compose.test.yaml up -d
```

### 11.3 Caddy Integration

```caddyfile
example.com {
    forward_auth localhost:51214 {
        uri /kt/0
    }
    reverse_proxy localhost:8080
}
```

### 11.4 Nginx Integration

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

## 12. CI/CD Pipeline

Defined in `.github/workflows/ci.yaml`. Triggers:

- **Push to `main` branch** → run tests
- **Push `v*` tag** → tests → create Release + build Docker image

### Pipeline Steps

```
┌──────────────────┐
│   go-test Job    │
│  1. Start DB     │
│  2. Install Go   │
│  3. go test ./.. │
└────────┬─────────┘
         │ (tag only)
    ┌────┴────┐
    │         │
    ▼         ▼
┌────────┐ ┌──────────┐
│ Release│ │  Docker  │
│ Zip    │ │ Build+Push│
└────────┘ └──────────┘
```

**Release artifacts:**
- `ktauth_{version}.zip` (contains `docker-compose.yaml`, `00-init.sql`, `.env.example`)
- `install.sh` script

**Docker image tags:**
- `stellashiina/ktauth:{version}`
- `stellashiina/ktauth:latest`

---

## 13. Extension Guide

### 13.1 Adding a New API Endpoint

1. **Define Handler** → `internal/handler/`
2. **Define Service (if business logic needed)** → `internal/service/`
3. **Define Repository (if data access needed)** → `internal/repository/`
4. **Register route** → `internal/router/`
5. **Wire dependencies** → `cmd/ktauth/main.go`

**Example: Adding a "Reset Password" endpoint**

```go
// internal/handler/user_handler.go — add method
func (h *UserHandler) ResetPassword(c *gin.Context) {
    // Get current user UUID (injected by middleware)
    uuid := c.GetString("uuid")
    // Parse request body...
    // Call service...
}

// internal/router/user_router.go — register route
user.POST("/reset-password", h.ResetPassword)
```

### 13.2 Adding a New Middleware

Middleware follows Gin's `gin.HandlerFunc` signature:

```go
func NewMyMiddleware(dep *SomeDependency) *MyMiddleware {
    return &MyMiddleware{dep}
}

func (m *MyMiddleware) Handle() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Pre-processing
        // ...

        c.Next()

        // Post-processing (optional)
        // ...
    }
}
```

Wire it in `main.go` and apply:

```go
myMiddleware := middleware.NewMyMiddleware(dep)
r.Use(myMiddleware.Handle())  // Global
// or
g := r.Group("/api/xxx", myMiddleware.Handle())  // Scoped
```

### 13.3 Adding a New Config Option

1. Add the variable to `.env.example`
2. Read it in `cmd/ktauth/main.go` (`os.Getenv()`)
3. Pass it to the Service/Middleware that needs it
4. Update this document's configuration section

### 13.4 Enabling Email Verification

The project already has `RegisterRepo` (email verification code storage) reserved. To activate:

1. Uncomment related code in `handler/user_handler.go`'s `RegisterUser`
2. Add an email sending Service (using Resend / SendGrid API)
3. Inject `RegisterRepo` and related services in `main.go`
4. Add `/api/users/send-code` and `/api/users/verify-code` endpoints

---

## 14. Code Conventions

### 14.1 Naming

| Kind | Convention | Example |
|------|-----------|---------|
| Package | lowercase, short | `iputils`, `access`, `identity` |
| File | snake_case | `user_repo.go`, `manage_iprule.go` |
| Struct | PascalCase | `IPAccessService`, `RateLimitRepo` |
| Method | PascalCase (exported) / camelCase (unexported) | `QueryRule()`, `connectPostgres()` |
| Variable | camelCase | `ipRepo`, `isWhitelist` |
| Constant | PascalCase or UPPER_SNAKE | `IPWhiteList`, `ErrIPNotFound` |
| Error var | `Err` prefix | `ErrUserNotFound`, `ErrIPExist` |

### 14.2 Project Standards

- **Error handling**: Repository layer defines sentinel errors (`var ErrXxx = errors.New(...)`), Service layer propagates, Handler layer uses `errors.As()` to branch
- **Logging**: Use `log/slog` standard library, not `fmt.Println`
- **Context propagation**: All data access methods take `context.Context` as first parameter
- **SQL safety**: Always use parameterized queries (`$1`, `$2`), never concatenate SQL
- **Redis key naming**: `{domain}:{sub}:{identifier}` format (e.g., `jwt:active:{uuid}:{jti}`)

### 14.3 Import Order

```go
import (
    // 1. Standard library
    "context"
    "fmt"

    // 2. Project packages
    "github.com/StellaShiina/ktauth/internal/model"
    "github.com/StellaShiina/ktauth/internal/repository"

    // 3. Third-party
    "github.com/gin-gonic/gin"
    "github.com/redis/go-redis/v9"
)
```

## 15. Roadmap

- [ ] Implement SMTP email verification code sending
- [ ] Optimize session management
- [ ] Administrator web panel

---

## Appendix: Quick Reference

### Common Commands

```bash
# Development
go run ./cmd/ktauth                                   # Start the service
go test ./... -v                                      # Run all tests
docker compose -f docker-compose.db.yaml up -d        # Start dev databases

# Build
go build -o ktauth ./cmd/ktauth                       # Local build
GOOS=linux GOARCH=amd64 go build -o ktauth ./cmd/ktauth  # Cross-compile

# Deploy
docker compose up -d                                  # Start full stack
docker compose logs -f ktauth                         # View logs
docker compose restart ktauth                         # Restart service
```

### Default Ports

| Service | Port |
|---------|------|
| KTAUTH API | 51214 |
| PostgreSQL | 5432 |
| Redis | 6379 |

### Repository Info

- **Module path**: `github.com/StellaShiina/ktauth`
- **Docker image**: `stellashiina/ktauth`
- **Install script**: `https://ktauth.kaju.win`
