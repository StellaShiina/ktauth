package access_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/StellaShiina/ktauth/internal/service/access"
)

type rateReaderMock struct {
	allowCalls  int
	abuseCalls  int
	allowKey    string
	allowLimit  int
	allowWindow time.Duration
	abuseKey    string
	abuseLimit  int
	abuseWindow time.Duration
	allow       bool
	abuse       bool
	allowErr    error
	abuseErr    error
}

func (m *rateReaderMock) Allow(_ context.Context, key string, limit int, window time.Duration) (bool, error) {
	m.allowCalls++
	m.allowKey, m.allowLimit, m.allowWindow = key, limit, window
	return m.allow, m.allowErr
}

func (m *rateReaderMock) Abuse(_ context.Context, key string, limit int, window time.Duration) (bool, error) {
	m.abuseCalls++
	m.abuseKey, m.abuseLimit, m.abuseWindow = key, limit, window
	return m.abuse, m.abuseErr
}

func TestRateLimitServiceAllow(t *testing.T) {
	reader := &rateReaderMock{allow: true}
	service := access.NewRateLimitService(reader, 12, true, 3, 30*time.Second)

	allowed, err := service.Allow(context.Background(), "192.0.2.18")
	if err != nil || !allowed {
		t.Fatalf("Allow = %v, %v; want true, nil", allowed, err)
	}
	if reader.allowCalls != 1 || reader.allowKey != "192.0.2.18/32" || reader.allowLimit != 12 || reader.allowWindow != time.Minute {
		t.Fatalf("Allow dependency call = %#v, want normalized key and configured limit", reader)
	}
}

func TestRateLimitServiceDisabledSkipsReader(t *testing.T) {
	reader := &rateReaderMock{}
	service := access.NewRateLimitService(reader, 12, false, 3, time.Minute)

	allowed, err := service.Allow(context.Background(), "not-an-ip")
	if err != nil || !allowed {
		t.Fatalf("Allow = %v, %v; want true, nil when disabled", allowed, err)
	}
	if reader.allowCalls != 0 {
		t.Fatal("rate reader was called while rate limiting was disabled")
	}
}

func TestRateLimitServiceAllowWrapsReaderError(t *testing.T) {
	readerErr := errors.New("redis unavailable")
	reader := &rateReaderMock{allowErr: readerErr}
	service := access.NewRateLimitService(reader, 12, true, 3, time.Minute)

	_, err := service.Allow(context.Background(), "192.0.2.18")
	if err == nil || !errors.Is(err, readerErr) {
		t.Fatalf("error = %v, want wrapped reader error", err)
	}
}

func TestRateLimitServiceAbuse(t *testing.T) {
	reader := &rateReaderMock{abuse: true}
	service := access.NewRateLimitService(reader, 12, true, 3, 30*time.Second)

	abuse, err := service.Abuse(context.Background(), "2001:db8::1")
	if err != nil || !abuse {
		t.Fatalf("Abuse = %v, %v; want true, nil", abuse, err)
	}
	if reader.abuseCalls != 1 || reader.abuseKey != "2001:db8::/64" || reader.abuseLimit != 3 || reader.abuseWindow != 30*time.Second {
		t.Fatalf("Abuse dependency call = %#v, want normalized key and configured abuse settings", reader)
	}
}

func TestRateLimitServiceRejectsInvalidIP(t *testing.T) {
	reader := &rateReaderMock{}
	service := access.NewRateLimitService(reader, 12, true, 3, time.Minute)

	if _, err := service.Allow(context.Background(), "not-an-ip"); err == nil {
		t.Fatal("Allow accepted an invalid IP")
	}
	if _, err := service.Abuse(context.Background(), "not-an-ip"); err == nil {
		t.Fatal("Abuse accepted an invalid IP")
	}
	if reader.allowCalls != 0 || reader.abuseCalls != 0 {
		t.Fatal("rate reader was called for an invalid IP")
	}
}
