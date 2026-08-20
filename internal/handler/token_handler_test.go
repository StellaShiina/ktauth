package handler_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/StellaShiina/ktauth/internal/handler"
	"github.com/gin-gonic/gin"
)

type adminTokenManagerMock struct {
	restockCalls int
	flushCalls   int
	getCalls     int
	listCalls    int
	token        string
	tokens       []string
	err          error
}

func (m *adminTokenManagerMock) Restock(context.Context) error     { m.restockCalls++; return m.err }
func (m *adminTokenManagerMock) FlushTokens(context.Context) error { m.flushCalls++; return m.err }
func (m *adminTokenManagerMock) GetToken(context.Context) (string, error) {
	m.getCalls++
	return m.token, m.err
}
func (m *adminTokenManagerMock) GetTokens(context.Context) ([]string, error) {
	m.listCalls++
	return m.tokens, m.err
}

func TestTokenHandlerOperations(t *testing.T) {
	manager := &adminTokenManagerMock{token: "token-1", tokens: []string{"token-1", "token-2"}}
	h := handler.NewTokenHandler(manager)

	tests := []struct {
		name string
		call func(*gin.Context)
		want int
	}{
		{name: "restock", call: h.Restock, want: http.StatusCreated},
		{name: "flush", call: h.FlushTokens, want: http.StatusOK},
		{name: "get", call: h.GetToken, want: http.StatusOK},
		{name: "get all", call: h.GetTokens, want: http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, recorder := newUserHandlerContext(http.MethodGet, "/tokens", "")
			tt.call(c)
			if recorder.Code != tt.want {
				t.Fatalf("status = %d, want %d; body = %q", recorder.Code, tt.want, recorder.Body.String())
			}
		})
	}
	if manager.restockCalls != 1 || manager.flushCalls != 1 || manager.getCalls != 1 || manager.listCalls != 1 {
		t.Fatalf("manager calls = %#v", manager)
	}
}

func TestTokenHandlerReturnsServiceErrors(t *testing.T) {
	manager := &adminTokenManagerMock{err: errors.New("redis unavailable")}
	h := handler.NewTokenHandler(manager)

	tests := []struct {
		call func(*gin.Context)
		want int
	}{
		{call: h.Restock, want: http.StatusInternalServerError},
		{call: h.FlushTokens, want: http.StatusOK},
		{call: h.GetToken, want: http.StatusOK},
		{call: h.GetTokens, want: http.StatusOK},
	}
	for _, tt := range tests {
		c, recorder := newUserHandlerContext(http.MethodGet, "/tokens", "")
		tt.call(c)
		if recorder.Code != tt.want {
			t.Fatalf("status = %d, want %d", recorder.Code, tt.want)
		}
	}
}
