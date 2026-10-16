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

// TestHomeAndLoginFlow verifies anonymous and authenticated pages, login cookies, and redirects. TestHomeAndLoginFlow 验证匿名与已登录页面、登录 Cookie 及回跳。
func TestHomeAndLoginFlow(t *testing.T) {
	anonymous := httptest.NewRecorder()
	home(anonymous, httptest.NewRequest(http.MethodGet, "/", nil))
	if anonymous.Code != http.StatusOK || !strings.Contains(anonymous.Body.String(), "not logged in") {
		t.Fatalf("anonymous home status=%d body=%q, want not logged in", anonymous.Code, anonymous.Body.String())
	}

	page := httptest.NewRecorder()
	login(page, httptest.NewRequest(http.MethodGet, "/login?back=%2Fprotected%3Ffrom%3Dlogin", nil))
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "/protected?from=login") {
		t.Fatalf("login page status=%d body=%q, want local back path", page.Code, page.Body.String())
	}

	form := "loginId=alice&back=%2Fprotected"
	request := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loggedIn := httptest.NewRecorder()
	login(loggedIn, request)
	if loggedIn.Code != http.StatusFound || loggedIn.Header().Get("Location") != "/protected" {
		t.Fatalf("login redirect status=%d location=%q, want /protected", loggedIn.Code, loggedIn.Header().Get("Location"))
	}
	cookies := loggedIn.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != cookie.Name || cookies[0].Value == "" || cookies[0].Value == "alice" || !cookies[0].HttpOnly || cookies[0].MaxAge <= 0 {
		t.Fatalf("login cookies = %+v, want signed alice session cookie", cookies)
	}

	loginCookie := cookies[0]
	homeRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	homeRequest.AddCookie(loginCookie)
	authenticated := httptest.NewRecorder()
	home(authenticated, homeRequest)
	if authenticated.Code != http.StatusOK || !strings.Contains(authenticated.Body.String(), "loginId: alice") {
		t.Fatalf("authenticated home status=%d body=%q, want alice", authenticated.Code, authenticated.Body.String())
	}

	unsafe := httptest.NewRecorder()
	unsafeRequest := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("back=https%3A%2F%2Fevil.example%2F"))
	unsafeRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	login(unsafe, unsafeRequest)
	if unsafe.Code != http.StatusFound || unsafe.Header().Get("Location") != "/" {
		t.Fatalf("unsafe back status=%d location=%q, want /", unsafe.Code, unsafe.Header().Get("Location"))
	}
	unsafeCookies := unsafe.Result().Cookies()
	if len(unsafeCookies) != 1 || unsafeCookies[0].Name != cookie.Name || unsafeCookies[0].Value == "" || unsafeCookies[0].Value == "user-1001" {
		t.Fatalf("default login cookie = %+v, want signed user-1001 cookie", unsafeCookies)
	}
}

// TestLoginRejectsMalformedFormAndUnsupportedMethod verifies login input and method errors. TestLoginRejectsMalformedFormAndUnsupportedMethod 验证登录表单错误与不支持的方法。
func TestLoginRejectsMalformedFormAndUnsupportedMethod(t *testing.T) {
	malformed := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("back=%zz"))
	malformed.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	malformedRecorder := httptest.NewRecorder()
	login(malformedRecorder, malformed)
	if malformedRecorder.Code != http.StatusBadRequest {
		t.Fatalf("malformed form status = %d, want %d", malformedRecorder.Code, http.StatusBadRequest)
	}

	unsupported := httptest.NewRecorder()
	login(unsupported, httptest.NewRequest(http.MethodPut, "/login", nil))
	if unsupported.Code != http.StatusMethodNotAllowed {
		t.Fatalf("unsupported method status = %d, want %d", unsupported.Code, http.StatusMethodNotAllowed)
	}
}

