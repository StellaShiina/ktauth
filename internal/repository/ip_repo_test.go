package repository_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/StellaShiina/ktauth/internal/db"
	"github.com/StellaShiina/ktauth/internal/model"
	"github.com/StellaShiina/ktauth/internal/repository"
	"github.com/StellaShiina/ktauth/pkg/iputils"
)

func TestIPRepo(t *testing.T) {
	mysql, err := db.NewMySQL()
	if err != nil {
		t.Fatal(err)
	}
	c := context.Background()
	IPRepo := repository.NewIPRepo(mysql)
	fmt.Println("Test GetIPs")
	ips, err := IPRepo.GetIPs(c)
	if err != nil {
		t.Fatal(err)
	}
	for _, ip := range ips {
		fmt.Println(ip)
	}

	fmt.Println("Test AddIP")
	cfdns := "2606:4700:4700::1111"
	version, ip, err := iputils.ProcessIP(cfdns)
	if err != nil {
		t.Fatal(err)
	}
	err = IPRepo.AddIP(c, version, ip, model.IPWhiteList)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("Test QueryIP")
	rule_type, err := IPRepo.QueryIP(c, version, ip)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("Rule_type of %s - %s\n", ip, rule_type)

	fmt.Println("Test DelIP")
	err = IPRepo.DelIP(c, version, ip)
	if err != nil {
		t.Fatal(err)
	}
}
