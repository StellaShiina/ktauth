package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/StellaShiina/ktauth/internal/db"
	"github.com/StellaShiina/ktauth/internal/handler"
	"github.com/StellaShiina/ktauth/internal/middleware"
	"github.com/StellaShiina/ktauth/internal/repository"
	"github.com/StellaShiina/ktauth/internal/router"
	"github.com/StellaShiina/ktauth/internal/service/access"
	"github.com/StellaShiina/ktauth/internal/service/admin"
	"github.com/StellaShiina/ktauth/internal/service/identity"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	logger := slog.New(
		slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelError,
		}),
	)
	slog.SetDefault(logger)

	redis, err := db.NewRedis()

	if err != nil {
		log.Fatal(err)
	} else {
		slog.Info("Connected to redis!")
	}
	defer redis.Close()

	postgres, err := connectPostgres(30 * time.Second)
	if err != nil {
		log.Fatal(err)
	} else {
		slog.Info("Connected to postgres!")
	}
	defer postgres.Close()

	// init repos
	ipRepo := repository.NewIPRepo(postgres)
	userRepo := repository.NewUserRepo(postgres)
	tokenRepo := repository.NewTokenRepo(redis)
	sessionRepo := repository.NewSessionRepo(redis)
	ipCache := repository.NewIPCache(redis)
	rateLimitRepo := repository.NewRateLimitRepo(redis)
	registerRepo := repository.NewRegisterRepo(redis)
	countDownRepo := repository.NewCountDownRepo(redis)

	// register services
	adminTokenService := admin.NewAdminTokenService(tokenRepo)
	adminIPRuleService := admin.NewAdminIPRuleService(ipRepo)
	ipAccessService := access.NewIPAccessService(ipRepo, ipCache)
	accountService := identity.NewAccountService(userRepo, registerRepo, countDownRepo)
	consumeTokenService := identity.NewConsumeTokenService(tokenRepo)
	sessionService := identity.NewSessionService(sessionRepo)
	rateLimitService := access.NewRateLimitService(rateLimitRepo)

	// register handlers
	tokenHandler := handler.NewTokenHandler(adminTokenService)
	userHandler := handler.NewUserHandler(sessionService, accountService, consumeTokenService)
	ipRuleHandler := handler.NewIPRuleHandler(adminIPRuleService)

	// register middlewares
	checkIPMiddleware := middleware.NewCheckIPMiddleware(ipAccessService)
	authMiddleWare := middleware.NewAuthMiddleWare(sessionService)
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(rateLimitService)

	r := gin.Default()

	r.SetTrustedProxies([]string{"127.0.0.0/8", "::1/128", "172.16.0.0/12", "192.168.0.0/16", "10.0.0.0/8"})

	// common route
	r.Use(checkIPMiddleware.DenyBlackList())

	// Kantan route
	// Allow non-blacklist access, set ratelimit to greylist
	r.GET("/kt/0", rateLimitMiddleware.RateLimit(), func(ctx *gin.Context) { ctx.Status(http.StatusNoContent) })
	// Only allow whitelist
	r.GET("/kt/1", checkIPMiddleware.WhiteListOnly(), func(ctx *gin.Context) { ctx.Status(http.StatusNoContent) })

	router.RegisterTokenRouter(r, tokenHandler, checkIPMiddleware)
	router.RegisterUserRouter(r, userHandler, authMiddleWare, rateLimitMiddleware)
	router.RegisterIPRouter(r, ipRuleHandler, checkIPMiddleware)

	r.Run(":10000")
}

func connectPostgres(timeout time.Duration) (*pgxpool.Pool, error) {
	var pool *pgxpool.Pool
	var err error

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	backoff := time.Second * 1

	for {
		slog.Info("Try to connect postgres...")
		pool, err = db.NewPostgres()
		if err == nil {
			return pool, nil
		}

		slog.Warn("Failed to connect postgres, will retry", "error", err, "next_retry", backoff)

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("Postgres connection timeout: %w", err)
		case <-time.After(backoff):
			backoff *= 2
			if backoff > 8*time.Second {
				backoff = 8 * time.Second
			}
		}
	}
}
