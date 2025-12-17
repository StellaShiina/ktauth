package admin

import (
	"context"

	"github.com/StellaShiina/ktauth/internal/repository"
)

type AdminTokenService struct {
	tokenRepo *repository.TokenRepo
}

func NewAdminTokenService(r *repository.TokenRepo) *AdminTokenService {
	return &AdminTokenService{r}
}

func (s *AdminTokenService) Restock(c context.Context) error {
	return s.tokenRepo.Restock(c)
}

func (s *AdminTokenService) FlushTokens(c context.Context) error {
	return s.tokenRepo.FlushAll(c)
}

func (s *AdminTokenService) GetToken(c context.Context) (string, error) {
	return s.tokenRepo.GetOne(c)
}

func (s *AdminTokenService) GetTokens(c context.Context) ([]string, error) {
	return s.tokenRepo.ListAll(c)
}
