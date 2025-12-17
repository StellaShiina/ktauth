package repository

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type CountDownRepo struct {
	rdb *redis.Client
}

func NewCountDownRepo(rdb *redis.Client) *CountDownRepo {
	return &CountDownRepo{rdb}
}

func (r *CountDownRepo) Set(c context.Context, key string) error {
	return r.rdb.Set(c, key, "", 1*time.Minute).Err()
}

// return true if the given key is in cd.
func (r *CountDownRepo) CD(c context.Context, key string) (bool, error) {
	_, err := r.rdb.Get(c, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
