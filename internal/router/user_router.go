package router

import (
	"net/http"

	"github.com/StellaShiina/ktauth/internal/handler"
	"github.com/StellaShiina/ktauth/internal/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterUserRouter(r *gin.Engine, h *handler.UserHandler, m *middleware.AuthMiddleWare, rm *middleware.RateLimitMiddleware) {
	r.POST("/api/user/register", rm.RateLimit(), h.RegisterUser)
	r.POST("/api/user/login", rm.RateLimit(), h.LoginUser)
	r.POST("/api/user/code", rm.RateLimit(), h.RequireCode)
	user := r.Group("/api/user", m.VerifySession())
	{
		user.GET("/auth", func(ctx *gin.Context) { ctx.String(http.StatusOK, "You've already logged in!") })
		user.GET("/logout", h.LogoutUser)
	}
}
