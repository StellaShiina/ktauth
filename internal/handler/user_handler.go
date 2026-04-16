package handler

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/StellaShiina/ktauth/internal/auth"
	"github.com/StellaShiina/ktauth/internal/crypto"
	"github.com/StellaShiina/ktauth/internal/service/identity"
	"github.com/StellaShiina/ktauth/pkg/iputils"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	sessionService      *identity.SessionService
	accountService      *identity.AccountService
	consumeTokenService *identity.ConsumeTokenService
}

type register struct {
	Token    *string `form:"token" json:"token" xml:"token"`
	Code     *string `form:"code" json:"code" xml:"code"`
	User     string  `form:"user" json:"user" xml:"user"  binding:"required"`
	Password string  `form:"password" json:"password" xml:"password" binding:"required"`
	Email    *string `form:"email" json:"email" xml:"email"`
}

type login struct {
	User     string `form:"user" json:"user" xml:"user"  binding:"required"`
	Password string `form:"password" json:"password" xml:"password" binding:"required"`
}

type requierCode struct {
	Email string `form:"email" json:"email" xml:"email" binding:"required"`
}

func NewUserHandler(sessionService *identity.SessionService, accountService *identity.AccountService, consumeTokenService *identity.ConsumeTokenService) *UserHandler {
	return &UserHandler{sessionService, accountService, consumeTokenService}
}

func (h *UserHandler) RequireCode(c *gin.Context) {
	var json requierCode
	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, _, ipNet, err := iputils.ProcessIP(c.ClientIP())
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	err = h.accountService.RequireCode(c.Request.Context(), json.Email, ipNet.String())
	if err != nil {
		if err.Error() == "Rate limit exceeded" {
			c.String(http.StatusTooManyRequests, "Rate limit exceeded")
			return
		}
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Verification code has sent(TTL:15min). Check your email(including spam, from ktauth.vvan.win)."})
}

func (h *UserHandler) RegisterUser(c *gin.Context) {
	var json register
	var email string
	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if json.Token != nil {
		if !h.consumeTokenService.Consume(c.Request.Context(), *json.Token) {
			c.JSON(http.StatusUnauthorized, gin.H{"status": "unauthorized"})
			return
		}
	} else if json.Code != nil && json.Email != nil {
		valid, err := h.accountService.VerifyCode(c.Request.Context(), *json.Email, *json.Code)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if !valid {
			c.JSON(http.StatusUnauthorized, gin.H{"status": "unauthorized"})
			return
		}
	} else {
		c.String(http.StatusBadRequest, "missing token or email+code")
		return
	}

	if json.Email == nil {
		email = ""
	} else {
		email = *json.Email
	}

	uuid, err := h.accountService.NewUser(c.Request.Context(), json.User, json.Password, email)
	if err != nil {
		fmt.Println("Register new user failed:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"your uuid": uuid})
}

func (h *UserHandler) LoginUser(c *gin.Context) {
	var json login
	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.accountService.GetUserByName(c.Request.Context(), json.User)

	if err != nil {
		slog.Error(err.Error())
		c.String(http.StatusUnauthorized, "Incorrect password or username...")
		return
	}

	if !crypto.VerifyPassword(user.PasswordHash, json.Password) {
		c.String(http.StatusUnauthorized, "Incorrect password or username...")
		return
	}

	tokenStr, jti, err := auth.SignToken(user.UUID, user.Name)

	if err != nil {
		c.String(http.StatusInternalServerError, "Server error")
		return
	}

	err = h.sessionService.CreateSession(c.Request.Context(), user.UUID, jti)

	if err != nil {
		c.String(http.StatusInternalServerError, "Server error")
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": tokenStr})
}

func (h *UserHandler) LogoutUser(c *gin.Context) {
	jti := c.GetString("jti")
	uuid := c.GetString("uuid")
	err := h.sessionService.DelSession(c.Request.Context(), uuid, jti)
	if err != nil {
		c.String(http.StatusInternalServerError, "Server error")
		return
	}
	c.String(http.StatusOK, "OK")
}
