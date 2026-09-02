package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/Zany2/dtoken-go/sso"
	"github.com/gin-gonic/gin"
)

// TestProtectedRedirectAndLocalSession verifies Gin protected-route behavior and local sessions. TestProtectedRedirectAndLocalSession 验证 Gin 受保护路由行为与本地会话。
func TestProtectedRedirectAndLocalSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetGinClientSessions(t)
	router := gin.New()
	router.GET("/protected", protected)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/protected", nil))
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
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "loginId: user-1001") {
		t.Fatalf("authenticated protected response status=%d body=%q", recorder.Code, recorder.Body.String())
	}
	if _, ok := localLoginID(nil); ok {
		t.Fatal("localLoginID(nil) = true, want false")
	}
}

// TestHome verifies the Gin client landing page. TestHome 验证 Gin 客户端首页响应。
func TestHome(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	home(context)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "Gin SSO Client") {
		t.Fatalf("home status=%d body=%q, want Gin client landing page", recorder.Code, recorder.Body.String())
	}
}

// TestCallbackAndSingleLogout verifies Gin ticket callback and logout callback wiring. TestCallbackAndSingleLogout 验证 Gin Ticket 回调与单点注销回调链路。
func TestCallbackAndSingleLogout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetGinClientSessions(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sso/token" {
			t.Fatalf("token request path = %q, want /sso/token", r.URL.Path)
		}
		_ = r.ParseForm()
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

	router := gin.New()
	router.GET("/sso/callback", callback)
	router.POST("/sso/logout-callback", ginWrap(clientApp.LogoutCallbackHandler(logoutCallback)))

	missing := httptest.NewRecorder()
	router.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/sso/callback", nil))
	if missing.Code != http.StatusBadRequest {
		t.Fatalf("missing ticket status = %d, want %d", missing.Code, http.StatusBadRequest)
	}

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/sso/callback?ticket=ticket-value", nil))
	if recorder.Code != http.StatusFound {
		t.Fatalf("callback status = %d, want %d", recorder.Code, http.StatusFound)
	}
	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != localCookie || cookies[0].Value == "" {
		t.Fatalf("callback cookies = %+v, want local session cookie", cookies)
	}

	second, err := newLocalSession("user-1001")
	if err != nil {
		t.Fatalf("newLocalSession(second) error = %v", err)
	}
	form := url.Values{"loginId": {"user-1001"}}
	logoutRequest := httptest.NewRequest(http.MethodPost, "/sso/logout-callback", strings.NewReader(form.Encode()))
	logoutRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	logoutRecorder := httptest.NewRecorder()
	router.ServeHTTP(logoutRecorder, logoutRequest)
	if logoutRecorder.Code != http.StatusOK {
		t.Fatalf("logout callback status = %d, want %d", logoutRecorder.Code, http.StatusOK)
	}
	for _, sessionID := range []string{cookies[0].Value, second} {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.AddCookie(&http.Cookie{Name: localCookie, Value: sessionID})
		if _, ok := localLoginID(request); ok {
			t.Fatalf("session %q remains after logout callback", sessionID)
		}
	}
}

// TestLogoutClearsLocalSession verifies Gin local logout removes the session and expires the cookie. TestLogoutClearsLocalSession 验证 Gin 本地注销会删除会话并过期 Cookie。
func TestLogoutClearsLocalSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetGinClientSessions(t)
	sessionID, err := newLocalSession("user-1001")
	if err != nil {
		t.Fatalf("newLocalSession() error = %v", err)
	}

	router := gin.New()
	router.GET("/logout", logout)
	request := httptest.NewRequest(http.MethodGet, "/logout", nil)
	request.AddCookie(&http.Cookie{Name: localCookie, Value: sessionID})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusFound || recorder.Header().Get("Location") != "/" {
		t.Fatalf("logout status=%d location=%q, want redirect to /", recorder.Code, recorder.Header().Get("Location"))
	}
	if _, ok := localLoginID(request); ok {
		t.Fatal("local session remains after logout")
	}
	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != localCookie || cookies[0].MaxAge != -1 {
		t.Fatalf("logout cookies = %+v, want expired local cookie", cookies)
	}
}

// TestCallbackRejectsIncompleteSSOResponse verifies Gin rejects an empty login subject. TestCallbackRejectsIncompleteSSOResponse 验证 Gin 不会接受缺少登录主体的响应。
func TestCallbackRejectsIncompleteSSOResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetGinClientSessions(t)
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

	router := gin.New()
	router.GET("/sso/callback", callback)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/sso/callback?ticket=ticket-value", nil))
	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("empty login ID status = %d, want %d", recorder.Code, http.StatusBadGateway)
	}
}

// TestCallbackHandlesExchangeFailure verifies upstream SSO failures become a gateway error. TestCallbackHandlesExchangeFailure 验证上游 SSO 失败会转换为网关错误。
func TestCallbackHandlesExchangeFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetGinClientSessions(t)
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

	router := gin.New()
	router.GET("/sso/callback", callback)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/sso/callback?ticket=ticket-value", nil))
	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("exchange failure status = %d, want %d", recorder.Code, http.StatusBadGateway)
	}
}

// TestNewLocalSessionRejectsEmptyLoginID verifies empty subjects are rejected before storage. TestNewLocalSessionRejectsEmptyLoginID 验证空登录主体不会写入会话。
func TestNewLocalSessionRejectsEmptyLoginID(t *testing.T) {
	resetGinClientSessions(t)
	if sessionID, err := newLocalSession(""); err == nil || sessionID != "" {
		t.Fatalf("newLocalSession(\"\") = %q, %v, want error and empty session", sessionID, err)
	}
}

func resetGinClientSessions(t *testing.T) {
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
