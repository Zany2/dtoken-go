package sso

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// registerNotificationClients registers callback bindings for one test account. registerNotificationClients 为测试账号登记客户端回调绑定。
func registerNotificationClients(t *testing.T, s *Server, ids ...string) {
	t.Helper()
	for _, id := range ids {
		client := newTestClient()
		client.ClientID = id
		if err := s.RegisterClient(client); err != nil {
			t.Fatal(err)
		}
		if _, err := s.RegisterClientSession(context.Background(), "alice", id, client.RedirectURIs[0]); err != nil {
			t.Fatal(err)
		}
	}
}

// notificationResponse creates bounded mock acknowledgement responses. notificationResponse 创建受限的模拟确认响应。
func notificationResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body))}
}

// TestLogoutNotificationDefaults keeps retries opt-in and handles empty notification sets. TestLogoutNotificationDefaults 保持重试显式启用并处理空通知集合。
func TestLogoutNotificationDefaults(t *testing.T) {
	s := NewServer()
	defer s.Close()
	h := NewHTTPServer(s, HTTPOptions{ServerOptions: ServerOptions{LogoutCallbackConcurrency: -1, LogoutCallbackMaxAttempts: -1, LogoutCallbackRetryInterval: -time.Second}})
	if h.options.LogoutCallbackConcurrency != 8 || h.options.LogoutCallbackMaxAttempts != 1 || h.options.LogoutCallbackRetryInterval != 100*time.Millisecond {
		t.Fatalf("defaults: %+v", h.options.ServerOptions)
	}
	report, err := h.NotifyLogoutClients(httptest.NewRequest("GET", "/", nil), "no-bindings")
	if err != nil || report == nil || !report.Complete || len(report.Clients) != 0 {
		t.Fatalf("empty report: %+v %v", report, err)
	}
}

// TestLogoutNotificationReadRetryClosesBodies releases failed responses before another attempt. TestLogoutNotificationReadRetryClosesBodies 在下一次尝试前释放失败响应资源。
func TestLogoutNotificationReadRetryClosesBodies(t *testing.T) {
	s := NewServer()
	defer s.Close()
	registerNotificationClients(t, s, "app-a")
	first := &logoutAcknowledgementBody{readErr: io.ErrUnexpectedEOF}
	second := &logoutAcknowledgementBody{reader: strings.NewReader(`{"code":0}`)}
	calls := 0
	h := NewHTTPServer(s, HTTPOptions{ServerOptions: ServerOptions{
		LogoutCallbackMaxAttempts: 2, LogoutCallbackRetryInterval: time.Nanosecond,
		LogoutHTTPClient: &http.Client{Transport: logoutReviewTransport(func(*http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return &http.Response{StatusCode: 200, Body: first}, nil
			}
			if !first.closed {
				t.Error("failed body still open at retry")
			}
			return &http.Response{StatusCode: 200, Body: second}, nil
		})},
	}})
	report, err := h.NotifyLogoutClients(httptest.NewRequest("GET", "/", nil), "alice")
	if err != nil || !report.Complete || calls != 2 || !first.closed || !second.closed {
		t.Fatalf("read retry: %+v %v calls=%d closed=%v/%v", report, err, calls, first.closed, second.closed)
	}
}

