package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/StellaShiina/ktauth/internal/auth"
	"github.com/StellaShiina/ktauth/internal/middleware"
	"github.com/gin-gonic/gin"
)

type sessionReaderMock struct {
	calls int
	uuid  string
	jti   string
	value string
	err   error
}

func (m *sessionReaderMock) GetSession(_ context.Context, uuid, jti string) (string, error) {
	m.calls++
	m.uuid, m.jti = uuid, jti
	return m.value, m.err
}

func authRequest() (*gin.Engine, *httptest.ResponseRecorder, *string, *string) {
	recorder := httptest.NewRecorder()
	engine := gin.New()
	var gotUUID, gotJTI string
	return engine, recorder, &gotUUID, &gotJTI
}

func TestAuthMiddlewareRejectsMissingAndInvalidToken(t *testing.T) {
	mock := &sessionReaderMock{}
	m := middleware.NewAuthMiddleWare(mock)

	for _, token := range []string{"", "not-a-token"} {
		engine, recorder, _, _ := authRequest()
		engine.GET("/protected", m.VerifySession(""), func(c *gin.Context) { c.Status(http.StatusOK) })
		engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/protected", nil))
		if token != "" {
			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			request.Header.Set("Authorization", "Bearer "+token)
			recorder = httptest.NewRecorder()
			engine.ServeHTTP(recorder, request)
		}
		if recorder.Code != http.StatusUnauthorized {
			t.Errorf("token %q status = %d, want 401", token, recorder.Code)
		}
	}
	if mock.calls != 0 {
		t.Fatalf("session calls = %d, want 0", mock.calls)
	}
}

func TestAuthMiddlewareAcceptsSessionAndSetsClaims(t *testing.T) {
	token, jti, err := auth.SignToken("u1", "alice", "user")
	if err != nil {
		t.Fatal(err)
	}
	mock := &sessionReaderMock{value: "u1"}
	m := middleware.NewAuthMiddleWare(mock)
	engine, recorder, gotUUID, gotJTI := authRequest()
	nextCalls := 0
	engine.GET("/protected", m.VerifySession(""), func(c *gin.Context) {
		nextCalls++
		*gotUUID = c.GetString("uuid")
		*gotJTI = c.GetString("jti")
		c.Status(http.StatusOK)
	})
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK || mock.calls != 1 || mock.uuid != "u1" || mock.jti != jti || nextCalls != 1 {
		t.Fatalf("status/session/next = %d/%#v/%d", recorder.Code, mock, nextCalls)
	}
	if *gotUUID != "u1" || *gotJTI != jti {
		t.Fatalf("claims in context = uuid %q, jti %q", *gotUUID, *gotJTI)
	}
}

func TestAuthMiddlewareRejectsSessionMismatchAndRole(t *testing.T) {
	token, _, err := auth.SignToken("u1", "alice", "user")
	if err != nil {
		t.Fatal(err)
	}

	mock := &sessionReaderMock{value: "u2"}
	m := middleware.NewAuthMiddleWare(mock)
	engine, recorder, _, _ := authRequest()
	engine.GET("/protected", m.VerifySession(""), func(c *gin.Context) { c.Status(http.StatusOK) })
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("session mismatch status = %d, want 401", recorder.Code)
	}

	mock = &sessionReaderMock{value: "u1"}
	m = middleware.NewAuthMiddleWare(mock)
	engine, recorder, _, _ = authRequest()
	engine.GET("/protected", m.VerifySession("admin"), func(c *gin.Context) { c.Status(http.StatusOK) })
	request = httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("role mismatch status = %d, want 401", recorder.Code)
	}
}

func TestAuthMiddlewareRejectsSessionError(t *testing.T) {
	token, _, err := auth.SignToken("u1", "alice", "user")
	if err != nil {
		t.Fatal(err)
	}
	m := middleware.NewAuthMiddleWare(&sessionReaderMock{err: errors.New("redis unavailable")})
	engine, recorder, _, _ := authRequest()
	engine.GET("/protected", m.VerifySession(""), func(c *gin.Context) { c.Status(http.StatusOK) })
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("session error status = %d, want 401", recorder.Code)
	}
}
