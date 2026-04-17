package middleware

import (
	"log/slog"
	"net/http"

	"github.com/StellaShiina/ktauth/internal/model"
	"github.com/StellaShiina/ktauth/internal/service/access"
	"github.com/gin-gonic/gin"
)

type CheckIPMiddleware struct {
	ipAccessService *access.IPAccessService
}

func NewCheckIPMiddleware(s *access.IPAccessService) *CheckIPMiddleware {
	return &CheckIPMiddleware{s}
}

// level 0 to deny blacklist, level 1 to only allow whitelist
func (m *CheckIPMiddleware) ACL(level int) gin.HandlerFunc {
	return func(c *gin.Context) {
		rule_type, err := m.ipAccessService.QueryRule(c, c.ClientIP())
		if err != nil {
			slog.Error(err.Error())
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
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
			c.AbortWithStatus(http.StatusInternalServerError)
		}
	}
}