// TestHTTPSSORoutesIssueAndExchangeTicket verifies standard SSO route registration and Ticket flow. TestHTTPSSORoutesIssueAndExchangeTicket 验证标准 SSO 路由注册及 Ticket 流程。
func TestHTTPSSORoutesIssueAndExchangeTicket(t *testing.T) {
	server := sso.NewServer()
	defer server.Close()
	mux, err := newDemoHandler(server)
	if err != nil {
		t.Fatal(err)
	}

	authorizeRequest := httptest.NewRequest(http.MethodGet, "/sso/authorize?client="+url.QueryEscape(clientID)+"&redirect="+url.QueryEscape(callbackURL), nil)
	anonymous := httptest.NewRecorder()
	mux.ServeHTTP(anonymous, authorizeRequest)
	loginURL, err := url.Parse(anonymous.Header().Get("Location"))
	if err != nil || anonymous.Code != http.StatusFound || loginURL.Path != "/login" || loginURL.Query().Get("back") != authorizeRequest.URL.RequestURI() {
		t.Fatalf("anonymous authorization: status=%d, location=%v, error=%v", anonymous.Code, loginURL, err)
	}

	// Complete the actual login form before resuming authorization. 先提交实际登录表单，再继续授权。
	cookieRecorder := httptest.NewRecorder()
	loginRequest := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(url.Values{
		"loginId": {"alice"}, "back": {loginURL.Query().Get("back")},
	}.Encode()))
	loginRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	mux.ServeHTTP(cookieRecorder, loginRequest)
	if cookieRecorder.Code != http.StatusFound || cookieRecorder.Header().Get("Location") != authorizeRequest.URL.RequestURI() || len(cookieRecorder.Result().Cookies()) != 1 {
		t.Fatalf("login did not resume authorization: status=%d headers=%v", cookieRecorder.Code, cookieRecorder.Header())
	}
	authorizeRequest.AddCookie(cookieRecorder.Result().Cookies()[0])
	authorizeRecorder := httptest.NewRecorder()
	mux.ServeHTTP(authorizeRecorder, authorizeRequest)
	if authorizeRecorder.Code != http.StatusFound {
		t.Fatalf("authorize status = %d, want %d", authorizeRecorder.Code, http.StatusFound)
	}
	location, err := url.Parse(authorizeRecorder.Header().Get("Location"))
	if err != nil {
		t.Fatalf("parse authorize location: %v", err)
	}
	ticket := location.Query().Get("ticket")
	if ticket == "" || location.Path != "/sso/callback" {
		t.Fatalf("authorize location = %q, want callback with ticket", location.String())
	}

	form := url.Values{
		"ticket":       {ticket},
		"client":       {clientID},
		"clientSecret": {clientSecret},
		"redirect":     {callbackURL},
	}

	// Invalid client credentials must not consume the Ticket. 错误客户端凭证不能消费 Ticket。
	form.Set("clientSecret", "wrong-secret")
	invalidRequest := httptest.NewRequest(http.MethodPost, "/sso/token", strings.NewReader(form.Encode()))
	invalidRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	invalidRecorder := httptest.NewRecorder()
	mux.ServeHTTP(invalidRecorder, invalidRequest)
	if invalidRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("invalid client credentials status = %d", invalidRecorder.Code)
	}
	form.Set("clientSecret", clientSecret)
	tokenRequest := httptest.NewRequest(http.MethodPost, "/sso/token", strings.NewReader(form.Encode()))
	tokenRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	tokenRecorder := httptest.NewRecorder()
	mux.ServeHTTP(tokenRecorder, tokenRequest)
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
	replayRequest := httptest.NewRequest(http.MethodPost, "/sso/token", strings.NewReader(form.Encode()))
	replayRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	replay := httptest.NewRecorder()
	mux.ServeHTTP(replay, replayRequest)
	if replay.Code != http.StatusBadRequest {
		t.Fatalf("replayed ticket status = %d, want 400", replay.Code)
	}
}
