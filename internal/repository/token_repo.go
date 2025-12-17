package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type TokenRepo struct {
	rdb *redis.Client
}

func NewTokenRepo(rdb *redis.Client) *TokenRepo {
	return &TokenRepo{rdb: rdb}
}

func (r *TokenRepo) Restock(ctx context.Context) error {
	n, err := r.rdb.SCard(ctx, "admin:tokens").Result()
	if err != nil {
		return fmt.Errorf("Redis scard error")
	}
	if n >= 10 {
		return fmt.Errorf("No need to restock")
	}
	numRestock := 10
	tokens := make([]any, 0, numRestock)
	for range numRestock {
		tokens = append(tokens, uuid.NewString())
	}
	return r.rdb.SAdd(ctx, "admin:tokens", tokens...).Err()
}

func (r *TokenRepo) Consume(ctx context.Context, token string) bool {
	n, err := r.rdb.SRem(ctx, "admin:tokens", token).Result()
	if n == 0 || err != nil {
		return false
	}
	return true
}

func (r *TokenRepo) ListAll(ctx context.Context) ([]string, error) {
	return r.rdb.SMembers(ctx, "admin:tokens").Result()
}

func (r *TokenRepo) GetOne(ctx context.Context) (string, error) {
	n, err := r.rdb.SCard(ctx, "admin:tokens").Result()
	if err != nil {
		return "", fmt.Errorf("Error when getting scard...")
	}
	if n == 0 {
		return "", fmt.Errorf("No tokens, try to restock...")
	}
	return r.rdb.SRandMember(ctx, "admin:tokens").Result()
}

func (r *TokenRepo) FlushAll(ctx context.Context) error {
	return r.rdb.Del(ctx, "admin:tokens").Err()
}
