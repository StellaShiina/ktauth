package middleware_test

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/StellaShiina/ktauth/internal/auth"
	"github.com/StellaShiina/ktauth/internal/logctx"
	"github.com/StellaShiina/ktauth/internal/middleware"
	"github.com/StellaShiina/ktauth/internal/model"
	"github.com/gin-gonic/gin"
)

// captureSlog swaps the global default logger for one writing into buf,
// returning a restore function.
func captureSlog(buf *bytes.Buffer) func() {
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(buf, nil)))
	return func() { slog.SetDefault(prev) }
}

func TestAccessLog(t *testing.T) {
	var buf bytes.Buffer

	tests := []struct {
		name      string
		status    int
		host      string
		headers   map[string]string
		setup     func(c *gin.Context)
		wantLevel string // "" = expect silence
		want      []string
	}{
		{
			name:   "2xx silent",
			status: http.StatusNoContent,
		},
		{
			name:   "403 blacklist with original uri",
			status: http.StatusForbidden,
			host:   "auth.example.com",
			headers: map[string]string{
				"X-Original-URI":    "/admin",
				"X-Original-Method": "GET",
			},
			setup:     func(c *gin.Context) { c.Set("rule", "blacklist") },
			wantLevel: "WARN",
			want: []string{
				`msg="request rejected"`,
				"status=403",
				"method=GET",
				"path=/kt/0",
				`query="q=1"`,
				"host=auth.example.com",
				"clientIP=",
				"durationMs=",
				"originalURI=/admin",
				"originalMethod=GET",
				"rule=blacklist",
				// this route writes the status directly, so nothing recorded a
				// reason: the audit line must say so rather than stay silent.
				"reason=unattributed",
			},
		},
		{
			name:   "429 greylist with caddy forwarded uri",
			status: http.StatusTooManyRequests,
			headers: map[string]string{
				"X-Forwarded-Uri":    "/secret",
				"X-Forwarded-Method": "POST",
			},
			setup:     func(c *gin.Context) { c.Set("rule", "greylist") },
			wantLevel: "WARN",
			want: []string{
				`msg="request rejected"`,
				"status=429",
				"originalURI=/secret",
				"originalMethod=POST",
				"rule=greylist",
				"reason=unattributed",
			},
		},
		{
			name:      "500 server error",
			status:    http.StatusInternalServerError,
			wantLevel: "ERROR",
			want:      []string{`msg="request failed"`, "status=500", "reason=unattributed"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			restore := captureSlog(&buf)
			defer restore()

			engine := gin.New()
			engine.Use(middleware.AccessLog())
			engine.GET("/kt/0", func(c *gin.Context) {
				if tt.setup != nil {
					tt.setup(c)
				}
				c.Status(tt.status)
			})

			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/kt/0?q=1", nil)
			req.Host = tt.host
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			engine.ServeHTTP(recorder, req)

			out := buf.String()
			if tt.wantLevel == "" {
				if out != "" {
					t.Fatalf("expected silence, got:\n%s", out)
				}
				return
			}
			if !strings.Contains(out, "level="+tt.wantLevel) {
				t.Errorf("missing level %s:\n%s", tt.wantLevel, out)
			}
			for _, w := range tt.want {
				if !strings.Contains(out, w) {
					t.Errorf("log missing %q:\n%s", w, out)
				}
			}
		})
	}
}

// TestAccessLogWithACLRule wires the real ACL middleware in front of the
// audit log and checks that a blacklist rejection carries rule=blacklist.
func TestAccessLogWithACLRule(t *testing.T) {
	var buf bytes.Buffer
	restore := captureSlog(&buf)
	defer restore()

	mock := &ipRuleQuerierMock{rule: model.IPBlackList}
	engine := gin.New()
	engine.Use(middleware.AccessLog())
	engine.GET("/kt/0", middleware.NewCheckIPMiddleware(mock).ACL(0), func(c *gin.Context) { c.Status(http.StatusNoContent) })

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/kt/0", nil)
	req.Host = "auth.example.com"
	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", recorder.Code)
	}
	out := buf.String()
	for _, w := range []string{`msg="request rejected"`, "status=403", "rule=blacklist", "host=auth.example.com", "reason=ip_blacklisted"} {
		if !strings.Contains(out, w) {
			t.Errorf("log missing %q:\n%s", w, out)
		}
	}
}

