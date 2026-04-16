package middleware

import (
	"log/slog"
	"net/http"

	"github.com/StellaShiina/ktauth/internal/service/access"
	"github.com/StellaShiina/ktauth/pkg/iputils"
	"github.com/gin-gonic/gin"
)

type RateLimitMiddleware struct {
	rateLimitService *access.RateLimitService
}

func NewRateLimitMiddleware(s *access.RateLimitService) *RateLimitMiddleware {
	return &RateLimitMiddleware{s}
}

func (m *RateLimitMiddleware) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		isWhiteList := c.GetBool("whitelist")
		if isWhiteList {
			c.Next()
			return
		}
		_, _, ipNet, err := iputils.ProcessIP(c.ClientIP())
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			slog.Error(err.Error())
			return
		}
		allow, err := m.rateLimitService.Allow(c.Request.Context(), ipNet.String())
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			slog.Error(err.Error())
			return
		}
		if !allow {
			c.String(http.StatusTooManyRequests, "Rate limit exceed!")
			c.Abort()
			return
		}
		c.Next()
	}
}
