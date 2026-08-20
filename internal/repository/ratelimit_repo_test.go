//go:build integration

package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/StellaShiina/ktauth/internal/db"
	"github.com/StellaShiina/ktauth/internal/repository"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func newRateLimitRepo(t *testing.T) (*repository.RateLimitRepo, *redis.Client) {
	t.Helper()

	rdb, err := db.NewRedis()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := rdb.Close(); err != nil {
			t.Errorf("close redis client: %v", err)
		}
	})

	return repository.NewRateLimitRepo(rdb), rdb
}

func uniqueIP() string {
	return "test-" + uuid.NewString()
}

func TestRateLimitRepoAllow(t *testing.T) {
	repo, rdb := newRateLimitRepo(t)
	ctx := context.Background()
	ip := uniqueIP()
	key := "ratelimit:ip:" + ip
	t.Cleanup(func() {
		if err := rdb.Del(ctx, key).Err(); err != nil {
			t.Errorf("clean up rate limit key: %v", err)
		}
	})

	for i, want := range []bool{true, true, false} {
		got, err := repo.Allow(ctx, ip, 2, time.Second)
		if err != nil {
			t.Fatalf("request %d: %v", i+1, err)
		}
		if got != want {
			t.Errorf("request %d: got allowed=%v, want %v", i+1, got, want)
		}
	}

	if got, err := rdb.ZCard(ctx, key).Result(); err != nil {
		t.Fatalf("count requests: %v", err)
	} else if got != 2 {
		t.Errorf("stored request count: got %d, want 2", got)
	}

	if err := repo.Delete(ctx, ip); err != nil {
		t.Fatalf("delete rate limit state: %v", err)
	}
	if got, err := repo.Allow(ctx, ip, 2, time.Second); err != nil {
		t.Fatalf("request after delete: %v", err)
	} else if !got {
		t.Error("request after delete was denied")
	}
}

func TestRateLimitRepoAllowWindowExpiry(t *testing.T) {
	repo, rdb := newRateLimitRepo(t)
	ctx := context.Background()
	ip := uniqueIP()
	key := "ratelimit:ip:" + ip
	t.Cleanup(func() {
		if err := rdb.Del(ctx, key).Err(); err != nil {
			t.Errorf("clean up rate limit key: %v", err)
		}
	})

	window := 100 * time.Millisecond
	if got, err := repo.Allow(ctx, ip, 1, window); err != nil {
		t.Fatal(err)
	} else if !got {
		t.Fatal("first request was denied")
	}
	if got, err := repo.Allow(ctx, ip, 1, window); err != nil {
		t.Fatal(err)
	} else if got {
		t.Fatal("second request was allowed inside the window")
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		got, err := repo.Allow(ctx, ip, 1, window)
		if err != nil {
			t.Fatal(err)
		}
		if got {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}

	t.Fatal("request remained denied after the sliding window expired")
}

func TestRateLimitRepoAbuse(t *testing.T) {
	repo, rdb := newRateLimitRepo(t)
	ctx := context.Background()
	ip := uniqueIP()
	key := "abuse:429:" + ip
	t.Cleanup(func() {
		if err := rdb.Del(ctx, key).Err(); err != nil {
			t.Errorf("clean up abuse key: %v", err)
		}
	})

	for i, wantAbuse := range []bool{false, false, true, false} {
		got, err := repo.Abuse(ctx, ip, 3, 2*time.Second)
		if err != nil {
			t.Fatalf("request %d: %v", i+1, err)
		}
		if got != wantAbuse {
			t.Errorf("request %d: got abuse=%v, want %v", i+1, got, wantAbuse)
		}
	}

	if got, err := rdb.Exists(ctx, key).Result(); err != nil {
		t.Fatalf("check abuse state: %v", err)
	} else if got != 1 {
		t.Error("abuse counter was not restarted after reaching the limit")
	}
}
