package admin

import (
	"context"

	"github.com/StellaShiina/ktauth/internal/repository"
)

type UserManageService struct {
	userRepo *repository.UserRepo
}

func NewUserManageService(userRepo *repository.UserRepo) *UserManageService {
	return &UserManageService{userRepo}
}

func (s *UserManageService) ListUsers(c context.Context) ([]UserResponse, error) {
	var userres []UserResponse
	data, err := s.userRepo.ListUsers(c)
	if err != nil {
		return nil, err
	}
	for _, user := range data {
		email := ""
		if user.Email != nil {
			email = *user.Email
		}
		userres = append(userres, UserResponse{
			ID:    user.UUID,
			Name:  user.Name,
			Email: email,
			Role:  user.Role,
		})
	}
	return userres, nil
}
