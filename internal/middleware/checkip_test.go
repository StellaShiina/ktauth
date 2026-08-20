package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/StellaShiina/ktauth/internal/middleware"
	"github.com/StellaShiina/ktauth/internal/model"
	"github.com/gin-gonic/gin"
)

type ipRuleQuerierMock struct {
	rule  model.IPRuleType
	err   error
	calls int
	ip    string
}

func (m *ipRuleQuerierMock) QueryRule(_ context.Context, ip string) (model.IPRuleType, error) {
	m.calls++
	m.ip = ip
	return m.rule, m.err
}

func TestCheckIPMiddlewareACL(t *testing.T) {
	tests := []struct {
		name          string
		rule          model.IPRuleType
		level         int
		status        int
		wantNext      bool
		wantWhitelist any
	}{
		{name: "blacklist", rule: model.IPBlackList, level: 0, status: http.StatusForbidden},
		{name: "greylist allowed", rule: model.IPGreyList, level: 0, status: http.StatusOK, wantNext: true, wantWhitelist: false},
		{name: "greylist denied", rule: model.IPGreyList, level: 1, status: http.StatusForbidden},
		{name: "whitelist", rule: model.IPWhiteList, level: 1, status: http.StatusOK, wantNext: true, wantWhitelist: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &ipRuleQuerierMock{rule: tt.rule}
			nextCalls := 0
			var whitelist any
			m := middleware.NewCheckIPMiddleware(mock)
			engine := gin.New()
			engine.GET("/", m.ACL(tt.level), func(c *gin.Context) {
				nextCalls++
				whitelist, _ = c.Get("whitelist")
				c.Status(http.StatusOK)
			})
			recorder := httptest.NewRecorder()

			engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

			if recorder.Code != tt.status || (nextCalls > 0) != tt.wantNext {
				t.Fatalf("status/next = %d/%d, want %d/%t", recorder.Code, nextCalls, tt.status, tt.wantNext)
			}
			if tt.wantNext && whitelist != tt.wantWhitelist {
				t.Fatalf("whitelist value = %#v, want %v", whitelist, tt.wantWhitelist)
			}
		})
	}
}

func TestCheckIPMiddlewareHandlesQueryErrorAndUnknownRule(t *testing.T) {
	for _, mock := range []*ipRuleQuerierMock{
		{err: errors.New("database unavailable")},
		{rule: model.IPRuleType("unknown")},
	} {
		engine := gin.New()
		engine.GET("/", middleware.NewCheckIPMiddleware(mock).ACL(0), func(c *gin.Context) { c.Status(http.StatusOK) })
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
		if recorder.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want 500", recorder.Code)
		}
	}
}
