package repository_test

import (
	"context"
	"fmt"
	"net"
	"testing"

	"github.com/StellaShiina/ktauth/internal/db"
	"github.com/StellaShiina/ktauth/internal/repository"
	"github.com/StellaShiina/ktauth/pkg/iputils"
)

func TestIPRepo(t *testing.T) {
	postgres, err := db.NewPostgres()
	if err != nil {
		t.Fatal(err)
	}
	c := context.Background()
	IPRepo := repository.NewIPRepo(postgres)
	fmt.Println("Test GetIPs")
	_, err = IPRepo.GetIPs(c)
	if err != nil {
		t.Error(err)
		t.Fail()
	}

	fmt.Println("Test AddIP")
	alidns := "2400:3200::1/32"
	alidns2 := net.ParseIP("2400:3200:baba::1")
	version, _, ipNet, err := iputils.ProcessIP(alidns)
	if err != nil {
		t.Fatal(err)
	}
	err = IPRepo.AddIP(c, version, ipNet, true)
	if err != nil {
		t.Fatal(err)
	}
	err = IPRepo.AddIP(c, version, ipNet, true)
	if err != repository.ErrIPExist {
		t.Fatal(err)
	}

	fmt.Println("Test QueryIP")
	isWhitelist, err := IPRepo.QueryIP(c, 4, net.ParseIP("127.0.0.1"))
	if err != nil || !isWhitelist {
		t.Error(err)
		t.Fail()
	}
	isWhitelist, err = IPRepo.QueryIP(c, version, alidns2)
	if err != nil || !isWhitelist {
		t.Error(err)
		t.Fail()
	}

	fmt.Println("Test DelIP")
	err = IPRepo.DelIP(c, version, ipNet)
	if err != nil {
		t.Error(err)
		t.Fail()
	}
	err = IPRepo.DelIP(c, version, ipNet)
	if err != repository.ErrIPNotFound {
		t.Error(err)
		t.Fail()
	}
}
