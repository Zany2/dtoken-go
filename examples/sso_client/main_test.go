package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/Zany2/dtoken-go/sso"
)

// TestProtectedRedirectAndLocalSession verifies redirect behavior and local session lookup. TestProtectedRedirectAndLocalSession 验证未登录重定向与本地会话解析。
func TestProtectedRedirectAndLocalSession(t *testing.T) {
	resetClientSessions(t)

	recorder := httptest.NewRecorder()
	protected(recorder, httptest.NewRequest(http.MethodGet, "/protected", nil))
	if recorder.Code != http.StatusFound {
		t.Fatalf("protected redirect status = %d, want %d", recorder.Code, http.StatusFound)
	}
	location, err := url.Parse(recorder.Header().Get("Location"))
	if err != nil {
		t.Fatalf("parse redirect location: %v", err)
	}
	if location.Path != "/sso/authorize" || location.Query().Get("redirect") != callbackURL {
		t.Fatalf("redirect location = %q, want authorize URL with callback", location.String())
	}

	sessionID, err := newLocalSession("user-1001")
	if err != nil {
		t.Fatalf("newLocalSession() error = %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.AddCookie(&http.Cookie{Name: localCookie, Value: sessionID})
	recorder = httptest.NewRecorder()
	protected(recorder, request)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "loginId: user-1001") {
		t.Fatalf("authenticated protected response status=%d body=%q", recorder.Code, recorder.Body.String())
	}
	if _, ok := localLoginID(nil); ok {
		t.Fatal("localLoginID(nil) = true, want false")
	}
}

// TestHome verifies the client landing page. TestHome 验证客户端首页响应。
func TestHome(t *testing.T) {
	recorder := httptest.NewRecorder()
	home(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "SSO Client") {
		t.Fatalf("home status=%d body=%q, want client landing page", recorder.Code, recorder.Body.String())
	}
}

// TestCallbackAndLogout verifies ticket exchange, callback failures, and local logout. TestCallbackAndLogout 验证 Ticket 交换、回调失败与本地注销。
func TestCallbackAndLogout(t *testing.T) {
	resetClientSessions(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sso/token" {
			t.Fatalf("token request path = %q, want /sso/token", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm() error = %v", err)
		}
		if r.Form.Get("ticket") != "ticket-value" || r.Form.Get("redirect") != callbackURL {
			t.Fatalf("token request form = %v, want ticket and redirect", r.Form)
		}
		_ = json.NewEncoder(w).Encode(sso.OKResponse(sso.TicketExchangeResult{LoginID: "user-1001"}))
	}))
	defer server.Close()

	previous := clientApp
	clientApp = sso.NewClientApp(sso.ClientConfig{
		Mode:         sso.ModeTicket,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		ServerURL:    server.URL,
		CheckSign:    false,
		Endpoints:    sso.DefaultEndpoints(),
		Params:       sso.DefaultParamNames(),
	})
	t.Cleanup(func() { clientApp = previous })

	missing := httptest.NewRecorder()
	callback(missing, httptest.NewRequest(http.MethodGet, "/sso/callback", nil))
	if missing.Code != http.StatusBadRequest {
		t.Fatalf("missing ticket status = %d, want %d", missing.Code, http.StatusBadRequest)
	}

	recorder := httptest.NewRecorder()
	callback(recorder, httptest.NewRequest(http.MethodGet, "/sso/callback?ticket=ticket-value", nil))
	if recorder.Code != http.StatusFound {
		t.Fatalf("callback status = %d, want %d", recorder.Code, http.StatusFound)
	}
	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != localCookie || cookies[0].Value == "" {
		t.Fatalf("callback cookies = %+v, want local session cookie", cookies)
	}

	logoutRequest := httptest.NewRequest(http.MethodGet, "/logout", nil)
	logoutRequest.AddCookie(cookies[0])
	logoutRecorder := httptest.NewRecorder()
	logout(logoutRecorder, logoutRequest)
	if logoutRecorder.Code != http.StatusFound {
		t.Fatalf("logout status = %d, want %d", logoutRecorder.Code, http.StatusFound)
	}
	if _, ok := localLoginID(logoutRequest); ok {
		t.Fatal("local session remains after logout")
	}
	if cleared := logoutRecorder.Result().Cookies(); len(cleared) != 1 || cleared[0].MaxAge != -1 {
		t.Fatalf("logout cookies = %+v, want expired cookie", cleared)
	}
}

