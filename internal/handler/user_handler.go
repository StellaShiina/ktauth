package handler

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/StellaShiina/ktauth/internal/auth"
	"github.com/StellaShiina/ktauth/internal/crypto"
	"github.com/StellaShiina/ktauth/internal/service/identity"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	sessionService      *identity.SessionService
	accountService      *identity.AccountService
	consumeTokenService *identity.ConsumeTokenService
	emailService        *identity.EmailService
}

type register struct {
	Token    *string `form:"token" json:"token" xml:"token"`
	User     string  `form:"user" json:"user" xml:"user"  binding:"required"`
	Password string  `form:"password" json:"password" xml:"password" binding:"required"`
	Email    *string `form:"email" json:"email" xml:"email"`
	Code     *string `form:"code" json:"code" xml:"code"`
}

type login struct {
	User     string `form:"user" json:"user" xml:"user"  binding:"required"`
	Password string `form:"password" json:"password" xml:"password" binding:"required"`
}

type emailCode struct {
	Email string `form:"email" json:"email" xml:"email" binding:"required,email"`
	Code  string `form:"code" json:"code" xml:"code"`
}

func NewUserHandler(sessionService *identity.SessionService, accountService *identity.AccountService, consumeTokenService *identity.ConsumeTokenService, emailService ...*identity.EmailService) *UserHandler {
	var es *identity.EmailService
	if len(emailService) > 0 {
		es = emailService[0]
	}
	return &UserHandler{sessionService, accountService, consumeTokenService, es}
}

func (h *UserHandler) SendEmailCode(c *gin.Context) {
	if h.emailService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "SMTP is not configured"})
		return
	}
	var req emailCode
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.emailService.SendCode(c.Request.Context(), req.Email); err != nil {
		switch err.Error() {
		case "verification code recently sent":
			c.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
			return
		case "invalid email address":
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "verification code sent"})
}

func (h *UserHandler) VerifyEmailCode(c *gin.Context) {
	if h.emailService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "SMTP is not configured"})
		return
	}
	var req emailCode
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
		return
	}
	valid, err := h.emailService.VerifyCode(c.Request.Context(), req.Email, req.Code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"valid": false})
		return
	}
	c.JSON(http.StatusOK, gin.H{"valid": true})
}

func (h *UserHandler) RegisterUser(c *gin.Context) {
	var json register
	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if json.Token != nil && *json.Token != "" {
		if !h.consumeTokenService.Consume(c.Request.Context(), *json.Token) {
			c.JSON(http.StatusUnauthorized, gin.H{"status": "unauthorized"})
			return
		}
	} else {
		if json.Email == nil || *json.Email == "" || json.Code == nil || *json.Code == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "token or email and code are required"})
			return
		}
		if h.emailService == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "SMTP is not configured"})
			return
		}
		valid, err := h.emailService.VerifyCode(c.Request.Context(), *json.Email, *json.Code)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if !valid {
			c.JSON(http.StatusUnauthorized, gin.H{"status": "unauthorized"})
			return
		}
	}

	uuid, err := h.accountService.NewUser(c.Request.Context(), json.User, json.Password, json.Email, "user")
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

	tokenStr, jti, err := auth.SignToken(user.UUID, user.Name, user.Role)

	if err != nil {
		c.String(http.StatusInternalServerError, "Server error")
		return
	}

	err = h.sessionService.CreateSession(c.Request.Context(), user.UUID, jti)

	if err != nil {
		c.String(http.StatusInternalServerError, "Server error")
		return
	}

	if c.Query("format") == "string" {
		c.String(http.StatusOK, tokenStr)
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
