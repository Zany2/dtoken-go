package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/sso"
)

// clientReviewTransport handles protocol requests without network connections. clientReviewTransport 无需网络连接即可处理协议请求。
type clientReviewTransport func(*http.Request) (*http.Response, error)

// RoundTrip delegates requests to the test. RoundTrip 将请求交给测试处理。
func (f clientReviewTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// TestLocalSessionExpiry verifies copied Cookies cannot extend sessions and new logins reclaim stale records. TestLocalSessionExpiry 验证复制 Cookie 不能延长会话，新登录会回收过期记录。
func TestLocalSessionExpiry(t *testing.T) {
	resetClientSessions(t)
	localSessions.mu.Lock()
	localSessions.values["expired"] = localSession{loginID: "alice", expiresAt: time.Now().Add(-time.Second)}
	localSessions.values["stale"] = localSession{loginID: "bob", expiresAt: time.Now().Add(-time.Second)}
	localSessions.mu.Unlock()
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.AddCookie(&http.Cookie{Name: localCookie, Value: "expired"})
	if id, ok := localLoginID(request); ok || id != "" {
		t.Fatalf("expired session accepted: %q, %v", id, ok)
	}
	id, err := newLocalSession("active")
	if err != nil {
		t.Fatal(err)
	}
	localSessions.mu.RLock()
	defer localSessions.mu.RUnlock()
	if len(localSessions.values) != 1 || localSessions.values[id].loginID != "active" || !localSessions.values[id].expiresAt.After(time.Now()) {
		t.Fatalf("expiry cleanup failed: %+v", localSessions.values)
	}
}

// TestCallbackRequiresBrowserState verifies invalid callbacks never exchange a Ticket or create a session. TestCallbackRequiresBrowserState 验证无效回调不会交换 Ticket 或创建会话。
func TestCallbackRequiresBrowserState(t *testing.T) {
	resetClientSessions(t)
	previous := clientApp
	t.Cleanup(func() { clientApp = previous })
	calls := 0
	cfg := clientApp.Config()
	cfg.HTTPClient = &http.Client{Transport: clientReviewTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"code":0,"data":{"loginId":"alice"}}`))}, nil
	})}
	clientApp = sso.NewClientApp(cfg)
	for _, scenario := range []string{"no cookie", "wrong state", "no state", "expired state", "wrong method"} {
		t.Run(scenario, func(t *testing.T) {
			request := newCallbackRequest(t, "ticket")
			query := request.URL.Query()
			wantStatus := http.StatusBadRequest
			switch scenario {
			case "no cookie":
				request.Header.Del("Cookie")
			case "wrong state":
				query.Set(cfg.Params.Back, "other-browser")
			case "no state":
				query.Del(cfg.Params.Back)
			case "expired state":
				options := loginStateCookie
				options.MaxAge = time.Millisecond
				recorder := httptest.NewRecorder()
				sso.SetLoginIDCookie(recorder, options, query.Get(cfg.Params.Back))
				request.Header.Del("Cookie")
				request.AddCookie(recorder.Result().Cookies()[0])
				time.Sleep(2 * time.Millisecond)
			case "wrong method":
				request.Method = http.MethodPost
				wantStatus = http.StatusMethodNotAllowed
			}
			request.URL.RawQuery = query.Encode()
			recorder := httptest.NewRecorder()
			newDemoHandler().ServeHTTP(recorder, request)
			if recorder.Code != wantStatus || len(recorder.Result().Cookies()) != 0 {
				t.Fatalf("invalid callback: status=%d cookies=%v", recorder.Code, recorder.Result().Cookies())
			}
		})
	}
	if calls != 0 {
		t.Fatalf("invalid callbacks made %d upstream requests", calls)
	}
	localSessions.mu.RLock()
	defer localSessions.mu.RUnlock()
	if len(localSessions.values) != 0 {
		t.Fatal("invalid callback created local identity")
	}
}

// TestCallbackRotatesOnlyAfterSuccessfulExchange verifies failure preserves the old session and success replaces it. TestCallbackRotatesOnlyAfterSuccessfulExchange 验证交换失败保留旧会话，成功后才替换。
func TestCallbackRotatesOnlyAfterSuccessfulExchange(t *testing.T) {
	resetClientSessions(t)
	previous := clientApp
	t.Cleanup(func() { clientApp = previous })
	fail := true
	cfg := clientApp.Config()
	cfg.HTTPClient = &http.Client{Transport: clientReviewTransport(func(r *http.Request) (*http.Response, error) {
		if fail {
			return &http.Response{StatusCode: http.StatusServiceUnavailable, Body: io.NopCloser(strings.NewReader("offline"))}, nil
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"code":0,"data":{"loginId":"new-user"}}`))}, nil
	})}
	clientApp = sso.NewClientApp(cfg)
	oldID, err := newLocalSession("old-user")
	if err != nil {
		t.Fatal(err)
	}
	for attempt := 0; attempt < 2; attempt++ {
		request := newCallbackRequest(t, "ticket")
		request.AddCookie(&http.Cookie{Name: localCookie, Value: oldID})
		recorder := httptest.NewRecorder()
		newDemoHandler().ServeHTTP(recorder, request)
		var issued *http.Cookie
		stateCleared := false
		for _, item := range recorder.Result().Cookies() {
			if item.Name == localCookie {
				issued = item
			}
			if item.Name == loginStateCookie.Name && item.MaxAge == -1 {
				stateCleared = true
			}
		}
		if !stateCleared {
			t.Fatal("accepted callback did not clear the browser state Cookie")
		}
		_, oldValid := localLoginID(request)
		if fail {
			if recorder.Code != http.StatusBadGateway || !oldValid || issued != nil {
				t.Fatal("failed exchange changed local identity")
			}
			fail = false
			continue
		}
		if recorder.Code != http.StatusFound || oldValid || issued == nil || issued.Value == oldID || !issued.HttpOnly || issued.Expires.IsZero() {
			t.Fatalf("successful rotation failed: status=%d cookie=%v oldValid=%v", recorder.Code, issued, oldValid)
		}
		check := httptest.NewRequest(http.MethodGet, "/protected", nil)
		check.AddCookie(issued)
		if id, ok := localLoginID(check); !ok || id != "new-user" {
			t.Fatalf("new session = %q, %v", id, ok)
		}
	}
}

