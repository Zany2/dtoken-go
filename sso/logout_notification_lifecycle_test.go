package sso

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// TestLogoutNotificationFreshSignatures verifies retry nonces and current signing keys after rotation. TestLogoutNotificationFreshSignatures 验证重试随机值及轮换后的当前签名密钥。
func TestLogoutNotificationFreshSignatures(t *testing.T) {
	s := NewServer()
	defer s.Close()
	registerNotificationClients(t, s, "app-a")
	client, err := s.GetClient("app-a")
	if err != nil {
		t.Fatal(err)
	}
	client.SecretKey = "first-key"
	if err := s.RegisterClient(client); err != nil {
		t.Fatal(err)
	}
	protection := newReplayTestProtection(t, nil)
	params := normalizeParamNames(ParamNames{Client: "application", Nonce: "requestID", Action: "operation", Sign: "signature"})
	firstNonce := ""
	calls := 0
	h := NewHTTPServer(s, HTTPOptions{ReplayProtection: newReplayTestProtection(t, nil), ServerOptions: ServerOptions{
		CheckSign: true, Params: params, LogoutCallbackMaxAttempts: 2, LogoutCallbackRetryInterval: time.Nanosecond,
		LogoutHTTPClient: &http.Client{Transport: logoutReviewTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			if err := r.ParseForm(); err != nil {
				return nil, err
			}
			key := "first-key"
			if calls > 1 {
				key = "rotated-key"
			}
			app := NewClientApp(ClientConfig{ClientID: "app-a", CheckSign: true, SecretKey: key, Params: params, ReplayProtection: protection})
			if _, err := app.VerifyLogoutCallback(r); err != nil {
				t.Errorf("retry callback verification: %v", err)
				return notificationResponse(401, ""), nil
			}
			if calls == 1 {
				firstNonce = r.Form.Get(params.Nonce)
				client.SecretKey = "rotated-key"
				if err := s.RegisterClient(client); err != nil {
					return nil, err
				}
				return notificationResponse(503, ""), nil
			}
			if firstNonce == "" || firstNonce == r.Form.Get(params.Nonce) {
				t.Error("retry reused nonce")
			}
			return notificationResponse(200, `{"code":0}`), nil
		})},
	}})
	report, err := h.NotifyLogoutClients(httptest.NewRequest("POST", "/sso/logout", nil), "alice")
	if err != nil || report == nil || !report.Complete || calls != 2 {
		t.Fatalf("signed retry: report=%+v calls=%d err=%v", report, calls, err)
	}
}

// TestLogoutNotificationRechecksRegistration stops retries after removal or allow-list changes. TestLogoutNotificationRechecksRegistration 在登记删除或白名单变更后停止继续投递。
func TestLogoutNotificationRechecksRegistration(t *testing.T) {
	for _, removed := range []bool{false, true} {
		s := NewServer()
		registerNotificationClients(t, s, "app-a")
		calls := 0
		h := NewHTTPServer(s, HTTPOptions{ServerOptions: ServerOptions{
			LogoutCallbackMaxAttempts: 3, LogoutCallbackRetryInterval: time.Nanosecond,
			LogoutHTTPClient: &http.Client{Transport: logoutReviewTransport(func(*http.Request) (*http.Response, error) {
				calls++
				if removed {
					if err := s.UnregisterClient("app-a"); err != nil {
						return nil, err
					}
				} else {
					client, err := s.GetClient("app-a")
					if err != nil {
						return nil, err
					}
					client.RedirectURIs = []string{"https://other.example/callback"}
					if err := s.RegisterClient(client); err != nil {
						return nil, err
					}
				}
				return notificationResponse(503, ""), nil
			})},
		}})
		report, err := h.NotifyLogoutClients(httptest.NewRequest("POST", "/sso/logout", nil), "alice")
		if report == nil || calls != 1 || report.Clients[0].Attempts != 2 {
			t.Fatalf("changed registration: report=%+v calls=%d err=%v", report, calls, err)
		}
		if removed {
			if err != nil || !report.Complete || report.Clients[0].Status != LogoutCallbackSkipped || report.Clients[0].Reason != "unregistered" {
				t.Fatalf("removed: %+v %v", report, err)
			}
		} else if !errors.Is(err, ErrInvalidCallbackURL) || report.Clients[0].Reason != "configuration" || report.Clients[0].Status != LogoutCallbackFailed {
			t.Fatalf("changed allow-list: %+v %v", report, err)
		}
		_ = s.Close()
	}
}

