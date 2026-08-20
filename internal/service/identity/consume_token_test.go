package identity_test

import (
	"context"
	"testing"

	"github.com/StellaShiina/ktauth/internal/service/identity"
)

type tokenConsumerMock struct {
	calls int
	token string
	value bool
}

func (m *tokenConsumerMock) Consume(_ context.Context, token string) bool {
	m.calls++
	m.token = token
	return m.value
}

func TestConsumeTokenServiceDelegates(t *testing.T) {
	consumer := &tokenConsumerMock{value: true}
	service := identity.NewConsumeTokenService(consumer)

	if !service.Consume(context.Background(), "token-1") {
		t.Fatal("Consume = false, want true")
	}
	if consumer.calls != 1 || consumer.token != "token-1" {
		t.Fatalf("consumer call = %#v, want one call with token-1", consumer)
	}
}

func TestConsumeTokenServiceReturnsConsumerResult(t *testing.T) {
	consumer := &tokenConsumerMock{}
	service := identity.NewConsumeTokenService(consumer)

	if service.Consume(context.Background(), "invalid") {
		t.Fatal("Consume = true, want consumer result false")
	}
}
