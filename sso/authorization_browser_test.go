package sso

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"testing"
)

// TestAuthorizationPolicyBrowserFlow combines policy, PKCE, signing, replay protection, and reduced local identity grants. TestAuthorizationPolicyBrowserFlow 组合策略、PKCE、签名、防重放与本地身份的缩减授权。
func TestAuthorizationPolicyBrowserFlow(t *testing.T) {
	for _, mode := range []Mode{ModeTicket, ModeOAuth2} {
		t.Run(string(mode), func(t *testing.T) {
			s := NewServer()
			defer s.Close()
			client := newTestClient()
			client.Modes, client.RequirePKCE, client.SecretKey = []Mode{mode}, true, "signing-key"
			if err := s.RegisterClient(client); err != nil {
				t.Fatal(err)
			}
			params := normalizeParamNames(ParamNames{Client: "application", Back: "state", CodeVerifier: "proof"})
			var phases []AuthorizationPhase
			h := NewHTTPServer(s, HTTPOptions{
				ServerOptions:    ServerOptions{CheckSign: true, Params: params},
				ReplayProtection: newReplayTestProtection(t, nil),
				LoginIDResolver:  func(*http.Request) (string, bool) { return "alice", true },
				AuthorizationPolicy: func(_ context.Context, request AuthorizationRequest) (AuthorizationDecision, error) {
					phases = append(phases, request.Phase)
					if request.LoginID != "alice" {
						t.Fatal("browser-controlled identity reached policy")
					}
					if request.Phase == AuthorizationIssue {
						return AuthorizationDecision{Allowed: true, Scopes: []string{"profile"}}, nil
					}
					if !slices.Equal(request.Scopes, []string{"profile"}) {
						t.Fatal("exchange policy lost issued grant")
					}
					return AuthorizationDecision{Allowed: true, Scopes: []string{}}, nil
				},
			})
			app := NewClientApp(ClientConfig{ClientID: client.ClientID, ClientSecret: client.ClientSecret, Mode: mode, ServerURL: "https://center.example", CheckSign: true, SecretKey: client.SecretKey, Params: params, ReplayProtection: newReplayTestProtection(t, nil), HTTPClient: &http.Client{Transport: logoutReviewTransport(func(r *http.Request) (*http.Response, error) {
				out := httptest.NewRecorder()
				h.Handler().ServeHTTP(out, r)
				return out.Result(), nil
			})}})
			memory := NewMemoryStorage()
			browser, err := NewBrowserClient(app, BrowserClientOptions{PKCE: true, Storage: memory, CallbackURL: client.RedirectURIs[0], StateCookie: CookieOptions{SecretKey: "cookie-key"}})
			if err != nil {
				t.Fatal(err)
			}
			start := httptest.NewRecorder()
			if err := browser.BeginLogin(start, httptest.NewRequest("GET", "/login", nil), "/profile", url.Values{params.Scope: {"profile email"}, params.LoginID: {"forged"}}); err != nil {
				t.Fatal(err)
			}
			authURL := start.Header().Get("Location")
			tampered, err := url.Parse(authURL)
			if err != nil {
				t.Fatal(err)
			}
			badQuery := tampered.Query()
			badQuery.Set(params.Scope, "admin")
			tampered.RawQuery = badQuery.Encode()
			bad := httptest.NewRecorder()
			h.HandleAuthorize(bad, httptest.NewRequest("GET", tampered.String(), nil))
			if bad.Code != 401 || len(phases) != 0 {
				t.Fatal("bad signature reached policy")
			}
			center := httptest.NewRecorder()
			h.HandleAuthorize(center, httptest.NewRequest("GET", authURL, nil))
			if center.Code != 302 {
				t.Fatalf("center: %d %s", center.Code, center.Body.String())
			}
			repeated := httptest.NewRecorder()
			h.HandleAuthorize(repeated, httptest.NewRequest("GET", authURL, nil))
			if repeated.Code != 401 || len(phases) != 1 {
				t.Fatal("replayed authorization reached policy")
			}
			callback := httptest.NewRequest("GET", center.Header().Get("Location"), nil)
			callback.AddCookie(start.Result().Cookies()[0])
			result, err := browser.CompleteLogin(httptest.NewRecorder(), callback)
			if err != nil {
				t.Fatal(err)
			}
			if result.Identity.LoginID != "alice" || len(result.Identity.Scopes) != 0 || !slices.Equal(phases, []AuthorizationPhase{AuthorizationIssue, AuthorizationExchange}) {
				t.Fatalf("browser result: %+v phases=%v", result, phases)
			}
		})
	}
}

// TestAuthorizationPolicyCanceledExchange preserves the credential when a policy returns after cancellation. TestAuthorizationPolicyCanceledExchange 在策略取消后返回时保留凭证。
func TestAuthorizationPolicyCanceledExchange(t *testing.T) {
	s := NewServer()
	defer s.Close()
	client := newTestClient()
	if err := s.RegisterClient(client); err != nil {
		t.Fatal(err)
	}
	ticket, err := s.GenerateTicket(context.Background(), client.ClientID, "alice", client.RedirectURIs[0], nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h := NewHTTPServer(s, HTTPOptions{AuthorizationPolicy: func(context.Context, AuthorizationRequest) (AuthorizationDecision, error) {
		cancel()
		return AuthorizationDecision{Allowed: true}, nil
	}})
	values := url.Values{"ticket": {ticket.Ticket}, "client": {client.ClientID}, "clientSecret": {client.ClientSecret}, "redirect": {client.RedirectURIs[0]}}
	_, err = h.exchangeCredential(httptest.NewRequest("POST", "/sso/token", nil).WithContext(ctx), values)
	if !errors.Is(err, context.Canceled) || !errors.Is(err, ErrAuthorizationPolicyUnavailable) {
		t.Fatalf("canceled policy: %v", err)
	}
	if _, err := s.ConsumeTicket(context.Background(), ticket.Ticket, client.ClientID, client.ClientSecret, client.RedirectURIs[0]); err != nil {
		t.Fatalf("canceled policy consumed credential: %v", err)
	}
}
