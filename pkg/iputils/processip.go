package iputils

import (
	"fmt"
	"net"
	"strings"

	"github.com/StellaShiina/ktauth/internal/model"
)

type IPError struct {
	ip string
}

func (e *IPError) Error() string {
	return fmt.Sprintf("Invalid IP: %s", e.ip)
}

// Input origin IP str. Return version, IPv4-Mapped-IPv6 or IPv6/64 IP, error.
func ProcessIP(ipStr string) (model.IPVersion, net.IP, error) {
	ipStr = strings.TrimSpace(ipStr)
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return "", nil, &IPError{ipStr}
	}
	if ip.To4() != nil {
		return model.V4, ip.To16(), nil
	}
	mask := net.CIDRMask(64, 128)
	ip = ip.Mask(mask)
	return model.V6, ip, nil
}
