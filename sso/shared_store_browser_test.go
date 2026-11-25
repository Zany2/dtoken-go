package sso

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"
)

// TestSharedStoreBrowserSSO reuses one center login across two origins and revokes both application sessions. TestSharedStoreBrowserSSO 跨两个来源复用一次中心登录并撤销两个应用会话。
func TestSharedStoreBrowserSSO(t *testing.T) {
	store := NewMemoryStorage()
	center := NewServer(WithStorage(store))
	defer center.Close()
	options := DefaultHTTPOptions()
	options.SecretKey, options.LoginPageURL = "center-signing-key", "https://center.example/login"
	options.Cookie.SecretKey = "center-cookie-key"
	options.ReplayProtection = newReplayTestProtection(t, store)
	centerLoggedIn := true
	readCenterCookie := LoginIDFromCookie(options.Cookie)
	options.LoginIDResolver = func(r *http.Request) (string, bool) {
		id, ok := readCenterCookie(r)
		return id, ok && centerLoggedIn
	}
	options.CenterLogout = func(_ *http.Request, id string) error {
		if id != "alice" {
			return ErrClientMismatch
		}
		centerLoggedIn = false
		return nil
	}
	options.EnrichExchangeResult = func(_ *http.Request, _ string, result *TicketExchangeResult) error {
		result.RemainSessionTimeout = 60
		return nil
	}
	handlers := make(map[string]http.HandlerFunc)
	var mu sync.Mutex
	localSessions := make(map[string]string)
	options.LogoutHTTPClient = &http.Client{Transport: logoutReviewTransport(func(r *http.Request) (*http.Response, error) {
		handler := handlers[r.URL.Host]
		if handler == nil || r.URL.Path != "/logout-callback" {
			return nil, errors.New("unexpected logout destination")
		}
		out := httptest.NewRecorder()
		handler(out, r)
		return out.Result(), nil
	})}
	h, err := NewHTTPServerChecked(center, options)
	if err != nil {
		t.Fatal(err)
	}
	loginCookie := httptest.NewRecorder()
	SetLoginIDCookie(loginCookie, options.Cookie, "alice")
	centerCookie := loginCookie.Result().Cookies()[0]
	var firstApp *ClientApp
	var pending CredentialRequest
	for _, origin := range []string{"https://alpha.example.org", "https://beta.example.net"} {
		u, _ := url.Parse(origin)
		client := &Client{ClientID: u.Host, ClientSecret: "exchange-secret", SecretKey: "client-signing-key", RedirectURIs: []string{origin + "/sso/callback"}, Modes: []Mode{ModeTicket}, RequirePKCE: true}
		if err := center.RegisterClientChecked(client); err != nil {
			t.Fatal(err)
		}
		peer := NewServer(WithStorage(store))
		t.Cleanup(func() { _ = peer.Close() })
		local, err := NewHTTPServerChecked(peer, options)
		if err != nil {
			t.Fatal(err)
		}
		config := DefaultClientConfig()
		config.ClientID, config.ClientSecret, config.SecretKey = client.ClientID, client.ClientSecret, client.SecretKey
		config.ServerURL, config.SharedStoreServer = "https://center.example", local
		config.ReplayProtection = newReplayTestProtection(t, NewMemoryStorage())
		config.RegisterCallback, config.LogoutCallbackURL = true, origin+"/logout-callback"
		config.HTTPClient = &http.Client{Transport: logoutReviewTransport(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("HTTP exchange is unavailable")
		})}
		app, err := NewClientAppChecked(config)
		if err != nil {
			t.Fatal(err)
		}
		handlers[u.Host] = app.LogoutCallbackHandler(func(_ *http.Request, callback LogoutCallback) error {
			mu.Lock()
			defer mu.Unlock()
			if callback.ClientID != client.ClientID || callback.LoginID != "alice" {
				return ErrClientMismatch
			}
			delete(localSessions, client.ClientID)
			return nil
		})
		browser, err := NewBrowserClient(app, BrowserClientOptions{PKCE: true, Storage: NewMemoryStorage(), CallbackURL: client.RedirectURIs[0], StateCookie: CookieOptions{SecretKey: "local-state-key"}})
		if err != nil {
			t.Fatal(err)
		}
		authorize, cookie := beginBrowserTestLogin(t, browser, "/orders")
		request := httptest.NewRequest(http.MethodGet, authorize.String(), nil)
		request.AddCookie(centerCookie)
		out := httptest.NewRecorder()
		h.HandleAuthorize(out, request)
		if out.Code != http.StatusFound {
			t.Fatalf("center authorization: %d %s", out.Code, out.Body.String())
		}
		callbackURL := out.Header().Get("Location")
		callback := httptest.NewRequest(http.MethodGet, callbackURL, nil)
		callback.AddCookie(cookie)
		out = httptest.NewRecorder()
		browser.CallbackHandler(func(_ http.ResponseWriter, _ *http.Request, result *BrowserLoginResult) error {
			if result.Identity.LoginID != "alice" || result.ReturnTo != "/orders" || time.Until(result.ExpiresAt) > time.Minute {
				return ErrInvalidRequest
			}
			mu.Lock()
			defer mu.Unlock()
			localSessions[client.ClientID] = result.Identity.LoginID
			return nil
		})(out, callback)
		if out.Code != http.StatusSeeOther || out.Header().Get("Location") != "/orders" {
			t.Fatalf("application login: %d %s", out.Code, out.Body.String())
		}
		callback = httptest.NewRequest(http.MethodGet, callbackURL, nil)
		callback.AddCookie(cookie)
		if _, err := browser.CompleteLogin(httptest.NewRecorder(), callback); !errors.Is(err, ErrInvalidLoginState) {
			t.Fatalf("callback replay: %v", err)
		}
		if firstApp == nil {
			firstApp = app
			pending = sharedStoreCredential(t, center, app, client)
		}
	}
	bindings, err := center.GetClientSessions(context.Background(), "alice")
	if err != nil || len(bindings) != 2 || len(localSessions) != 2 {
		t.Fatalf("missing application sessions: %+v, %v", bindings, err)
	}
	target, err := firstApp.SignoutURL("", nil)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodGet, target, nil)
	r.AddCookie(centerCookie)
	out := httptest.NewRecorder()
	h.HandleLogout(out, r)
	if out.Code != http.StatusOK || centerLoggedIn || len(localSessions) != 0 {
		t.Fatalf("global logout: %d %s", out.Code, out.Body.String())
	}
	bindings, err = center.GetClientSessions(context.Background(), "alice")
	if err != nil || len(bindings) != 0 {
		t.Fatalf("bindings survived logout: %+v, %v", bindings, err)
	}
	if _, err := firstApp.ExchangeCredential(context.Background(), pending); err == nil {
		t.Fatal("pending ticket survived global logout")
	}
	authorize, err := firstApp.AuthURL("https://alpha.example.org/sso/callback", nil)
	if err != nil {
		t.Fatal(err)
	}
	r = httptest.NewRequest(http.MethodGet, authorize, nil)
	r.AddCookie(centerCookie)
	out = httptest.NewRecorder()
	h.HandleAuthorize(out, r)
	u, err := url.Parse(out.Header().Get("Location"))
	if err != nil || out.Code != http.StatusFound || u.Host != "center.example" || u.Path != "/login" {
		t.Fatal("old center session still authorized after logout")
	}
}
