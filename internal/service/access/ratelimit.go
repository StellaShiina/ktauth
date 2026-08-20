package access

import (
	"context"
	"fmt"
	"time"

	"github.com/StellaShiina/ktauth/pkg/iputils"
)

type RateReader interface {
	Allow(ctx context.Context, ip string, limit int, window time.Duration) (bool, error)
	Abuse(ctx context.Context, ip string, limit int, window time.Duration) (bool, error)
}

// RateLimitService provides request rate limiting services based on IP address.
// It uses a sliding window algorithm to ensure precise rate limiting.
type RateLimitService struct {
	rateReader     RateReader
	limitPerMinute int
	enabled        bool
	abuseLimit     int
	abuseWindow    time.Duration
}

// NewRateLimitService creates a new instance of RateLimitService. Set limitPerMinute and enabled.
func NewRateLimitService(rateReader RateReader, limitPerMinute int, enabled bool, abuseLimit int, abuseWindow time.Duration) *RateLimitService {
	return &RateLimitService{rateReader, limitPerMinute, enabled, abuseLimit, abuseWindow}
}

// Allow checks if the incoming request from the specified IP is allowed.
// It enforces a limit of 60 requests per 1 minute window.
func (s *RateLimitService) Allow(ctx context.Context, ip string) (bool, error) {
	if !s.enabled {
		return true, nil
	}

	// Process IPstr
	_, _, ipNet, err := iputils.ProcessIP(ip)

	if err != nil {
		return false, err
	}

	// Define the rate limit rules: 60 requests per 1 minute
	const (
		window = 1 * time.Minute
	)

	// Delegate the check to the repository
	allowed, err := s.rateReader.Allow(ctx, ipNet.String(), s.limitPerMinute, window)
	if err != nil {
		// In case of error (e.g., Redis down), we might want to default to deny or allow.
		// Returning error allows the caller to decide.
		return false, fmt.Errorf("rate limit check failed: %w", err)
	}

	return allowed, nil
}

func (s *RateLimitService) Abuse(ctx context.Context, ip string) (bool, error) {
	_, _, ipNet, err := iputils.ProcessIP(ip)

	if err != nil {
		return false, err
	}

	return s.rateReader.Abuse(ctx, ipNet.String(), s.abuseLimit, s.abuseWindow)
}
