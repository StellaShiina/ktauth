package middleware

import (
	"net/http"

	"github.com/StellaShiina/ktauth/internal/service/access"
	"github.com/gin-gonic/gin"
)

type CheckIPMiddleware struct {
	ipAccessService *access.IPAccessService
}

func NewCheckIPMiddleware(s *access.IPAccessService) *CheckIPMiddleware {
	return &CheckIPMiddleware{s}
}

func (m *CheckIPMiddleware) VerifyWhileList() gin.HandlerFunc {
	return func(c *gin.Context) {
		valid, err := m.ipAccessService.VerifyWhileList(c.Request.Context(), c.ClientIP())
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		if !valid {
			c.Set("whitelist", false)
			c.Next()
			return
		}
		c.Set("whitelist", true)
		c.Next()
	}
}

func (m *CheckIPMiddleware) WhiteListOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		isWhiteList := c.GetBool("whitelist")
		if !isWhiteList {
			c.String(http.StatusUnauthorized, "You are not allowed to access.\nYour IP: "+c.ClientIP())
			c.Abort()
			return
		}
		c.Next()
	}
}
