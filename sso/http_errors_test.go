package sso

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// assertHTTPFailure checks the public envelope and response protections without accepting identity data. assertHTTPFailure 检查公开错误格式及响应保护，不接受身份数据。
func assertHTTPFailure(t *testing.T, w *httptest.ResponseRecorder, status int) {
	t.Helper()
	var body Response
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || w.Code != status || body.Code != status || body.Message == "" {
		t.Fatalf("invalid failure envelope: HTTP=%d decode=%v", w.Code, err)
	}
	if body.Data != nil || strings.Contains(w.Body.String(), "private-") || w.Header().Get("Location") != "" || len(w.Result().Cookies()) != 0 {
		t.Fatal("failure exposed diagnostics or emitted login/logout state")
	}
	if w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Pragma") != "no-cache" || w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("failure lost response protections")
	}
}

// TestHTTPErrorClassification keeps infrastructure causes from masquerading as credential rejections. TestHTTPErrorClassification 防止基础设施错误被底层原因伪装成凭证拒绝。
func TestHTTPErrorClassification(t *testing.T) {
	for _, tc := range []struct {
		name    string
		err     error
		status  int
		message string
	}{
		{"unknown", errors.New("private-database-address"), 500, "SSO operation failed"},
		{"request", fmt.Errorf("private-field: %w", ErrInvalidRequest), 400, ErrInvalidRequest.Error()},
		{"signature", fmt.Errorf("private-key: %w", ErrInvalidSign), 401, ErrInvalidSign.Error()},
		{"policy denied", ErrAuthorizationDenied, 403, ErrAuthorizationDenied.Error()},
		{"missing client", ErrClientNotFound, 404, ErrClientNotFound.Error()},
		{"method", ErrMethodNotAllowed, 405, ErrMethodNotAllowed.Error()},
		{"size", ErrRequestTooLarge, 413, ErrRequestTooLarge.Error()},
		{"policy infrastructure", &authorizationPolicyError{ErrAuthorizationPolicyUnavailable, ErrInvalidTicket}, 503, ErrAuthorizationPolicyUnavailable.Error()},
		{"storage infrastructure", errors.Join(ErrStorageUnavailable, ErrRequestReplayed), 500, ErrStorageUnavailable.Error()},
		{"initialization", errors.Join(ErrServerNotInitialized, ErrInvalidRequest), 500, ErrServerNotInitialized.Error()},
		{"canceled", errors.Join(context.Canceled, ErrInvalidTicket), 500, context.Canceled.Error()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, message := protocolError(tc.err)
			if status != tc.status || statusFromError(tc.err) != status || message != tc.message {
				t.Fatalf("classification=%d %q", status, message)
			}
		})
	}
}

// TestHTTPErrorInitialization covers direct handlers, registered routes and failed Handler construction. TestHTTPErrorInitialization 覆盖直接处理器、注册路由及失败的 Handler 构造。
func TestHTTPErrorInitialization(t *testing.T) {
	store := &registrationFailureStorage{MemoryStorage: NewMemoryStorage(), failWrite: true}
	s := NewServer(WithStorage(store))
	defer s.Close()
	calls := 0
	h := NewHTTPServer(s, HTTPOptions{ServerOptions: ServerOptions{Clients: map[string]Client{"app": {}}},
		OnHTTPError: func(r *http.Request, status int, err error) {
			calls++
			if status != 500 || !errors.Is(err, ErrServerNotInitialized) || !errors.Is(err, ErrStorageUnavailable) || r == nil {
				t.Fatal("initialization observer lost the retained cause")
			}
		},
	})
	store.failWrite = false
	mux := http.NewServeMux()
	h.Register(mux)
	for _, tc := range []struct {
		path    string
		handler http.HandlerFunc
	}{
		{"/sso/authorize", h.HandleAuthorize}, {"/sso/token", h.HandleToken},
		{"/sso/introspect", h.HandleIntrospect}, {"/sso/userinfo", h.HandleUserInfo},
		{"/sso/revoke", h.HandleRevoke}, {"/sso/logout", h.HandleLogout},
	} {
		for _, handler := range []http.Handler{tc.handler, mux, h.Handler()} {
			before := calls
			w := replayTestRequest(handler, http.MethodPost, tc.path, nil)
			assertHTTPFailure(t, w, 500)
			if calls != before+1 || strings.Contains(w.Body.String(), "registration write failed") {
				t.Fatal("initialization diagnostic escaped or observer count differed")
			}
		}
	}
}

