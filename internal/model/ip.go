package model

import "time"

type IPRuleType string

const (
	IPWhiteList IPRuleType = "whitelist"
	IPBlackList IPRuleType = "blacklist"
	IPGreyList  IPRuleType = "greylist"
)

type IPVersion string

const (
	V4 IPVersion = "4"
	V6 IPVersion = "6"
)

type IP struct {
	ID       int64
	Version  IPVersion
	IP_bin   []byte
	RuleType IPRuleType
	CreateAt time.Time
	UpdateAt time.Time
	Note     *string
	IP_str   string
}
