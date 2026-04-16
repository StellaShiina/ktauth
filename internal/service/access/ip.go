package access

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/StellaShiina/ktauth/internal/model"
	"github.com/StellaShiina/ktauth/internal/repository"
	"github.com/StellaShiina/ktauth/pkg/iputils"
)

type IPAccessService struct {
	ipRepo  *repository.IPRepo
	ipCache *repository.IPCache
}

func NewIPAccessService(r *repository.IPRepo, c *repository.IPCache) *IPAccessService {
	return &IPAccessService{r, c}
}

// Return rule_type, error
func (s *IPAccessService) QueryRule(c context.Context, ipStr string) (model.IPRuleType, error) {
	var rule_type model.IPRuleType

	version, ip, _, err := iputils.ProcessIP(ipStr)

	if err != nil {
		return "", fmt.Errorf("Invalid IP")
	}

	ruleStr, err := s.ipCache.Get(c, ip.String())

	if err != nil && err.Error() != "Cache not found" {
		slog.Error("Redis error, fail to access cached rules")
	} else if err == nil {
		slog.Debug("Cached rule")
		return model.IPRuleType(ruleStr), nil
	}

	isWhitelist, err := s.ipRepo.QueryIP(c, version, ip)

	if err != nil {
		if err == repository.ErrIPNotFound {
			slog.Info("Cache not hit, greylist", "ip", ip.String())
			rule_type = model.IPGreyList
			err = s.ipCache.Cache(c, model.IPGreyList, ip.String())
		} else {
			return "", fmt.Errorf("Error when getting ip_rule from db: %v", err)
		}
	} else {
		if isWhitelist {
			slog.Info("Cache not hit, whitelist", "ip", ip.String())
			rule_type = model.IPWhiteList
			err = s.ipCache.Cache(c, model.IPWhiteList, ip.String())
		} else {
			slog.Info("Cache not hit, blacklist", "ip", ip.String())
			rule_type = model.IPBlackList
			err = s.ipCache.Cache(c, model.IPBlackList, ip.String())
		}
	}
	if err != nil {
		slog.Error(err.Error())
	}
	return rule_type, nil
}
