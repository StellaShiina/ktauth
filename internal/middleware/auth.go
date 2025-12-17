package middleware

import (
	"net/http"
	"strings"

	"github.com/StellaShiina/ktauth/internal/auth"
	"github.com/StellaShiina/ktauth/internal/service/identity"
	"github.com/gin-gonic/gin"
)

type AuthMiddleWare struct {
	sessionService *identity.SessionService
}

func NewAuthMiddleWare(s *identity.SessionService) *AuthMiddleWare {
	return &AuthMiddleWare{s}
}

func (m *AuthMiddleWare) VerifySession() gin.HandlerFunc {
	return func(c *gin.Context) {
		authStr := c.GetHeader("Authorization")
		if !strings.HasPrefix(authStr, "Bearer ") {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		tokenStr := strings.TrimPrefix(authStr, "Bearer ")

		claims, err := auth.ParseToken(tokenStr)

		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		uuid, err := m.sessionService.GetSession(c.Request.Context(), claims.UUID, claims.ID)

		if err != nil || claims.UUID != uuid {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		c.Set("uuid", claims.UUID)
		c.Set("jti", claims.ID)

		c.Next()
	}
}
