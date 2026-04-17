package router

import (
	"net/http"

	"github.com/StellaShiina/ktauth/internal/handler"
	"github.com/StellaShiina/ktauth/internal/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterUserRouter(r *gin.Engine, h *handler.UserHandler, aclm *middleware.CheckIPMiddleware, m *middleware.AuthMiddleWare, rm *middleware.RateLimitMiddleware) {
	r.POST("/api/users/register", aclm.ACL(0), rm.RateLimit(), h.RegisterUser)
	r.POST("/api/users/login", aclm.ACL(0), rm.RateLimit(), h.LoginUser)
	user := r.Group("/api/users", aclm.ACL(0), m.VerifySession(""))
	{
		user.GET("/auth", func(ctx *gin.Context) { ctx.Status(http.StatusNoContent) })
		user.GET("/logout", h.LogoutUser)
	}
}
