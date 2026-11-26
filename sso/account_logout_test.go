package sso

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/contract"
)

// TestAccountLogoutRevokesBeforeNotifications checks all modes, shared instances, account isolation and retry after delivery failure. TestAccountLogoutRevokesBeforeNotifications 检查全部模式、共享实例、账号隔离及通知失败后的重试。
func TestAccountLogoutRevokesBeforeNotifications(t *testing.T) {
	for _, basic := range []bool{false, true} {
		t.Run(map[bool]string{false: "atomic", true: "ordinary"}[basic], func(t *testing.T) {
			memory := NewMemoryStorage()
			var storage contract.Storage = memory
			if basic {
				storage = &basicSSOStorage{inner: memory}
			}
			s := NewServer(WithStorage(storage))
			peer := NewServer(WithStorage(storage))
			defer s.Close()
			defer peer.Close()
			registerNotificationClients(t, s, "app-a", "app-b")
			client, err := s.GetClient("app-a")
			if err != nil {
				t.Fatal(err)
			}
			client.Modes = []Mode{ModeTicket, ModeSharedToken, ModeRemoteSession, ModeOAuth2}
			if err := s.RegisterClient(client); err != nil {
				t.Fatal(err)
			}
			old := issueLogoutCredentials(t, s, client, "alice")
			secondClient := *client
			secondClient.ClientID = "app-b"
			if err := s.RegisterClient(&secondClient); err != nil {
				t.Fatal(err)
			}
			otherApp := issueLogoutCredentials(t, s, &secondClient, "alice")
			other := issueLogoutCredentials(t, s, client, "bob")
			var centerCalls, callbacks atomic.Int32
			fail := true
			h := NewHTTPServer(peer, HTTPOptions{
				ServerOptions: ServerOptions{EnableSLO: true, LogoutCallbackBestEffort: true,
					LogoutHTTPClient: &http.Client{Transport: logoutReviewTransport(func(r *http.Request) (*http.Response, error) {
						callbacks.Add(1)
						if centerCalls.Load() == 0 {
							t.Error("notification preceded center revocation")
						}
						if _, err := s.ValidateSharedToken(r.Context(), old.token.Token, client.ClientID); !errors.Is(err, ErrInvalidSharedToken) {
							t.Errorf("credential still active during notification: %v", err)
						}
						if err := r.ParseForm(); err != nil {
							return nil, err
						}
						if fail && r.Form.Get(DefaultParamNames().Client) == "app-b" {
							return notificationResponse(503, "private-response"), nil
						}
						return notificationResponse(200, "ok"), nil
					})},
				},
				LoginIDResolver: func(*http.Request) (string, bool) {
					t.Error("trusted API resolved browser subject")
					return "bob", true
				},
				CenterLogout: func(_ *http.Request, id string) error {
					if id != "alice" {
						t.Errorf("revoked wrong subject: %s", id)
					}
					centerCalls.Add(1)
					return nil
				},
				OnLogoutResult: func(*http.Request, LogoutResult) { t.Error("trusted API invoked browser response observer") },
			})
			r := httptest.NewRequest(http.MethodPost, "/internal-operation?loginId=bob", nil)
			result, err := h.LogoutAccount(r, "alice")
			if err == nil || result.Cause == nil || result.Stage != "notifications" || result.Complete || !result.CenterRevoked || !result.CredentialsRevoked || result.BindingsCleared {
				t.Fatalf("partial progress = %+v, %v", result, err)
			}
			if result.Notifications == nil || result.Notifications.Complete || len(result.Notifications.Clients) != 2 || callbacks.Load() != 2 {
				t.Fatalf("notification report = %+v", result.Notifications)
			}
			assertLogoutCredentials(t, s, old, false)
			assertLogoutCredentials(t, peer, otherApp, false)
			assertLogoutCredentials(t, peer, other, true)
			sessions, err := s.GetClientSessions(context.Background(), "alice")
			if err != nil || len(sessions) != 2 {
				t.Fatalf("retry bindings = %v, %v", sessions, err)
			}
			encoded, err := json.Marshal(result)
			if err != nil || strings.Contains(string(encoded), "private-") || strings.Contains(string(encoded), "alice") {
				t.Fatalf("unsafe result JSON: %s, %v", encoded, err)
			}

			fail = false
			result, err = h.LogoutAccount(r, "alice")
			if err != nil || result.Cause != nil || !result.Complete || !result.BindingsCleared || result.Stage != "complete" || centerCalls.Load() != 2 || callbacks.Load() != 4 {
				t.Fatalf("retry = %+v, %v", result, err)
			}
			if sessions, err := s.GetClientSessions(context.Background(), "alice"); err != nil || len(sessions) != 0 {
				t.Fatalf("cleanup = %v, %v", sessions, err)
			}
			assertLogoutCredentials(t, peer, other, true)
			fresh := issueLogoutCredentials(t, s, client, "alice")
			assertLogoutCredentials(t, peer, fresh, true)
		})
	}
}

