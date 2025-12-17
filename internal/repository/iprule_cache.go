package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/StellaShiina/ktauth/internal/model"
	"github.com/redis/go-redis/v9"
)

var ttl = map[model.IPRuleType]time.Duration{
	model.IPBlackList: 1 * time.Hour,
	model.IPWhiteList: 30 * time.Minute,
	model.IPGreyList:  5 * time.Minute,
}

type IPCache struct {
	rdb *redis.Client
}

func NewIPCache(rdb *redis.Client) *IPCache {
	return &IPCache{rdb}
}

func (r *IPCache) Cache(c context.Context, rule_type model.IPRuleType, ips ...string) error {
	keyPrefix := fmt.Sprintf("%s:ip:", string(rule_type))
	pipe := r.rdb.Pipeline()
	for _, ip := range ips {
		pipe.Set(c, keyPrefix+ip, "", ttl[rule_type])
	}
	_, err := pipe.Exec(c)
	return err
}

func (r *IPCache) Check(c context.Context, rule_type model.IPRuleType, ip string) (bool, error) {
	keyPrefix := fmt.Sprintf("%s:ip:", string(rule_type))
	_, err := r.rdb.Get(c, keyPrefix+ip).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
