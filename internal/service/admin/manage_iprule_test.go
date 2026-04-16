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
	IPRepo := repository.NewIPRepo(postgres)
	s := admin.NewAdminIPRuleService(IPRepo)

	fmt.Println("AddRule test...")
	res, err := s.AddRule(c, "2606:4700:4700::1111", false)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("Add %s to %s\n", res, model.IPBlackList)
	_, err = s.AddRule(c, "2606:4700:4700::1001", false)
	if err == nil {
		t.Fatal("duplicate error")
	}

	fmt.Println("DelRule test...")
	err = s.DelRule(c, "2606:4700:4700::1001")
	if err != nil {
		t.Fatal(err)
	}
	err = s.DelRule(c, "2606:4700:4700::1111")
	if err == nil {
		t.Fatal("Delete a not exist ip")
	}

	fmt.Println("ListRules test...")
	_, err = s.ListRules(c)
	if err != nil {
		t.Fatal(err)
	}
}