// TestAccessLogReasons wires the real middlewares in front of the audit log and
// checks that each 4xx/5xx line reports the specific cause, not just a status.
func TestAccessLogReasons(t *testing.T) {
	userToken, _, err := auth.SignToken("u1", "alice", "user")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		route   func(engine *gin.Engine)
		request func() *http.Request
		want    []string
	}{
		{
			name: "auth missing bearer",
			route: func(e *gin.Engine) {
				m := middleware.NewAuthMiddleWare(&sessionReaderMock{})
				e.GET("/p", m.VerifySession(""), func(c *gin.Context) { c.Status(http.StatusNoContent) })
			},
			request: func() *http.Request { return httptest.NewRequest(http.MethodGet, "/p", nil) },
			want:    []string{`msg="request rejected"`, "status=401", "reason=missing_bearer_token"},
		},
		{
			name: "auth invalid token",
			route: func(e *gin.Engine) {
				m := middleware.NewAuthMiddleWare(&sessionReaderMock{})
				e.GET("/p", m.VerifySession(""), func(c *gin.Context) { c.Status(http.StatusNoContent) })
			},
			request: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/p", nil)
				r.Header.Set("Authorization", "Bearer not-a-token")
				return r
			},
			want: []string{"status=401", "reason=invalid_token", "detail="},
		},
		{
			name: "auth insufficient role names the actor",
			route: func(e *gin.Engine) {
				m := middleware.NewAuthMiddleWare(&sessionReaderMock{value: "u1"})
				e.GET("/p", m.VerifySession("admin"), func(c *gin.Context) { c.Status(http.StatusNoContent) })
			},
			request: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/p", nil)
				r.Header.Set("Authorization", "Bearer "+userToken)
				return r
			},
			want: []string{"status=401", "reason=insufficient_role", "uuid=u1"},
		},
		{
			name: "blacklisted ip",
			route: func(e *gin.Engine) {
				m := middleware.NewCheckIPMiddleware(&ipRuleQuerierMock{rule: model.IPBlackList})
				e.GET("/p", m.ACL(0), func(c *gin.Context) { c.Status(http.StatusNoContent) })
			},
			request: func() *http.Request { return httptest.NewRequest(http.MethodGet, "/p", nil) },
			want:    []string{"status=403", "rule=blacklist", "reason=ip_blacklisted"},
		},
		{
			name: "acl lookup failure carries the error",
			route: func(e *gin.Engine) {
				m := middleware.NewCheckIPMiddleware(&ipRuleQuerierMock{err: errors.New("redis unavailable")})
				e.GET("/p", m.ACL(0), func(c *gin.Context) { c.Status(http.StatusNoContent) })
			},
			request: func() *http.Request { return httptest.NewRequest(http.MethodGet, "/p", nil) },
			want: []string{`msg="request failed"`, "status=500", "reason=ip_rule_lookup_failed",
				`detail="redis unavailable"`},
		},
		{
			name: "rate limited",
			route: func(e *gin.Engine) {
				e.GET("/p", middleware.NewRateLimitMiddleware(&rateLimiterMock{}, &ipRuleAdderMock{}).RateLimit(),
					func(c *gin.Context) { c.Status(http.StatusNoContent) })
			},
			request: func() *http.Request { return httptest.NewRequest(http.MethodGet, "/p", nil) },
			want:    []string{"status=429", "reason=ratelimit_exceeded"},
		},
		{
			name: "rate limiter unavailable",
			route: func(e *gin.Engine) {
				m := middleware.NewRateLimitMiddleware(&rateLimiterMock{allowErr: errors.New("redis down")}, &ipRuleAdderMock{})
				e.GET("/p", m.RateLimit(), func(c *gin.Context) { c.Status(http.StatusNoContent) })
			},
			request: func() *http.Request { return httptest.NewRequest(http.MethodGet, "/p", nil) },
			want:    []string{"status=500", "reason=ratelimit_unavailable", `detail="redis down"`},
		},
		{
			name: "handler records its own reason",
			route: func(e *gin.Engine) {
				e.POST("/p", func(c *gin.Context) {
					logctx.SetReasonDetail(c, logctx.ReasonInvalidBody, errors.New("unexpected end of JSON input"))
					c.JSON(http.StatusBadRequest, gin.H{"error": "unexpected end of JSON input"})
				})
			},
			request: func() *http.Request { return httptest.NewRequest(http.MethodPost, "/p", nil) },
			want: []string{"status=400", "reason=invalid_body",
				`detail="unexpected end of JSON input"`},
		},
		{
			name:    "unmatched route",
			route:   func(e *gin.Engine) {},
			request: func() *http.Request { return httptest.NewRequest(http.MethodGet, "/nope", nil) },
			want:    []string{"status=404", "reason=route_not_found"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			restore := captureSlog(&buf)
			defer restore()

			engine := gin.New()
			engine.Use(middleware.AccessLog())
			tt.route(engine)

			engine.ServeHTTP(httptest.NewRecorder(), tt.request())

			out := buf.String()
			for _, w := range tt.want {
				if !strings.Contains(out, w) {
					t.Errorf("log missing %q:\n%s", w, out)
				}
			}
		})
	}
}
