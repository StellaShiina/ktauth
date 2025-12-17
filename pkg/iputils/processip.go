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
	return fmt.Sprintf("Invalid IP: %s", e.ip)
}

// Input IP string. Return IPv6/64 cidr or origin IPv4.
func IPv6ToCIDR64String(ip net.IP) (string, error) {
	if ip == nil {
		return "", fmt.Errorf("Invalid IP")
	}
	ip6 := ip.To16()
	if ip6 == nil || ip.To4() != nil {
		return ip.String(), nil
	}

	mask := net.CIDRMask(64, 128)
	prefix := ip.Mask(mask)

	CIDR := &net.IPNet{
		IP:   prefix,
		Mask: mask,
	}

	return CIDR.String(), nil
}

// Input IP/CIDR string. Return origin CIDR or IPv6/64 CIDR or IPv4/32 CIDR.
func ProcessIPToCIDR(ipStr string) (string, error) {
	ipStr = strings.TrimSpace(ipStr)
	_, cidr, err := net.ParseCIDR(ipStr)
	if err == nil {
		return cidr.String(), nil
	}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return "", &IPError{ipStr}
	}
	if ip.To4() != nil {
		return ip.String() + "/32", nil
	}
	mask := net.CIDRMask(64, 128)
	prefix := ip.Mask(mask)

	cidr = &net.IPNet{
		IP:   prefix,
		Mask: mask,
	}

	return cidr.String(), nil
}
