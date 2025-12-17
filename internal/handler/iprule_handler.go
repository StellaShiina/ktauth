package handler

import (
	"errors"
	"net/http"

	"github.com/StellaShiina/ktauth/internal/model"
	"github.com/StellaShiina/ktauth/internal/service/admin"
	"github.com/StellaShiina/ktauth/pkg/iputils"
	"github.com/gin-gonic/gin"
)

type rule struct {
	IP string `json:"ip"`
}

type IPRuleHandler struct {
	adminIPRuleService *admin.AdminIPRuleService
}

func NewIPRuleHandler(s *admin.AdminIPRuleService) *IPRuleHandler {
	return &IPRuleHandler{s}
}

func (h *IPRuleHandler) AddRule(c *gin.Context) {
	var json rule
	if err := c.ShouldBindJSON(&json); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	var rule_type model.IPRuleType
	isBan := c.Query("ban")
	if isBan != "" {
		rule_type = model.IPBlackList
	} else {
		rule_type = model.IPWhiteList
	}

	cidr, err := h.adminIPRuleService.AddRule(c.Request.Context(), json.IP, rule_type)

	if err != nil {
		var ipe *iputils.IPError
		if errors.As(err, &ipe) {
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
	err := h.adminIPRuleService.DelRule(c.Request.Context(), json.IP)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.String(http.StatusOK, "OK")
}
