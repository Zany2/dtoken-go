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

// cancellationReviewStorage models a backend that returns success without observing request cancellation. cancellationReviewStorage 模拟未遵守请求取消但返回成功的存储。
type cancellationReviewStorage struct {
	*MemoryStorage
	writes   int
	afterSet func()
}

// Get deliberately ignores cancellation to expose missing checks at application boundaries. Get 特意忽略取消，以暴露应用边界缺失的检查。
func (s *cancellationReviewStorage) Get(_ context.Context, key string) (any, error) {
	return s.MemoryStorage.Get(context.Background(), key)
}

// Set records a successful write and optionally ends the request afterward. Set 记录成功写入，并可在随后终止请求。
func (s *cancellationReviewStorage) Set(_ context.Context, key string, value any, ttl time.Duration) error {
	s.writes++
	err := s.MemoryStorage.Set(context.Background(), key, value, ttl)
	if s.afterSet != nil {
		s.afterSet()
	}
	return err
}

// TestCancellationReviewAuthorization stops all four modes before issuance after canceled identity or metadata hooks. TestCancellationReviewAuthorization 在身份或元数据钩子取消后阻止四种模式签发。
func TestCancellationReviewAuthorization(t *testing.T) {
	for _, mode := range []Mode{ModeTicket, ModeOAuth2, ModeSharedToken, ModeRemoteSession} {
		for _, stage := range []string{"before", "resolver", "metadata", "persistence"} {
			t.Run(string(mode)+"/"+stage, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				store := &cancellationReviewStorage{MemoryStorage: NewMemoryStorage()}
				s := NewServer(WithStorage(store))
				defer s.Close()
				client := newTestClient()
				client.Modes = []Mode{mode}
				if err := s.RegisterClient(client); err != nil {
					t.Fatal(err)
				}
				store.writes = 0
				resolverCalls, extraCalls := 0, 0
				h := NewHTTPServer(s, HTTPOptions{
					ServerOptions: ServerOptions{EnableSLO: true},
					LoginIDResolver: func(*http.Request) (string, bool) {
						resolverCalls++
						if stage == "resolver" {
							cancel()
						}
						return "alice", true
					},
					AuthorizationExtra: func(*http.Request, string, string) (map[string]any, error) {
						extraCalls++
						if stage == "metadata" {
							cancel()
						}
						return map[string]any{"private": "value"}, nil
					},
				})
				if stage == "before" {
					cancel()
				}
				if stage == "persistence" {
					store.afterSet = cancel
				}
				values := url.Values{"client": {client.ClientID}, "redirect": {client.RedirectURIs[0]}, "callback": {client.RedirectURIs[0]}, "mode": {string(mode)}}
				w := httptest.NewRecorder()
				h.HandleAuthorize(w, httptest.NewRequest("GET", "/sso/authorize?"+values.Encode(), nil).WithContext(ctx))
				if w.Code < 400 || w.Header().Get("Location") != "" {
					t.Fatalf("canceled authorization redirected: %d", w.Code)
				}
				if stage == "before" && resolverCalls != 0 || (stage == "before" || stage == "resolver") && extraCalls != 0 {
					t.Fatal("canceled authorization invoked later hooks")
				}
				wantWrites := 0
				if stage == "persistence" {
					wantWrites = 1
				}
				if store.writes != wantWrites {
					t.Fatalf("writes=%d, want %d", store.writes, wantWrites)
				}
				if bindings, err := s.GetClientSessions(context.Background(), "alice"); err != nil || len(bindings) != 0 {
					t.Fatalf("canceled issuance registered callbacks: %v, %v", bindings, err)
				}
			})
		}
	}
}

