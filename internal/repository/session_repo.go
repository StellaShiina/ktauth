package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type SessionRepo struct {
	rdb *redis.Client
}

func NewSessionRepo(rdb *redis.Client) *SessionRepo {
	return &SessionRepo{rdb: rdb}
}

func (r *SessionRepo) CreateSession(ctx context.Context, uuid, jti string) error {
	key := fmt.Sprintf("jwt:active:%s:%s", uuid, jti)
	return r.rdb.Set(ctx, key, uuid, 144*time.Hour).Err()
}

func (r *SessionRepo) GetSession(ctx context.Context, uuid, jti string) (string, error) {
	key := fmt.Sprintf("jwt:active:%s:%s", uuid, jti)
	uuid, err := r.rdb.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}
	return uuid, nil
}

func (r *SessionRepo) DelSession(ctx context.Context, uuid, jti string) error {
	key := fmt.Sprintf("jwt:active:%s:%s", uuid, jti)
	return r.rdb.Del(ctx, key).Err()
}
