package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

type RateLimiter interface {
	Allow(ctx context.Context, ip string) (bool, error)
	Abuse(ctx context.Context, ip string) (bool, error)
}

type IPRuleAdder interface {
	AddRule(ctx context.Context, ipStr string, isWhiteList bool, note *string) (string, error)
}

type RateLimitMiddleware struct {
	rateLimiter RateLimiter
	ipRuleAdder IPRuleAdder
}

func NewRateLimitMiddleware(rateLimiter RateLimiter, ipRuleAdder IPRuleAdder) *RateLimitMiddleware {
	return &RateLimitMiddleware{rateLimiter, ipRuleAdder}
}

func (m *RateLimitMiddleware) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		isWhiteList := c.GetBool("whitelist")
		if isWhiteList {
			c.Next()
			return
		}
		allow, err := m.rateLimiter.Allow(c.Request.Context(), c.ClientIP())
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			slog.Error(err.Error())
			return
		}
		if !allow {
			c.String(http.StatusTooManyRequests, "Rate limit exceed!")
			c.Abort()
			if abuse, err := m.rateLimiter.Abuse(c.Request.Context(), c.ClientIP()); err == nil {
				if abuse {
					note := "Abuse with too many 429. Host: " + c.Request.Host
					cidr, err := m.ipRuleAdder.AddRule(c.Request.Context(), c.ClientIP(), false, &note)
					if err != nil {
						slog.Error("Add abuse IP to database failed", "error", err)
					} else {
						slog.Warn("Ban abuse IP", "IPRange", cidr)
					}
				}
			} else {
				slog.Error("Error when evaluating abuse", "error", err)
			}
			return
		}
		c.Next()
	}
}