// logoutReportStorage injects failures in individual logout lifecycle stages. logoutReportStorage 在注销生命周期的不同阶段注入故障。
type logoutReportStorage struct {
	*MemoryStorage
	fail string
}

// Get fails callback-index discovery when requested. Get 按需使回调索引读取失败。
func (s *logoutReportStorage) Get(ctx context.Context, key string) (any, error) {
	if s.fail == "notifications" && strings.Contains(key, ClientSessionKeySuffix) {
		return nil, errors.New("private-storage-address")
	}
	return s.MemoryStorage.Get(ctx, key)
}

// Set fails account credential invalidation when requested. Set 按需使账号凭证失效操作失败。
func (s *logoutReportStorage) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	if s.fail == "credentials" {
		return errors.New("private-storage-address")
	}
	return s.MemoryStorage.Set(ctx, key, value, ttl)
}

// Delete fails callback-index cleanup when requested. Delete 按需使回调索引清理失败。
func (s *logoutReportStorage) Delete(ctx context.Context, keys ...string) error {
	if s.fail == "cleanup" {
		return errors.New("private-storage-address")
	}
	return s.MemoryStorage.Delete(ctx, keys...)
}

// TestLogoutNotificationHTTPStages verifies strict/best-effort outcomes, diagnostics, and observer isolation. TestLogoutNotificationHTTPStages 验证严格／尽力结果、阶段诊断与观察钩子隔离。
func TestLogoutNotificationHTTPStages(t *testing.T) {
	for _, scenario := range []string{"complete", "callback failure", "best effort", "notifications", "center_logout", "credentials", "cleanup"} {
		t.Run(scenario, func(t *testing.T) {
			storage := &logoutReportStorage{MemoryStorage: NewMemoryStorage()}
			s := NewServer(WithStorage(storage))
			defer s.Close()
			registerNotificationClients(t, s, "app-a")
			storage.fail = scenario
			centerCalls, observerCalls := 0, 0
			errorCalls := 0
			var observedError error
			var observed LogoutResult
			h := NewHTTPServer(s, HTTPOptions{
				ServerOptions: ServerOptions{EnableSLO: true, LogoutCallbackBestEffort: scenario == "best effort",
					LogoutHTTPClient: &http.Client{Transport: logoutReviewTransport(func(*http.Request) (*http.Response, error) {
						if scenario == "callback failure" || scenario == "best effort" {
							return nil, errors.New("private-callback-address")
						}
						return notificationResponse(200, "ok"), nil
					})},
				},
				LoginIDResolver: func(*http.Request) (string, bool) { return "alice", true },
				CenterLogout: func(*http.Request, string) error {
					centerCalls++
					if scenario == "center_logout" {
						return errors.New("private-center-details")
					}
					return nil
				},
				OnLogoutResult: func(_ *http.Request, result LogoutResult) {
					observerCalls++
					observed = result
					if result.Notifications != nil && len(result.Notifications.Clients) > 0 {
						result.Notifications.Clients[0].ClientID = "observer-mutation"
					}
				},
				OnHTTPError: func(_ *http.Request, status int, err error) {
					errorCalls++
					observedError = err
					if status < 400 {
						t.Fatal("successful logout emitted an HTTP error observation")
					}
				},
			})
			w := protocolReviewPost(h.HandleLogout, nil)
			var body struct {
				Code int          `json:"code"`
				Data LogoutResult `json:"data"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			wantStage := scenario
			if scenario == "callback failure" {
				wantStage = "notifications"
			}
			if scenario == "best effort" {
				wantStage = "complete"
			}
			completed := wantStage == "complete"
			if (completed && errorCalls != 0) || (!completed && (errorCalls != 1 || !errors.Is(observedError, observed.Cause))) {
				t.Fatal("HTTP error observer disagreed with the logout lifecycle result")
			}
			if body.Data.Stage != wantStage || (w.Code == 200) != completed || observerCalls != 1 || observed.LoginID != "alice" || (observed.Cause == nil) != completed {
				t.Fatalf("stage=%s response=%s observed=%+v", scenario, w.Body.String(), observed)
			}
			if wantStage == "notifications" && centerCalls != 0 {
				t.Fatal("center revoked after strict failure")
			}
			if (len(w.Result().Cookies()) == 1) != completed || (body.Data.Result == ResultOK) != completed {
				t.Fatalf("cleanup outcome: %s", w.Body.String())
			}
			if strings.Contains(w.Body.String(), "private-") || strings.Contains(w.Body.String(), "observer-mutation") || strings.Contains(w.Body.String(), "alice") {
				t.Fatalf("unsafe response: %s", w.Body.String())
			}
			if scenario == "best effort" && (body.Data.Notifications == nil || body.Data.Notifications.Complete || body.Data.Notifications.Clients[0].Status != LogoutCallbackFailed || observed.Notifications.Clients[0].Cause == nil) {
				t.Fatalf("lost best-effort failure: %+v", body.Data)
			}
		})
	}
}

// TestLogoutNotificationCancellationStopsBestEffort prevents canceled requests from revoking center state. TestLogoutNotificationCancellationStopsBestEffort 防止已取消请求在尽力模式中继续撤销中心状态。
func TestLogoutNotificationCancellationStopsBestEffort(t *testing.T) {
	s := NewServer()
	defer s.Close()
	registerNotificationClients(t, s, "app-a")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	centerCalls, observations := 0, 0
	h := NewHTTPServer(s, HTTPOptions{
		ServerOptions: ServerOptions{EnableSLO: true, LogoutCallbackBestEffort: true,
			LogoutHTTPClient: &http.Client{Transport: logoutReviewTransport(func(*http.Request) (*http.Response, error) { cancel(); return nil, ctx.Err() })},
		},
		LoginIDResolver: func(*http.Request) (string, bool) { return "alice", true },
		CenterLogout:    func(*http.Request, string) error { centerCalls++; return nil },
		OnLogoutResult: func(_ *http.Request, result LogoutResult) {
			observations++
			if !errors.Is(result.Cause, context.Canceled) {
				t.Errorf("lost cancellation: %v", result.Cause)
			}
		},
	})
	w := httptest.NewRecorder()
	h.HandleLogout(w, httptest.NewRequest("GET", "/sso/logout", nil).WithContext(ctx))
	if centerCalls != 0 || observations != 1 || w.Code != 500 || len(w.Result().Cookies()) != 0 {
		t.Fatalf("canceled logout: %d center=%d observer=%d", w.Code, centerCalls, observations)
	}
}

// TestLogoutNotificationSkipsAndAuthorization distinguishes missing targets and rejects untrusted HTTP identities. TestLogoutNotificationSkipsAndAuthorization 区分缺失目标并拒绝不可信 HTTP 身份。
func TestLogoutNotificationSkipsAndAuthorization(t *testing.T) {
	s := NewServer()
	defer s.Close()
	registerNotificationClients(t, s, "no-callback", "removed")
	if err := s.saveClientSession(context.Background(), &ClientSession{ClientID: "no-callback", LoginID: "alice"}); err != nil {
		t.Fatal(err)
	}
	if err := s.UnregisterClient("removed"); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	observations := 0
	h := NewHTTPServer(s, HTTPOptions{
		ServerOptions:   ServerOptions{EnableSLO: true, LogoutHTTPClient: &http.Client{Transport: logoutReviewTransport(func(*http.Request) (*http.Response, error) { calls.Add(1); return notificationResponse(200, "ok"), nil })}},
		LoginIDResolver: func(*http.Request) (string, bool) { return "alice", true },
		OnLogoutResult:  func(*http.Request, LogoutResult) { observations++ },
	})
	report, err := h.NotifyLogoutClients(httptest.NewRequest("GET", "/", nil), "alice")
	if err != nil || report == nil || !report.Complete || len(report.Clients) != 2 || calls.Load() != 0 {
		t.Fatalf("skipped targets: %+v %v calls=%d", report, err, calls.Load())
	}
	if report.Clients[0].Reason != "no_callback" || report.Clients[0].Attempts != 0 || report.Clients[1].Reason != "unregistered" {
		t.Fatalf("skip reasons: %+v", report)
	}
	w := protocolReviewPost(h.HandleLogout, url.Values{"loginId": {"another-account"}})
	if w.Code != 401 || observations != 0 {
		t.Fatalf("untrusted identity: status=%d observed=%d", w.Code, observations)
	}
	if _, err := h.NotifyLogoutClients(nil, "alice"); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("nil request: %v", err)
	}
	if _, err := h.NotifyLogoutClients(httptest.NewRequest("GET", "/", nil), ""); !errors.Is(err, ErrUserIDEmpty) {
		t.Fatalf("empty subject: %v", err)
	}
}
