package handler_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/StellaShiina/ktauth/internal/handler"
	"github.com/StellaShiina/ktauth/internal/repository"
	"github.com/StellaShiina/ktauth/internal/service/admin"
)

type ipRuleManagerMock struct {
	addCalls   int
	listCalls  int
	delCalls   int
	ip         string
	whitelist  bool
	note       *string
	version    *int16
	typeFilter *bool
	addResult  string
	listResult []admin.IPResponse
	addErr     error
	listErr    error
	delErr     error
}

func (m *ipRuleManagerMock) AddRule(_ context.Context, ip string, whitelist bool, note *string) (string, error) {
	m.addCalls++
	m.ip, m.whitelist, m.note = ip, whitelist, note
	return m.addResult, m.addErr
}

func (m *ipRuleManagerMock) ListRules(_ context.Context, version *int16, typeFilter *bool) ([]admin.IPResponse, error) {
	m.listCalls++
	m.version, m.typeFilter = version, typeFilter
	return m.listResult, m.listErr
}

func (m *ipRuleManagerMock) DelRule(_ context.Context, ip string) (string, error) {
	m.delCalls++
	m.ip = ip
	return m.addResult, m.delErr
}

type userManagerMock struct {
	users []admin.UserResponse
	err   error
}

func (m *userManagerMock) ListUsers(context.Context) ([]admin.UserResponse, error) {
	return m.users, m.err
}

func TestIPRuleHandlerAddRuleAndLegacyBan(t *testing.T) {
	note := "test"
	manager := &ipRuleManagerMock{addResult: "192.0.2.0/24"}
	h := handler.NewIPRuleHandler(manager)

	c, recorder := newUserHandlerContext(http.MethodPost, "/ips/new", `{"ip":"192.0.2.1","note":"test","isWhiteList":true}`)
	h.AddRule(c)
	if recorder.Code != http.StatusOK || manager.addCalls != 1 || !manager.whitelist || manager.ip != "192.0.2.1" || manager.note == nil || *manager.note != note {
		t.Fatalf("response/call = %d/%#v", recorder.Code, manager)
	}

	c, recorder = newUserHandlerContext(http.MethodPost, "/ips/new?ban=1", `{"ip":"192.0.2.1"}`)
	h.AddRule(c)
	if recorder.Code != http.StatusOK || manager.addCalls != 2 || manager.whitelist {
		t.Fatalf("legacy ban response/call = %d/%#v", recorder.Code, manager)
	}
}

func TestIPRuleHandlerListAndDelete(t *testing.T) {
	manager := &ipRuleManagerMock{listResult: []admin.IPResponse{{ID: 1, IPCIDR: "192.0.2.0/24", CreateAt: time.Now()}}, addResult: "192.0.2.0/24"}
	h := handler.NewIPRuleHandler(manager)

	c, recorder := newUserHandlerContext(http.MethodGet, "/ips?version=4&type=white", "")
	h.ListRules(c)
	if recorder.Code != http.StatusOK || manager.listCalls != 1 || manager.version == nil || *manager.version != 4 || manager.typeFilter == nil || !*manager.typeFilter {
		t.Fatalf("list response/call = %d/%#v", recorder.Code, manager)
	}

	c, recorder = newUserHandlerContext(http.MethodDelete, "/ips", `{"ip":"192.0.2.1"}`)
	h.DelRule(c)
	if recorder.Code != http.StatusOK || manager.delCalls != 1 {
		t.Fatalf("delete response/call = %d/%#v", recorder.Code, manager)
	}
}

func TestIPRuleHandlerMapsErrors(t *testing.T) {
	manager := &ipRuleManagerMock{addErr: repository.ErrIPExist, delErr: repository.ErrIPNotFound}
	h := handler.NewIPRuleHandler(manager)

	c, recorder := newUserHandlerContext(http.MethodPost, "/ips/new", `{"ip":"192.0.2.1"}`)
	h.AddRule(c)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("duplicate status = %d, want 400", recorder.Code)
	}

	c, recorder = newUserHandlerContext(http.MethodDelete, "/ips", `{"ip":"192.0.2.1"}`)
	h.DelRule(c)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("missing rule status = %d, want 400", recorder.Code)
	}

	manager = &ipRuleManagerMock{addErr: errors.New("database unavailable"), listErr: errors.New("database unavailable")}
	h = handler.NewIPRuleHandler(manager)
	c, recorder = newUserHandlerContext(http.MethodPost, "/ips/new", `{"ip":"192.0.2.1"}`)
	h.AddRule(c)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("generic add status = %d, want 500", recorder.Code)
	}
	c, recorder = newUserHandlerContext(http.MethodGet, "/ips", "")
	h.ListRules(c)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("generic list status = %d, want 500", recorder.Code)
	}
}

func TestUserManageHandlerListUsers(t *testing.T) {
	manager := &userManagerMock{users: []admin.UserResponse{{ID: "u1", Name: "alice"}}}
	h := handler.NewUserManageHandler(manager)
	c, recorder := newUserHandlerContext(http.MethodGet, "/users", "")
	h.ListUsers(c)
	if recorder.Code != http.StatusOK || recorder.Body.String() == "" {
		t.Fatalf("status/body = %d/%q, want 200 with JSON", recorder.Code, recorder.Body.String())
	}

	h = handler.NewUserManageHandler(&userManagerMock{err: errors.New("database unavailable")})
	c, recorder = newUserHandlerContext(http.MethodGet, "/users", "")
	h.ListUsers(c)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("error status = %d, want 500", recorder.Code)
	}
}
