package router

import (
	"github.com/StellaShiina/ktauth/internal/handler"
	"github.com/StellaShiina/ktauth/internal/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterTokenRouter(r *gin.Engine, h *handler.TokenHandler, aclm *middleware.CheckIPMiddleware, authm *middleware.AuthMiddleWare) {
	token := r.Group("/api/tokens", aclm.ACL(1), authm.VerifySession("admin"))
	{
		token.GET("/restock", h.Restock)
		token.DELETE("/flush", h.FlushTokens)
		token.GET("", h.GetToken)
		token.GET("/all", h.GetTokens)
	}
}
