package main

import (
	"fmt"
	"log"
	"log/slog"
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
)

func main() {
	logger := slog.New(
		slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}),
	)
	slog.SetDefault(logger)

	config := db.Config{
		User:      "ktauth",
		Passwd:    "ktauth",
		Net:       "tcp",
		Addr:      "127.0.0.1",
		DBName:    "ktauth",
		ParseTime: true,
		Loc:       *time.Local,
	}

	mysql, err := db.NewMySQL(config)
	if err != nil {
		log.Fatal(err)
	} else {
		fmt.Println("Connected to mysql!")
	}
	defer mysql.Close()

	redis, err := db.NewRedis()

	if err != nil {
		log.Fatal(err)
	} else {
		fmt.Println("Connected to redis!")
	}
	defer redis.Close()

	// init repos
	ipRepo := repository.NewIPRepo(mysql)
	userRepo := repository.NewUserRepo(mysql)
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

	r.Use(checkIPMiddleware.VerifyWhileList())

	router.RegisterTokenRouter(r, tokenHandler, checkIPMiddleware)
	router.RegisterUserRouter(r, userHandler, authMiddleWare, rateLimitMiddleware)
	router.RegisterIPRouter(r, ipRuleHandler, checkIPMiddleware)

	r.Run(":10000")
}
