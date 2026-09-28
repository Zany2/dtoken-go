package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/sso"
	"github.com/gin-gonic/gin"
)

// ginClientReviewTransport handles protocol requests without network connections. ginClientReviewTransport 无需网络连接即可处理协议请求。
type ginClientReviewTransport func(*http.Request) (*http.Response, error)

// RoundTrip delegates requests to the test. RoundTrip 将请求交给测试处理。
func (f ginClientReviewTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

// TestLocalSessionExpiry verifies copied Cookies cannot extend sessions and new logins reclaim stale records. TestLocalSessionExpiry 验证复制 Cookie 不能延长会话，新登录会回收过期记录。
func TestLocalSessionExpiry(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetGinClientSessions(t)
	localSessions.mu.Lock()
	localSessions.values["expired"] = localSession{loginID: "alice", expiresAt: time.Now().Add(-time.Second)}
	localSessions.values["stale"] = localSession{loginID: "bob", expiresAt: time.Now().Add(-time.Second)}
	localSessions.mu.Unlock()
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.AddCookie(&http.Cookie{Name: localCookie, Value: "expired"})
	recorder := httptest.NewRecorder()
	newDemoRouter().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusFound || recorder.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("expired session response: status=%d headers=%v", recorder.Code, recorder.Header())
	}
	localSessions.mu.RLock()
	_, retained := localSessions.values["expired"]
	localSessions.mu.RUnlock()
	if retained {
		t.Fatal("expired session was not reclaimed on access")
	}
	before := time.Now()
	id, err := newLocalSession("active")
	if err != nil {
		t.Fatal(err)
	}
	localSessions.mu.RLock()
	defer localSessions.mu.RUnlock()
	session := localSessions.values[id]
	if len(localSessions.values) != 1 || session.loginID != "active" || session.expiresAt.Before(before.Add(localCookieTTL)) || session.expiresAt.After(time.Now().Add(localCookieTTL)) {
		t.Fatalf("expiry cleanup or new deadline failed: %+v", localSessions.values)
	}
}

