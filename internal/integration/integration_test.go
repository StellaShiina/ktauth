//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/StellaShiina/ktauth/internal/auth"
	"github.com/StellaShiina/ktauth/internal/db"
	"github.com/StellaShiina/ktauth/internal/handler"
	"github.com/StellaShiina/ktauth/internal/middleware"
	"github.com/StellaShiina/ktauth/internal/repository"
	"github.com/StellaShiina/ktauth/internal/router"
	"github.com/StellaShiina/ktauth/internal/service/access"
	"github.com/StellaShiina/ktauth/internal/service/admin"
	"github.com/StellaShiina/ktauth/internal/service/identity"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const (
	testRuleIP = "203.0.113.77"
	userTestIP = "198.51.100.125"
	rateTestIP = "198.51.100.124"
)

type integrationApp struct {
	engine       *gin.Engine
	postgres     *pgxpool.Pool
	redis        *redis.Client
	adminSession string
	userSession  string
	token        string
	userName     string
}

func TestMain(m *testing.M) {
	if err := loadExampleEnv(); err != nil {
		fmt.Fprintf(os.Stderr, "load .env.example: %v\n", err)
		os.Exit(1)
	}

	// Keep the integration flow quick while still exercising the real limiter.
	_ = os.Setenv("RATELIMIT", "2")
	_ = os.Setenv("ABUSELIMIT", "2")
	_ = os.Setenv("ABUSEWINDOW", "1")
	_ = os.Setenv("ENABLE_RATELIMIT", "YES")
	_ = os.Setenv("LOGLEVEL", "error")

	os.Exit(m.Run())
}

func loadExampleEnv() error {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return fmt.Errorf("locate integration test")
	}

	path := filepath.Join(filepath.Dir(filename), "..", "..", ".env.example")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(key) == "" {
			continue
		}
		if err := os.Setenv(strings.TrimSpace(key), strings.TrimSpace(value)); err != nil {
			return err
		}
	}
	return nil
}

func newIntegrationApp(t *testing.T) *integrationApp {
	t.Helper()

	postgres, err := db.NewPostgres()
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	rdb, err := db.NewRedis()
	if err != nil {
		postgres.Close()
		t.Fatalf("connect redis: %v", err)
	}

	app := &integrationApp{
		postgres: postgres,
		redis:    rdb,
		userName: "integration-" + uuid.NewString(),
	}
	t.Cleanup(func() {
		app.cleanup(t)
	})

	ctx := context.Background()
	app.cleanupState(t, ctx)

	ipRepo := repository.NewIPRepo(postgres)
	userRepo := repository.NewUserRepo(postgres)
	tokenRepo := repository.NewTokenRepo(rdb)
	sessionRepo := repository.NewSessionRepo(rdb)
	ipCache := repository.NewIPCache(rdb)
	rateLimitRepo := repository.NewRateLimitRepo(rdb)
	registerRepo := repository.NewRegisterRepo(rdb)
	countdownRepo := repository.NewCountDownRepo(rdb)

	adminIPRuleService := admin.NewAdminIPRuleService(ipRepo, ipCache, rateLimitRepo)
	userManageService := admin.NewUserManageService(userRepo)
	ipAccessService := access.NewIPAccessService(ipRepo, ipCache)
	rateLimit, err := strconv.Atoi(os.Getenv("RATELIMIT"))
	if err != nil {
		t.Fatalf("parse RATELIMIT: %v", err)
	}
	abuseLimit, err := strconv.Atoi(os.Getenv("ABUSELIMIT"))
	if err != nil {
		t.Fatalf("parse ABUSELIMIT: %v", err)
	}
	abuseWindowMinutes, err := strconv.Atoi(os.Getenv("ABUSEWINDOW"))
	if err != nil {
		t.Fatalf("parse ABUSEWINDOW: %v", err)
	}
	rateLimitService := access.NewRateLimitService(rateLimitRepo, rateLimit, true, abuseLimit, time.Duration(abuseWindowMinutes)*time.Minute)
	accountService := identity.NewAccountService(userRepo)
	consumeTokenService := identity.NewConsumeTokenService(tokenRepo)
	sessionService := identity.NewSessionService(sessionRepo)
	emailService := identity.NewEmailService(registerRepo, countdownRepo,
		os.Getenv("SMTP_HOST"), os.Getenv("SMTP_PORT"), os.Getenv("SMTP_USERNAME"), os.Getenv("SMTP_PASSWORD"), os.Getenv("SMTP_FROM"))

	tokenHandler := handler.NewTokenHandler(admin.NewAdminTokenService(tokenRepo))
	userHandler := handler.NewUserHandler(sessionService, accountService, consumeTokenService, emailService)
	ipRuleHandler := handler.NewIPRuleHandler(adminIPRuleService)
	userManageHandler := handler.NewUserManageHandler(userManageService)

	checkIPMiddleware := middleware.NewCheckIPMiddleware(ipAccessService)
	authMiddleware := middleware.NewAuthMiddleWare(sessionService)
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(rateLimitService, adminIPRuleService)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	trustedProxies := strings.Split(os.Getenv("TRUSTED_PROXIES"), ",")
	if err := engine.SetTrustedProxies(trustedProxies); err != nil {
		t.Fatalf("configure trusted proxies: %v", err)
	}
	engine.GET("/kt/0", checkIPMiddleware.ACL(0), rateLimitMiddleware.RateLimit(), func(c *gin.Context) { c.Status(http.StatusNoContent) })
	engine.GET("/kt/1", checkIPMiddleware.ACL(1), func(c *gin.Context) { c.Status(http.StatusNoContent) })
	router.RegisterTokenRouter(engine, tokenHandler, checkIPMiddleware, authMiddleware)
	router.RegisterUserRouter(engine, userHandler, checkIPMiddleware, authMiddleware, rateLimitMiddleware)
	router.RegisterIPRouter(engine, ipRuleHandler, checkIPMiddleware, authMiddleware)
	router.RegisterUserManageRouter(engine, userManageHandler, checkIPMiddleware, authMiddleware)
	app.engine = engine

	return app
}

