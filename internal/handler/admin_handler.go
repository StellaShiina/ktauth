package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/StellaShiina/ktauth/internal/repository"
	"github.com/StellaShiina/ktauth/internal/service/admin"
	"github.com/StellaShiina/ktauth/pkg/iputils"
	"github.com/gin-gonic/gin"
)

var ipe *iputils.IPError

type IPRuleManager interface {
	AddRule(ctx context.Context, ipStr string, isWhiteList bool, note *string) (string, error)
	ListRules(ctx context.Context, version *int16, isWhiteList *bool) ([]admin.IPResponse, error)
	DelRule(ctx context.Context, ipStr string) (string, error)
}

type UserManager interface {
	ListUsers(ctx context.Context) ([]admin.UserResponse, error)
}

type rule struct {
	IP          string  `json:"ip"`
	Note        *string `json:"note"`
	IsWhiteList *bool   `json:"isWhiteList"`
}

type IPRuleHandler struct {
	ipRuleManager IPRuleManager
}

type UserManageHandler struct {
	userManager UserManager
}

func NewIPRuleHandler(ipRuleManager IPRuleManager) *IPRuleHandler {
	return &IPRuleHandler{ipRuleManager}
}

func NewUserManageHandler(userManager UserManager) *UserManageHandler {
	return &UserManageHandler{userManager}
}

func (h *IPRuleHandler) AddRule(c *gin.Context) {
	var json rule
	if err := c.ShouldBindJSON(&json); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	var isWhiteList bool
	isBan := c.Query("ban") // legacy, may be removed in future
	if isBan != "" || (json.IsWhiteList != nil && *json.IsWhiteList == false) {
		isWhiteList = false
	} else {
		isWhiteList = true
	}

	cidr, err := h.ipRuleManager.AddRule(c.Request.Context(), json.IP, isWhiteList, json.Note)

	if err != nil {
		if errors.As(err, &ipe) {
			c.String(http.StatusBadRequest, err.Error())
			return
		} else if err == repository.ErrIPExist {
			c.String(http.StatusBadRequest, err.Error())
			return
		} else {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
	}

	c.String(http.StatusOK, "CIDR "+cidr+" added.")
}

func (h *IPRuleHandler) ListRules(c *gin.Context) {
	var version *int16
	var isWhiteList *bool
	versionStr := c.Query("version")
	typeStr := c.Query("type")
	switch versionStr {
	case "4":
		versionInt := int16(4)
		version = &versionInt
	case "6":
		versionInt := int16(6)
		version = &versionInt
	}
	switch typeStr {
	case "white":
		typeBool := true
		isWhiteList = &typeBool
	case "black":
		typeBool := false
		isWhiteList = &typeBool
	}
	rules, err := h.ipRuleManager.ListRules(c.Request.Context(), version, isWhiteList)
	if err != nil {
		c.String(http.StatusInternalServerError, "Server error...")
		return
	}
	c.JSON(http.StatusOK, gin.H{"rules": rules})
}

func (h *IPRuleHandler) DelRule(c *gin.Context) {
	var json rule
	if err := c.ShouldBindJSON(&json); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	cidr, err := h.ipRuleManager.DelRule(c.Request.Context(), json.IP)
	if err != nil {
		if errors.As(err, &ipe) {
			c.String(http.StatusBadRequest, err.Error())
			return
		} else if err == repository.ErrIPNotFound {
			c.String(http.StatusBadRequest, err.Error())
			return
		}
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.String(http.StatusOK, "CIDR "+cidr+" deleted.")
}

func (h *UserManageHandler) ListUsers(c *gin.Context) {
	users, err := h.userManager.ListUsers(c.Request.Context())
	if err != nil {
		c.String(http.StatusInternalServerError, "Server error...")
		slog.Error("Error when listing users", "error", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"users": users})
}
