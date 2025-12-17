package router

import (
	"github.com/StellaShiina/ktauth/internal/handler"
	"github.com/StellaShiina/ktauth/internal/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterTokenRouter(r *gin.Engine, h *handler.TokenHandler, m *middleware.CheckIPMiddleware) {
	token := r.Group("/api/token", m.WhiteListOnly())
	{
		token.GET("/auth", func(ctx *gin.Context) { ctx.String(200, "From ip whilelist: "+ctx.ClientIP()) })
		token.GET("/restock", h.Restock)
		token.DELETE("/flush", h.FlushTokens)
		token.GET("", h.GetToken)
		token.GET("/all", h.GetTokens)
	}
}
