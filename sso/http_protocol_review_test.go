package sso

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// TestSignerPreservesRepeatedValueOrder verifies signing matches first-value parameter semantics. TestSignerPreservesRepeatedValueOrder 验证签名与参数取首值的语义一致。
func TestSignerPreservesRepeatedValueOrder(t *testing.T) {
	signer := NewSignerWithParams("secret", ParamNames{Client: "application"})
	values := url.Values{"application": {"app"}, "tag": {"second", "first"}}
	signed := signer.AttachSign(values)
	if signed.Get("sign") == "" || !NewSigner("secret").Verify(signed) {
		t.Fatal("partial parameter config did not retain the default signature name")
	}
	if values.Get("sign") != "" || values.Get("tag") != "second" {
		t.Fatal("signing mutated the caller's parameters")
	}
	signed["tag"] = []string{"first", "second"}
	if signer.Verify(signed) {
		t.Fatal("reordering repeated values must invalidate the signature")
	}
	signed = signer.AttachSign(values)
	signed.Add("sign", signed.Get("sign"))
	if signer.Verify(signed) {
		t.Fatal("multiple signature values must be rejected")
	}
	empty := NewSigner("")
	if empty.Verify(empty.AttachSign(values)) {
		t.Fatal("an empty signing secret must not validate requests")
	}
}

// protocolReviewPost sends a form through a handler without starting a server. protocolReviewPost 通过处理器提交表单，无需启动服务。
func protocolReviewPost(handler http.HandlerFunc, values url.Values) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	handler(recorder, request)
	return recorder
}