// TestLogoutNotificationRetryPolicy checks retryable HTTP/transport failures and terminal acknowledgement errors. TestLogoutNotificationRetryPolicy 检查可重试 HTTP／传输故障与不可重试确认错误。
func TestLogoutNotificationRetryPolicy(t *testing.T) {
	transportErr := errors.New("temporary network failure")
	for _, tc := range []struct {
		name         string
		status       int
		body         string
		transportErr error
		retry        bool
		reason       string
	}{
		{"unavailable", 503, "", nil, true, "http_status"},
		{"timeout response", 408, "", nil, true, "http_status"},
		{"rate limited", 429, "", nil, true, "http_status"},
		{"transport", 0, "", transportErr, true, "transport"},
		{"forbidden", 403, "", nil, false, "http_status"},
		{"redirect", 302, "", nil, false, "http_status"},
		{"business rejection", 200, `{"code":1}`, nil, false, "acknowledgement"},
		{"invalid acknowledgement", 200, `{"code":`, nil, false, "acknowledgement"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := NewServer()
			defer s.Close()
			registerNotificationClients(t, s, "app-a")
			calls := 0
			h := NewHTTPServer(s, HTTPOptions{ServerOptions: ServerOptions{
				LogoutCallbackMaxAttempts: 3, LogoutCallbackRetryInterval: time.Nanosecond,
				LogoutHTTPClient: &http.Client{Transport: logoutReviewTransport(func(*http.Request) (*http.Response, error) {
					calls++
					if calls > 1 {
						return notificationResponse(204, ""), nil
					}
					if tc.transportErr != nil {
						return nil, tc.transportErr
					}
					return notificationResponse(tc.status, tc.body), nil
				})},
			}})
			report, err := h.NotifyLogoutClients(httptest.NewRequest("POST", "/sso/logout", nil), "alice")
			if report == nil || len(report.Clients) != 1 {
				t.Fatalf("report=%+v error=%v", report, err)
			}
			result := report.Clients[0]
			if tc.retry {
				if err != nil || !report.Complete || calls != 2 || result.Attempts != 2 || result.Status != LogoutCallbackDelivered || result.HTTPStatus != 204 || result.Cause != nil || result.Reason != "" {
					t.Fatalf("retry result=%+v calls=%d err=%v", result, calls, err)
				}
			} else if err == nil || report.Complete || calls != 1 || result.Status != LogoutCallbackFailed || result.Reason != tc.reason || result.HTTPStatus != tc.status {
				t.Fatalf("terminal result=%+v calls=%d err=%v", result, calls, err)
			}
		})
	}
}

// TestLogoutNotificationOnlyRetriesFailedTargets verifies partial recovery and stable report ordering. TestLogoutNotificationOnlyRetriesFailedTargets 验证部分恢复、成功目标不重发与稳定报告顺序。
func TestLogoutNotificationOnlyRetriesFailedTargets(t *testing.T) {
	s := NewServer()
	defer s.Close()
	registerNotificationClients(t, s, "app-c", "app-a", "app-b")
	var mu sync.Mutex
	calls := map[string]int{}
	failure := errors.New("offline endpoint secret-path")
	h := NewHTTPServer(s, HTTPOptions{ServerOptions: ServerOptions{
		LogoutCallbackMaxAttempts: 3, LogoutCallbackRetryInterval: time.Nanosecond,
		LogoutHTTPClient: &http.Client{Transport: logoutReviewTransport(func(r *http.Request) (*http.Response, error) {
			if err := r.ParseForm(); err != nil {
				return nil, err
			}
			id := r.Form.Get("client")
			mu.Lock()
			calls[id]++
			count := calls[id]
			mu.Unlock()
			if id == "app-c" {
				return nil, failure
			}
			if id == "app-b" && count == 1 {
				return notificationResponse(503, ""), nil
			}
			return notificationResponse(200, `{"code":0}`), nil
		})},
	}})
	report, err := h.NotifyLogoutClients(httptest.NewRequest("POST", "/sso/logout", nil), "alice")
	if !errors.Is(err, failure) || report == nil || report.Complete {
		t.Fatalf("report=%+v err=%v", report, err)
	}
	for i, id := range []string{"app-a", "app-b", "app-c"} {
		if report.Clients[i].ClientID != id || calls[id] != i+1 || report.Clients[i].Attempts != i+1 {
			t.Fatalf("ordering/retry mismatch: %+v calls=%v", report, calls)
		}
	}
	payload, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(payload), "secret-path") || strings.Contains(string(payload), "https:") || strings.Contains(string(payload), "alice") {
		t.Fatalf("sensitive report: %s", payload)
	}
	if report.Clients[2].Reason != "transport" || report.Clients[2].Status != LogoutCallbackFailed {
		t.Fatalf("final failed target: %+v", report.Clients[2])
	}
}

