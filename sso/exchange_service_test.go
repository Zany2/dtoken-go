package sso

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// directExchangeTestClient replaces only the legacy local transport while retaining shared storage. directExchangeTestClient 仅替换旧本地传输，保留共享存储。
func directExchangeTestClient(t *testing.T, mode Mode, basic bool) (*Server, *ClientApp, *Client) {
	t.Helper()
	issuer, legacy, client := sharedStoreTestClient(t, mode, basic)
	local := legacy.config.SharedStoreServer
	service, err := NewExchangeService(local.server, local.exchangeCore(nil).options)
	if err != nil {
		t.Fatal(err)
	}
	config := legacy.config
	config.SharedStoreServer, config.ExchangeService = nil, service
	app, err := NewClientAppChecked(config)
	if err != nil {
		t.Fatal(err)
	}
	return issuer, app, client
}

// TestExchangeServiceLifecycle preserves proof validation, policy, enrichment and one-time consumption. TestExchangeServiceLifecycle 保留证明校验、策略、补充及一次性消费。
func TestExchangeServiceLifecycle(t *testing.T) {
	for _, mode := range []Mode{ModeTicket, ModeOAuth2} {
		for _, basic := range []bool{false, true} {
			t.Run(string(mode)+map[bool]string{false: "/atomic", true: "/ordinary"}[basic], func(t *testing.T) {
				issuer, app, client := directExchangeTestClient(t, mode, basic)
				req := sharedStoreCredential(t, issuer, app, client)
				service := app.config.ExchangeService
				allowed, calls := false, 0
				service.options.AuthorizationPolicy = func(_ context.Context, request AuthorizationRequest) (AuthorizationDecision, error) {
					if request.LoginID != "alice" || request.ClientID != client.ClientID || request.Mode != mode {
						t.Fatalf("untrusted policy input: %+v", request)
					}
					return AuthorizationDecision{Allowed: allowed, Scopes: []string{"profile"}}, nil
				}
				type contextKey struct{}
				ctx := context.WithValue(context.Background(), contextKey{}, "application-context")
				service.options.EnrichExchangeResult = func(ctx context.Context, id string, result *TicketExchangeResult) error {
					calls++
					if ctx.Value(contextKey{}) != "application-context" || id != client.ClientID || len(result.Scopes) != 1 {
						t.Fatal("lost context, client, or reduced grant")
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
					{"proof", func(_ *ClientConfig, r *CredentialRequest) { r.CodeVerifier = NewPKCEVerifier() }, ErrInvalidPKCE},
					{"redirect", func(_ *ClientConfig, r *CredentialRequest) { r.RedirectURI = "https://other.example/callback" }, ErrRedirectURIMismatch},
					{"callback", func(c *ClientConfig, _ *CredentialRequest) { c.LogoutCallbackURL = "https://other.example/logout" }, ErrInvalidCallbackURL},
				} {
					cfg, bad := app.config, req
					scenario.change(&cfg, &bad)
					if _, err := NewClientApp(cfg).ExchangeCredential(ctx, bad); !errors.Is(err, scenario.want) {
						t.Fatalf("%s: %v", scenario.name, err)
					}
				}
				if _, err := app.ExchangeCredential(ctx, req); !errors.Is(err, ErrAuthorizationDenied) {
					t.Fatalf("policy denial: %v", err)
				}
				if calls != 0 {
					t.Fatal("rejected request reached enrichment")
				}
				allowed = true
				result, err := app.ExchangeCredential(ctx, req)
				if err != nil || result.LoginID != "alice" || result.RemainTokenTimeout != 60 || calls != 1 {
					t.Fatalf("exchange: %+v, %v", result, err)
				}
				bindings, err := issuer.GetClientSessions(ctx, "alice")
				if err != nil || len(bindings) != 1 || bindings[0].ClientID != client.ClientID {
					t.Fatalf("registration: %+v, %v", bindings, err)
				}
				if _, err := app.ExchangeCredential(ctx, req); err == nil || calls != 1 {
					t.Fatal("credential consumed more than once")
				}
			})
		}
	}
}

// TestExchangeServiceBoundaries rejects ambiguous, oversized and cancelled input before consumption. TestExchangeServiceBoundaries 在消费前拒绝歧义、超限及取消输入。
func TestExchangeServiceBoundaries(t *testing.T) {
	issuer, app, client := directExchangeTestClient(t, ModeTicket, false)
	req := sharedStoreCredential(t, issuer, app, client)
	service := app.config.ExchangeService
	values, err := app.signRequest(app.credentialValues(req), ActionToken)
	if err != nil {
		t.Fatal(err)
	}
	duplicate := cloneValues(values)
	duplicate.Add(app.config.Params.Client, client.ClientID)
	if _, err := service.Exchange(nil, duplicate); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("duplicate: %v", err)
	}
	service.options.RequestLimits.MaxBodyBytes = 1
	if _, err := service.Exchange(nil, values); !errors.Is(err, ErrRequestTooLarge) {
		t.Fatalf("limit: %v", err)
	}
	service.options.RequestLimits = DefaultRequestLimits()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.Exchange(ctx, values); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel: %v", err)
	}
	if _, err := service.Exchange(nil, values); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Exchange(nil, values); !errors.Is(err, ErrRequestReplayed) {
		t.Fatalf("nonce: %v", err)
	}
	var missing *ExchangeService
	if _, err := missing.Exchange(nil, nil); !errors.Is(err, ErrServerNotInitialized) {
		t.Fatal(err)
	}
}

