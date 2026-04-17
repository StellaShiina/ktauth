package admin

import (
	"context"
	"log/slog"

	"github.com/StellaShiina/ktauth/internal/repository"
	"github.com/StellaShiina/ktauth/pkg/iputils"
)

type AdminIPRuleService struct {
	ipRepo        *repository.IPRepo
	ipCache       *repository.IPCache
	rateLimitRepo *repository.RateLimitRepo
}

func NewAdminIPRuleService(ipRepo *repository.IPRepo, ipCache *repository.IPCache, rateLimitRepo *repository.RateLimitRepo) *AdminIPRuleService {
	return &AdminIPRuleService{ipRepo, ipCache, rateLimitRepo}
}

// Return cidr string, err error
func (s *AdminIPRuleService) AddRule(c context.Context, ipStr string, isWhiteList bool, note *string) (string, error) {
	version, _, ipNet, err := iputils.ProcessIP(ipStr)
	if err != nil {
		return "", err
	}
	err = s.ipRepo.AddIP(c, version, ipNet, isWhiteList, note)
	if err == nil {
		if err := s.ipCache.Delete(c, ipNet.String()); err != nil {
			slog.Error("Failed to delete cached rule", "error", err)
		}
		if err := s.rateLimitRepo.Delete(c, ipNet.String()); err != nil {
			slog.Error("Failed to delete ratelimit record", "error", err)
		}
	}
	return ipNet.String(), err
}

func (s *AdminIPRuleService) ListRules(c context.Context) ([]IPResponse, error) {
	var ipres []IPResponse
	data, err := s.ipRepo.GetIPs(c)
	if err != nil {
		return nil, err
	}
	for _, ip := range data {
		note := ""
		if ip.Note != nil {
			note = *ip.Note
		}
		ipres = append(ipres, IPResponse{
			ID:          ip.ID,
			Version:     ip.Version,
			IPCIDR:      ip.IPRange.String(),
			IsWhitelist: ip.IsWhitelist,
			CreateAt:    ip.CreateAt,
			UpdateAt:    ip.UpdateAt,
			Note:        note,
		})
	}
	return ipres, nil
}

func (s *AdminIPRuleService) DelRule(c context.Context, ipStr string) (string, error) {
	version, _, ipNet, err := iputils.ProcessIP(ipStr)
	if err != nil {
		return "", err
	} else {
		if err := s.ipCache.Delete(c, ipNet.String()); err != nil {
			slog.Error("Failed to delete cached rule", "error", err)
		}
		if err := s.rateLimitRepo.Delete(c, ipNet.String()); err != nil {
			slog.Error("Failed to delete ratelimit record", "error", err)
		}
	}
	return ipNet.String(), s.ipRepo.DelIP(c, version, ipNet)
}
