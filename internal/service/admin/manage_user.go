package admin

import (
	"context"

	"github.com/StellaShiina/ktauth/internal/model"
)

type UserReader interface {
	ListUsers(ctx context.Context) ([]model.User, error)
}

type UserManageService struct {
	userReader UserReader
}

func NewUserManageService(userReader UserReader) *UserManageService {
	return &UserManageService{userReader}
}

func (s *UserManageService) ListUsers(c context.Context) ([]UserResponse, error) {
	var userres []UserResponse
	data, err := s.userReader.ListUsers(c)
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
