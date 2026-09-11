package handler

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/StellaShiina/ktauth/internal/logctx"
	"github.com/gin-gonic/gin"
)

type AdminTokenManager interface {
	Restock(ctx context.Context) error
	FlushTokens(ctx context.Context) error
	GetToken(ctx context.Context) (string, error)
	GetTokens(ctx context.Context) ([]string, error)
}

type TokenHandler struct {
	adminTokenManager AdminTokenManager
}

func NewTokenHandler(s AdminTokenManager) *TokenHandler {
	return &TokenHandler{s}
}

func (h *TokenHandler) Restock(c *gin.Context) {
	err := h.adminTokenManager.Restock(c.Request.Context())
	if err != nil {
		slog.Error("failed to restock tokens", "error", err)
		logctx.SetReasonDetail(c, logctx.ReasonRestockTokensFailed, err)
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.String(http.StatusCreated, "Restock OK!")
}

func (h *TokenHandler) FlushTokens(c *gin.Context) {
	err := h.adminTokenManager.FlushTokens(c.Request.Context())
	if err != nil {
		slog.Error("failed to flush tokens", "error", err)
		logctx.SetReasonDetail(c, logctx.ReasonFlushTokensFailed, err)
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.String(http.StatusOK, "OK")
}

func (h *TokenHandler) GetToken(c *gin.Context) {
	token, err := h.adminTokenManager.GetToken(c.Request.Context())
	if err != nil {
		slog.Error("failed to get token", "error", err)
		c.String(http.StatusOK, err.Error())
		return
	}
	c.String(http.StatusOK, token)
}

func (h *TokenHandler) GetTokens(c *gin.Context) {
	tokens, err := h.adminTokenManager.GetTokens(c.Request.Context())
	if err != nil {
		slog.Error("failed to get tokens", "error", err)
		c.String(http.StatusOK, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"tokens": tokens})
}
