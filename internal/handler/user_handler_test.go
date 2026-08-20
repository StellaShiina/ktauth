package handler_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/StellaShiina/ktauth/internal/crypto"
	"github.com/StellaShiina/ktauth/internal/handler"
	"github.com/StellaShiina/ktauth/internal/model"
	"github.com/gin-gonic/gin"
)

type sessionManagerMock struct {
	createCalls int
	deleteCalls int
	uuid        string
	jti         string
	err         error
}

func (m *sessionManagerMock) CreateSession(_ context.Context, uuid, jti string) error {
	m.createCalls++
	m.uuid, m.jti = uuid, jti
	return m.err
}

func (m *sessionManagerMock) DelSession(_ context.Context, uuid, jti string) error {
	m.deleteCalls++
	m.uuid, m.jti = uuid, jti
	return m.err
}

type accountManagerMock struct {
	newCalls int
	getCalls int
	newUUID  string
	getName  string
	user     model.User
	err      error
}

func (m *accountManagerMock) NewUser(_ context.Context, name, _ string, _ *string, _ string) (string, error) {
	m.newCalls++
	m.getName = name
	return m.newUUID, m.err
}

func (m *accountManagerMock) GetUserByName(_ context.Context, name string) (model.User, error) {
	m.getCalls++
	m.getName = name
	return m.user, m.err
}

type tokenConsumerMock struct {
	calls int
	token string
	valid bool
}

func (m *tokenConsumerMock) Consume(_ context.Context, token string) bool {
	m.calls++
	m.token = token
	return m.valid
}

type emailManagerMock struct {
	sendCalls   int
	verifyCalls int
	email       string
	code        string
	valid       bool
	err         error
}

func (m *emailManagerMock) SendCode(_ context.Context, email string) error {
	m.sendCalls++
	m.email = email
	return m.err
}

func (m *emailManagerMock) VerifyCode(_ context.Context, email, code string) (bool, error) {
	m.verifyCalls++
	m.email, m.code = email, code
	return m.valid, m.err
}

func newUserHandlerContext(method, target, body string) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	context, _ := gin.CreateTestContext(recorder)
	context.Request = request
	return context, recorder
}

func TestUserHandlerRegisterWithToken(t *testing.T) {
	account := &accountManagerMock{newUUID: "new-user"}
	token := &tokenConsumerMock{valid: true}
	h := handler.NewUserHandler(&sessionManagerMock{}, account, token)
	c, recorder := newUserHandlerContext(http.MethodPost, "/register", `{"token":"invite","user":"alice","password":"secret"}`)

	h.RegisterUser(c)

	if recorder.Code != http.StatusCreated || !strings.Contains(recorder.Body.String(), "new-user") {
		t.Fatalf("status/body = %d/%q, want 201 with new-user", recorder.Code, recorder.Body.String())
	}
	if token.calls != 1 || token.token != "invite" || account.newCalls != 1 {
		t.Fatalf("dependency calls = token %#v, account %#v", token, account)
	}
}

func TestUserHandlerRegisterRejectsInvalidToken(t *testing.T) {
	account := &accountManagerMock{newUUID: "new-user"}
	token := &tokenConsumerMock{}
	h := handler.NewUserHandler(&sessionManagerMock{}, account, token)
	c, recorder := newUserHandlerContext(http.MethodPost, "/register", `{"token":"invite","user":"alice","password":"secret"}`)

	h.RegisterUser(c)

	if recorder.Code != http.StatusUnauthorized || account.newCalls != 0 {
		t.Fatalf("status/account calls = %d/%d, want 401/0", recorder.Code, account.newCalls)
	}
}

func TestUserHandlerRegisterWithEmailCode(t *testing.T) {
	account := &accountManagerMock{newUUID: "new-user"}
	email := &emailManagerMock{valid: true}
	h := handler.NewUserHandler(&sessionManagerMock{}, account, &tokenConsumerMock{}, email)
	c, recorder := newUserHandlerContext(http.MethodPost, "/register", `{"user":"alice","password":"secret","email":"alice@example.com","code":"123456"}`)

	h.RegisterUser(c)

	if recorder.Code != http.StatusCreated || email.verifyCalls != 1 || account.newCalls != 1 {
		t.Fatalf("status/calls = %d/%d/%d, want 201/1/1", recorder.Code, email.verifyCalls, account.newCalls)
	}
}

