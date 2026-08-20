package identity

import (
	"context"
)

type TokenConsumer interface {
	Consume(ctx context.Context, token string) bool
}

type ConsumeTokenService struct {
	consumer TokenConsumer
}

func NewConsumeTokenService(r TokenConsumer) *ConsumeTokenService {
	return &ConsumeTokenService{r}
}

func (s *ConsumeTokenService) Consume(c context.Context, token string) bool {
	return s.consumer.Consume(c, token)
}
