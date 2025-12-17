package identity

import (
	"context"

	"github.com/StellaShiina/ktauth/internal/repository"
)

type ConsumeTokenService struct {
	tokenRepo *repository.TokenRepo
}

func NewConsumeTokenService(r *repository.TokenRepo) *ConsumeTokenService {
	return &ConsumeTokenService{r}
}

func (s *ConsumeTokenService) Consume(c context.Context, token string) bool {
	return s.tokenRepo.Consume(c, token)
}
