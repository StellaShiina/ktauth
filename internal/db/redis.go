package db

import (
	"context"

	"github.com/redis/go-redis/v9"
)

func NewRedis() (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
	})

	ctx := context.Background()

	err := rdb.Ping(ctx).Err()

	if err != nil {
		return nil, err
	}

	return rdb, nil
}
