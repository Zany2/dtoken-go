package sso

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/contract"
)

// sharedStoreTestClient builds independent issuer and consumer instances over one store. sharedStoreTestClient 在同一存储上构建独立的签发和消费实例。
func sharedStoreTestClient(t *testing.T, mode Mode, basic bool) (*Server, *ClientApp, *Client) {
	t.Helper()
	memory := NewMemoryStorage()
	var store contract.Storage = memory
	if basic {
		store = &basicSSOStorage{inner: memory}
	}
	issuer := NewServer(WithStorage(store))
	peer := NewServer(WithStorage(store))
	t.Cleanup(func() { _ = issuer.Close(); _ = peer.Close() })
	client := newTestClient()
	client.Modes, client.RequirePKCE, client.SecretKey = []Mode{mode}, true, "request-signing-key"
	if err := issuer.RegisterClientChecked(client); err != nil {
		t.Fatal(err)
	}
	options := DefaultHTTPOptions()
	options.SecretKey = "request-signing-key"
	options.ReplayProtection = newReplayTestProtection(t, store)
	local, err := NewHTTPServerChecked(peer, options)
	if err != nil {
		t.Fatal(err)
	}
	config := DefaultClientConfig()
	config.Mode, config.ClientID, config.ClientSecret = mode, client.ClientID, client.ClientSecret
	config.SecretKey, config.ServerURL = client.SecretKey, "https://center.example"
	config.SharedStoreServer, config.ReplayProtection = local, newReplayTestProtection(t, store)
	config.RegisterCallback, config.LogoutCallbackURL = true, "https://app.example.com/logout"
	config.HTTPClient = &http.Client{Transport: logoutReviewTransport(func(*http.Request) (*http.Response, error) {
		t.Error("shared-store exchange attempted HTTP")
		return nil, errors.New("unexpected transport")
	})}
	app, err := NewClientAppChecked(config)
	if err != nil {
		t.Fatal(err)
	}
	return issuer, app, client
}

// sharedStoreCredential issues a proof-bound credential through the separate center instance. sharedStoreCredential 通过独立中心实例签发绑定证明的凭证。
func sharedStoreCredential(t *testing.T, issuer *Server, app *ClientApp, client *Client) CredentialRequest {
	t.Helper()
	req := CredentialRequest{Mode: app.config.Mode, RedirectURI: client.RedirectURIs[0], CodeVerifier: NewPKCEVerifier()}
	challenge, err := PKCEChallenge(req.CodeVerifier)
	if err != nil {
		t.Fatal(err)
	}
	options := OneTimeCredentialOptions{CodeChallenge: challenge, CodeChallengeMethod: PKCEMethodS256}
	if req.Mode == ModeTicket {
		credential, err := issuer.GenerateTicketWithOptions(context.Background(), client.ClientID, "alice", req.RedirectURI, []string{"profile", "email"}, nil, options)
		if err != nil {
			t.Fatal(err)
		}
		req.Ticket = credential.Ticket
	} else {
		credential, err := issuer.GenerateOAuth2CodeWithOptions(context.Background(), client.ClientID, "alice", req.RedirectURI, []string{"profile", "email"}, nil, options)
		if err != nil {
			t.Fatal(err)
		}
		req.Code = credential.Code
	}
	return req
}

