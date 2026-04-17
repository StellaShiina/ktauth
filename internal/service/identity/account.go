package identity

import (
	"context"
	"fmt"

	"github.com/StellaShiina/ktauth/internal/crypto"
	"github.com/StellaShiina/ktauth/internal/model"
	"github.com/StellaShiina/ktauth/internal/repository"
	"github.com/google/uuid"
)

type AccountService struct {
	userRepo *repository.UserRepo
	// TODO may activate in later version
	// registerRepo *repository.RegisterRepo
}

func NewAccountService(userRepo *repository.UserRepo) *AccountService {
	return &AccountService{userRepo}
}

// return uuid, error
func (s *AccountService) NewUser(c context.Context, name, password string, email *string, role string) (string, error) {
	UUID := uuid.NewString()
	password_hash, hashErr := crypto.HashPassword(password)
	if hashErr != nil {
		return "", fmt.Errorf("Hash error: %v", hashErr)
	}
	err := s.userRepo.NewUser(c, UUID, name, password_hash, email, role)
	if err != nil {
		return "", err
	}
	return UUID, nil
}

func (s *AccountService) GetUserByName(c context.Context, name string) (model.User, error) {
	return s.userRepo.GetUserByName(c, name)
}

func (s *AccountService) UpdateUser(c context.Context, uuid, name, password string, email *string, role string) error {
	password_hash, hashErr := crypto.HashPassword(password)
	if hashErr != nil {
		return hashErr
	}
	return s.userRepo.UpdateUser(c, uuid, name, password_hash, email, role)
}