// TestCancellationReviewInspection rejects both active and inactive hook results after cancellation in all modes. TestCancellationReviewInspection 在四种模式中拒绝取消后返回的有效或无效状态结果。
func TestCancellationReviewInspection(t *testing.T) {
	s := NewServer()
	defer s.Close()
	client := newTestClient()
	client.Modes = []Mode{ModeTicket, ModeOAuth2, ModeSharedToken, ModeRemoteSession}
	if err := s.RegisterClient(client); err != nil {
		t.Fatal(err)
	}
	issued := issueLogoutCredentials(t, s, client, "alice")
	for _, tc := range []struct {
		mode       Mode
		key, value string
	}{
		{ModeTicket, "ticket", issued.ticket.Ticket}, {ModeOAuth2, "code", issued.code.Code},
		{ModeSharedToken, "tokenValue", issued.token.Token}, {ModeRemoteSession, "sessionId", issued.session.SessionID},
	} {
		for _, active := range []bool{true, false} {
			for _, endpoint := range []string{"/sso/introspect", "/sso/userinfo"} {
				ctx, cancel := context.WithCancel(context.Background())
				calls := 0
				h := NewHTTPServer(s, HTTPOptions{CheckCredentialActive: func(*http.Request, CredentialInfo) (bool, error) { calls++; cancel(); return active, nil }})
				values := url.Values{"client": {client.ClientID}, "clientSecret": {client.ClientSecret}, "mode": {string(tc.mode)}, tc.key: {tc.value}}
				r := httptest.NewRequest("POST", endpoint, strings.NewReader(values.Encode())).WithContext(ctx)
				r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				w := httptest.NewRecorder()
				h.Handler().ServeHTTP(w, r)
				cancel()
				if calls != 1 || w.Code < 400 || strings.Contains(w.Body.String(), "alice") || strings.Contains(w.Body.String(), `"active"`) {
					t.Fatalf("canceled %s/%s active=%v: %d %s calls=%d", tc.mode, endpoint, active, w.Code, w.Body.String(), calls)
				}
			}
		}
	}
	assertLogoutCredentials(t, s, issued, true)
}

// TestCancellationReviewExchange keeps consumed credentials consumed and rejects late enrichment without returning identity. TestCancellationReviewExchange 保持凭证已消费，拒绝延迟补充结果且不返回身份。
func TestCancellationReviewExchange(t *testing.T) {
	for _, mode := range []Mode{ModeTicket, ModeOAuth2} {
		for _, register := range []bool{false, true} {
			s := NewServer()
			t.Cleanup(func() { _ = s.Close() })
			client := newTestClient()
			client.Modes = []Mode{ModeTicket, ModeOAuth2, ModeSharedToken, ModeRemoteSession}
			if err := s.RegisterClient(client); err != nil {
				t.Fatal(err)
			}
			issued := issueLogoutCredentials(t, s, client, "alice")
			ctx, cancel := context.WithCancel(context.Background())
			calls := 0
			h := NewHTTPServer(s, HTTPOptions{ServerOptions: ServerOptions{EnableSLO: true}, EnrichExchangeResult: func(_ *http.Request, _ string, result *TicketExchangeResult) error {
				calls++
				result.TokenValue = "private-token"
				cancel()
				return nil
			}})
			values := url.Values{"client": {client.ClientID}, "clientSecret": {client.ClientSecret}, "redirect": {client.RedirectURIs[0]}, "mode": {string(mode)}}
			if mode == ModeTicket {
				values.Set("ticket", issued.ticket.Ticket)
			} else {
				values.Set("code", issued.code.Code)
			}
			if register {
				values.Set("callback", client.RedirectURIs[0])
			}
			result, err := h.exchangeCredential(httptest.NewRequest("POST", "/sso/token", nil).WithContext(ctx), values)
			cancel()
			if !errors.Is(err, context.Canceled) || result != nil || calls != 1 {
				t.Fatalf("canceled %s exchange = %v, %v", mode, result, err)
			}
			if bindings, err := s.GetClientSessions(context.Background(), "alice"); err != nil || len(bindings) != 0 {
				t.Fatal("canceled exchange registered callback")
			}
			if _, err := h.exchangeCredential(httptest.NewRequest("POST", "/sso/token", nil), values); err == nil || calls != 1 {
				t.Fatal("canceled exchange restored a consumed credential")
			}
		}
	}
}
