package router

import (
	"github.com/StellaShiina/ktauth/internal/handler"
	"github.com/StellaShiina/ktauth/internal/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterIPRouter(r *gin.Engine, h *handler.IPRuleHandler, m *middleware.CheckIPMiddleware) {
	g := r.Group("/api/ip", m.WhiteListOnly())
	{
		g.GET("", h.ListRules)
		g.DELETE("", h.DelRule)
		g.POST("/new", h.AddRule)
	}
}
