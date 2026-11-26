package consumer_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/sso"
	"github.com/Zany2/dtoken-go/sso/httpserver"
)

// TestSSOConsumerBrowserAndAccountLogout verifies browser, rotation, limits and account logout through public APIs only. TestSSOConsumerBrowserAndAccountLogout 仅通过公开 API 验证浏览器、密钥轮换、大小限制及账号下线。
func TestSSOConsumerBrowserAndAccountLogout(t *testing.T) {
	server := sso.NewServer()
	t.Cleanup(func() { _ = server.Close() })
	protection, err := sso.NewReplayProtection(sso.ReplayProtectionOptions{Storage: sso.NewMemoryStorage()})
	if err != nil {
		t.Fatal(err)
	}
	const clientID = "browser-consumer"
	const callbackURL = "https://app.example/sso/callback"
	var callback http.Handler
	centerLogouts, applicationLogouts := 0, 0
	localActive := false
	options := httpserver.DefaultOptions()
	options.RequestLimits = httpserver.DefaultRequestLimits()
	options.Clients = map[string]sso.Client{clientID: {
		ClientSecret: "exchange-key", SecretKey: "new-signing-key", VerificationKey: "old-signing-key",
		RedirectURIs: []string{callbackURL}, AllowOrigins: []string{"https://app.example"},
		Modes: []sso.Mode{sso.ModeTicket, sso.ModeSharedToken}, RequirePKCE: true,
	}}
	options.ReplayProtection = protection
	options.LoginIDResolver = func(*http.Request) (string, bool) { return "alice", true }
	options.EnrichExchangeResult = func(_ *http.Request, _ string, result *sso.TicketExchangeResult) error {
		result.RemainSessionTimeout = 60
		return nil
	}
	options.CenterLogout = func(_ *http.Request, loginID string) error {
		if loginID != "alice" {
			return errors.New("unexpected center logout subject")
		}
		centerLogouts++
		return nil
	}
	options.LogoutHTTPClient = &http.Client{Transport: ssoConsumerTransport{handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if callback == nil {
			http.Error(w, "logout callback is not wired", http.StatusInternalServerError)
			return
		}
		callback.ServeHTTP(w, r)
	})}}
	center, err := httpserver.NewChecked(server, options)
	if err != nil {
		t.Fatal(err)
	}
	config := sso.DefaultClientConfig()
	config.ClientID, config.ClientSecret = clientID, "exchange-key"
	config.SecretKey, config.VerificationKey = "old-signing-key", "new-signing-key"
	config.ServerURL = "https://center.example"
	config.RegisterCallback, config.LogoutCallbackURL = true, "https://app.example/sso/logout"
	config.ReplayProtection = protection
	config.RequestLimits = sso.RequestLimits{MaxQueryBytes: 256, MaxBodyBytes: 1024}
	config.HTTPClient = &http.Client{Transport: ssoConsumerTransport{handler: center.Handler()}}
	app, err := sso.NewClientAppChecked(config)
	if err != nil {
		t.Fatal(err)
	}
	callback = app.LogoutCallbackHandler(func(_ *http.Request, result sso.LogoutCallback) error {
		if result.LoginID != "alice" || result.ClientID != clientID {
			return errors.New("invalid logout identity")
		}
		applicationLogouts++
		localActive = false
		return nil
	})
	browser, err := sso.NewBrowserClient(app, sso.BrowserClientOptions{
		PKCE: true, Storage: sso.NewMemoryStorage(), CallbackURL: callbackURL,
		StateCookie: sso.CookieOptions{SecretKey: "browser-cookie-key"}, SessionMaxAge: time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	start := httptest.NewRecorder()
	if err := browser.BeginLogin(start, httptest.NewRequest(http.MethodGet, "https://app.example/login", nil), "/orders", nil); err != nil {
		t.Fatal(err)
	}
	issued := httptest.NewRecorder()
	center.Handler().ServeHTTP(issued, httptest.NewRequest(http.MethodGet, start.Header().Get("Location"), nil))
	if issued.Code != http.StatusFound || len(start.Result().Cookies()) != 1 {
		t.Fatalf("authorization status %d", issued.Code)
	}
	request := httptest.NewRequest(http.MethodGet, issued.Header().Get("Location"), nil)
	request.AddCookie(start.Result().Cookies()[0])
	oversized := request.Clone(request.Context())
	oversized.URL.RawQuery += "&extension=" + strings.Repeat("a", 256)
	if result, err := browser.CompleteLogin(httptest.NewRecorder(), oversized); result != nil || !errors.Is(err, sso.ErrRequestTooLarge) {
		t.Fatalf("oversized callback: result=%v err=%v", result, err)
	}
	login := browser.CallbackHandler(func(_ http.ResponseWriter, _ *http.Request, result *sso.BrowserLoginResult) error {
		if result.Identity.LoginID != "alice" || result.ReturnTo != "/orders" || !result.ExpiresAt.After(time.Now()) || time.Until(result.ExpiresAt) > time.Minute {
			t.Fatal("invalid browser login result")
		}
		localActive = true
		return nil
	})
	response := httptest.NewRecorder()
	login(response, request)
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/orders" || !localActive {
		t.Fatalf("login failed: %d", response.Code)
	}
	if _, err := browser.CompleteLogin(httptest.NewRecorder(), request); !errors.Is(err, sso.ErrInvalidLoginState) {
		t.Fatalf("consumed state accepted: %v", err)
	}
	credential, err := server.GenerateSharedToken(context.Background(), clientID, "alice", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	result, err := center.LogoutAccount(httptest.NewRequest(http.MethodPost, "/trusted-account-logout", nil), "alice")
	if err != nil || result == nil || !result.Complete || centerLogouts != 1 || applicationLogouts != 1 || localActive {
		t.Fatalf("account logout: result=%+v err=%v center=%d application=%d", result, err, centerLogouts, applicationLogouts)
	}
	if _, err := server.ValidateSharedToken(context.Background(), credential.Token, clientID); !errors.Is(err, sso.ErrInvalidSharedToken) {
		t.Fatalf("account credential remained valid: %v", err)
	}
}