// TestHTTPExchangeRejectsConflictingModes verifies rejected forms preserve both one-time credentials. TestHTTPExchangeRejectsConflictingModes 验证冲突表单被拒绝且不会消费一次性凭证。
func TestHTTPExchangeRejectsConflictingModes(t *testing.T) {
	ctx := context.Background()
	s := NewServer()
	defer s.Close()
	client := newTestClient()
	client.Modes = []Mode{ModeTicket, ModeOAuth2}
	if err := s.RegisterClient(client); err != nil {
		t.Fatal(err)
	}
	ticket, err := s.GenerateTicket(ctx, client.ClientID, "user", client.RedirectURIs[0], nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	code, err := s.GenerateOAuth2Code(ctx, client.ClientID, "user", client.RedirectURIs[0], nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	h := NewHTTPServer(s, HTTPOptions{})
	base := url.Values{"client": {client.ClientID}, "clientSecret": {client.ClientSecret}, "redirect": {client.RedirectURIs[0]}}
	for _, tc := range []struct {
		mode, ticket, code string
	}{
		{"", ticket.Ticket, code.Code},
		{string(ModeTicket), "", code.Code},
		{string(ModeOAuth2), ticket.Ticket, ""},
		{string(ModeSharedToken), ticket.Ticket, ""},
		{"unknown", "", code.Code},
	} {
		values := cloneValues(base)
		values.Set("mode", tc.mode)
		values.Set("ticket", tc.ticket)
		values.Set("code", tc.code)
		if response := protocolReviewPost(h.HandleToken, values); response.Code != http.StatusBadRequest {
			t.Fatalf("conflicting form status = %d, body = %s", response.Code, response.Body.String())
		}
	}
	// Omitted mode remains compatible with both credential kinds. 省略模式时仍兼容两类凭证。
	for key, value := range map[string]string{"ticket": ticket.Ticket, "code": code.Code} {
		values := cloneValues(base)
		values.Set(key, value)
		if response := protocolReviewPost(h.HandleToken, values); response.Code != http.StatusOK {
			t.Fatalf("%s exchange after rejection = %d, body = %s", key, response.Code, response.Body.String())
		}
	}
}

// TestHTTPInspectionHonorsDisabledOneTimeModes verifies introspection and user info enforce current client policy while revocation remains available. TestHTTPInspectionHonorsDisabledOneTimeModes 验证内省和用户信息遵循当前客户端策略，同时仍允许撤销。
func TestHTTPInspectionHonorsDisabledOneTimeModes(t *testing.T) {
	ctx := context.Background()
	s := NewServer()
	defer s.Close()
	client := newTestClient()
	client.Modes = []Mode{ModeTicket, ModeOAuth2}
	if err := s.RegisterClient(client); err != nil {
		t.Fatal(err)
	}
	ticket, err := s.GenerateTicket(ctx, client.ClientID, "user", client.RedirectURIs[0], nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	code, err := s.GenerateOAuth2Code(ctx, client.ClientID, "user", client.RedirectURIs[0], nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	client.Modes = []Mode{ModeRemoteSession}
	if err = s.RegisterClient(client); err != nil {
		t.Fatal(err)
	}
	h := NewHTTPServer(s, HTTPOptions{})
	for _, tc := range []struct {
		mode       Mode
		key, value string
	}{
		{ModeTicket, "ticket", ticket.Ticket},
		{ModeOAuth2, "code", code.Code},
	} {
		values := url.Values{"client": {client.ClientID}, "clientSecret": {client.ClientSecret}, "mode": {string(tc.mode)}, tc.key: {tc.value}}
		for _, handler := range []http.HandlerFunc{h.HandleIntrospect, h.HandleUserInfo} {
			if response := protocolReviewPost(handler, values); response.Code != http.StatusBadRequest {
				t.Fatalf("%s disabled mode status = %d, body = %s", tc.mode, response.Code, response.Body.String())
			}
		}
		for range 2 {
			if response := protocolReviewPost(h.HandleRevoke, values); response.Code != http.StatusOK {
				t.Fatalf("%s revoke status = %d, body = %s", tc.mode, response.Code, response.Body.String())
			}
		}
	}
}

// registrationFailureStorage injects startup read or write failures. registrationFailureStorage 注入启动时的读取或写入错误。
type registrationFailureStorage struct {
	*MemoryStorage
	failRead  bool
	failWrite bool
}

// Get injects a client lookup failure when requested. Get 按需注入客户端查询错误。
func (s *registrationFailureStorage) Get(ctx context.Context, key string) (any, error) {
	if s.failRead {
		return nil, errors.New("registration read failed")
	}
	return s.MemoryStorage.Get(ctx, key)
}

// Set injects a client registration failure when requested. Set 按需注入客户端注册错误。
func (s *registrationFailureStorage) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	if s.failWrite {
		return errors.New("registration write failed")
	}
	return s.MemoryStorage.Set(ctx, key, value, ttl)
}

// TestHTTPStartupRegistrationErrorsBlockHandlers verifies startup errors remain visible to every route. TestHTTPStartupRegistrationErrorsBlockHandlers 验证所有路由都保留并报告启动错误。
func TestHTTPStartupRegistrationErrorsBlockHandlers(t *testing.T) {
	for _, scenario := range []string{"initial clients", "anonymous read", "anonymous write"} {
		t.Run(scenario, func(t *testing.T) {
			storage := &registrationFailureStorage{MemoryStorage: NewMemoryStorage(), failWrite: scenario != "anonymous read", failRead: scenario == "anonymous read"}
			s := NewServer(WithStorage(storage))
			defer s.Close()
			options := HTTPOptions{}
			if scenario == "initial clients" {
				options.Clients = map[string]Client{"app": {}}
			} else {
				options.AllowAnonymousClient = true
			}
			h := NewHTTPServer(s, options)
			if err := h.initializationError(); !errors.Is(err, ErrServerNotInitialized) || !errors.Is(err, ErrStorageUnavailable) {
				t.Fatalf("startup error = %v", err)
			}
			storage.failRead, storage.failWrite = false, false
			for _, handler := range []http.HandlerFunc{h.HandleAuthorize, h.HandleToken, h.HandleIntrospect, h.HandleUserInfo, h.HandleRevoke, h.HandleLogout} {
				if response := protocolReviewPost(handler, nil); response.Code != http.StatusInternalServerError {
					t.Fatalf("incomplete initialization status = %d, body = %s", response.Code, response.Body.String())
				}
			}
		})
	}
}
