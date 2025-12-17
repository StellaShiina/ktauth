package access

import (
	"context"
	"fmt"
	"time"

	"github.com/StellaShiina/ktauth/internal/repository"
)

// RateLimitService provides request rate limiting services based on IP address.
// It uses a sliding window algorithm to ensure precise rate limiting.
type RateLimitService struct {
	rateLimitRepo *repository.RateLimitRepo
}

// NewRateLimitService creates a new instance of RateLimitService.
func NewRateLimitService(rateLimitRepo *repository.RateLimitRepo) *RateLimitService {
	return &RateLimitService{rateLimitRepo}
}

// Allow checks if the incoming request from the specified IP is allowed.
// It enforces a limit of 60 requests per 1 minute window.
func (s *RateLimitService) Allow(ctx context.Context, ip string) (bool, error) {
	// Construct a unique key for the IP address
	key := fmt.Sprintf("ratelimit:ip:%s", ip)

	// Define the rate limit rules: 60 requests per 1 minute
	const (
		limit  = 60
		window = 1 * time.Minute
	)

	// Delegate the check to the repository
	allowed, err := s.rateLimitRepo.Allow(ctx, key, limit, window)
	if err != nil {
		// In case of error (e.g., Redis down), we might want to default to deny or allow.
		// Returning error allows the caller to decide.
		return false, fmt.Errorf("rate limit check failed: %w", err)
	}

	return allowed, nil
}
