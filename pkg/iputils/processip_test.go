package iputils_test

import (
	"fmt"
	"testing"

	"github.com/StellaShiina/ktauth/internal/model"
	"github.com/StellaShiina/ktauth/pkg/iputils"
)

func TestMaskIP(t *testing.T) {
	v, ip, err := iputils.ProcessIP("127.0.0.1")
	if err != nil || v != model.V4 || len(ip) != 16 {
		t.Error("Fail to processe 127.0.0.1\n")
	}
	fmt.Println(v, ip, err)
	v, ip, err = iputils.ProcessIP("2001:4860:4860::8888")
	if err != nil || v != model.V6 || len(ip) != 16 || ip.String() != "2001:4860:4860::" {
		t.Error("Fail to processe 2001:4860:4860::8888\n")
	}
	fmt.Println(v, ip, err)
	v, ip, err = iputils.ProcessIP("abc")
	if err == nil && v != "" && ip == nil {
		t.Error("Fail to processe abc\n")
	}
	fmt.Println(v, ip, err)
}
