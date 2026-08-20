package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/StellaShiina/ktauth/internal/auth"
	"github.com/gin-gonic/gin"
)

type SessionReader interface {
	GetSession(ctx context.Context, uuid, jti string) (string, error)
}

type AuthMiddleWare struct {
	SessionReader SessionReader
}

func NewAuthMiddleWare(s SessionReader) *AuthMiddleWare {
	return &AuthMiddleWare{s}
}

func (m *AuthMiddleWare) VerifySession(requireRole string) gin.HandlerFunc {
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

		uuid, err := m.SessionReader.GetSession(c.Request.Context(), claims.UUID, claims.ID)

		if err != nil || claims.UUID != uuid {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		c.Set("uuid", claims.UUID)
		c.Set("jti", claims.ID)

		if requireRole == "admin" && claims.Role != "admin" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		c.Next()
	}
}
