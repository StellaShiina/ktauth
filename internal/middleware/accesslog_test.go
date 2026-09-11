package middleware_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
			},
		},
		{
			name:      "500 server error",
			status:    http.StatusInternalServerError,
			wantLevel: "ERROR",
			want:      []string{`msg="request failed"`, "status=500"},
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
	for _, w := range []string{`msg="request rejected"`, "status=403", "rule=blacklist", "host=auth.example.com"} {
		if !strings.Contains(out, w) {
			t.Errorf("log missing %q:\n%s", w, out)
		}
	}
}
