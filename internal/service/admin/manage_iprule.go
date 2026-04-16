package admin

import (
	"context"

	"github.com/StellaShiina/ktauth/internal/model"
	"github.com/StellaShiina/ktauth/internal/repository"
	"github.com/StellaShiina/ktauth/pkg/iputils"
)

type AdminIPRuleService struct {
	ipRepo *repository.IPRepo
}

func NewAdminIPRuleService(ipRepo *repository.IPRepo) *AdminIPRuleService {
	return &AdminIPRuleService{ipRepo: ipRepo}
}

// Return cidr string, err error
func (s *AdminIPRuleService) AddRule(c context.Context, ipStr string, isWhiteList bool) (string, error) {
	version, _, ipNet, err := iputils.ProcessIP(ipStr)
	if err != nil {
		return "", err
	}
	err = s.ipRepo.AddIP(c, version, ipNet, isWhiteList)
	return ipNet.String(), err
}

func (s *AdminIPRuleService) ListRules(c context.Context) ([]model.IP, error) {
	return s.ipRepo.GetIPs(c)
}

func (s *AdminIPRuleService) DelRule(c context.Context, ipStr string) error {
	version, _, ipNet, err := iputils.ProcessIP(ipStr)
	if err != nil {
		return err
	}
	return s.ipRepo.DelIP(c, version, ipNet)
}
