package consumer_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/Zany2/dtoken-go/sso"
	"github.com/Zany2/dtoken-go/sso/httpserver"
)

// ssoConsumerTransport exercises public HTTP handlers without opening a listening port. ssoConsumerTransport 通过公开 HTTP 处理器验证协议，不监听端口。
type ssoConsumerTransport struct {
	handler http.Handler
}

// RoundTrip delivers a request to the configured center handler. RoundTrip 将请求交给配置的中心处理器。
func (transport ssoConsumerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	w := httptest.NewRecorder()
	transport.handler.ServeHTTP(w, r)
	return w.Result(), nil
}

// TestSSOConsumerLifecycle verifies public APIs from a module outside the checkout. TestSSOConsumerLifecycle 从仓库外模块验证公开 API。
func TestSSOConsumerLifecycle(t *testing.T) {
	for _, mode := range []sso.Mode{sso.ModeTicket, sso.ModeOAuth2, sso.ModeSharedToken, sso.ModeRemoteSession} {
		t.Run(string(mode), func(t *testing.T) {
			ctx := context.Background()
			server := sso.NewServer()
			t.Cleanup(func() { _ = server.Close() })
			oneTime := mode == sso.ModeTicket || mode == sso.ModeOAuth2
			registration := sso.Client{
				ClientID: "consumer", ClientSecret: "consumer-exchange-key", SecretKey: "consumer-signing-key",
				RedirectURIs: []string{"https://app.example/sso/callback"}, Modes: []sso.Mode{mode},
				Scopes: []string{"profile"}, RequirePKCE: oneTime,
			}
			protection, err := sso.NewReplayProtection(sso.ReplayProtectionOptions{Storage: sso.NewMemoryStorage()})
			if err != nil {
				t.Fatal(err)
			}
			options := sso.DefaultHTTPOptions()
			options.Clients = map[string]sso.Client{registration.ClientID: registration}
			options.ReplayProtection = protection
			options.Params.Client = "application"
			options.Params.Ticket = "proof"
			options.LoginIDResolver = func(*http.Request) (string, bool) { return "alice", true }
			center, err := httpserver.NewChecked(server, options)
			if err != nil {
				t.Fatal(err)
			}
			config := sso.DefaultClientConfig()
			config.Mode, config.ClientID = mode, registration.ClientID
			config.ClientSecret, config.SecretKey = registration.ClientSecret, registration.SecretKey
			config.ServerURL, config.Params = "https://center.example", options.Params
			config.ReplayProtection = protection
			config.HTTPClient = &http.Client{Transport: ssoConsumerTransport{handler: center.Handler()}}
			app, err := sso.NewClientAppChecked(config)
			if err != nil {
				t.Fatal(err)
			}
			var target, verifier string
			extra := url.Values{"scope": {"profile"}}
			if oneTime {
				verifier = sso.NewPKCEVerifier()
				target, err = app.AuthURLWithPKCE(registration.RedirectURIs[0], verifier, extra)
			} else {
				target, err = app.AuthURL(registration.RedirectURIs[0], extra)
			}
			if err != nil {
				t.Fatal(err)
			}
			names := app.Config().Params
			tampered, err := url.Parse(target)
			if err != nil {
				t.Fatal(err)
			}
			query := tampered.Query()
			query.Set(names.Sign, "invalid-signature")
			tampered.RawQuery = query.Encode()
			rejected := httptest.NewRecorder()
			center.Handler().ServeHTTP(rejected, httptest.NewRequest(http.MethodGet, tampered.String(), nil))
			if rejected.Code != http.StatusUnauthorized {
				t.Fatalf("invalid signature status = %d", rejected.Code)
			}
			w := httptest.NewRecorder()
			center.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, target, nil))
			if w.Code != http.StatusFound {
				t.Fatalf("authorize status = %d", w.Code)
			}
			callback, err := url.Parse(w.Header().Get("Location"))
			if err != nil {
				t.Fatal(err)
			}
			replayed := httptest.NewRecorder()
			center.Handler().ServeHTTP(replayed, httptest.NewRequest(http.MethodGet, target, nil))
			if replayed.Code != http.StatusUnauthorized {
				t.Fatalf("authorization replay status = %d", replayed.Code)
			}
			values := callback.Query()
			req := sso.CredentialRequest{
				Mode: mode, RedirectURI: registration.RedirectURIs[0], CodeVerifier: verifier,
				Ticket: values.Get(names.Ticket), Code: values.Get(names.Code),
				TokenValue: values.Get(names.TokenValue), SessionID: values.Get(names.SessionID),
			}
			info, err := app.UserInfo(ctx, req)
			if err != nil || info == nil || !info.Active || info.LoginID != "alice" || info.ClientID != registration.ClientID || info.Mode != mode {
				t.Fatalf("userinfo = %+v, %v", info, err)
			}
			if oneTime {
				missingProof := req
				missingProof.CodeVerifier = ""
				if _, err := app.ExchangeCredential(ctx, missingProof); err == nil {
					t.Fatal("PKCE credential accepted without proof")
				}
				result, err := app.ExchangeCredential(ctx, req)
				if err != nil || result == nil || result.LoginID != "alice" {
					t.Fatalf("exchange = %+v, %v", result, err)
				}
				if _, err := app.ExchangeCredential(ctx, req); err == nil {
					t.Fatal("consumed credential accepted")
				}
			} else if err := app.Revoke(ctx, req); err != nil {
				t.Fatal(err)
			}
			info, err = app.Introspect(ctx, req)
			if err != nil || info == nil || info.Active {
				t.Fatalf("consumed/revoked credential = %+v, %v", info, err)
			}
		})
	}
}
