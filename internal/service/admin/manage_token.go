package admin

import (
	"context"
)

type TokenStore interface {
	Restock(ctx context.Context) error
	FlushAll(ctx context.Context) error
	GetOne(ctx context.Context) (string, error)
	ListAll(ctx context.Context) ([]string, error)
}

type AdminTokenService struct {
	ts TokenStore
}

func NewAdminTokenService(r TokenStore) *AdminTokenService {
	return &AdminTokenService{r}
}

func (s *AdminTokenService) Restock(c context.Context) error {
	return s.ts.Restock(c)
}

func (s *AdminTokenService) FlushTokens(c context.Context) error {
	return s.ts.FlushAll(c)
}

func (s *AdminTokenService) GetToken(c context.Context) (string, error) {
	return s.ts.GetOne(c)
}

func (s *AdminTokenService) GetTokens(c context.Context) ([]string, error) {
	return s.ts.ListAll(c)
}
