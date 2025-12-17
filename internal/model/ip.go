package model

import "time"

type IPRuleType string

const (
	IPWhiteList IPRuleType = "whitelist"
	IPBlackList IPRuleType = "blacklist"
	IPGreyList  IPRuleType = "greylist"
)

type IP struct {
	ID       int64
	CIDR     string
	RuleType IPRuleType
	Note     *string
	CreateAt time.Time
	UpdateAt time.Time
}
