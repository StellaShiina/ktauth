package middleware

import (
	"log/slog"
	"net/http"

	"github.com/StellaShiina/ktauth/internal/service/access"
	"github.com/StellaShiina/ktauth/internal/service/admin"
	"github.com/gin-gonic/gin"
)

type RateLimitMiddleware struct {
	rateLimitService   *access.RateLimitService
	adminIPRuleService *admin.AdminIPRuleService
}

func NewRateLimitMiddleware(rateLimitService *access.RateLimitService, adminIPRuleService *admin.AdminIPRuleService) *RateLimitMiddleware {
	return &RateLimitMiddleware{rateLimitService, adminIPRuleService}
}

func (m *RateLimitMiddleware) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		isWhiteList := c.GetBool("whitelist")
		if isWhiteList {
			c.Next()
			return
		}
		allow, err := m.rateLimitService.Allow(c.Request.Context(), c.ClientIP())
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			slog.Error(err.Error())
			return
		}
		if !allow {
			c.String(http.StatusTooManyRequests, "Rate limit exceed!")
			c.Abort()
			if abuse, err := m.rateLimitService.Abuse(c.Request.Context(), c.ClientIP()); err == nil {
				if abuse {
					note := "Abuse with too many 429"
					cidr, err := m.adminIPRuleService.AddRule(c.Request.Context(), c.ClientIP(), false, &note)
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
