package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/StellaShiina/ktauth/internal/model"
	"github.com/gin-gonic/gin"
)

type IPRuleQuerier interface {
	QueryRule(ctx context.Context, ip string) (model.IPRuleType, error)
}

type CheckIPMiddleware struct {
	ipQuerier IPRuleQuerier
}

func NewCheckIPMiddleware(s IPRuleQuerier) *CheckIPMiddleware {
	return &CheckIPMiddleware{s}
}

// level 0 to deny blacklist, level 1 to only allow whitelist
func (m *CheckIPMiddleware) ACL(level int) gin.HandlerFunc {
	return func(c *gin.Context) {
		rule_type, err := m.ipQuerier.QueryRule(c, c.ClientIP())
		if err != nil {
			slog.Error("failed to query ip rule", "error", err, "clientIP", c.ClientIP())
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		// audit-log context only: lets the access log report which rule fired
		c.Set("rule", string(rule_type))
		switch rule_type {
		case model.IPBlackList:
			c.JSON(http.StatusForbidden, gin.H{
				"message": "Sorry, you are not allow to access",
				"ip":      c.ClientIP(),
			})
			c.Abort()
			return
		case model.IPGreyList:
			if level == 1 {
				c.JSON(http.StatusForbidden, gin.H{
					"message": "Sorry, you are not allow to access",
					"ip":      c.ClientIP(),
				})
				c.Abort()
				return
			}
			c.Set("whitelist", false)
			c.Next()
			return
		case model.IPWhiteList:
			c.Set("whitelist", true)
			c.Next()
			return
		default:
			slog.Error("unknown ip rule type", "rule", string(rule_type), "clientIP", c.ClientIP())
			c.AbortWithStatus(http.StatusInternalServerError)
		}
	}
}
