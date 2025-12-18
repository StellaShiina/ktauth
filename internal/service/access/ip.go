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

type IPAccessService struct {
	ipRepo  *repository.IPRepo
	ipCache *repository.IPCache
}

func NewIPAccessService(r *repository.IPRepo, c *repository.IPCache) *IPAccessService {
	return &IPAccessService{r, c}
}

// Verify whether the given ip is in the whitelist. Error when parsing IP or accessing repo.
func (s *IPAccessService) VerifyWhileList(c context.Context, ipStr string) (bool, error) {
	// 解析完后原始IP字符串就只用作redis储存的key，如果是ipv6会被修改为cidr
	ip := net.ParseIP(ipStr)

	if ip == nil {
		return false, fmt.Errorf("Invalid IP")
	}

	ipStr, err := iputils.IPv6ToCIDR64String(ip)

	ruleStr, err := s.ipCache.Get(c, ipStr)

	if err != nil && err.Error() != "Cache not found" {
		slog.Error("Redis error, fail to access cached rules")
	} else if err == nil {
		slog.Debug("Cached rule")
		rule_type := model.IPRuleType(ruleStr)
		switch rule_type {
		case model.IPWhiteList:
			return true, nil
		default:
			return false, nil
		}
	}

	slog.Debug("Not cached rule")
	whiteList, err := s.ipRepo.GetIPsByType(c, model.IPWhiteList)
	if err != nil {
		return false, fmt.Errorf("Error when getting whitelist: %v", err)
	}
	for _, wip := range whiteList {
		_, cidr, err := net.ParseCIDR(wip)
		if err != nil {
			fmt.Println("Error parsing whitelist cidr in db...")
			continue
		}
		if cidr.Contains(ip) {
			err := s.ipCache.Cache(c, model.IPWhiteList, ipStr)
			if err != nil {
				slog.Error(err.Error())
			}
			return true, nil
		}
	}
	err = s.ipCache.Cache(c, model.IPGreyList, ipStr)
	if err != nil {
		slog.Error(err.Error())
	}
	return false, nil
}