// TestSharedStoreExchange validates rejection before consumption, shared policy, enrichment, and logout bindings. TestSharedStoreExchange 验证消费前拒绝、共享策略、返回补充及退出绑定。
func TestSharedStoreExchange(t *testing.T) {
	for _, mode := range []Mode{ModeTicket, ModeOAuth2} {
		for _, basic := range []bool{false, true} {
			t.Run(string(mode)+map[bool]string{false: "/atomic", true: "/ordinary"}[basic], func(t *testing.T) {
				issuer, app, client := sharedStoreTestClient(t, mode, basic)
				req := sharedStoreCredential(t, issuer, app, client)
				local := app.config.SharedStoreServer
				allowed, policyCalls, enrichCalls := false, 0, 0
				local.options.AuthorizationPolicy = func(_ context.Context, request AuthorizationRequest) (AuthorizationDecision, error) {
					policyCalls++
					if request.Phase != AuthorizationExchange || request.LoginID != "alice" || request.ClientID != client.ClientID || request.Mode != mode {
						t.Fatalf("untrusted policy input: %+v", request)
					}
					return AuthorizationDecision{Allowed: allowed, Scopes: []string{"profile"}}, nil
				}
				local.options.EnrichExchangeResult = func(r *http.Request, id string, result *TicketExchangeResult) error {
					enrichCalls++
					if r.Method != http.MethodPost || r.URL.RawQuery != "" || r.PostForm.Get("code_verifier") != req.CodeVerifier || id != client.ClientID || len(result.Scopes) != 1 {
						t.Fatal("local exchange did not preserve protocol hook inputs")
					}
					result.RemainTokenTimeout = 60
					return nil
				}
				for _, scenario := range []struct {
					name   string
					change func(*ClientConfig, *CredentialRequest)
					want   error
				}{
					{"secret", func(c *ClientConfig, _ *CredentialRequest) { c.ClientSecret = "wrong" }, ErrInvalidClientCredentials},
					{"signature", func(c *ClientConfig, _ *CredentialRequest) { c.SecretKey = "wrong" }, ErrInvalidSign},
					{"redirect", func(_ *ClientConfig, r *CredentialRequest) { r.RedirectURI = "https://other.example/callback" }, ErrRedirectURIMismatch},
					{"proof", func(_ *ClientConfig, r *CredentialRequest) { r.CodeVerifier = NewPKCEVerifier() }, ErrInvalidPKCE},
					{"callback", func(c *ClientConfig, _ *CredentialRequest) { c.LogoutCallbackURL = "https://other.example/logout" }, ErrInvalidCallbackURL},
					{"mixed credential", func(_ *ClientConfig, r *CredentialRequest) { r.TokenValue = "foreign" }, ErrInvalidRequest},
				} {
					config, bad := app.config, req
					scenario.change(&config, &bad)
					if _, err := NewClientApp(config).ExchangeCredential(context.Background(), bad); !errors.Is(err, scenario.want) {
						t.Fatalf("%s: %v", scenario.name, err)
					}
				}
				if policyCalls != 0 || enrichCalls != 0 {
					t.Fatal("invalid request reached policy or enrichment")
				}
				if _, err := app.ExchangeCredential(nil, req); !errors.Is(err, ErrAuthorizationDenied) {
					t.Fatalf("policy denial: %v", err)
				}
				allowed = true
				result, err := app.ExchangeCredential(context.Background(), req)
				if err != nil || result.LoginID != "alice" || len(result.Scopes) != 1 || result.Scopes[0] != "profile" || result.RemainTokenTimeout != 60 || enrichCalls != 1 {
					t.Fatalf("exchange = %+v, %v", result, err)
				}
				bindings, err := issuer.GetClientSessions(context.Background(), "alice")
				if err != nil || len(bindings) != 1 || bindings[0].ClientID != client.ClientID {
					t.Fatalf("center did not see shared logout binding: %+v, %v", bindings, err)
				}
				if _, err := app.ExchangeCredential(context.Background(), req); err == nil || enrichCalls != 1 {
					t.Fatal("credential was consumed more than once")
				}
			})
		}
	}
}

// TestSharedStoreRouting preserves legacy HTTP defaults and explicit HTTP override without fallback. TestSharedStoreRouting 保留旧 HTTP 默认值及显式 HTTP 覆盖，不在失败时切换路径。
func TestSharedStoreRouting(t *testing.T) {
	issuer, app, client := sharedStoreTestClient(t, ModeTicket, false)
	req := sharedStoreCredential(t, issuer, app, client)
	for _, withLocal := range []bool{false, true} {
		config := app.config
		config.UseHTTPCheck = withLocal
		if !withLocal {
			config.SharedStoreServer = nil
		}
		calls := 0
		config.HTTPClient = &http.Client{Transport: logoutReviewTransport(func(*http.Request) (*http.Response, error) {
			calls++
			return nil, errors.New("offline")
		})}
		if _, err := NewClientApp(config).ExchangeCredential(context.Background(), req); err == nil || calls != 1 {
			t.Fatalf("HTTP routing failed: %v, calls=%d", err, calls)
		}
	}
	if _, err := app.ExchangeCredential(context.Background(), req); err != nil {
		t.Fatalf("HTTP failure unexpectedly consumed shared credential: %v", err)
	}

	config := app.config
	config.Mode = ModeRemoteSession
	if _, err := NewClientAppChecked(config); !errors.Is(err, ErrModeUnsupported) {
		t.Fatalf("unsupported direct mode: %v", err)
	}
	config = app.config
	config.Params.Ticket = "custom-ticket"
	if _, err := NewClientAppChecked(config); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("mismatched parameter names: %v", err)
	}
	config = app.config
	config.SharedStoreServer = NewHTTPServer(nil, DefaultHTTPOptions())
	if _, err := NewClientAppChecked(config); !errors.Is(err, ErrServerNotInitialized) {
		t.Fatalf("uninitialized local server: %v", err)
	}
}

