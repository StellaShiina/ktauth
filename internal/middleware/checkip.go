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

func (m *CheckIPMiddleware) DenyBlackList() gin.HandlerFunc {
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

func (m *CheckIPMiddleware) WhiteListOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		isWhiteList := c.GetBool("whitelist")
		if !isWhiteList {
			c.JSON(http.StatusForbidden, gin.H{
				"message": "Sorry, you are not allow to access",
				"ip":      c.ClientIP(),
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
