package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type RateLimitRepo struct {
	rdb *redis.Client
}

func NewRateLimitRepo(rdb *redis.Client) *RateLimitRepo {
	return &RateLimitRepo{rdb: rdb}
}

// slidingWindowScript is a Lua script for sliding window rate limiting
// It atomically removes old entries, counts current entries, and adds a new entry if allowed.
var slidingWindowScript = redis.NewScript(`
local key = KEYS[1]
local window = tonumber(ARGV[1])
local limit = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local member = ARGV[4]

-- Remove requests that are outside the sliding window
-- Score is timestamp. We keep entries with score > (now - window)
redis.call('ZREMRANGEBYSCORE', key, '-inf', now - window)

-- Count requests in the current window
local count = redis.call('ZCARD', key)

if count < limit then
	-- Add current request
	-- We use the timestamp as the score, and a unique member ID
	redis.call('ZADD', key, now, member)
	-- Update expiration time to avoid stale keys (window size in milliseconds)
	redis.call('PEXPIRE', key, window)
	return 1 -- Allowed
else
	return 0 -- Denied
end
`)

// Allow checks if a request is allowed based on the sliding window algorithm
// key: Unique identifier for the limit (e.g., "ratelimit:ip:127.0.0.1")
// limit: Maximum number of requests allowed in the window
// window: The duration of the sliding window
func (r *RateLimitRepo) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	now := time.Now()
	nowMs := now.UnixMilli()

	// Generate a unique member ID to prevent overwriting requests with the same timestamp
	member := uuid.New().String()

	keys := []string{key}
	args := []any{
		window.Milliseconds(), // ARGV[1]: Window size in ms
		limit,                 // ARGV[2]: Max requests
		nowMs,                 // ARGV[3]: Current timestamp in ms
		member,                // ARGV[4]: Unique member
	}

	result, err := slidingWindowScript.Run(ctx, r.rdb, keys, args...).Result()
	if err != nil {
		return false, fmt.Errorf("failed to execute rate limit script: %w", err)
	}

	// Result should be 1 (allowed) or 0 (denied)
	if res, ok := result.(int64); ok {
		return res == 1, nil
	}
	return false, fmt.Errorf("unexpected result type from redis script: %T", result)
}
