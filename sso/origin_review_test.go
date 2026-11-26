package sso

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// TestOriginComparisonPreservesRedirectBinding permits equivalent origins without relaxing exact authorization callbacks. TestOriginComparisonPreservesRedirectBinding 允许等价来源，不放宽精确授权回调绑定。
func TestOriginComparisonPreservesRedirectBinding(t *testing.T) {
	s := NewServer()
	defer s.Close()
	client := &Client{ClientID: "app", RedirectURIs: []string{"https://APP.example:443/sso/callback"}}
	for _, tc := range []struct {
		target  string
		allowed bool
	}{
		{"https://app.example/logout", true}, {"https://app.example:0443/logout", true},
		{"https://app.example:8443/logout", false}, {"http://app.example/logout", false},
		{"https://app.example.evil/logout", false}, {"https://user@app.example/logout", false},
		{"https://app.example/logout#", false}, {"https://app.example:0/logout", false},
		{"https://app.example:/logout", false}, {"https://app.example:65536/logout", false},
		{"https://app.example/%5Clogout", false},
	} {
		if got := s.isValidLogoutCallbackURL(client, tc.target); got != tc.allowed {
			t.Fatalf("callback %q allowed=%v", tc.target, got)
		}
	}
	if s.isValidRedirectURI(client, "https://app.example/sso/callback") {
		t.Fatal("exact redirect binding was relaxed")
	}
	for _, origin := range []string{"https://app.example/path", "https://app.example?", "https://user@app.example", "https://app.example#"} {
		client.RedirectURIs, client.AllowOrigins = nil, []string{origin}
		if s.isValidLogoutCallbackURL(client, "https://app.example/logout") {
			t.Fatalf("malformed allowed origin %q accepted", origin)
		}
	}
	for _, pair := range [][2]string{{"http://APP.example:80/a", "http://app.example/b"}, {"https://[0:0:0:0:0:0:0:1]:443/a", "https://[::1]/b"}} {
		a, _ := url.Parse(pair[0])
		b, _ := url.Parse(pair[1])
		if !sameURLOrigin(a, b) {
			t.Fatalf("equivalent origins rejected: %v", pair)
		}
	}
	v4, _ := url.Parse("https://127.0.0.1/logout")
	mapped, _ := url.Parse("https://[::ffff:127.0.0.1]/logout")
	if sameURLOrigin(v4, mapped) {
		t.Fatal("IPv4 and IPv4-mapped IPv6 are distinct browser origins")
	}
}

// TestBrowserLogoutOriginSerialization exercises default ports, invalid Origin syntax, and hook side effects. TestBrowserLogoutOriginSerialization 验证默认端口、非法 Origin 语法及钩子副作用。
func TestBrowserLogoutOriginSerialization(t *testing.T) {
	app := NewClientApp(ClientConfig{ClientID: "app", ServerURL: "https://center.example"})
	browser, err := NewBrowserClient(app, BrowserClientOptions{Storage: NewMemoryStorage(), CallbackURL: "https://APP.example:443/sso/callback", StateCookie: CookieOptions{SecretKey: "state-key"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		origin  string
		allowed bool
	}{
		{"https://app.example", true}, {"https://app.example:443", true}, {"https://APP.example", true},
		{"https://app.example:8443", false}, {"https://app.example/", false}, {"https://app.example?", false},
		{"https://app.example#", false}, {"https://user@app.example", false}, {"null", false},
		{"https://app.example https://other.example", false}, {"http://app.example", false},
	} {
		calls := 0
		handler := browser.LogoutHandler(func(http.ResponseWriter, *http.Request) error { calls++; return nil }, false, "/")
		r := httptest.NewRequest(http.MethodPost, "https://app.example/logout", nil)
		r.Header.Set("Origin", tc.origin)
		w := httptest.NewRecorder()
		handler(w, r)
		if (w.Code == http.StatusSeeOther) != tc.allowed || (calls == 1) != tc.allowed {
			t.Fatalf("origin %q: status=%d calls=%d", tc.origin, w.Code, calls)
		}
	}
}

// TestBrowserConfigurationRejectsUnusableCallbacks checks invalid ports, decoded controls, and unsupported SameSite values. TestBrowserConfigurationRejectsUnusableCallbacks 检查非法端口、解码控制字符及不支持的 SameSite 值。
func TestBrowserConfigurationRejectsUnusableCallbacks(t *testing.T) {
	app := NewClientApp(ClientConfig{ClientID: "app", ServerURL: "https://center.example"})
	for _, target := range []string{"https://app.example:0/callback", "https://app.example:65536/callback", "https://app.example:/callback", "https://app.example/call%0aback"} {
		_, err := NewBrowserClient(app, BrowserClientOptions{Storage: NewMemoryStorage(), CallbackURL: target, StateCookie: CookieOptions{SecretKey: "key"}})
		if !errors.Is(err, ErrBrowserConfig) {
			t.Fatalf("callback config = %v", err)
		}
	}
	for _, mode := range []http.SameSite{-1, 99} {
		_, err := NewBrowserClient(app, BrowserClientOptions{Storage: NewMemoryStorage(), CallbackURL: "https://app.example/callback", StateCookie: CookieOptions{SecretKey: "key", SameSite: mode}})
		if !errors.Is(err, ErrBrowserConfig) {
			t.Fatalf("SameSite config = %v", err)
		}
	}
}
