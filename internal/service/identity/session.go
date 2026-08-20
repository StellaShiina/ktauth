package identity

import (
	"context"
)

type SessionStore interface {
	CreateSession(ctx context.Context, uuid, jti string) error
	DelSession(ctx context.Context, uuid, jti string) error
	GetSession(ctx context.Context, uuid, jti string) (string, error)
}

type SessionService struct {
	sessionStore SessionStore
}

func NewSessionService(r SessionStore) *SessionService {
	return &SessionService{r}
}

// Create a user session with uuid jti. Return error when redis set error
func (s *SessionService) CreateSession(c context.Context, UUID, jti string) error {
	return s.sessionStore.CreateSession(c, UUID, jti)
}

// Delete a session with a specific jti. Return error when redis del error
func (s *SessionService) DelSession(c context.Context, UUID, jti string) error {
	return s.sessionStore.DelSession(c, UUID, jti)
}

// Get a user session with jti. Return uuid, err.
func (s *SessionService) GetSession(c context.Context, UUID, jti string) (string, error) {
	return s.sessionStore.GetSession(c, UUID, jti)
}