// TestExchangeServiceHTTPParity rejects invalid proof in both channels and consumes the same valid grant. TestExchangeServiceHTTPParity 验证两种通道均拒绝无效证明并消费相同授权。
func TestExchangeServiceHTTPParity(t *testing.T) {
	issuer, app, client := directExchangeTestClient(t, ModeTicket, false)
	req := sharedStoreCredential(t, issuer, app, client)
	options := DefaultHTTPOptions()
	options.SecretKey = client.SecretKey
	options.ReplayProtection = app.config.ExchangeService.options.ReplayProtection
	options.ExchangeEnricher = func(_ context.Context, _ string, result *TicketExchangeResult) error {
		result.RemainSessionTimeout = 120
		return nil
	}
	handler, err := NewHTTPServerChecked(issuer, options)
	if err != nil {
		t.Fatal(err)
	}
	bad := req
	bad.CodeVerifier = NewPKCEVerifier()
	if _, err := app.ExchangeCredential(nil, bad); !errors.Is(err, ErrInvalidPKCE) {
		t.Fatal(err)
	}
	for _, input := range []CredentialRequest{bad, req} {
		values, err := app.signRequest(app.credentialValues(input), ActionToken)
		if err != nil {
			t.Fatal(err)
		}
		r := httptest.NewRequest(http.MethodPost, "/sso/token", strings.NewReader(values.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		result, err := handler.exchangeRequest(r)
		if input.CodeVerifier == bad.CodeVerifier {
			if !errors.Is(err, ErrInvalidPKCE) {
				t.Fatalf("HTTP proof: %v", err)
			}
		} else if err != nil || result.LoginID != "alice" || result.RemainSessionTimeout != 120 {
			t.Fatalf("HTTP exchange: %+v, %v", result, err)
		}
	}
	if _, err := app.ExchangeCredential(nil, req); err == nil {
		t.Fatal("HTTP consumption was invisible to local exchange")
	}
}

// TestExchangeServiceHookFailure keeps consumed credentials consumed and prevents identity substitution. TestExchangeServiceHookFailure 保持失败钩子后的凭证已消费状态并阻止身份替换。
func TestExchangeServiceHookFailure(t *testing.T) {
	issuer, app, client := directExchangeTestClient(t, ModeTicket, false)
	req := sharedStoreCredential(t, issuer, app, client)
	app.config.ExchangeService.options.EnrichExchangeResult = func(_ context.Context, _ string, result *TicketExchangeResult) error {
		result.LoginID = "mallory"
		return nil
	}
	if _, err := app.ExchangeCredential(nil, req); err == nil {
		t.Fatal("identity substitution accepted")
	}
	app.config.ExchangeService.options.EnrichExchangeResult = nil
	if _, err := app.ExchangeCredential(nil, req); err == nil {
		t.Fatal("consumed ticket restored")
	}
	bindings, err := issuer.GetClientSessions(context.Background(), "alice")
	if err != nil || len(bindings) != 0 {
		t.Fatalf("failed grant registered: %+v, %v", bindings, err)
	}
}

// TestExchangeServiceConfiguration rejects conflicting routes and hook ownership before use. TestExchangeServiceConfiguration 在使用前拒绝路由及钩子职责冲突。
func TestExchangeServiceConfiguration(t *testing.T) {
	issuer, app, _ := directExchangeTestClient(t, ModeTicket, false)
	for _, change := range []func(*ClientConfig){
		func(c *ClientConfig) { c.UseHTTPCheck = true },
		func(c *ClientConfig) { c.SharedStoreServer = NewHTTPServer(issuer, DefaultHTTPOptions()) },
		func(c *ClientConfig) { c.Params.Ticket = "different" },
		func(c *ClientConfig) { c.Mode = ModeRemoteSession },
		func(c *ClientConfig) { c.ExchangeService = &ExchangeService{} },
	} {
		cfg := app.config
		change(&cfg)
		if _, err := NewClientAppChecked(cfg); !errors.Is(err, ErrSSOConfig) {
			t.Fatalf("configuration: %v", err)
		}
	}
	options := DefaultHTTPOptions()
	options.SecretKey = "key"
	options.ExchangeEnricher = func(context.Context, string, *TicketExchangeResult) error { return nil }
	options.EnrichExchangeResult = func(*http.Request, string, *TicketExchangeResult) error { return nil }
	if _, err := NewHTTPServerChecked(issuer, options); !errors.Is(err, ErrSSOConfig) {
		t.Fatal(err)
	}
	if err := NewHTTPServer(issuer, options).InitializationError(); !errors.Is(err, ErrSSOConfig) {
		t.Fatal(err)
	}
	if _, err := NewExchangeService(nil, DefaultExchangeOptions()); !errors.Is(err, ErrServerNotInitialized) {
		t.Fatal(err)
	}
	invalid := DefaultExchangeOptions()
	invalid.Params.Ticket = invalid.Params.Client
	if _, err := NewExchangeService(issuer, invalid); !errors.Is(err, ErrSSOConfig) {
		t.Fatal(err)
	}
}