func TestUserHandlerSendAndVerifyEmailCode(t *testing.T) {
	email := &emailManagerMock{valid: true}
	h := handler.NewUserHandler(&sessionManagerMock{}, &accountManagerMock{}, &tokenConsumerMock{}, email)

	c, recorder := newUserHandlerContext(http.MethodPost, "/send-code", `{"email":"alice@example.com"}`)
	h.SendEmailCode(c)
	if recorder.Code != http.StatusOK || email.sendCalls != 1 || email.email != "alice@example.com" {
		t.Fatalf("send status/call = %d/%d, want 200/1", recorder.Code, email.sendCalls)
	}

	c, recorder = newUserHandlerContext(http.MethodPost, "/verify-code", `{"email":"alice@example.com","code":"123456"}`)
	h.VerifyEmailCode(c)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"valid":true`) || email.verifyCalls != 1 {
		t.Fatalf("verify status/body/calls = %d/%q/%d", recorder.Code, recorder.Body.String(), email.verifyCalls)
	}
}

func TestUserHandlerEmailConfigurationAndValidationErrors(t *testing.T) {
	h := handler.NewUserHandler(&sessionManagerMock{}, &accountManagerMock{}, &tokenConsumerMock{})
	c, recorder := newUserHandlerContext(http.MethodPost, "/send-code", `{"email":"alice@example.com"}`)
	h.SendEmailCode(c)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("SendEmailCode status = %d, want 500", recorder.Code)
	}

	email := &emailManagerMock{err: errors.New("verification code recently sent")}
	h = handler.NewUserHandler(&sessionManagerMock{}, &accountManagerMock{}, &tokenConsumerMock{}, email)
	c, recorder = newUserHandlerContext(http.MethodPost, "/send-code", `{"email":"alice@example.com"}`)
	h.SendEmailCode(c)
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("SendEmailCode status = %d, want 429", recorder.Code)
	}

	c, recorder = newUserHandlerContext(http.MethodPost, "/verify-code", `{"email":"alice@example.com"}`)
	h.VerifyEmailCode(c)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("VerifyEmailCode status = %d, want 400", recorder.Code)
	}
}

func TestUserHandlerLoginAndLogout(t *testing.T) {
	passwordHash, err := crypto.HashPassword("secret")
	if err != nil {
		t.Fatal(err)
	}
	session := &sessionManagerMock{}
	account := &accountManagerMock{user: model.User{UUID: "u1", Name: "alice", Role: "user", PasswordHash: passwordHash}}
	h := handler.NewUserHandler(session, account, &tokenConsumerMock{})

	c, recorder := newUserHandlerContext(http.MethodPost, "/login", `{"user":"alice","password":"secret"}`)
	h.LoginUser(c)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"token"`) || session.createCalls != 1 {
		t.Fatalf("login status/body/session = %d/%q/%d", recorder.Code, recorder.Body.String(), session.createCalls)
	}
	if session.uuid != "u1" || session.jti == "" {
		t.Fatalf("session call = %#v", session)
	}

	c, recorder = newUserHandlerContext(http.MethodGet, "/logout", "")
	c.Set("uuid", "u1")
	c.Set("jti", session.jti)
	h.LogoutUser(c)
	if recorder.Code != http.StatusOK || recorder.Body.String() != "OK" || session.deleteCalls != 1 {
		t.Fatalf("logout status/body/calls = %d/%q/%d", recorder.Code, recorder.Body.String(), session.deleteCalls)
	}
}

func TestUserHandlerLoginRejectsWrongPassword(t *testing.T) {
	passwordHash, err := crypto.HashPassword("secret")
	if err != nil {
		t.Fatal(err)
	}
	session := &sessionManagerMock{}
	account := &accountManagerMock{user: model.User{PasswordHash: passwordHash}}
	h := handler.NewUserHandler(session, account, &tokenConsumerMock{})
	c, recorder := newUserHandlerContext(http.MethodPost, "/login", `{"user":"alice","password":"wrong"}`)

	h.LoginUser(c)

	if recorder.Code != http.StatusUnauthorized || session.createCalls != 0 {
		t.Fatalf("status/session calls = %d/%d, want 401/0", recorder.Code, session.createCalls)
	}
}
