package identity_test

import (
	"context"
	"errors"
	"testing"

	"github.com/StellaShiina/ktauth/internal/crypto"
	"github.com/StellaShiina/ktauth/internal/model"
	"github.com/StellaShiina/ktauth/internal/service/identity"
	"github.com/google/uuid"
)

type userStoreMock struct {
	newUserCalls    int
	getUserCalls    int
	updateUserCalls int
	newUUID         string
	newName         string
	newPasswordHash string
	newEmail        *string
	newRole         string
	getName         string
	getUser         model.User
	newErr          error
	getErr          error
	updateErr       error
	updatePassword  string
}

func (m *userStoreMock) NewUser(_ context.Context, uuid, name, passwordHash string, email *string, role string) error {
	m.newUserCalls++
	m.newUUID, m.newName, m.newPasswordHash, m.newEmail, m.newRole = uuid, name, passwordHash, email, role
	return m.newErr
}

func (m *userStoreMock) GetUserByName(_ context.Context, name string) (model.User, error) {
	m.getUserCalls++
	m.getName = name
	return m.getUser, m.getErr
}

func (m *userStoreMock) UpdateUser(_ context.Context, _, _, passwordHash string, _ *string, _ string) error {
	m.updateUserCalls++
	m.updatePassword = passwordHash
	return m.updateErr
}

func TestAccountServiceNewUserHashesPasswordAndDelegates(t *testing.T) {
	email := "user@example.com"
	store := &userStoreMock{}
	service := identity.NewAccountService(store)

	gotUUID, err := service.NewUser(context.Background(), "alice", "secret", &email, "user")
	if err != nil {
		t.Fatalf("NewUser returned error: %v", err)
	}
	if _, err := uuid.Parse(gotUUID); err != nil {
		t.Fatalf("UUID = %q is invalid: %v", gotUUID, err)
	}
	if store.newUserCalls != 1 || store.newUUID != gotUUID || store.newName != "alice" || store.newEmail != &email || store.newRole != "user" {
		t.Fatalf("NewUser dependency call = %#v", store)
	}
	if store.newPasswordHash == "" || !crypto.VerifyPassword(store.newPasswordHash, "secret") {
		t.Fatal("password was not hashed correctly")
	}
}

func TestAccountServiceNewUserReturnsStoreError(t *testing.T) {
	storeErr := errors.New("user already exists")
	store := &userStoreMock{newErr: storeErr}
	service := identity.NewAccountService(store)

	gotUUID, err := service.NewUser(context.Background(), "alice", "secret", nil, "user")
	if !errors.Is(err, storeErr) {
		t.Fatalf("error = %v, want store error", err)
	}
	if gotUUID != "" {
		t.Fatalf("UUID = %q, want empty on failure", gotUUID)
	}
}

func TestAccountServiceGetUserByName(t *testing.T) {
	want := model.User{UUID: "u1", Name: "alice", Role: "user"}
	store := &userStoreMock{getUser: want}
	service := identity.NewAccountService(store)

	got, err := service.GetUserByName(context.Background(), "alice")
	if err != nil {
		t.Fatalf("GetUserByName returned error: %v", err)
	}
	if got != want || store.getUserCalls != 1 || store.getName != "alice" {
		t.Fatalf("result/call = %#v, want %#v and one call", store, want)
	}
}

func TestAccountServiceUpdateUserHashesPassword(t *testing.T) {
	store := &userStoreMock{}
	service := identity.NewAccountService(store)

	if err := service.UpdateUser(context.Background(), "u1", "alice", "new-secret", nil, "user"); err != nil {
		t.Fatalf("UpdateUser returned error: %v", err)
	}
	if store.updateUserCalls != 1 || !crypto.VerifyPassword(store.updatePassword, "new-secret") {
		t.Fatal("UpdateUser did not pass a valid password hash")
	}
}
