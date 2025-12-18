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
	keyPrefix := "rule:ip:"
	pipe := r.rdb.Pipeline()
	for _, ip := range ips {
		pipe.Set(c, keyPrefix+ip, string(rule_type), ttl[rule_type])
	}
	_, err := pipe.Exec(c)
	return err
}

// return ruletype(string) or "", err"Cache not found" or "", err
func (r *IPCache) Get(c context.Context, ip string) (string, error) {
	keyPrefix := "rule:ip:"
	ruleStr, err := r.rdb.Get(c, keyPrefix+ip).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("Cache not found")
	}
	if err != nil {
		return "", err
	}
	return ruleStr, nil
}