// TestAccountLogoutFailureStages preserves partial progress and callback bindings across recoverable failures. TestAccountLogoutFailureStages 在可恢复故障下保留部分进度和回调绑定。
func TestAccountLogoutFailureStages(t *testing.T) {
	for _, stage := range []string{"center_logout", "credentials", "notifications", "cleanup"} {
		t.Run(stage, func(t *testing.T) {
			storage := &logoutReportStorage{MemoryStorage: NewMemoryStorage()}
			s := NewServer(WithStorage(storage))
			defer s.Close()
			registerNotificationClients(t, s, "app-a")
			client, err := s.GetClient("app-a")
			if err != nil {
				t.Fatal(err)
			}
			ticket, err := s.GenerateTicket(context.Background(), client.ClientID, "alice", client.RedirectURIs[0], nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			failure := errors.New("private-center-failure")
			var delivered atomic.Int32
			storage.fail = stage
			h := NewHTTPServer(s, HTTPOptions{
				ServerOptions: ServerOptions{EnableSLO: true, LogoutHTTPClient: &http.Client{Transport: logoutReviewTransport(func(*http.Request) (*http.Response, error) {
					delivered.Add(1)
					return notificationResponse(200, "ok"), nil
				})}},
				CenterLogout: func(*http.Request, string) error {
					if storage.fail == "center_logout" {
						return failure
					}
					return nil
				},
			})
			result, err := h.LogoutAccount(httptest.NewRequest("POST", "/", nil), "alice")
			if err == nil || result.Stage != stage || result.Complete || result.BindingsCleared || result.CenterRevoked != (stage != "center_logout") || result.CredentialsRevoked != (stage == "notifications" || stage == "cleanup") {
				t.Fatalf("stage %s = %+v, %v", stage, result, err)
			}
			if stage == "center_logout" && !errors.Is(err, failure) {
				t.Fatal("lost hook error")
			}
			if (delivered.Load() == 1) != (stage == "cleanup") {
				t.Fatalf("unexpected delivery count: %d", delivered.Load())
			}
			encoded, _ := json.Marshal(result)
			if strings.Contains(string(encoded), "private-") {
				t.Fatalf("unsafe diagnostics: %s", encoded)
			}
			storage.fail = ""
			_, err = s.ValidateTicket(context.Background(), ticket.Ticket)
			if result.CredentialsRevoked && !errors.Is(err, ErrInvalidTicket) || !result.CredentialsRevoked && err != nil {
				t.Fatalf("credential state = %v", err)
			}
			if bindings, err := s.GetClientSessions(context.Background(), "alice"); err != nil || len(bindings) != 1 {
				t.Fatalf("lost retry bindings: %v, %v", bindings, err)
			}
			if result, err := h.LogoutAccount(httptest.NewRequest("POST", "/", nil), "alice"); err != nil || !result.Complete {
				t.Fatalf("recovery = %+v, %v", result, err)
			}
		})
	}
}

// TestAccountLogoutValidationAndDisabledSLO rejects invalid calls before side effects and preserves disabled notification bindings. TestAccountLogoutValidationAndDisabledSLO 在副作用前拒绝无效调用，禁用通知时保留绑定。
func TestAccountLogoutValidationAndDisabledSLO(t *testing.T) {
	s := NewServer()
	defer s.Close()
	registerNotificationClients(t, s, "app-a")
	centerCalls := 0
	h := NewHTTPServer(s, HTTPOptions{CenterLogout: func(*http.Request, string) error { centerCalls++; return nil }})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r := httptest.NewRequest("POST", "/", nil)
	for _, tc := range []struct {
		server  *HTTPServer
		request *http.Request
		id      string
		want    error
	}{
		{nil, r, "alice", ErrServerNotInitialized}, {NewHTTPServer(s, HTTPOptions{}), r, "alice", ErrAccountLogoutConfig},
		{h, nil, "alice", ErrInvalidRequest}, {h, r, "", ErrUserIDEmpty}, {h, r.WithContext(ctx), "alice", context.Canceled},
		{NewHTTPServer(s, HTTPOptions{ServerOptions: ServerOptions{Endpoints: Endpoints{Logout: "/{bad}"}}}), r, "alice", ErrSSOConfig},
	} {
		result, err := tc.server.LogoutAccount(tc.request, tc.id)
		if !errors.Is(err, tc.want) || !errors.Is(result.Cause, tc.want) || result.Stage != "validation" || result.CenterRevoked || result.CredentialsRevoked || result.Complete {
			t.Fatalf("validation = %+v, %v, want %v", result, err, tc.want)
		}
	}
	if centerCalls != 0 {
		t.Fatal("validation invoked center hook")
	}
	result, err := h.LogoutAccount(r, "alice")
	if err != nil || !result.Complete || !result.CenterRevoked || !result.CredentialsRevoked || result.Notifications != nil || result.BindingsCleared || centerCalls != 1 {
		t.Fatalf("disabled SLO = %+v, %v", result, err)
	}
	if bindings, err := s.GetClientSessions(r.Context(), "alice"); err != nil || len(bindings) != 1 {
		t.Fatalf("disabled SLO cleared bindings: %v, %v", bindings, err)
	}
}

// accountCancelStorage simulates cancellation after a successful storage mutation. accountCancelStorage 模拟存储修改成功后发生取消。
type accountCancelStorage struct {
	*MemoryStorage
	stage  string
	cancel context.CancelFunc
}

// Set cancels only after writing the account credential generation. Set 仅在写入账号凭证版本后取消。
func (s *accountCancelStorage) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	err := s.MemoryStorage.Set(ctx, key, value, ttl)
	if err == nil && s.stage == "credentials" && strings.Contains(key, "sso:logout-version:") {
		s.cancel()
	}
	return err
}

