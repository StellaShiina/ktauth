package handler

import (
	"net/http"

	"github.com/StellaShiina/ktauth/internal/service/admin"
	"github.com/gin-gonic/gin"
)

type TokenHandler struct {
	adminTokenService *admin.AdminTokenService
}

func NewTokenHandler(s *admin.AdminTokenService) *TokenHandler {
	return &TokenHandler{s}
}

func (h *TokenHandler) Restock(c *gin.Context) {
	err := h.adminTokenService.Restock(c.Request.Context())
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.String(http.StatusCreated, "Restock OK!")
}

func (h *TokenHandler) FlushTokens(c *gin.Context) {
	err := h.adminTokenService.FlushTokens(c.Request.Context())
	if err != nil {
		c.String(http.StatusOK, err.Error())
		return
	}
	c.String(http.StatusOK, "OK")
}

func (h *TokenHandler) GetToken(c *gin.Context) {
	token, err := h.adminTokenService.GetToken(c.Request.Context())
	if err != nil {
		c.String(http.StatusOK, err.Error())
		return
	}
	c.String(http.StatusOK, token)
}

func (h *TokenHandler) GetTokens(c *gin.Context) {
	tokens, err := h.adminTokenService.GetTokens(c.Request.Context())
	if err != nil {
		c.String(http.StatusOK, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"tokens": tokens})
}
