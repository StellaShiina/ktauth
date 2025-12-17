package identity

import (
	"context"
	"fmt"
	"log/slog"
	"net/mail"
	"strings"

	"github.com/StellaShiina/ktauth/internal/auth"
	"github.com/StellaShiina/ktauth/internal/crypto"
	"github.com/StellaShiina/ktauth/internal/model"
	"github.com/StellaShiina/ktauth/internal/repository"
	"github.com/google/uuid"
)

type AccountService struct {
	userRepo      *repository.UserRepo
	registerRepo  *repository.RegisterRepo
	countDownRepo *repository.CountDownRepo
}

func NewAccountService(userRepo *repository.UserRepo, registerRepo *repository.RegisterRepo, countDownRepo *repository.CountDownRepo) *AccountService {
	return &AccountService{userRepo, registerRepo, countDownRepo}
}

func (s *AccountService) RequireCode(c context.Context, email, ip string) error {
	cdKey := "cd:ip:" + ip
	// check cd
	isCD, err := s.countDownRepo.CD(c, cdKey)
	if isCD {
		return fmt.Errorf("Rate limit exceeded")
	}
	if err != nil {
		return err
	}

	// check email
	email = strings.TrimSpace(email)
	_, err = mail.ParseAddress(email)
	if err != nil {
		return err
	}

	// send code
	code, err := auth.Resend(email)
	if err != nil {
		return err
	}

	// set cd
	err = s.countDownRepo.Set(c, cdKey)
	if err != nil {
		return err
	}

	// set code
	err = s.registerRepo.Set(c, email, code)
	if err != nil {
		return err
	}

	return nil
}

func (s *AccountService) VerifyCode(c context.Context, email, code string) (bool, error) {
	valid, err := s.registerRepo.Validate(c, email, code)
	if err != nil || !valid {
		return false, err
	}
	return true, nil
}

// return uuid, error
func (s *AccountService) NewUser(c context.Context, name, password, email string) (string, error) {
	UUID := uuid.NewString()
	password_hash, hassErr := crypto.HashPassword(password)
	if hassErr != nil {
		return "", fmt.Errorf("Hash error: %v", hassErr)
	}
	slog.Debug(password)
	slog.Debug(password_hash)
	err := s.userRepo.NewUser(c, UUID, name, password_hash, email)
	if err != nil {
		return "", err
	}
	return UUID, nil
}

func (s *AccountService) GetUserByName(c context.Context, name string) (model.User, error) {
	return s.userRepo.GetUserByName(c, name)
}
