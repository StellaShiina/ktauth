package access

import (
	"context"

	"github.com/StellaShiina/ktauth/internal/repository"
)

type CDService struct {
	cdRepo *repository.CountDownRepo
}

func NewCDService(r *repository.CountDownRepo) *CDService {
	return &CDService{r}
}

func (s *CDService) Set(c context.Context, key string) error {
	return s.cdRepo.Set(c, key)
}

// Check if the given key is in cd
func (s *CDService) Check(c context.Context, key string) (bool, error) {
	return s.cdRepo.CD(c, key)
}
