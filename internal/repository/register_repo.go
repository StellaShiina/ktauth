package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RegisterRepo struct {
	rdb *redis.Client
}

func NewRegisterRepo(rdb *redis.Client) *RegisterRepo {
	return &RegisterRepo{rdb}
}

func (r *RegisterRepo) Set(c context.Context, email, code string) error {
	key := fmt.Sprintf("register:%s:%s", email, code)
	return r.rdb.Set(c, key, "", 15*time.Minute).Err()
}

func (r *RegisterRepo) Validate(c context.Context, email, code string) (bool, error) {
	key := fmt.Sprintf("register:%s:%s", email, code)
	_, err := r.rdb.GetDel(c, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
