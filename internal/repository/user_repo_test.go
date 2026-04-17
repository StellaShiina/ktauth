package repository_test

import (
	"context"
	"testing"

	"github.com/StellaShiina/ktauth/internal/db"
	"github.com/StellaShiina/ktauth/internal/repository"
)

func TestUserRepo(t *testing.T) {
	postgres, err := db.NewPostgres()
	if err != nil {
		t.Fatal(err)
	}
	c := context.Background()
	r := repository.NewUserRepo(postgres)

	uuid := "f03c6866-3bf2-40d4-96e1-37bca3fe9cf3"
	name := "testuser"
	password_hash := "$2a$10$4/wdVsUh/qaN76xt9ly80uwIbMlhtHFkT6gMIP.g3InF4d/hbZ6/m"
	email := "testuser@example.com"
	role := "system"

	if err := r.NewUser(c, uuid, name, password_hash, &email, role); err != nil {
		t.Fatal(err)
	}

	if err := r.UpdateUser(c, uuid, name, "$2a$10$4/wdVsUh/qaN76xt9ly80uwIbMlhtHFkT6gMIP.g3InF4d/hbZ6/m", nil, "admin"); err != nil {
		t.Error(err)
		t.Fail()
	}

	if err := r.NewUser(c, uuid, name, password_hash, &email, role); err != repository.ErrUserExist {
		t.Fatal(err)
	}

	if user, err := r.GetUserByName(c, name); err != nil || user.UUID != uuid {
		t.Errorf("error: %v, uuid: %s", err, user.UUID)
		t.Fail()
	}

	if _, err := r.ListUsers(c); err != nil {
		t.Error(err)
		t.Fail()
	}

	if err := r.DelUser(c, name); err != nil {
		t.Error(err)
		t.Fail()
	}

	if err := r.DelUser(c, name); err != repository.ErrUserNotFound {
		t.Error(err)
		t.Fail()
	}

	if _, err := r.GetUserByName(c, name); err == nil {
		t.Error(err)
		t.Fail()
	}
}