// TestAuthorizationStateRoundTrip verifies the real SSO handler echoes browser state before Ticket exchange. TestAuthorizationStateRoundTrip 验证真实 SSO 处理器回传浏览器状态并完成 Ticket 交换。
func TestAuthorizationStateRoundTrip(t *testing.T) {
	resetClientSessions(t)
	center := sso.NewServer()
	defer center.Close()
	if err := center.RegisterClient(&sso.Client{
		ClientID: clientID, ClientSecret: clientSecret, RedirectURIs: []string{callbackURL}, Modes: []sso.Mode{sso.ModeTicket},
	}); err != nil {
		t.Fatal(err)
	}
	centerHandler := sso.NewHTTPServer(center, sso.HTTPOptions{
		LoginIDResolver: func(*http.Request) (string, bool) { return "alice", true },
	}).Handler()
	previous := clientApp
	t.Cleanup(func() { clientApp = previous })
	cfg := clientApp.Config()
	cfg.HTTPClient = &http.Client{Transport: clientReviewTransport(func(r *http.Request) (*http.Response, error) {
		recorder := httptest.NewRecorder()
		centerHandler.ServeHTTP(recorder, r)
		return recorder.Result(), nil
	})}
	clientApp = sso.NewClientApp(cfg)
	handler := newDemoHandler()
	start := httptest.NewRecorder()
	handler.ServeHTTP(start, httptest.NewRequest(http.MethodGet, "/protected", nil))
	if start.Code != http.StatusFound {
		t.Fatalf("login start status = %d", start.Code)
	}
	authorized := httptest.NewRecorder()
	centerHandler.ServeHTTP(authorized, httptest.NewRequest(http.MethodGet, start.Header().Get("Location"), nil))
	if authorized.Code != http.StatusFound {
		t.Fatalf("authorize status = %d, body=%s", authorized.Code, authorized.Body.String())
	}
	location, err := url.Parse(authorized.Header().Get("Location"))
	if err != nil || location.Query().Get("back") == "" || location.Query().Get("ticket") == "" {
		t.Fatalf("invalid callback location: %v, %v", location, err)
	}
	request := httptest.NewRequest(http.MethodGet, location.String(), nil)
	for _, item := range start.Result().Cookies() {
		request.AddCookie(item)
	}
	completed := httptest.NewRecorder()
	handler.ServeHTTP(completed, request)
	if completed.Code != http.StatusFound {
		t.Fatalf("callback status = %d, body=%s", completed.Code, completed.Body.String())
	}
	check := httptest.NewRequest(http.MethodGet, "/protected", nil)
	for _, item := range completed.Result().Cookies() {
		if item.Name == localCookie {
			check.AddCookie(item)
		}
	}
	if id, ok := localLoginID(check); !ok || id != "alice" {
		t.Fatalf("round-trip identity = %q, %v", id, ok)
	}
}
