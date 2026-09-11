//go:build integration

package db_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/StellaShiina/ktauth/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

const connectTimeout = 30 * time.Second

func connectPostgres(ctx context.Context, t *testing.T) (*pgxpool.Pool, error) {
	t.Helper()

	backoff := time.Second
	var lastErr error

	for attempt := 1; ; attempt++ {
		pool, err := db.NewPostgres()
		if err == nil {
			t.Logf("postgres became ready after %d attempt(s)", attempt)
			return pool, nil
		}
		lastErr = err
		t.Logf("postgres not ready (attempt %d): %v; retrying in %s", attempt, err, backoff)

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("postgres not ready after %s: %w", connectTimeout, lastErr)
		case <-time.After(backoff):
		}

		if backoff < 8*time.Second {
			backoff *= 2
		}
	}
}

func TestPostgres(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), connectTimeout)
	defer cancel()

	postgres, err := connectPostgres(ctx, t)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(postgres.Close)

	if err := postgres.Ping(t.Context()); err != nil {
		t.Fatal(err)
	}
}