// TestCallbackRequiresBrowserState verifies invalid callbacks never exchange a Ticket or create a session. TestCallbackRequiresBrowserState 验证无效回调不会交换 Ticket 或创建会话。
func TestCallbackRequiresBrowserState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetGinClientSessions(t)
	previous := clientApp
	t.Cleanup(func() { clientApp = previous })
	calls := 0
	cfg := clientApp.Config()
	cfg.HTTPClient = &http.Client{Transport: ginClientReviewTransport(func(*http.Request) (*http.Response, error) {
		calls++
		recorder := httptest.NewRecorder()
		_ = json.NewEncoder(recorder).Encode(sso.OKResponse(sso.TicketExchangeResult{LoginID: "alice"}))
		return recorder.Result(), nil
	})}
	clientApp = sso.NewClientApp(cfg)
	router := newDemoRouter()
	for _, scenario := range []string{"no cookie", "wrong state", "no state", "tampered cookie", "expired state", "no ticket", "wrong method"} {
		t.Run(scenario, func(t *testing.T) {
			request := newCallbackRequest(t, "ticket")
			query := request.URL.Query()
			wantStatus := http.StatusBadRequest
			switch scenario {
			case "no cookie":
				request.Header.Del("Cookie")
			case "wrong state":
				other := newCallbackRequest(t, "other-ticket")
				query.Set(cfg.Params.Back, other.URL.Query().Get(cfg.Params.Back))
			case "no state":
				query.Del(cfg.Params.Back)
			case "tampered cookie":
				cookie, _ := request.Cookie(loginStateCookie.Name)
				cookie.Value += "tampered"
				request.Header.Del("Cookie")
				request.AddCookie(cookie)
			case "expired state":
				options := loginStateCookie
				options.MaxAge = time.Millisecond
				recorder := httptest.NewRecorder()
				sso.SetLoginIDCookie(recorder, options, query.Get(cfg.Params.Back))
				request.Header.Del("Cookie")
				request.AddCookie(recorder.Result().Cookies()[0])
				time.Sleep(2 * time.Millisecond)
			case "no ticket":
				query.Del("ticket")
			case "wrong method":
				request.Method = http.MethodPost
				wantStatus = http.StatusNotFound
			}
			request.URL.RawQuery = query.Encode()
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)
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
	gin.SetMode(gin.TestMode)
	resetGinClientSessions(t)
	previous := clientApp
	t.Cleanup(func() { clientApp = previous })
	fail := true
	cfg := clientApp.Config()
	cfg.HTTPClient = &http.Client{Transport: ginClientReviewTransport(func(r *http.Request) (*http.Response, error) {
		recorder := httptest.NewRecorder()
		if fail {
			http.Error(recorder, "offline", http.StatusServiceUnavailable)
		} else {
			_ = json.NewEncoder(recorder).Encode(sso.OKResponse(sso.TicketExchangeResult{LoginID: "new-user"}))
		}
		return recorder.Result(), nil
	})}
	clientApp = sso.NewClientApp(cfg)
	router := newDemoRouter()
	oldID, err := newLocalSession("old-user")
	if err != nil {
		t.Fatal(err)
	}
	for attempt := 0; attempt < 2; attempt++ {
		request := newCallbackRequest(t, "ticket")
		request.AddCookie(&http.Cookie{Name: localCookie, Value: oldID})
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
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
		if !stateCleared || recorder.Header().Get("Cache-Control") != "no-store" || recorder.Header().Get("Referrer-Policy") != "no-referrer" {
			t.Fatalf("callback state or response headers: %v", recorder.Header())
		}
		_, oldValid := localLoginID(request)
		if fail {
			if recorder.Code != http.StatusBadGateway || !oldValid || issued != nil {
				t.Fatal("failed exchange changed local identity")
			}
			fail = false
			continue
		}
		if recorder.Code != http.StatusFound || recorder.Header().Get("Location") != "/protected" || oldValid || issued == nil || issued.Value == oldID || !issued.HttpOnly || issued.SameSite != http.SameSiteLaxMode || issued.MaxAge != int(localCookieTTL.Seconds()) || issued.Expires.IsZero() {
			t.Fatalf("successful rotation failed: status=%d cookie=%v oldValid=%v", recorder.Code, issued, oldValid)
		}
		check := httptest.NewRequest(http.MethodGet, "/protected", nil)
		check.AddCookie(issued)
		if id, ok := localLoginID(check); !ok || id != "new-user" {
			t.Fatalf("new session = %q, %v", id, ok)
		}
	}
}

// TestAuthorizationStateRoundTrip verifies real SSO state echo and Ticket exchange through the Gin routes. TestAuthorizationStateRoundTrip 验证真实 SSO 状态回传及 Gin 路由中的 Ticket 交换。
func TestAuthorizationStateRoundTrip(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetGinClientSessions(t)
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
	calls := 0
	cfg := clientApp.Config()
	cfg.HTTPClient = &http.Client{Transport: ginClientReviewTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != http.MethodPost || r.URL.Path != cfg.Endpoints.Token {
			t.Errorf("unexpected exchange: %s %s", r.Method, r.URL.Path)
		}
		recorder := httptest.NewRecorder()
		centerHandler.ServeHTTP(recorder, r)
		return recorder.Result(), nil
	})}
	clientApp = sso.NewClientApp(cfg)
	router := newDemoRouter()
	start := httptest.NewRecorder()
	router.ServeHTTP(start, httptest.NewRequest(http.MethodGet, "/protected", nil))
	if start.Code != http.StatusFound {
		t.Fatalf("login start status = %d", start.Code)
	}
	authorizationURL, err := url.Parse(start.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	authorized := httptest.NewRecorder()
	centerHandler.ServeHTTP(authorized, httptest.NewRequest(http.MethodGet, authorizationURL.String(), nil))
	if authorized.Code != http.StatusFound {
		t.Fatalf("authorize status = %d, body=%s", authorized.Code, authorized.Body.String())
	}
	location, err := url.Parse(authorized.Header().Get("Location"))
	if err != nil || location.Query().Get(cfg.Params.Back) == "" || location.Query().Get(cfg.Params.Back) != authorizationURL.Query().Get(cfg.Params.Back) || location.Query().Get(cfg.Params.Ticket) == "" {
		t.Fatalf("invalid callback location: %v, %v", location, err)
	}
	request := httptest.NewRequest(http.MethodGet, location.String(), nil)
	for _, item := range start.Result().Cookies() {
		request.AddCookie(item)
	}
	completed := httptest.NewRecorder()
	router.ServeHTTP(completed, request)
	if completed.Code != http.StatusFound || calls != 1 {
		t.Fatalf("callback status=%d exchanges=%d body=%s", completed.Code, calls, completed.Body.String())
	}
	check := httptest.NewRequest(http.MethodGet, "/protected", nil)
	for _, item := range completed.Result().Cookies() {
		if item.Name == localCookie {
			check.AddCookie(item)
		}
	}
	resource := httptest.NewRecorder()
	router.ServeHTTP(resource, check)
	if resource.Code != http.StatusOK || !strings.Contains(resource.Body.String(), "loginId: alice") {
		t.Fatalf("protected resource status=%d body=%s", resource.Code, resource.Body.String())
	}
}

