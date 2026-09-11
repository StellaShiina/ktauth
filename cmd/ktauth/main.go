package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
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
	var ratelimit int
	var enableRatelimit bool
	var logLevel slog.Leveler
	var abuseLimit int
	var abuseWindow time.Duration
	var trustedProxies []string

	// Set up logger
	logLevelstr := strings.TrimSpace(strings.ToLower(os.Getenv("LOGLEVEL")))

	switch logLevelstr {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelWarn
	}

	logger := slog.New(
		slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: logLevel,
		}),
	)
	slog.SetDefault(logger)

	// Route gin's internal output (debug prints, panic traces) through slog.
	gin.DefaultWriter = middleware.NewSlogWriter(slog.LevelInfo)
	gin.DefaultErrorWriter = middleware.NewSlogWriter(slog.LevelError)

	// Configure rate limit
	ratelimit, err := strconv.Atoi(os.Getenv("RATELIMIT"))
	if err != nil {
		slog.Warn("no available ratelimit conf, use default 60/min")
		ratelimit = 60
	}

	if os.Getenv("ENABLE_RATELIMIT") == "NO" {
		slog.Warn("ratelimit is inactive")
		enableRatelimit = false
	} else {
		slog.Info("ratelimit is active")
		enableRatelimit = true
	}

	abuseLimit, err = strconv.Atoi(os.Getenv("ABUSELIMIT"))
	if err != nil {
		abuseLimit = 100
	}

	abuseWindowMin, err := strconv.Atoi(os.Getenv("ABUSEWINDOW"))
	if err != nil {
		abuseWindow = 5 * time.Minute
	} else {
		abuseWindow = time.Duration(abuseWindowMin) * time.Minute
	}

	slog.Info("abuse settings", "abuseLimit", abuseLimit, "abuseWindow", abuseWindow)

	redis, err := db.NewRedis()

	if err != nil {
		slog.Error("failed to connect to redis", "error", err)
	} else {
		slog.Info("connected to redis")
	}
	defer redis.Close()

	postgres, err := connectPostgres(30 * time.Second)
	if err != nil {
		slog.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	} else {
		slog.Info("connected to postgres")
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
	countdownRepo := repository.NewCountDownRepo(redis)

	// register services
	adminTokenService := admin.NewAdminTokenService(tokenRepo)
	adminIPRuleService := admin.NewAdminIPRuleService(ipRepo, ipCache, rateLimitRepo)
	userManageService := admin.NewUserManageService(userRepo)
	ipAccessService := access.NewIPAccessService(ipRepo, ipCache)
	rateLimitService := access.NewRateLimitService(rateLimitRepo, ratelimit, enableRatelimit, abuseLimit, abuseWindow)
	accountService := identity.NewAccountService(userRepo)
	consumeTokenService := identity.NewConsumeTokenService(tokenRepo)
	sessionService := identity.NewSessionService(sessionRepo)
	emailService := identity.NewEmailService(registerRepo, countdownRepo,
		os.Getenv("SMTP_HOST"), os.Getenv("SMTP_PORT"), os.Getenv("SMTP_USERNAME"), os.Getenv("SMTP_PASSWORD"), os.Getenv("SMTP_FROM"))

	// register handlers
	tokenHandler := handler.NewTokenHandler(adminTokenService)
	userHandler := handler.NewUserHandler(sessionService, accountService, consumeTokenService, emailService)
	ipRuleHandler := handler.NewIPRuleHandler(adminIPRuleService)
	userManageHandler := handler.NewUserManageHandler(userManageService)

	// register middlewares
	checkIPMiddleware := middleware.NewCheckIPMiddleware(ipAccessService)
	authMiddleWare := middleware.NewAuthMiddleWare(sessionService)
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(rateLimitService, adminIPRuleService)

	if err := updateAdmin(accountService); err != nil {
		slog.Error("failed to update admin info", "error", err)
		panic(err)
	}

	r := gin.New()
	// AccessLog outermost: it must observe the status that Recovery writes
	// on panics, and the abort statuses written by /kt middleware.
	r.Use(middleware.AccessLog(), gin.Recovery())

	trustedProxiesRaw := os.Getenv("TRUSTED_PROXIES")
	if trustedProxiesRaw == "" {
		trustedProxies = []string{"127.0.0.0/8", "::1/128"}
		slog.Warn("no trusted proxies found, using default", "trustedProxies", trustedProxies)
	} else {
		trustedProxies = strings.Split(trustedProxiesRaw, ",")
		slog.Info("trusted proxies set from env", "trustedProxies", trustedProxies)
	}

	err = r.SetTrustedProxies(trustedProxies)
	if err != nil {
		slog.Error("invalid TRUSTED_PROXIES settings", "error", err)
		panic(err)
	}

	// Kantan route
	// Allow non-blacklist access, set ratelimit to greylist
	r.GET("/kt/0", checkIPMiddleware.ACL(0), rateLimitMiddleware.RateLimit(), func(ctx *gin.Context) { ctx.Status(http.StatusNoContent) })
	// Only allow whitelist
	r.GET("/kt/1", checkIPMiddleware.ACL(1), func(ctx *gin.Context) { ctx.Status(http.StatusNoContent) })

	router.RegisterTokenRouter(r, tokenHandler, checkIPMiddleware, authMiddleWare)
	router.RegisterUserRouter(r, userHandler, checkIPMiddleware, authMiddleWare, rateLimitMiddleware)
	router.RegisterIPRouter(r, ipRuleHandler, checkIPMiddleware, authMiddleWare)
	router.RegisterUserManageRouter(r, userManageHandler, checkIPMiddleware, authMiddleWare)

	srv := &http.Server{
		Addr:     ":51214",
		Handler:  r,
		ErrorLog: slog.NewLogLogger(slog.Default().Handler(), slog.LevelError),
	}
	slog.Info("listening on :51214")
	if err := srv.ListenAndServe(); err != nil {
		slog.Error("http server stopped", "error", err)
	}
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

func updateAdmin(s *identity.AccountService) error {
	adminName := os.Getenv("ADMIN_NAME")
	adminPasswd := os.Getenv("ADMIN_PASSWD")
	if adminName == "" || adminPasswd == "" {
		slog.Warn("No admin conf, use default admin:admin")
		adminName, adminPasswd = "admin", "admin"
	}
	return s.UpdateUser(context.Background(), "00000000-0000-0000-0000-000000000000", adminName, adminPasswd, nil, "admin")
}
