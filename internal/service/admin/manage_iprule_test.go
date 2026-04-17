package admin_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/StellaShiina/ktauth/internal/db"
	"github.com/StellaShiina/ktauth/internal/model"
	"github.com/StellaShiina/ktauth/internal/repository"
	"github.com/StellaShiina/ktauth/internal/service/admin"
)

func TestManageIPRule(t *testing.T) {
	postgres, err := db.NewPostgres()
	if err != nil {
		t.Fatal(err)
	}
	c := context.Background()
	ipRepo := repository.NewIPRepo(postgres)

	rdb, err := db.NewRedis()
	if err != nil {
		t.Fatal(err)
	}
	ipCache := repository.NewIPCache(rdb)
	rateLimitRepo := repository.NewRateLimitRepo(rdb)

	s := admin.NewAdminIPRuleService(ipRepo, ipCache, rateLimitRepo)

	note := "test"

	fmt.Println("AddRule test...")
	res, err := s.AddRule(c, "2606:4700:4700::1111", false, &note)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("Add %s to %s\n", res, model.IPBlackList)
	_, err = s.AddRule(c, "2606:4700:4700::1001", false, &note)
	if err == nil {
		t.Fatal("duplicate error")
	}

	fmt.Println("DelRule test...")
	_, err = s.DelRule(c, "2606:4700:4700::1001")
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.DelRule(c, "2606:4700:4700::1111")
	if err == nil {
		t.Fatal("Delete a not exist ip")
	}

	fmt.Println("ListRules test...")
	_, err = s.ListRules(c)
	if err != nil {
		t.Fatal(err)
	}
}