// TestCallbackRejectsIncompleteSSOResponse verifies an empty login subject is not stored locally. TestCallbackRejectsIncompleteSSOResponse 验证不会保存缺少登录主体的 SSO 响应。
func TestCallbackRejectsIncompleteSSOResponse(t *testing.T) {
	resetClientSessions(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(sso.OKResponse(sso.TicketExchangeResult{}))
	}))
	defer server.Close()

	previous := clientApp
	clientApp = sso.NewClientApp(sso.ClientConfig{
		Mode:      sso.ModeTicket,
		ServerURL: server.URL,
		CheckSign: false,
		Endpoints: sso.DefaultEndpoints(),
		Params:    sso.DefaultParamNames(),
	})
	t.Cleanup(func() { clientApp = previous })

	recorder := httptest.NewRecorder()
	callback(recorder, httptest.NewRequest(http.MethodGet, "/sso/callback?ticket=ticket-value", nil))
	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("empty login ID status = %d, want %d", recorder.Code, http.StatusBadGateway)
	}
}

// TestCallbackHandlesExchangeFailure verifies upstream SSO failures become a gateway error. TestCallbackHandlesExchangeFailure 验证上游 SSO 失败会转换为网关错误。
func TestCallbackHandlesExchangeFailure(t *testing.T) {
	resetClientSessions(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "upstream unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	previous := clientApp
	clientApp = sso.NewClientApp(sso.ClientConfig{
		Mode:      sso.ModeTicket,
		ServerURL: server.URL,
		CheckSign: false,
		Endpoints: sso.DefaultEndpoints(),
		Params:    sso.DefaultParamNames(),
	})
	t.Cleanup(func() { clientApp = previous })

	recorder := httptest.NewRecorder()
	callback(recorder, httptest.NewRequest(http.MethodGet, "/sso/callback?ticket=ticket-value", nil))
	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("exchange failure status = %d, want %d", recorder.Code, http.StatusBadGateway)
	}
}

// TestNewLocalSessionRejectsEmptyLoginID verifies empty subjects are rejected before storage. TestNewLocalSessionRejectsEmptyLoginID 验证空登录主体不会写入会话。
func TestNewLocalSessionRejectsEmptyLoginID(t *testing.T) {
	resetClientSessions(t)
	if sessionID, err := newLocalSession(""); err == nil || sessionID != "" {
		t.Fatalf("newLocalSession(\"\") = %q, %v, want error and empty session", sessionID, err)
	}
}

// TestLogoutCallbackRemovesAllSessions verifies single logout clears every local session for one login ID. TestLogoutCallbackRemovesAllSessions 验证单点注销会清理指定用户的全部本地会话。
func TestLogoutCallbackRemovesAllSessions(t *testing.T) {
	resetClientSessions(t)
	first, err := newLocalSession("user-1001")
	if err != nil {
		t.Fatalf("newLocalSession(first) error = %v", err)
	}
	second, err := newLocalSession("user-1001")
	if err != nil {
		t.Fatalf("newLocalSession(second) error = %v", err)
	}
	other, err := newLocalSession("user-2002")
	if err != nil {
		t.Fatalf("newLocalSession(other) error = %v", err)
	}
	if err = logoutCallback(nil, sso.LogoutCallback{LoginID: "user-1001"}); err != nil {
		t.Fatalf("logoutCallback() error = %v", err)
	}
	for _, sessionID := range []string{first, second} {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.AddCookie(&http.Cookie{Name: localCookie, Value: sessionID})
		if _, ok := localLoginID(request); ok {
			t.Fatalf("session %q remains after single logout", sessionID)
		}
	}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.AddCookie(&http.Cookie{Name: localCookie, Value: other})
	if loginID, ok := localLoginID(request); !ok || loginID != "user-2002" {
		t.Fatalf("other session lookup = %q, %v, want user-2002 true", loginID, ok)
	}
}

func resetClientSessions(t *testing.T) {
	t.Helper()
	localSessions.mu.Lock()
	localSessions.values = make(map[string]string)
	localSessions.mu.Unlock()
	t.Cleanup(func() {
		localSessions.mu.Lock()
		localSessions.values = make(map[string]string)
		localSessions.mu.Unlock()
	})
}