// TestSharedStoreFailureLifecycle checks limits, cancellation, capacity, and post-consumption hook failures. TestSharedStoreFailureLifecycle 检查限制、取消、容量和消费后钩子失败。
func TestSharedStoreFailureLifecycle(t *testing.T) {
	issuer, app, client := sharedStoreTestClient(t, ModeTicket, false)
	local := app.config.SharedStoreServer
	req := sharedStoreCredential(t, issuer, app, client)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := app.ExchangeCredential(ctx, req); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation: %v", err)
	}
	local.options.RequestLimits.MaxBodyBytes = 32
	if _, err := app.ExchangeCredential(context.Background(), req); !errors.Is(err, ErrRequestTooLarge) {
		t.Fatalf("form limit: %v", err)
	}
	local.options.RequestLimits = DefaultRequestLimits()
	other := *client
	other.ClientID = "other-app"
	if err := issuer.RegisterClient(&other); err != nil {
		t.Fatal(err)
	}
	if _, err := issuer.RegisterClientSession(context.Background(), "alice", other.ClientID, app.config.LogoutCallbackURL); err != nil {
		t.Fatal(err)
	}
	local.options.MaxRegisteredClient = 1
	if _, err := app.ExchangeCredential(context.Background(), req); !errors.Is(err, ErrClientSessionLimit) {
		t.Fatalf("session capacity: %v", err)
	}
	if err := issuer.ClearClientSessions(context.Background(), "alice"); err != nil {
		t.Fatal(err)
	}
	private := errors.New("private enrichment failure")
	local.options.EnrichExchangeResult = func(*http.Request, string, *TicketExchangeResult) error { return private }
	if _, err := app.ExchangeCredential(context.Background(), req); !errors.Is(err, private) {
		t.Fatalf("hook diagnosis: %v", err)
	}
	local.options.EnrichExchangeResult = nil
	if _, err := app.ExchangeCredential(context.Background(), req); err == nil {
		t.Fatal("post-consumption failure restored ticket")
	}
	bindings, err := issuer.GetClientSessions(context.Background(), "alice")
	if err != nil || len(bindings) != 0 {
		t.Fatalf("failed hook created binding: %+v, %v", bindings, err)
	}
	req = sharedStoreCredential(t, issuer, app, client)
	if err := issuer.RevokeLoginCredentials(context.Background(), "alice"); err != nil {
		t.Fatal(err)
	}
	if _, err := app.ExchangeCredential(context.Background(), req); err == nil {
		t.Fatal("shared consumer accepted credential revoked by center")
	}
}

// TestSharedStoreReplay keeps local exchange subject to signed request freshness and one-time nonce claims. TestSharedStoreReplay 保证本地兑换仍受签名请求新鲜度及一次性随机值约束。
func TestSharedStoreReplay(t *testing.T) {
	issuer, app, client := sharedStoreTestClient(t, ModeTicket, false)
	req := sharedStoreCredential(t, issuer, app, client)
	values, err := app.signRequest(app.credentialValues(req), ActionToken)
	if err != nil {
		t.Fatal(err)
	}
	local := app.config.SharedStoreServer
	local.options.AuthorizationPolicy = func(context.Context, AuthorizationRequest) (AuthorizationDecision, error) {
		return AuthorizationDecision{}, nil
	}
	request := func() *http.Request {
		r := httptest.NewRequest(http.MethodPost, "https://center.example/sso/token", strings.NewReader(values.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		return r
	}
	if _, err := local.exchangeRequest(request()); !errors.Is(err, ErrAuthorizationDenied) {
		t.Fatalf("first request: %v", err)
	}
	local.options.AuthorizationPolicy = nil
	if _, err := local.exchangeRequest(request()); !errors.Is(err, ErrRequestReplayed) {
		t.Fatalf("nonce reuse: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := app.ExchangeCredential(ctx, req); err != nil {
		t.Fatalf("fresh request failed after policy denial: %v", err)
	}
}
