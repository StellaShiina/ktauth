package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/StellaShiina/ktauth/internal/repository"
	"github.com/StellaShiina/ktauth/internal/service/admin"
	"github.com/StellaShiina/ktauth/pkg/iputils"
	"github.com/gin-gonic/gin"
)

var ipe *iputils.IPError

type rule struct {
	IP   string  `json:"ip"`
	Note *string `json:"note"`
}

type IPRuleHandler struct {
	adminIPRuleService *admin.AdminIPRuleService
}

type UserManageHandler struct {
	userManageService *admin.UserManageService
}

func NewIPRuleHandler(adminIPRuleService *admin.AdminIPRuleService) *IPRuleHandler {
	return &IPRuleHandler{adminIPRuleService}
}

func NewUserManageHandler(userManageService *admin.UserManageService) *UserManageHandler {
	return &UserManageHandler{userManageService}
}

func (h *IPRuleHandler) AddRule(c *gin.Context) {
	var json rule
	if err := c.ShouldBindJSON(&json); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	var isWhiteList bool
	isBan := c.Query("ban")
	if isBan != "" {
		isWhiteList = false
	} else {
		isWhiteList = true
	}

	cidr, err := h.adminIPRuleService.AddRule(c.Request.Context(), json.IP, isWhiteList, json.Note)

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
	rules, err := h.adminIPRuleService.ListRules(c.Request.Context())
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
	cidr, err := h.adminIPRuleService.DelRule(c.Request.Context(), json.IP)
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
	users, err := h.userManageService.ListUsers(c.Request.Context())
	if err != nil {
		c.String(http.StatusInternalServerError, "Server error...")
		slog.Error("Error when listing users", "error", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"users": users})
}
