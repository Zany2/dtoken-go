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
	if err := server.RegisterClient(&sso.Client{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Name:         "Demo Client",
		RedirectURIs: []string{callbackURL},
		Modes:        []sso.Mode{sso.ModeTicket},
	}); err != nil {
		t.Fatalf("RegisterClient() error = %v", err)
	}
	httpSSO := sso.NewHTTPServer(server, sso.HTTPOptions{
		ServerOptions: sso.ServerOptions{
			CheckSign: false,
			Endpoints: sso.DefaultEndpoints(),
			Params:    sso.DefaultParamNames(),
		},
		LoginIDResolver: sso.LoginIDFromCookie(cookie),
		LoginPageURL:    "http://localhost:9000/login",
		Cookie:          cookie,
	})
	mux := http.NewServeMux()
	httpSSO.Register(mux)

	authorizeRequest := httptest.NewRequest(http.MethodGet, "/sso/authorize?client="+url.QueryEscape(clientID)+"&redirect="+url.QueryEscape(callbackURL), nil)
	cookieRecorder := httptest.NewRecorder()
	sso.SetLoginIDCookie(cookieRecorder, cookie, "alice")
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
}
