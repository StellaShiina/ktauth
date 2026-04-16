package iputils_test

import (
	"fmt"
	"testing"

	"github.com/StellaShiina/ktauth/pkg/iputils"
)

func TestMaskIP(t *testing.T) {
	v, ip, ipNet, err := iputils.ProcessIP("127.0.32.1/8")
	if err != nil || v != 4 || ip.String() != "127.0.32.1" || ipNet.String() != "127.0.0.0/8" {
		t.Error("Fail to processe 127.0.32.1/8\n", err)
		fmt.Println(v, ip, ipNet, err)
		t.Fail()
	}
	v, ip, ipNet, err = iputils.ProcessIP("127.0.0.1")
	if err != nil || v != 4 || ip.String() != "127.0.0.1" || ipNet.String() != "127.0.0.1/32" {
		t.Error("Fail to processe 127.0.0.1\n")
		fmt.Println(v, ip, ipNet, err)
		t.Fail()
	}
	v, ip, ipNet, err = iputils.ProcessIP("2001:4860:4860::8888")
	if err != nil || v != 6 || ip.String() != "2001:4860:4860::8888" || ipNet.String() != "2001:4860:4860::/64" {
		t.Error("Fail to processe 2001:4860:4860::8888\n", err)
		fmt.Println(v, ip, ipNet, err)
		t.Fail()
	}
	v, ip, ipNet, err = iputils.ProcessIP("2001:4860:4860::8888/32")
	if err != nil || v != 6 || ip.String() != "2001:4860:4860::8888" || ipNet.String() != "2001:4860::/32" {
		t.Error("Fail to processe 2001:4860:4860::8888/32\n", err)
		fmt.Println(v, ip, ipNet, err)
		t.Fail()
	}
	v, ip, ipNet, err = iputils.ProcessIP("abc")
	if err == nil && v != 0 && ip == nil {
		t.Error("Fail to processe abc\n", err)
		t.Fail()
	}
}
