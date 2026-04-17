package router

import (
	"github.com/StellaShiina/ktauth/internal/handler"
	"github.com/StellaShiina/ktauth/internal/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterIPRouter(r *gin.Engine, h *handler.IPRuleHandler, aclm *middleware.CheckIPMiddleware, authm *middleware.AuthMiddleWare) {
	g := r.Group("/api/ips", aclm.ACL(1), authm.VerifySession("admin"))
	{
		g.GET("", h.ListRules)
		g.DELETE("", h.DelRule)
		g.POST("/new", h.AddRule)
	}
}

func RegisterUserManageRouter(r *gin.Engine, h *handler.UserManageHandler, aclm *middleware.CheckIPMiddleware, authm *middleware.AuthMiddleWare) {
	r.GET("/api/users", aclm.ACL(1), authm.VerifySession("admin"), h.ListUsers)
}
