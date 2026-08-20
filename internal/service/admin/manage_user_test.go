package admin_test

import (
	"context"
	"errors"
	"testing"

	"github.com/StellaShiina/ktauth/internal/model"
	"github.com/StellaShiina/ktauth/internal/service/admin"
)

type userReaderMock struct {
	users []model.User
	err   error
}

func (m *userReaderMock) ListUsers(context.Context) ([]model.User, error) {
	return m.users, m.err
}

func TestUserManageServiceMapsUsers(t *testing.T) {
	email := "alice@example.com"
	reader := &userReaderMock{users: []model.User{
		{UUID: "u1", Name: "alice", Email: &email, Role: "user"},
		{UUID: "u2", Name: "bob", Role: "admin"},
	}}
	service := admin.NewUserManageService(reader)

	users, err := service.ListUsers(context.Background())
	if err != nil {
		t.Fatalf("ListUsers returned error: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("users count = %d, want 2", len(users))
	}
	if users[0].ID != "u1" || users[0].Name != "alice" || users[0].Email != email || users[0].Role != "user" {
		t.Errorf("first user = %#v", users[0])
	}
	if users[1].ID != "u2" || users[1].Email != "" || users[1].Role != "admin" {
		t.Errorf("second user = %#v", users[1])
	}
}

func TestUserManageServiceReturnsReaderError(t *testing.T) {
	readerErr := errors.New("database unavailable")
	service := admin.NewUserManageService(&userReaderMock{err: readerErr})

	users, err := service.ListUsers(context.Background())
	if !errors.Is(err, readerErr) {
		t.Fatalf("error = %v, want reader error", err)
	}
	if users != nil {
		t.Fatalf("users = %#v, want nil", users)
	}
}