func (a *integrationApp) cleanupState(t *testing.T, ctx context.Context) {
	t.Helper()
	if _, err := a.postgres.Exec(ctx, "DELETE FROM users WHERE name = $1", a.userName); err != nil {
		t.Fatalf("clean test user: %v", err)
	}
	for _, cidr := range []string{testRuleIP + "/32", userTestIP + "/32", rateTestIP + "/32"} {
		if _, err := a.postgres.Exec(ctx, "DELETE FROM ip WHERE ip_range = $1::cidr", cidr); err != nil {
			t.Fatalf("clean test IP %s: %v", cidr, err)
		}
	}
	keys := []string{
		"rule:ip:" + testRuleIP + "/32",
		"rule:ip:" + userTestIP + "/32",
		"rule:ip:" + rateTestIP + "/32",
		"ratelimit:ip:" + testRuleIP + "/32",
		"ratelimit:ip:" + userTestIP + "/32",
		"ratelimit:ip:" + rateTestIP + "/32",
		"abuse:429:" + rateTestIP + "/32",
	}
	if err := a.redis.Del(ctx, keys...).Err(); err != nil {
		t.Fatalf("clean redis state: %v", err)
	}
}

func (a *integrationApp) cleanup(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	if a.adminSession != "" {
		_ = a.redis.Del(ctx, a.adminSession).Err()
	}
	if a.userSession != "" {
		_ = a.redis.Del(ctx, a.userSession).Err()
	}
	if a.token != "" {
		_ = a.redis.SRem(ctx, "admin:tokens", a.token).Err()
	}
	a.cleanupState(t, ctx)
	a.redis.Close()
	a.postgres.Close()
}

func request(engine *gin.Engine, method, path, remoteIP, token, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.RemoteAddr = remoteIP + ":45678"
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	engine.ServeHTTP(recorder, req)
	return recorder
}

