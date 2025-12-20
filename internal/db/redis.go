package db

import (
	"context"
	"os"

	"github.com/redis/go-redis/v9"
)

func NewRedis() (*redis.Client, error) {
	host := os.Getenv("REDIS_HOST")
	if host == "" {
		host = "127.0.0.1"
	}
	rdb := redis.NewClient(&redis.Options{
		Addr: host + ":6379",
	})

	ctx := context.Background()

	err := rdb.Ping(ctx).Err()

	if err != nil {
		return nil, err
	}

	return rdb, nil
}
