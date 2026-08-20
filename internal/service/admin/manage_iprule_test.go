package admin_test

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/StellaShiina/ktauth/internal/model"
	"github.com/StellaShiina/ktauth/internal/repository"
	"github.com/StellaShiina/ktauth/internal/service/admin"
)

type ipRuleStoreMock struct {
	addCalls    int
	delCalls    int
	listCalls   int
	version     int16
	ipRange     *net.IPNet
	isWhitelist bool
	note        *string
	listVersion *int16
	listWhite   *bool
	addErr      error
	delErr      error
	listErr     error
	users       []model.IP
}

func (m *ipRuleStoreMock) AddIP(_ context.Context, version int16, ipRange *net.IPNet, isWhitelist bool, note *string) error {
	m.addCalls++
	m.version, m.ipRange, m.isWhitelist, m.note = version, ipRange, isWhitelist, note
	return m.addErr
}

func (m *ipRuleStoreMock) DelIP(_ context.Context, version int16, ipRange *net.IPNet) error {
	m.delCalls++
	m.version, m.ipRange = version, ipRange
	return m.delErr
}

func (m *ipRuleStoreMock) GetIPs(_ context.Context, version *int16, isWhiteList *bool) ([]model.IP, error) {
	m.listCalls++
	m.listVersion, m.listWhite = version, isWhiteList
	return m.users, m.listErr
}

type deleteMock struct {
	calls []string
	err   error
}

func (m *deleteMock) Delete(_ context.Context, ip string) error {
	m.calls = append(m.calls, ip)
	return m.err
}

func TestAdminIPRuleServiceAddRule(t *testing.T) {
	note := "test note"
	store := &ipRuleStoreMock{}
	cache := &deleteMock{}
	rateLimit := &deleteMock{}
	service := admin.NewAdminIPRuleService(store, cache, rateLimit)

	cidr, err := service.AddRule(context.Background(), "192.0.2.18", true, &note)
	if err != nil || cidr != "192.0.2.18/32" {
		t.Fatalf("AddRule = %q, %v; want 192.0.2.18/32, nil", cidr, err)
	}
	if store.addCalls != 1 || store.version != 4 || store.ipRange.String() != cidr || !store.isWhitelist || store.note != &note {
		t.Fatalf("store call = %#v", store)
	}
	if len(cache.calls) != 1 || cache.calls[0] != cidr || len(rateLimit.calls) != 1 || rateLimit.calls[0] != cidr {
		t.Fatalf("invalidation calls = cache %#v, rate limit %#v", cache.calls, rateLimit.calls)
	}
}

func TestAdminIPRuleServiceAddRuleDoesNotInvalidateOnStoreError(t *testing.T) {
	storeErr := errors.New("database unavailable")
	store := &ipRuleStoreMock{addErr: storeErr}
	cache := &deleteMock{}
	rateLimit := &deleteMock{}
	service := admin.NewAdminIPRuleService(store, cache, rateLimit)

	_, err := service.AddRule(context.Background(), "192.0.2.18", false, nil)
	if !errors.Is(err, storeErr) {
		t.Fatalf("error = %v, want store error", err)
	}
	if len(cache.calls) != 0 || len(rateLimit.calls) != 0 {
		t.Fatal("state was invalidated after a failed add")
	}
}

func TestAdminIPRuleServiceDelRuleInvalidatesBeforeDelete(t *testing.T) {
	store := &ipRuleStoreMock{}
	cache := &deleteMock{}
	rateLimit := &deleteMock{}
	service := admin.NewAdminIPRuleService(store, cache, rateLimit)

	cidr, err := service.DelRule(context.Background(), "2001:db8::1")
	if err != nil || cidr != "2001:db8::/64" {
		t.Fatalf("DelRule = %q, %v; want 2001:db8::/64, nil", cidr, err)
	}
	if store.delCalls != 1 || store.version != 6 || store.ipRange.String() != cidr {
		t.Fatalf("store call = %#v", store)
	}
	if len(cache.calls) != 1 || cache.calls[0] != cidr || len(rateLimit.calls) != 1 || rateLimit.calls[0] != cidr {
		t.Fatalf("invalidation calls = cache %#v, rate limit %#v", cache.calls, rateLimit.calls)
	}
}

func TestAdminIPRuleServiceListRulesMapsData(t *testing.T) {
	note := "allow office"
	_, network, err := net.ParseCIDR("192.0.2.0/24")
	if err != nil {
		t.Fatal(err)
	}
	version := int16(4)
	isWhitelist := true
	created := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	updated := created.Add(time.Hour)
	store := &ipRuleStoreMock{users: []model.IP{
		{ID: 1, Version: 4, IPRange: network, IsWhitelist: true, CreateAt: created, UpdateAt: updated, Note: &note},
		{ID: 2, Version: 4, IPRange: network, IsWhitelist: false},
	}}
	service := admin.NewAdminIPRuleService(store, &deleteMock{}, &deleteMock{})

	rules, err := service.ListRules(context.Background(), &version, &isWhitelist)
	if err != nil {
		t.Fatalf("ListRules returned error: %v", err)
	}
	if store.listCalls != 1 || store.listVersion != &version || store.listWhite != &isWhitelist {
		t.Fatalf("list call = %#v", store)
	}
	if len(rules) != 2 || rules[0].IPCIDR != "192.0.2.0/24" || rules[0].Note != note || rules[1].Note != "" || rules[1].IsWhitelist {
		t.Fatalf("rules = %#v", rules)
	}
	if !rules[0].CreateAt.Equal(created) || !rules[0].UpdateAt.Equal(updated) {
		t.Fatalf("timestamps were not mapped: %#v", rules[0])
	}
}

func TestAdminIPRuleServiceRejectsInvalidIP(t *testing.T) {
	store := &ipRuleStoreMock{}
	cache := &deleteMock{}
	rateLimit := &deleteMock{}
	service := admin.NewAdminIPRuleService(store, cache, rateLimit)

	if _, err := service.AddRule(context.Background(), "not-an-ip", false, nil); err == nil {
		t.Fatal("AddRule accepted an invalid IP")
	}
	if _, err := service.DelRule(context.Background(), "not-an-ip"); err == nil {
		t.Fatal("DelRule accepted an invalid IP")
	}
	if store.addCalls != 0 || store.delCalls != 0 || len(cache.calls) != 0 || len(rateLimit.calls) != 0 {
		t.Fatal("dependencies were called for an invalid IP")
	}
}

func TestAdminIPRuleServiceListRulesReturnsStoreError(t *testing.T) {
	storeErr := errors.New("database unavailable")
	service := admin.NewAdminIPRuleService(&ipRuleStoreMock{listErr: storeErr}, &deleteMock{}, &deleteMock{})

	rules, err := service.ListRules(context.Background(), nil, nil)
	if !errors.Is(err, storeErr) {
		t.Fatalf("error = %v, want store error", err)
	}
	if rules != nil {
		t.Fatalf("rules = %#v, want nil", rules)
	}
}

func TestAdminIPRuleServiceDeleteErrorIsReturned(t *testing.T) {
	storeErr := repository.ErrIPNotFound
	service := admin.NewAdminIPRuleService(&ipRuleStoreMock{delErr: storeErr}, &deleteMock{}, &deleteMock{})

	cidr, err := service.DelRule(context.Background(), "192.0.2.18")
	if cidr != "192.0.2.18/32" || !errors.Is(err, storeErr) {
		t.Fatalf("DelRule = %q, %v; want normalized CIDR and delete error", cidr, err)
	}
}
