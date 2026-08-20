package access_test

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"

	"github.com/StellaShiina/ktauth/internal/model"
	"github.com/StellaShiina/ktauth/internal/repository"
	"github.com/StellaShiina/ktauth/internal/service/access"
)

type ipReaderMock struct {
	queried  bool
	version  int16
	clientIP net.IP
	result   bool
	err      error
}

func (m *ipReaderMock) QueryIP(_ context.Context, version int16, clientIP net.IP) (bool, error) {
	m.queried = true
	m.version = version
	m.clientIP = clientIP
	return m.result, m.err
}

type cacheCall struct {
	rule model.IPRuleType
	ips  []string
}

type ipCacheMock struct {
	rules    map[string]string
	getErr   error
	cacheErr error
	calls    []cacheCall
}

func (m *ipCacheMock) Get(_ context.Context, ip string) (string, error) {
	if m.getErr != nil {
		return "", m.getErr
	}
	if rule, ok := m.rules[ip]; ok {
		return rule, nil
	}
	return "", errors.New("Cache not found")
}

func (m *ipCacheMock) Cache(_ context.Context, rule model.IPRuleType, ips ...string) error {
	m.calls = append(m.calls, cacheCall{rule: rule, ips: append([]string(nil), ips...)})
	return m.cacheErr
}

func TestIPAccessServiceQueryRuleCacheHit(t *testing.T) {
	reader := &ipReaderMock{}
	cache := &ipCacheMock{rules: map[string]string{"192.0.2.0/24": string(model.IPBlackList)}}
	service := access.NewIPAccessService(reader, cache)

	rule, err := service.QueryRule(context.Background(), "192.0.2.18/24")
	if err != nil {
		t.Fatalf("QueryRule returned error: %v", err)
	}
	if rule != model.IPBlackList {
		t.Fatalf("rule = %q, want %q", rule, model.IPBlackList)
	}
	if reader.queried {
		t.Fatal("database was queried after a cache hit")
	}
	if len(cache.calls) != 0 {
		t.Fatalf("cache writes = %d, want 0", len(cache.calls))
	}
}

func TestIPAccessServiceQueryRuleDatabaseResult(t *testing.T) {
	tests := []struct {
		name      string
		whitelist bool
		wantRule  model.IPRuleType
	}{
		{name: "whitelist", whitelist: true, wantRule: model.IPWhiteList},
		{name: "blacklist", whitelist: false, wantRule: model.IPBlackList},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := &ipReaderMock{result: tt.whitelist}
			cache := &ipCacheMock{}
			service := access.NewIPAccessService(reader, cache)

			rule, err := service.QueryRule(context.Background(), "192.0.2.18/24")
			if err != nil {
				t.Fatalf("QueryRule returned error: %v", err)
			}
			if rule != tt.wantRule {
				t.Fatalf("rule = %q, want %q", rule, tt.wantRule)
			}
			if reader.version != 4 || !reader.clientIP.Equal(net.ParseIP("192.0.2.18")) {
				t.Fatalf("database query = version %d, ip %s; want version 4, ip 192.0.2.18", reader.version, reader.clientIP)
			}
			if len(cache.calls) != 1 || cache.calls[0].rule != tt.wantRule || len(cache.calls[0].ips) != 1 || cache.calls[0].ips[0] != "192.0.2.0/24" {
				t.Fatalf("cache calls = %#v, want one %q write for 192.0.2.0/24", cache.calls, tt.wantRule)
			}
		})
	}
}

func TestIPAccessServiceQueryRuleNotFoundIsGreylist(t *testing.T) {
	reader := &ipReaderMock{err: repository.ErrIPNotFound}
	cache := &ipCacheMock{}
	service := access.NewIPAccessService(reader, cache)

	rule, err := service.QueryRule(context.Background(), "2001:db8::1")
	if err != nil {
		t.Fatalf("QueryRule returned error: %v", err)
	}
	if rule != model.IPGreyList {
		t.Fatalf("rule = %q, want %q", rule, model.IPGreyList)
	}
	if len(cache.calls) != 1 || cache.calls[0].rule != model.IPGreyList || cache.calls[0].ips[0] != "2001:db8::/64" {
		t.Fatalf("cache calls = %#v, want greylist for 2001:db8::/64", cache.calls)
	}
}

func TestIPAccessServiceQueryRuleReturnsDatabaseError(t *testing.T) {
	dbErr := errors.New("database unavailable")
	reader := &ipReaderMock{err: dbErr}
	cache := &ipCacheMock{}
	service := access.NewIPAccessService(reader, cache)

	_, err := service.QueryRule(context.Background(), "192.0.2.1")
	if err == nil || !strings.Contains(err.Error(), dbErr.Error()) {
		t.Fatalf("error = %v, want wrapped database error", err)
	}
	if len(cache.calls) != 0 {
		t.Fatal("cache was written after a database error")
	}
}

func TestIPAccessServiceQueryRuleRejectsInvalidIP(t *testing.T) {
	reader := &ipReaderMock{}
	cache := &ipCacheMock{}
	service := access.NewIPAccessService(reader, cache)

	if _, err := service.QueryRule(context.Background(), "not-an-ip"); err == nil {
		t.Fatal("QueryRule accepted an invalid IP")
	}
	if reader.queried || len(cache.calls) != 0 {
		t.Fatal("dependencies were called for an invalid IP")
	}
}