// TestLogoutNotificationConcurrency bounds deliveries across concurrent requests to the same handler. TestLogoutNotificationConcurrency 限制同一处理器多个并发请求的通知总数。
func TestLogoutNotificationConcurrency(t *testing.T) {
	s := NewServer()
	defer s.Close()
	registerNotificationClients(t, s, "app-a", "app-b", "app-c", "app-d")
	var active, peak, calls atomic.Int32
	started := make(chan struct{}, 8)
	release := make(chan struct{})
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(release) })
	h := NewHTTPServer(s, HTTPOptions{ServerOptions: ServerOptions{
		LogoutCallbackConcurrency: 2,
		LogoutHTTPClient: &http.Client{Transport: logoutReviewTransport(func(r *http.Request) (*http.Response, error) {
			current := active.Add(1)
			defer active.Add(-1)
			calls.Add(1)
			for old := peak.Load(); current > old; old = peak.Load() {
				if peak.CompareAndSwap(old, current) {
					break
				}
			}
			started <- struct{}{}
			select {
			case <-release:
			case <-r.Context().Done():
				return nil, r.Context().Err()
			}
			return notificationResponse(200, "ok"), nil
		})},
	}})
	done := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			_, err := h.NotifyLogoutClients(httptest.NewRequest("POST", "/sso/logout", nil), "alice")
			done <- err
		}()
	}
	for i := 0; i < 2; i++ {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("workers did not start")
		}
	}
	releaseOnce.Do(func() { close(release) })
	for i := 0; i < 2; i++ {
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(time.Second):
			t.Fatal("workers did not finish")
		}
	}
	if peak.Load() != 2 || calls.Load() != 8 || active.Load() != 0 {
		t.Fatalf("peak=%d calls=%d active=%d", peak.Load(), calls.Load(), active.Load())
	}
}

// TestLogoutNotificationTimeoutAndCancellation distinguishes attempt timeouts from parent cancellation. TestLogoutNotificationTimeoutAndCancellation 区分单次尝试超时与父请求取消。
func TestLogoutNotificationTimeoutAndCancellation(t *testing.T) {
	s := NewServer()
	defer s.Close()
	registerNotificationClients(t, s, "app-a", "app-b")
	h := NewHTTPServer(s, HTTPOptions{ServerOptions: ServerOptions{
		LogoutCallbackConcurrency: 1, LogoutCallbackMaxAttempts: 2,
		LogoutCallbackTimeout: 5 * time.Millisecond, LogoutCallbackRetryInterval: time.Nanosecond,
		LogoutHTTPClient: &http.Client{Transport: logoutReviewTransport(func(r *http.Request) (*http.Response, error) { <-r.Context().Done(); return nil, r.Context().Err() })},
	}})
	report, err := h.NotifyLogoutClients(httptest.NewRequest("POST", "/sso/logout", nil), "alice")
	if !errors.Is(err, context.DeadlineExceeded) || report == nil || report.Complete {
		t.Fatalf("timeout report=%+v err=%v", report, err)
	}
	for _, result := range report.Clients {
		if result.Attempts != 2 || result.Reason != "timeout" || result.Status != LogoutCallbackFailed {
			t.Fatalf("attempt timeout: %+v", result)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h.options.LogoutHTTPClient = &http.Client{Transport: logoutReviewTransport(func(*http.Request) (*http.Response, error) { cancel(); return nil, ctx.Err() })}
	report, err = h.NotifyLogoutClients(httptest.NewRequest("POST", "/sso/logout", nil).WithContext(ctx), "alice")
	if !errors.Is(err, context.Canceled) || report == nil {
		t.Fatalf("cancel report=%+v err=%v", report, err)
	}
	if report.Clients[0].Status != LogoutCallbackCanceled || report.Clients[0].Attempts != 1 || report.Clients[1].Attempts != 0 || report.Clients[1].Status != LogoutCallbackCanceled {
		t.Fatalf("canceled dispatch: %+v", report)
	}

	// Cancellation interrupts retry delay without waiting for the configured interval. 取消会立即中断重试等待，无需等待配置的间隔。
	waitCtx, waitCancel := context.WithCancel(context.Background())
	defer waitCancel()
	firstAttempt := make(chan struct{}, 1)
	h.options.LogoutCallbackRetryInterval = time.Hour
	h.options.LogoutHTTPClient = &http.Client{Transport: logoutReviewTransport(func(*http.Request) (*http.Response, error) {
		firstAttempt <- struct{}{}
		return notificationResponse(503, ""), nil
	})}
	waitDone := make(chan struct{})
	go func() {
		report, err = h.NotifyLogoutClients(httptest.NewRequest("POST", "/sso/logout", nil).WithContext(waitCtx), "alice")
		close(waitDone)
	}()
	select {
	case <-firstAttempt:
	case <-time.After(time.Second):
		t.Fatal("retry test did not reach first attempt")
	}
	waitCancel()
	select {
	case <-waitDone:
	case <-time.After(time.Second):
		t.Fatal("retry delay did not stop on cancellation")
	}
	if !errors.Is(err, context.Canceled) || report == nil || report.Clients[0].Attempts != 1 {
		t.Fatalf("retry wait cancellation: report=%+v err=%v", report, err)
	}
}
