package identity

import (
	"context"
	"fmt"

	"github.com/StellaShiina/ktauth/internal/crypto"
	"github.com/StellaShiina/ktauth/internal/model"
	"github.com/google/uuid"
)

type UserStore interface {
	NewUser(ctx context.Context, UUID, name, passwordHash string, email *string, role string) error
	GetUserByName(ctx context.Context, name string) (model.User, error)
	UpdateUser(ctx context.Context, uuid, name, passwordHash string, email *string, role string) error
}

type AccountService struct {
	userStore UserStore
}

func NewAccountService(userStore UserStore) *AccountService {
	return &AccountService{userStore}
}

// return uuid, error
func (s *AccountService) NewUser(c context.Context, name, password string, email *string, role string) (string, error) {
	UUID := uuid.NewString()
	password_hash, hashErr := crypto.HashPassword(password)
	if hashErr != nil {
		return "", fmt.Errorf("Hash error: %v", hashErr)
	}
	err := s.userStore.NewUser(c, UUID, name, password_hash, email, role)
	if err != nil {
		return "", err
	}
	return UUID, nil
}

func (s *AccountService) GetUserByName(c context.Context, name string) (model.User, error) {
	return s.userStore.GetUserByName(c, name)
}

func (s *AccountService) UpdateUser(c context.Context, uuid, name, password string, email *string, role string) error {
	password_hash, hashErr := crypto.HashPassword(password)
	if hashErr != nil {
		return hashErr
	}
	return s.userStore.UpdateUser(c, uuid, name, password_hash, email, role)
}
