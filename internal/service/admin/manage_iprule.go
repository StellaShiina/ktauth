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
func (s *AdminIPRuleService) AddRule(c context.Context, ipStr string, rule_type model.IPRuleType) (string, error) {
	cidr, err := iputils.ProcessIPToCIDR(ipStr)
	if err != nil {
		return "", err
	}
	_, err = s.ipRepo.AddIP(c, cidr, rule_type)
	return cidr, err
}

func (s *AdminIPRuleService) ListRules(c context.Context) ([]model.IP, error) {
	return s.ipRepo.GetIPs(c)
}

func (s *AdminIPRuleService) DelRule(c context.Context, cidr string) error {
	return s.ipRepo.DelIP(c, cidr)
}
