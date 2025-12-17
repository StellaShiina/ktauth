package identity

import (
	"context"

	"github.com/StellaShiina/ktauth/internal/repository"
)

type SessionService struct {
	sessionRepo *repository.SessionRepo
}

func NewSessionService(r *repository.SessionRepo) *SessionService {
	return &SessionService{r}
}

// Create a user session with uuid jti. Return error when redis set error
func (s *SessionService) CreateSession(c context.Context, UUID, jti string) error {
	return s.sessionRepo.CreateSession(c, UUID, jti)
}

// Delete a session with a specific jti. Return error when redis del error
func (s *SessionService) DelSession(c context.Context, UUID, jti string) error {
	return s.sessionRepo.DelSession(c, UUID, jti)
}

// Get a user session with jti. Return uuid, err.
func (s *SessionService) GetSession(c context.Context, UUID, jti string) (string, error) {
	return s.sessionRepo.GetSession(c, UUID, jti)
}