// TestHTTPErrorReplayStorage checks all six endpoints before business effects, then retries with the unclaimed nonce. TestHTTPErrorReplayStorage 检查六个端点在业务副作用前的存储失败，再以未占用随机值重试。
func TestHTTPErrorReplayStorage(t *testing.T) {
	for _, tc := range []struct {
		action, path, method string
		success              int
	}{
		{ActionAuthorize, "/sso/authorize", http.MethodGet, 302},
		{ActionToken, "/sso/token", http.MethodPost, 200},
		{ActionIntrospect, "/sso/introspect", http.MethodPost, 200},
		{ActionUserInfo, "/sso/userinfo", http.MethodPost, 200},
		{ActionRevoke, "/sso/revoke", http.MethodPost, 200},
		{ActionLogout, "/sso/logout", http.MethodPost, 200},
	} {
		t.Run(tc.action, func(t *testing.T) {
			s := NewServer()
			defer s.Close()
			client := newTestClient()
			client.SecretKey = "key"
			if err := s.RegisterClient(client); err != nil {
				t.Fatal(err)
			}
			ticket, err := s.GenerateTicket(context.Background(), client.ClientID, "alice", client.RedirectURIs[0], nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			failure := errors.New("private-replay-backend")
			store := &replayFaultStorage{Storage: NewMemoryStorage(), readErr: failure}
			protection := newReplayTestProtection(t, store)
			observed, effects := 0, 0
			h := NewHTTPServer(s, HTTPOptions{
				ServerOptions: ServerOptions{CheckSign: true}, ReplayProtection: protection,
				LoginIDResolver:       func(*http.Request) (string, bool) { return "alice", true },
				AuthorizationExtra:    func(*http.Request, string, string) (map[string]any, error) { effects++; return nil, nil },
				EnrichExchangeResult:  func(*http.Request, string, *TicketExchangeResult) error { effects++; return nil },
				CheckCredentialActive: func(*http.Request, CredentialInfo) (bool, error) { effects++; return true, nil },
				CenterLogout:          func(*http.Request, string) error { effects++; return nil },
				OnHTTPError: func(_ *http.Request, status int, err error) {
					observed++
					if status != 500 || !errors.Is(err, ErrStorageUnavailable) || !errors.Is(err, failure) {
						t.Fatal("observer lost replay storage cause")
					}
				},
			})
			values := url.Values{"client": {client.ClientID}, "clientSecret": {client.ClientSecret}, "redirect": {client.RedirectURIs[0]}}
			if tc.action != ActionAuthorize && tc.action != ActionLogout {
				values.Set("ticket", ticket.Ticket)
			}
			values = NewSigner("key").AttachSign(protection.stamp(values, DefaultParamNames(), tc.action))
			assertHTTPFailure(t, replayTestRequest(h.Handler(), tc.method, tc.path, values), 500)
			if observed != 1 || effects != 0 {
				t.Fatal("failed claim invoked business effects or repeated observation")
			}
			if _, err := s.ValidateTicket(context.Background(), ticket.Ticket); err != nil {
				t.Fatal("failed claim changed the credential")
			}
			store.readErr = nil
			w := replayTestRequest(h.Handler(), tc.method, tc.path, values)
			if w.Code != tc.success || observed != 1 {
				t.Fatalf("recovery HTTP=%d observers=%d", w.Code, observed)
			}
		})
	}
}
