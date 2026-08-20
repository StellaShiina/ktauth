package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/StellaShiina/ktauth/internal/middleware"
	"github.com/gin-gonic/gin"
)

type rateLimiterMock struct {
	allowCalls int
	abuseCalls int
	allow      bool
	abuse      bool
	allowErr   error
	abuseErr   error
	ip         string
}

func (m *rateLimiterMock) Allow(_ context.Context, ip string) (bool, error) {
	m.allowCalls++
	m.ip = ip
	return m.allow, m.allowErr
}

func (m *rateLimiterMock) Abuse(_ context.Context, ip string) (bool, error) {
	m.abuseCalls++
	m.ip = ip
	return m.abuse, m.abuseErr
}

type ipRuleAdderMock struct {
	calls     int
	ip        string
	whitelist bool
	note      string
	cidr      string
	err       error
}

func (m *ipRuleAdderMock) AddRule(_ context.Context, ip string, whitelist bool, note *string) (string, error) {
	m.calls++
	m.ip, m.whitelist, m.cidr = ip, whitelist, "192.0.2.0/32"
	if note != nil {
		m.note = *note
	}
	return m.cidr, m.err
}

func TestRateLimitMiddlewareSkipsWhitelist(t *testing.T) {
	limiter := &rateLimiterMock{}
	adder := &ipRuleAdderMock{}
	nextCalls := 0
	engine := gin.New()
	engine.GET("/protected", func(c *gin.Context) { c.Set("whitelist", true); c.Next() }, middleware.NewRateLimitMiddleware(limiter, adder).RateLimit(), func(c *gin.Context) { nextCalls++; c.Status(http.StatusOK) })
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/protected", nil))

	if recorder.Code != http.StatusOK || nextCalls != 1 || limiter.allowCalls != 0 || adder.calls != 0 {
		t.Fatalf("status/next/calls = %d/%d/%d/%d", recorder.Code, nextCalls, limiter.allowCalls, adder.calls)
	}
}

func TestRateLimitMiddlewareAllowsRequest(t *testing.T) {
	limiter := &rateLimiterMock{allow: true}
	nextCalls := 0
	engine := gin.New()
	engine.GET("/protected", func(c *gin.Context) { c.Set("whitelist", false); c.Next() }, middleware.NewRateLimitMiddleware(limiter, &ipRuleAdderMock{}).RateLimit(), func(c *gin.Context) { nextCalls++; c.Status(http.StatusOK) })
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/protected", nil))

	if recorder.Code != http.StatusOK || nextCalls != 1 || limiter.allowCalls != 1 || limiter.abuseCalls != 0 {
		t.Fatalf("status/next/calls = %d/%d/%d/%d", recorder.Code, nextCalls, limiter.allowCalls, limiter.abuseCalls)
	}
}

func TestRateLimitMiddlewareHandlesAllowError(t *testing.T) {
	limiter := &rateLimiterMock{allowErr: errors.New("redis unavailable")}
	engine := gin.New()
	engine.GET("/protected", func(c *gin.Context) { c.Set("whitelist", false); c.Next() }, middleware.NewRateLimitMiddleware(limiter, &ipRuleAdderMock{}).RateLimit(), func(c *gin.Context) { c.Status(http.StatusOK) })
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/protected", nil))

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", recorder.Code)
	}
}

func TestRateLimitMiddlewareBansAbusiveIP(t *testing.T) {
	limiter := &rateLimiterMock{allow: false, abuse: true}
	adder := &ipRuleAdderMock{}
	engine := gin.New()
	engine.GET("/protected", func(c *gin.Context) { c.Set("whitelist", false); c.Next() }, middleware.NewRateLimitMiddleware(limiter, adder).RateLimit(), func(c *gin.Context) { c.Status(http.StatusOK) })
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Host = "example.test"
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusTooManyRequests || !strings.Contains(recorder.Body.String(), "Rate limit exceed") {
		t.Fatalf("status/body = %d/%q, want 429", recorder.Code, recorder.Body.String())
	}
	if limiter.allowCalls != 1 || limiter.abuseCalls != 1 || adder.calls != 1 || adder.whitelist || adder.note == "" {
		t.Fatalf("dependency calls = limiter %#v, adder %#v", limiter, adder)
	}
}

func TestRateLimitMiddlewareDoesNotBanWhenAbuseCheckFails(t *testing.T) {
	limiter := &rateLimiterMock{allow: false, abuseErr: errors.New("redis unavailable")}
	adder := &ipRuleAdderMock{}
	engine := gin.New()
	engine.GET("/protected", func(c *gin.Context) { c.Set("whitelist", false); c.Next() }, middleware.NewRateLimitMiddleware(limiter, adder).RateLimit(), func(c *gin.Context) { c.Status(http.StatusOK) })
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/protected", nil))

	if recorder.Code != http.StatusTooManyRequests || adder.calls != 0 || limiter.abuseCalls != 1 {
		t.Fatalf("status/calls = %d/%d/%d, want 429/0/1", recorder.Code, adder.calls, limiter.abuseCalls)
	}
}
