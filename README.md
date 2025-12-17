# KTAUTH (簡単auth)

[English](./README.md) | [简体中文](./README_zh-CN.md)

**KTAUTH** acts as a robust and lightweight authentication and authorization service built with Go. It provides a secure foundation for managing user identities, controlling access based on IP addresses, and enforcing rate limits to protect your APIs.

The name "KTAUTH" is derived from "Kantan Auth" (Japanese: 簡単), meaning "Simple Auth".

## Tech Stack

- **Language:** [Go 1.25+](https://go.dev/)
- **Web Framework:** [Gin](https://github.com/gin-gonic/gin)
- **Database:** [MySQL](https://www.mysql.com/)
- **Cache & Rate Limiting:** [Redis](https://redis.io/)
- **Authentication:** [JWT (JSON Web Tokens)](https://jwt.io/)
- **Email Service:** [Resend](https://resend.com/)
- **Containerization:** Docker & Docker Compose

## Key Features

### Secure Authentication
- **JWT Implementation:** Stateless authentication using JSON Web Tokens for secure API access.
- **Session Management:** Robust session handling backed by Redis.
- **Email Verification:** Integrated email verification flow using Resend for user registration.
- **Password Security:** Secure password hashing using `bcrypt`.

### Access Control & Security
- **IP Access Management:**
  - **Whitelist/Blacklist:** Flexible IP rule management to allow or deny traffic from specific sources.
  - **CIDR Support:** Supports processing of IP ranges.
- **Advanced Rate Limiting:**
  - Implements a **Sliding Window Algorithm** using Redis Lua scripts and Sorted Sets (ZSET).
  - Provides precise traffic control (default: 60 requests/minute) to prevent abuse and DDoS attacks.

### Clean Architecture
- Follows a structured **Layered Architecture** (Handler -> Service -> Repository -> DB).
- Separation of concerns ensures maintainability and testability.

## Getting Started

### Prerequisites

- [Go](https://go.dev/dl/) (version 1.25 or later)
- [Docker](https://www.docker.com/) & [Docker Compose](https://docs.docker.com/compose/)

### Installation

1. **Clone the repository:**
   ```bash
   git clone https://github.com/StellaShiina/ktauth.git
   cd ktauth
   ```

2. **Setup Environment:**
   Ensure you have the necessary environment configuration files (`resend.env`, `sendgrid.env`) if you plan to use email services.
   *(Note: Database credentials are currently configured in `cmd/ktauth/main.go`. For production, please externalize these configurations.)*

### Running the Application

#### Using Docker Compose (Recommended)

Start the MySQL and Redis dependencies:

```bash
docker-compose up -d
```

This will spin up:
- **MySQL** on port `3306` (Pre-configured with `ktauth` database and user)
- **Redis** on port `6379`

#### Running Locally

Once the dependencies are up, you can run the application:

```bash
go mod download
go run cmd/ktauth/main.go
```

The server will start on port `10000`.

## Project Structure

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