func TestApplicationEndToEnd(t *testing.T) {
	app := newIntegrationApp(t)

	ctx := context.Background()
	token := "integration-token-" + uuid.NewString()
	if err := app.redis.SAdd(ctx, "admin:tokens", token).Err(); err != nil {
		t.Fatalf("seed registration token: %v", err)
	}
	app.token = token

	registerBody := fmt.Sprintf(`{"token":%q,"user":%q,"password":"integration-password"}`, token, app.userName)
	response := request(app.engine, http.MethodPost, "/api/users/register", userTestIP, "", registerBody)
	if response.Code != http.StatusCreated {
		t.Fatalf("register status = %d, body = %s", response.Code, response.Body.String())
	}

	loginBody := fmt.Sprintf(`{"user":%q,"password":"integration-password"}`, app.userName)
	response = request(app.engine, http.MethodPost, "/api/users/login", userTestIP, "", loginBody)
	if response.Code != http.StatusOK {
		t.Fatalf("login status = %d, body = %s", response.Code, response.Body.String())
	}
	var loginResult struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &loginResult); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if loginResult.Token == "" {
		t.Fatal("login returned an empty token")
	}
	claims, err := auth.ParseToken(loginResult.Token)
	if err != nil {
		t.Fatalf("parse login token: %v", err)
	}
	app.userSession = fmt.Sprintf("jwt:active:%s:%s", claims.UUID, claims.ID)

	response = request(app.engine, http.MethodGet, "/api/users/auth", userTestIP, loginResult.Token, "")
	if response.Code != http.StatusNoContent {
		t.Fatalf("authenticated user status = %d, body = %s", response.Code, response.Body.String())
	}
	response = request(app.engine, http.MethodGet, "/api/users/logout", userTestIP, loginResult.Token, "")
	if response.Code != http.StatusOK {
		t.Fatalf("logout status = %d, body = %s", response.Code, response.Body.String())
	}
	response = request(app.engine, http.MethodGet, "/api/users/auth", userTestIP, loginResult.Token, "")
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("post-logout auth status = %d, want 401", response.Code)
	}

	adminToken, adminJTI, err := auth.SignToken("00000000-0000-0000-0000-000000000000", "admin", "admin")
	if err != nil {
		t.Fatalf("sign admin token: %v", err)
	}
	adminSession := fmt.Sprintf("jwt:active:%s:%s", "00000000-0000-0000-0000-000000000000", adminJTI)
	app.adminSession = adminSession
	if err := app.redis.Set(ctx, adminSession, "00000000-0000-0000-0000-000000000000", 24*time.Hour).Err(); err != nil {
		t.Fatalf("seed admin session: %v", err)
	}

	response = request(app.engine, http.MethodGet, "/api/ips?version=4&type=white", "127.0.0.1", adminToken, "")
	if response.Code != http.StatusOK {
		t.Fatalf("list IP rules status = %d, body = %s", response.Code, response.Body.String())
	}
	addRuleBody := fmt.Sprintf(`{"ip":%q,"isWhiteList":false,"note":"integration"}`, testRuleIP)
	response = request(app.engine, http.MethodPost, "/api/ips/new", "127.0.0.1", adminToken, addRuleBody)
	if response.Code != http.StatusOK {
		t.Fatalf("add IP rule status = %d, body = %s", response.Code, response.Body.String())
	}
	response = request(app.engine, http.MethodGet, "/kt/0", testRuleIP, "", "")
	if response.Code != http.StatusForbidden {
		t.Fatalf("blacklisted IP status = %d, want 403", response.Code)
	}
	response = request(app.engine, http.MethodDelete, "/api/ips", "127.0.0.1", adminToken, fmt.Sprintf(`{"ip":%q}`, testRuleIP))
	if response.Code != http.StatusOK {
		t.Fatalf("delete IP rule status = %d, body = %s", response.Code, response.Body.String())
	}

	for i := 0; i < 3; i++ {
		response = request(app.engine, http.MethodGet, "/kt/0", rateTestIP, "", "")
		want := http.StatusNoContent
		if i == 2 {
			want = http.StatusTooManyRequests
		}
		if response.Code != want {
			t.Fatalf("rate-limit request %d status = %d, want %d; body = %s", i+1, response.Code, want, response.Body.String())
		}
	}
	response = request(app.engine, http.MethodGet, "/kt/0", rateTestIP, "", "")
	if response.Code != http.StatusTooManyRequests {
		t.Fatalf("abuse-triggering request status = %d, want 429", response.Code)
	}
	response = request(app.engine, http.MethodGet, "/kt/0", rateTestIP, "", "")
	if response.Code != http.StatusForbidden {
		t.Fatalf("auto-banned IP status = %d, want 403", response.Code)
	}
}
