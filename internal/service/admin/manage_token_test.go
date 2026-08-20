package admin_test

import (
	"context"
	"errors"
	"testing"

	"github.com/StellaShiina/ktauth/internal/service/admin"
)

type tokenStoreMock struct {
	restockCalls int
	flushCalls   int
	getCalls     int
	listCalls    int
	token        string
	tokens       []string
	restockErr   error
	flushErr     error
	getErr       error
	listErr      error
}

func (m *tokenStoreMock) Restock(context.Context) error {
	m.restockCalls++
	return m.restockErr
}

func (m *tokenStoreMock) FlushAll(context.Context) error {
	m.flushCalls++
	return m.flushErr
}

func (m *tokenStoreMock) GetOne(context.Context) (string, error) {
	m.getCalls++
	return m.token, m.getErr
}

func (m *tokenStoreMock) ListAll(context.Context) ([]string, error) {
	m.listCalls++
	return m.tokens, m.listErr
}

func TestAdminTokenServiceDelegates(t *testing.T) {
	store := &tokenStoreMock{token: "token-1", tokens: []string{"token-1", "token-2"}}
	service := admin.NewAdminTokenService(store)

	if err := service.Restock(context.Background()); err != nil {
		t.Fatalf("Restock returned error: %v", err)
	}
	if err := service.FlushTokens(context.Background()); err != nil {
		t.Fatalf("FlushTokens returned error: %v", err)
	}
	token, err := service.GetToken(context.Background())
	if err != nil || token != "token-1" {
		t.Fatalf("GetToken = %q, %v; want token-1, nil", token, err)
	}
	tokens, err := service.GetTokens(context.Background())
	if err != nil || len(tokens) != 2 {
		t.Fatalf("GetTokens = %#v, %v; want two tokens, nil", tokens, err)
	}
	if store.restockCalls != 1 || store.flushCalls != 1 || store.getCalls != 1 || store.listCalls != 1 {
		t.Fatalf("store calls = %#v, want one call for each operation", store)
	}
}

func TestAdminTokenServiceReturnsStoreErrors(t *testing.T) {
	errWant := errors.New("redis unavailable")
	store := &tokenStoreMock{restockErr: errWant, flushErr: errWant, getErr: errWant, listErr: errWant}
	service := admin.NewAdminTokenService(store)

	if err := service.Restock(context.Background()); !errors.Is(err, errWant) {
		t.Errorf("Restock error = %v, want store error", err)
	}
	if err := service.FlushTokens(context.Background()); !errors.Is(err, errWant) {
		t.Errorf("FlushTokens error = %v, want store error", err)
	}
	if _, err := service.GetToken(context.Background()); !errors.Is(err, errWant) {
		t.Errorf("GetToken error = %v, want store error", err)
	}
	if _, err := service.GetTokens(context.Background()); !errors.Is(err, errWant) {
		t.Errorf("GetTokens error = %v, want store error", err)
	}
}
