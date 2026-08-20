package access

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"github.com/StellaShiina/ktauth/internal/model"
	"github.com/StellaShiina/ktauth/internal/repository"
	"github.com/StellaShiina/ktauth/pkg/iputils"
)

type IPRuleReader interface {
	QueryIP(ctx context.Context, version int16, clientIP net.IP) (bool, error)
}

type IPRuleCache interface {
	Cache(c context.Context, rule_type model.IPRuleType, ips ...string) error
	Get(c context.Context, ip string) (string, error)
}

type IPAccessService struct {
	ipRuleReader IPRuleReader
	ipRuleCache  IPRuleCache
}

func NewIPAccessService(r IPRuleReader, c IPRuleCache) *IPAccessService {
	return &IPAccessService{r, c}
}

// Return rule_type, error
func (s *IPAccessService) QueryRule(c context.Context, ipStr string) (model.IPRuleType, error) {
	var rule_type model.IPRuleType

	version, ip, ipNet, err := iputils.ProcessIP(ipStr)

	if err != nil {
		return "", fmt.Errorf("Invalid IP")
	}

	ruleStr, err := s.ipRuleCache.Get(c, ipNet.String())

	if err != nil && err.Error() != "Cache not found" {
		slog.Error("Redis error, fail to access cached rules")
	} else if err == nil {
		slog.Debug("Cached rule", "ip", ipNet.String(), "rule", ruleStr)
		return model.IPRuleType(ruleStr), nil
	}

	isWhitelist, err := s.ipRuleReader.QueryIP(c, version, ip)

	if err != nil {
		if err == repository.ErrIPNotFound {
			slog.Debug("Cache not hit, greylist", "ip", ip.String())
			rule_type = model.IPGreyList
			err = s.ipRuleCache.Cache(c, model.IPGreyList, ipNet.String())
		} else {
			return "", fmt.Errorf("Error when getting ip_rule from db: %v", err)
		}
	} else {
		if isWhitelist {
			slog.Debug("Cache not hit, whitelist", "ip", ip.String())
			rule_type = model.IPWhiteList
			err = s.ipRuleCache.Cache(c, model.IPWhiteList, ipNet.String())
		} else {
			slog.Debug("Cache not hit, blacklist", "ip", ip.String())
			rule_type = model.IPBlackList
			err = s.ipRuleCache.Cache(c, model.IPBlackList, ipNet.String())
		}
	}
	if err != nil {
		slog.Error(err.Error())
	}
	return rule_type, nil
}
