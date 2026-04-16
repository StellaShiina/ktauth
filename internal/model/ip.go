package model

import (
	"net"
	"time"
)

type IPRuleType string

const (
	IPWhiteList IPRuleType = "whitelist"
	IPBlackList IPRuleType = "blacklist"
	IPGreyList  IPRuleType = "greylist"
)

type IP struct {
	ID          int64      // 对应 BIGSERIAL
	Version     int16      // 对应 SMALLINT (4 或 6)
	IPRange     *net.IPNet // 对应 CIDR 类型
	IsWhitelist bool       // 对应 BOOLEAN (true=白名单, false=黑名单)
	CreateAt    time.Time  // 对应 TIMESTAMPTZ
	UpdateAt    time.Time  // 对应 TIMESTAMPTZ
	Note        *string    // 对应 TEXT
}
