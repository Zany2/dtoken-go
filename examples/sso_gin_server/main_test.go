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

// TestSafeBackAllowsLocalPathsOnly verifies login redirects cannot leave the site. TestSafeBackAllowsLocalPathsOnly 验证登录重定向只能指向站内路径。
func TestSafeBackAllowsLocalPathsOnly(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "empty", raw: "", want: "/"},
		{name: "relative", raw: "/protected?from=login", want: "/protected?from=login"},
		{name: "absolute", raw: "https://evil.example/", want: "/"},
		{name: "scheme relative", raw: "//evil.example/", want: "/"},
		{name: "backslash", raw: `/\\evil.example/`, want: "/"},
		{name: "no leading slash", raw: "protected", want: "/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := safeBack(tt.raw); got != tt.want {
				t.Fatalf("safeBack(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

// TestGinLoginAndHome verifies login page rendering, login cookie creation, and home status. TestGinLoginAndHome 验证登录页渲染、登录 Cookie 创建与首页状态。
func TestGinLoginAndHome(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := sso.NewServer()
	defer server.Close()
	router, err := newDemoRouter(server)
	if err != nil {
		t.Fatal(err)
	}

	page := performGinRequest(router, http.MethodGet, "/login?back=%2Fprotected%3Ffrom%3Dlogin", "")
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "/protected?from=login") {
		t.Fatalf("login page status=%d body=%q", page.Code, page.Body.String())
	}

	login := performGinRequest(router, http.MethodPost, "/login", "loginId=alice&back=%2Fprotected")
	if login.Code != http.StatusFound || login.Header().Get("Location") != "/protected" {
		t.Fatalf("login redirect status=%d location=%q", login.Code, login.Header().Get("Location"))
	}
	cookies := login.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != cookie.Name || cookies[0].Value == "" || cookies[0].Value == "alice" {
		t.Fatalf("login cookies = %+v, want signed alice login cookie", cookies)
	}

	homeRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	homeRequest.AddCookie(cookies[0])
	homeRecorder := httptest.NewRecorder()
	router.ServeHTTP(homeRecorder, homeRequest)
	if homeRecorder.Code != http.StatusOK || !strings.Contains(homeRecorder.Body.String(), "loginId: alice") {
		t.Fatalf("home status=%d body=%q", homeRecorder.Code, homeRecorder.Body.String())
	}

	unsafe := performGinRequest(router, http.MethodPost, "/login", "back=https%3A%2F%2Fevil.example%2F")
	if unsafe.Code != http.StatusFound || unsafe.Header().Get("Location") != "/" {
		t.Fatalf("unsafe back status=%d location=%q, want /", unsafe.Code, unsafe.Header().Get("Location"))
	}
}

// TestGinSSORoutesIssueAndExchangeTicket verifies Gin protocol route wiring. TestGinSSORoutesIssueAndExchangeTicket 验证 Gin 协议路由注册、Ticket 签发与交换。
func TestGinSSORoutesIssueAndExchangeTicket(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := sso.NewServer()
	defer server.Close()
	router, err := newDemoRouter(server)
	if err != nil {
		t.Fatal(err)
	}

	authorizeRequest := httptest.NewRequest(http.MethodGet, "/sso/authorize?client="+url.QueryEscape(clientID)+"&redirect="+url.QueryEscape(callbackURL)+"&back=browser-state", nil)
	anonymous := httptest.NewRecorder()
	router.ServeHTTP(anonymous, authorizeRequest)
	loginURL, err := url.Parse(anonymous.Header().Get("Location"))
	if err != nil || anonymous.Code != http.StatusFound || loginURL.Path != "/login" || loginURL.Query().Get("back") != authorizeRequest.URL.RequestURI() {
		t.Fatalf("anonymous authorize: status=%d location=%v error=%v", anonymous.Code, loginURL, err)
	}
	cookieRecorder := performGinRequest(router, http.MethodPost, "/login", url.Values{
		"loginId": {"alice"}, "back": {loginURL.Query().Get("back")},
	}.Encode())
	if cookieRecorder.Code != http.StatusFound || cookieRecorder.Header().Get("Location") != authorizeRequest.URL.RequestURI() || len(cookieRecorder.Result().Cookies()) != 1 {
		t.Fatalf("login did not resume authorization: status=%d headers=%v", cookieRecorder.Code, cookieRecorder.Header())
	}
	authorizeRequest.AddCookie(cookieRecorder.Result().Cookies()[0])
	authorizeRecorder := httptest.NewRecorder()
	router.ServeHTTP(authorizeRecorder, authorizeRequest)
	if authorizeRecorder.Code != http.StatusFound {
		t.Fatalf("authorize status = %d, want %d", authorizeRecorder.Code, http.StatusFound)
	}
	location, err := url.Parse(authorizeRecorder.Header().Get("Location"))
	if err != nil {
		t.Fatalf("parse authorize location: %v", err)
	}
	ticket := location.Query().Get("ticket")
	if ticket == "" || location.Path != "/sso/callback" || location.Query().Get("back") != "browser-state" {
		t.Fatalf("authorize location = %q, want callback with ticket", location.String())
	}

	form := url.Values{
		"ticket":       {ticket},
		"client":       {clientID},
		"clientSecret": {clientSecret},
		"redirect":     {callbackURL},
	}

	// Invalid credentials must not consume the Ticket. 无效凭据不能消费 Ticket。
	form.Set("clientSecret", "wrong-secret")
	invalid := performGinRequest(router, http.MethodPost, "/sso/token", form.Encode())
	if invalid.Code != http.StatusUnauthorized {
		t.Fatalf("invalid credentials status = %d", invalid.Code)
	}
	form.Set("clientSecret", clientSecret)
	tokenRequest := httptest.NewRequest(http.MethodPost, "/sso/token", strings.NewReader(form.Encode()))
	tokenRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	tokenRecorder := httptest.NewRecorder()
	router.ServeHTTP(tokenRecorder, tokenRequest)
	if tokenRecorder.Code != http.StatusOK {
		t.Fatalf("token status = %d, want %d, body=%s", tokenRecorder.Code, http.StatusOK, tokenRecorder.Body.String())
	}
	var response sso.Response
	if err = json.Unmarshal(tokenRecorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode token response: %v", err)
	}
	data, ok := response.Data.(map[string]any)
	if !ok || data["loginId"] != "alice" {
		t.Fatalf("token response data = %#v, want loginId alice", response.Data)
	}
	if replay := performGinRequest(router, http.MethodPost, "/sso/token", form.Encode()); replay.Code != http.StatusBadRequest {
		t.Fatalf("replayed ticket status = %d, want 400", replay.Code)
	}
}

// performGinRequest submits a request without opening a network listener. performGinRequest 不启动网络监听即可提交请求。
func performGinRequest(router http.Handler, method, path, form string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(form))
	if form != "" {
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}
