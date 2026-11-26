package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/sso"
)

// newTestCenter owns an isolated center Auth for each test. newTestCenter 为每个测试持有隔离的中心 Auth。
func newTestCenter(t *testing.T) *demoCenter {
	t.Helper()
	auth, err := newCenterAuth()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(auth.Close)
	return &demoCenter{auth: auth}
}

// centerRequest exercises actual routes without a network listener. centerRequest 无需网络监听即可调用实际路由。
func centerRequest(h http.Handler, method, path, form string, loginCookie *http.Cookie) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(form))
	if method == http.MethodPost {
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if loginCookie != nil {
		r.AddCookie(loginCookie)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

// centerLogin submits the real demo form and returns its server-backed Cookie. centerLogin 提交实际示例表单并返回关联服务端会话的 Cookie。
func centerLogin(t *testing.T, h http.Handler, loginID string, previous *http.Cookie) *http.Cookie {
	t.Helper()
	w := centerRequest(h, http.MethodPost, "/login", url.Values{"loginId": {loginID}}.Encode(), previous)
	cookies := w.Result().Cookies()
	if w.Code != http.StatusFound || len(cookies) != 1 {
		t.Fatalf("login: %d %s", w.Code, w.Body.String())
	}
	return cookies[0]
}

// centerTicket authorizes a browser and extracts the issued Ticket. centerTicket 为浏览器授权并提取签发的 Ticket。
func centerTicket(t *testing.T, h http.Handler, loginCookie *http.Cookie) string {
	t.Helper()
	values := url.Values{"client": {clientID}, "redirect": {callbackURL}, "centerSessionID": {"forged"}}
	w := centerRequest(h, http.MethodGet, "/sso/authorize?"+values.Encode(), "", loginCookie)
	location, err := url.Parse(w.Header().Get("Location"))
	if err != nil || w.Code != http.StatusFound || location.Query().Get("ticket") == "" {
		t.Fatalf("authorize: %d %s", w.Code, w.Body.String())
	}
	return location.Query().Get("ticket")
}

// centerExchange redeems a Ticket without requiring a browser Cookie. centerExchange 无需浏览器 Cookie 即可兑换 Ticket。
func centerExchange(h http.Handler, ticket string, loginCookie *http.Cookie) *httptest.ResponseRecorder {
	return centerRequest(h, http.MethodPost, "/sso/token", url.Values{
		"client": {clientID}, "clientSecret": {clientSecret}, "redirect": {callbackURL}, "ticket": {ticket},
	}.Encode(), loginCookie)
}

// TestCenterLogoutRevokesCopiedCookies verifies account-wide logout blocks new issuance and preserves other users. TestCenterLogoutRevokesCopiedCookies 验证账号注销阻止旧 Cookie 重新签发凭证且不影响其他用户。
func TestCenterLogoutRevokesCopiedCookies(t *testing.T) {
	center := newTestCenter(t)
	server := sso.NewServer()
	defer server.Close()
	h, err := newDemoHandler(server, center.auth)
	if err != nil {
		t.Fatal(err)
	}
	first := centerLogin(t, h, "alice", nil)
	second := centerLogin(t, h, "alice", nil)
	other := centerLogin(t, h, "bob", nil)
	oldTicket := centerTicket(t, h, first)
	w := centerRequest(h, http.MethodPost, "/sso/logout", "", first)
	if w.Code != http.StatusOK {
		t.Fatalf("logout: %d %s", w.Code, w.Body.String())
	}
	for _, copied := range []*http.Cookie{first, second} {
		w := centerRequest(h, http.MethodGet, "/sso/authorize?"+url.Values{"client": {clientID}, "redirect": {callbackURL}}.Encode(), "", copied)
		location, err := url.Parse(w.Header().Get("Location"))
		if err != nil || w.Code != http.StatusFound || location.Path != "/login" {
			t.Fatalf("copied Cookie authenticated: %d %v", w.Code, location)
		}
	}
	if w := centerExchange(h, oldTicket, nil); w.Code == http.StatusOK {
		t.Fatal("pre-logout Ticket exchanged")
	}
	centerTicket(t, h, other)
	fresh := centerLogin(t, h, "alice", first)
	if w := centerExchange(h, centerTicket(t, h, fresh), nil); w.Code != http.StatusOK {
		t.Fatalf("fresh login failed: %d %s", w.Code, w.Body.String())
	}
}

// TestCenterExchangeRequiresOriginalSession verifies exact binding, finite lifetime, and absence of bearer-token disclosure. TestCenterExchangeRequiresOriginalSession 验证精确会话绑定、有限有效期及不泄露登录 Token。
func TestCenterExchangeRequiresOriginalSession(t *testing.T) {
	center := newTestCenter(t)
	server := sso.NewServer()
	defer server.Close()
	h, err := newDemoHandler(server, center.auth)
	if err != nil {
		t.Fatal(err)
	}
	first := centerLogin(t, h, "alice", nil)
	second := centerLogin(t, h, "alice", nil)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.AddCookie(first)
	token, ok := sso.LoginIDFromCookie(cookie)(request)
	if !ok {
		t.Fatal("missing center token")
	}
	good := centerTicket(t, h, first)
	w := centerExchange(h, good, second)
	var response struct {
		Code int                      `json:"code"`
		Data sso.TicketExchangeResult `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	result := response.Data
	if w.Code != http.StatusOK || response.Code != 0 || result.LoginID != "alice" || result.RemainTokenTimeout <= 0 || result.RemainTokenTimeout > int64(cookie.MaxAge/time.Second) || result.DeviceID == "" || result.Extra["centerSessionID"] != centerSessionRef(token) {
		t.Fatalf("exchange: %d %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), token) || strings.Contains(w.Body.String(), first.Value) {
		t.Fatal("exchange disclosed center credentials")
	}

	stale := centerTicket(t, h, first)
	if err := center.auth.LogoutByToken(context.Background(), token); err != nil {
		t.Fatal(err)
	}
	if w := centerExchange(h, stale, second); w.Code != http.StatusBadRequest {
		t.Fatalf("another browser rescued retired Ticket: %d %s", w.Code, w.Body.String())
	}
	if w := centerExchange(h, stale, nil); w.Code != http.StatusBadRequest {
		t.Fatal("failed enrichment restored consumed Ticket")
	}
	centerTicket(t, h, second)

	// Directly issued unbound credentials cannot bypass this application's center-session policy. 直接签发的未绑定凭证不能绕过本应用的中心会话策略。
	unbound, err := server.GenerateTicket(context.Background(), clientID, "alice", callbackURL, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if w := centerExchange(h, unbound.Ticket, second); w.Code != http.StatusBadRequest {
		t.Fatalf("unbound Ticket accepted: %d", w.Code)
	}
	activeBinding, err := server.ValidateTicket(context.Background(), centerTicket(t, h, second))
	if err != nil {
		t.Fatal(err)
	}
	wrongOwner, err := server.GenerateTicket(context.Background(), clientID, "bob", callbackURL, nil, activeBinding.Extra)
	if err != nil {
		t.Fatal(err)
	}
	if w := centerExchange(h, wrongOwner.Ticket, second); w.Code != http.StatusBadRequest {
		t.Fatalf("foreign session binding accepted: %d", w.Code)
	}
}

// TestCenterLoginReplacesBrowserSession verifies login replacement retires the previous Cookie without affecting another browser. TestCenterLoginReplacesBrowserSession 验证登录替换使旧 Cookie 失效且不影响其他浏览器。
func TestCenterLoginReplacesBrowserSession(t *testing.T) {
	center := newTestCenter(t)
	server := sso.NewServer()
	defer server.Close()
	h, err := newDemoHandler(server, center.auth)
	if err != nil {
		t.Fatal(err)
	}
	first := centerLogin(t, h, "alice", nil)
	other := centerLogin(t, h, "alice", nil)
	replacement := centerLogin(t, h, "bob", first)
	for _, tc := range []struct {
		cookie *http.Cookie
		want   string
	}{{first, ""}, {other, "alice"}, {replacement, "bob"}} {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.AddCookie(tc.cookie)
		id, ok := center.loginID(r)
		if id != tc.want || ok != (tc.want != "") {
			t.Fatalf("identity: %q %v, want %q", id, ok, tc.want)
		}
	}
}

// TestCenterReadEndpointsRequireOriginalSession keeps reads consistent with exchange while exposing backend failures. TestCenterReadEndpointsRequireOriginalSession 使读取接口与换票保持一致，同时保留后端故障。
func TestCenterReadEndpointsRequireOriginalSession(t *testing.T) {
	center := newTestCenter(t)
	server := sso.NewServer()
	defer server.Close()
	h, err := newDemoHandler(server, center.auth)
	if err != nil {
		t.Fatal(err)
	}
	first := centerLogin(t, h, "alice", nil)
	second := centerLogin(t, h, "alice", nil)
	ticket := centerTicket(t, h, first)
	values := url.Values{"client": {clientID}, "clientSecret": {clientSecret}, "ticket": {ticket}}
	read := func(path string) *httptest.ResponseRecorder {
		return centerRequest(h, http.MethodPost, path, values.Encode(), second)
	}
	for _, path := range []string{"/sso/introspect", "/sso/userinfo"} {
		if w := read(path); w.Code != http.StatusOK {
			t.Fatalf("initial read: %d %s", w.Code, w.Body.String())
		}
	}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.AddCookie(first)
	token, ok := sso.LoginIDFromCookie(cookie)(request)
	if !ok {
		t.Fatal("missing center token")
	}
	if err := center.auth.LogoutByToken(context.Background(), token); err != nil {
		t.Fatal(err)
	}

	// The SSO record still exists; application-session validation alone must stop disclosure. SSO 记录仍存在，仅应用会话校验就必须阻止信息披露。
	if _, err := server.ValidateTicket(context.Background(), ticket); err != nil {
		t.Fatalf("SSO record unexpectedly revoked: %v", err)
	}
	w := read("/sso/introspect")
	var response struct {
		Data sso.CredentialInfo `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusOK || response.Data.Active || response.Data.LoginID != "" || len(response.Data.Extra) != 0 {
		t.Fatalf("retired session disclosed metadata: %d %s", w.Code, w.Body.String())
	}
	if w := read("/sso/userinfo"); w.Code != http.StatusUnauthorized {
		t.Fatalf("retired userinfo: %d", w.Code)
	}
	values.Set("ticket", centerTicket(t, h, second))
	if w := read("/sso/introspect"); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "\"active\":true") {
		t.Fatalf("other browser affected: %d %s", w.Code, w.Body.String())
	}

	// A missing backend is an error, not an ordinary inactive session. 后端不可用属于错误，不应被视为普通会话失效。
	center.auth.Close()
	for _, path := range []string{"/sso/introspect", "/sso/userinfo"} {
		if w := read(path); w.Code != http.StatusInternalServerError {
			t.Fatalf("backend failure hidden: %d %s", w.Code, w.Body.String())
		}
	}
}
