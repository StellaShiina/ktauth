package admin

import (
	"context"
	"log/slog"
	"net"

	"github.com/StellaShiina/ktauth/internal/model"
	"github.com/StellaShiina/ktauth/pkg/iputils"
)

type IPRuleStore interface {
	AddIP(ctx context.Context, version int16, ipRange *net.IPNet, isWhitelist bool, note *string) error
	DelIP(ctx context.Context, version int16, ipRange *net.IPNet) error
	GetIPs(ctx context.Context, version *int16, isWhiteList *bool) ([]model.IP, error)
}

type IPRuleCacheInvalidator interface {
	Delete(ctx context.Context, ip string) error
}

type RateLimitInvalidator interface {
	Delete(ctx context.Context, ip string) error
}

type AdminIPRuleService struct {
	store IPRuleStore
	cache IPRuleCacheInvalidator
	rlInv RateLimitInvalidator
}

func NewAdminIPRuleService(store IPRuleStore, cache IPRuleCacheInvalidator, rlInv RateLimitInvalidator) *AdminIPRuleService {
	return &AdminIPRuleService{store, cache, rlInv}
}

// Return cidr string, err error
func (s *AdminIPRuleService) AddRule(c context.Context, ipStr string, isWhiteList bool, note *string) (string, error) {
	version, _, ipNet, err := iputils.ProcessIP(ipStr)
	if err != nil {
		return "", err
	}
	err = s.store.AddIP(c, version, ipNet, isWhiteList, note)
	if err == nil {
		if err := s.cache.Delete(c, ipNet.String()); err != nil {
			slog.Error("Failed to delete cached rule", "error", err)
		}
		if err := s.rlInv.Delete(c, ipNet.String()); err != nil {
			slog.Error("Failed to delete ratelimit record", "error", err)
		}
	}
	return ipNet.String(), err
}

func (s *AdminIPRuleService) ListRules(c context.Context, version *int16, isWhiteList *bool) ([]IPResponse, error) {
	var ipres []IPResponse
	data, err := s.store.GetIPs(c, version, isWhiteList)
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
		if err := s.cache.Delete(c, ipNet.String()); err != nil {
			slog.Error("Failed to delete cached rule", "error", err)
		}
		if err := s.rlInv.Delete(c, ipNet.String()); err != nil {
			slog.Error("Failed to delete ratelimit record", "error", err)
		}
	}
	return ipNet.String(), s.store.DelIP(c, version, ipNet)
}