// Delete cancels after callback-index deletion to verify truthful progress without overall success. Delete 在删除回调索引后取消，验证进度真实且不报告整体成功。
func (s *accountCancelStorage) Delete(ctx context.Context, keys ...string) error {
	err := s.MemoryStorage.Delete(ctx, keys...)
	if err == nil && s.stage == "cleanup" {
		s.cancel()
	}
	return err
}

// TestAccountLogoutCancellation records acknowledged effects but stops each canceled lifecycle. TestAccountLogoutCancellation 记录已确认副作用，并在各阶段取消时停止流程。
func TestAccountLogoutCancellation(t *testing.T) {
	for _, stage := range []string{"center_logout", "credentials", "notifications", "cleanup"} {
		t.Run(stage, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			storage := &accountCancelStorage{MemoryStorage: NewMemoryStorage(), stage: stage, cancel: cancel}
			s := NewServer(WithStorage(storage))
			defer s.Close()
			registerNotificationClients(t, s, "app-a")
			h := NewHTTPServer(s, HTTPOptions{
				ServerOptions: ServerOptions{EnableSLO: true, LogoutCallbackBestEffort: true, LogoutHTTPClient: &http.Client{Transport: logoutReviewTransport(func(*http.Request) (*http.Response, error) {
					if stage == "notifications" {
						cancel()
					}
					return notificationResponse(200, "ok"), nil
				})}},
				CenterLogout: func(*http.Request, string) error {
					if stage == "center_logout" {
						cancel()
					}
					return nil
				},
			})
			result, err := h.LogoutAccount(httptest.NewRequest("POST", "/", nil).WithContext(ctx), "alice")
			if !errors.Is(err, context.Canceled) || !errors.Is(result.Cause, context.Canceled) || result.Stage != stage || result.Complete || !result.CenterRevoked || result.CredentialsRevoked != (stage != "center_logout") || result.BindingsCleared != (stage == "cleanup") {
				t.Fatalf("cancel %s = %+v, %v", stage, result, err)
			}
		})
	}
}
