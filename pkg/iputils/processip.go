package iputils

import (
	"fmt"
	"net"
	"strings"
)

type IPError struct {
	ip string
}

func (e *IPError) Error() string {
	return fmt.Sprintf("Invalid IP: %v", e.ip)
}

// Input origin IP str. Return version, origin ip, processed cidr, error.
func ProcessIP(ipStr string) (int16, net.IP, *net.IPNet, error) {
	ipStr = strings.TrimSpace(ipStr)
	var ipNet *net.IPNet
	var ip net.IP
	ip, ipNet, err := net.ParseCIDR(ipStr)
	if err == nil {
		if ip.To4() != nil {
			return 4, ip, ipNet, nil
		} else {
			return 6, ip, ipNet, nil
		}
	}
	ip = net.ParseIP(ipStr)
	if ip == nil {
		return 0, nil, nil, &IPError{ipStr}
	}
	if ip.To4() != nil {
		mask := net.CIDRMask(32, 32)
		maskedip := ip.Mask(mask)
		ipNet = &net.IPNet{
			IP:   maskedip,
			Mask: mask,
		}
		return 4, ip, ipNet, nil
	}
	mask := net.CIDRMask(64, 128)
	maskedip := ip.Mask(mask)
	ipNet = &net.IPNet{
		IP:   maskedip,
		Mask: mask,
	}
	return 6, ip, ipNet, nil
}