// TestLogoutCallbackRejectsInvalidRequests verifies rejected callbacks retain the user's sessions. TestLogoutCallbackRejectsInvalidRequests 验证被拒绝的注销回调不会删除用户会话。
func TestLogoutCallbackRejectsInvalidRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetGinClientSessions(t)
	id, err := newLocalSession("alice")
	if err != nil {
		t.Fatal(err)
	}
	router := newDemoRouter()
	for _, scenario := range []string{"missing client", "wrong client", "missing timestamp", "stale timestamp", "missing loginId", "wrong method"} {
		t.Run(scenario, func(t *testing.T) {
			form := url.Values{"loginId": {"alice"}, "client": {clientID}, "timestamp": {time.Now().Format(time.RFC3339)}}
			method := http.MethodPost
			switch scenario {
			case "missing client":
				form.Del("client")
			case "wrong client":
				form.Set("client", "another-client")
			case "missing timestamp":
				form.Del("timestamp")
			case "stale timestamp":
				form.Set("timestamp", time.Now().Add(-time.Hour).Format(time.RFC3339))
			case "missing loginId":
				form.Del("loginId")
			case "wrong method":
				method = http.MethodGet
			}
			request := httptest.NewRequest(method, "/sso/logout-callback", strings.NewReader(form.Encode()))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)
			if recorder.Code < 400 || recorder.Code >= 500 {
				t.Fatalf("invalid logout status = %d, want client error", recorder.Code)
			}
			check := httptest.NewRequest(http.MethodGet, "/protected", nil)
			check.AddCookie(&http.Cookie{Name: localCookie, Value: id})
			if loginID, ok := localLoginID(check); !ok || loginID != "alice" {
				t.Fatal("rejected logout removed local identity")
			}
		})
	}
}

// TestCallbackIsExcludedFromAccessLog verifies callback credentials are omitted while ordinary requests remain logged. TestCallbackIsExcludedFromAccessLog 验证回调凭证不进入访问日志，普通请求仍被记录。
func TestCallbackIsExcludedFromAccessLog(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var output bytes.Buffer
	previous := gin.DefaultWriter
	gin.DefaultWriter = &output
	t.Cleanup(func() { gin.DefaultWriter = previous })
	router := newDemoRouter()
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/sso/callback?ticket=private-ticket&back=private-state", nil))
	if output.Len() != 0 {
		t.Fatalf("callback was access-logged: %s", output.String())
	}
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/protected", nil))
	if !strings.Contains(output.String(), "/protected") {
		t.Fatalf("ordinary request was not logged: %s", output.String())
	}
}
