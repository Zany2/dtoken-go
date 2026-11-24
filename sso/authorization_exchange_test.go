package sso

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/Zany2/dtoken-go/contract"
)

// TestAuthorizationExchangePolicy checks proof-before-policy ordering, non-consuming failures, reduced grants, and callback registration. TestAuthorizationExchangePolicy 检查证明先于策略、失败不消费、缩减授权与回调登记。
func TestAuthorizationExchangePolicy(t *testing.T) {
	for _, mode := range []Mode{ModeTicket, ModeOAuth2} {
		for _, basic := range []bool{false, true} {
			t.Run(string(mode)+map[bool]string{false: "/atomic", true: "/ordinary"}[basic], func(t *testing.T) {
				memory := NewMemoryStorage()
				var storage contract.Storage = memory
				if basic {
					storage = &basicSSOStorage{inner: memory}
				}
				s := NewServer(WithStorage(storage))
				client := newTestClient()
				client.Modes, client.RequirePKCE = []Mode{mode}, true
				if err := s.RegisterClient(client); err != nil {
					t.Fatal(err)
				}
				verifier := NewPKCEVerifier()
				challenge, _ := PKCEChallenge(verifier)
				options := OneTimeCredentialOptions{CodeChallenge: challenge, CodeChallengeMethod: PKCEMethodS256}
				key, value := "ticket", ""
				if mode == ModeTicket {
					ticket, err := s.GenerateTicketWithOptions(context.Background(), client.ClientID, "alice", client.RedirectURIs[0], []string{"profile", "email"}, nil, options)
					if err != nil {
						t.Fatal(err)
					}
					value = ticket.Ticket
				} else {
					code, err := s.GenerateOAuth2CodeWithOptions(context.Background(), client.ClientID, "alice", client.RedirectURIs[0], []string{"profile", "email"}, nil, options)
					if err != nil {
						t.Fatal(err)
					}
					key, value = "code", code.Code
				}
				values := url.Values{key: {value}, "mode": {string(mode)}, "client": {client.ClientID}, "clientSecret": {client.ClientSecret}, "redirect": {client.RedirectURIs[0]}, "code_verifier": {verifier}, "callback": {"https://app.example.com/logout"}, "loginId": {"forged"}, "scope": {"admin"}}
				calls, enriched := 0, 0
				decision := AuthorizationDecision{}
				var policyErr error
				h := NewHTTPServer(s, HTTPOptions{
					ServerOptions: ServerOptions{EnableSLO: true},
					AuthorizationPolicy: func(_ context.Context, request AuthorizationRequest) (AuthorizationDecision, error) {
						calls++
						if request.Phase != AuthorizationExchange || request.LoginID != "alice" || request.ClientID != client.ClientID || request.Mode != mode || !slices.Equal(request.Scopes, []string{"profile", "email"}) {
							t.Fatalf("policy input: %+v", request)
						}
						return decision, policyErr
					},
					EnrichExchangeResult: func(_ *http.Request, _ string, result *TicketExchangeResult) error {
						enriched++
						if !slices.Equal(result.Scopes, []string{"profile"}) {
							t.Fatal("enrichment saw unreduced scopes")
						}
						return nil
					},
				})
				for _, invalid := range []struct{ key, value string }{{"clientSecret", "wrong"}, {"redirect", "https://other.example/callback"}, {"code_verifier", NewPKCEVerifier()}, {"callback", "https://other.example/logout"}} {
					bad := cloneValues(values)
					bad.Set(invalid.key, invalid.value)
					if out := protocolReviewPost(h.HandleToken, bad); out.Code == 200 || calls != 0 {
						t.Fatalf("unvalidated request reached policy: %d %s", out.Code, out.Body.String())
					}
				}
				for _, scenario := range []struct {
					decision AuthorizationDecision
					err      error
					code     int
				}{
					{AuthorizationDecision{}, nil, 403},
					{AuthorizationDecision{Allowed: true}, errors.New("private-access-store-secret"), 503},
					{AuthorizationDecision{Allowed: true}, ErrAuthorizationDenied, 503},
					{AuthorizationDecision{Allowed: true, Scopes: []string{"admin"}}, nil, 500},
				} {
					decision, policyErr = scenario.decision, scenario.err
					out := protocolReviewPost(h.HandleToken, values)
					if out.Code != scenario.code || enriched != 0 || strings.Contains(out.Body.String(), "private-access-store-secret") {
						t.Fatalf("failure: %d %s", out.Code, out.Body.String())
					}
					bindings, err := s.GetClientSessions(context.Background(), "alice")
					if err != nil || len(bindings) != 0 {
						t.Fatal("failed decision registered callback")
					}
				}
				decision, policyErr = AuthorizationDecision{Allowed: true, Scopes: []string{"profile"}}, nil
				out := protocolReviewPost(h.HandleToken, values)
				if out.Code != 200 || enriched != 1 {
					t.Fatalf("credential was consumed on failure: %d %s", out.Code, out.Body.String())
				}
				var response struct {
					Data TicketExchangeResult `json:"data"`
				}
				if err := json.Unmarshal(out.Body.Bytes(), &response); err != nil {
					t.Fatal(err)
				}
				if response.Data.LoginID != "alice" || !slices.Equal(response.Data.Scopes, []string{"profile"}) {
					t.Fatalf("unexpected grant: %+v", response.Data)
				}
				bindings, err := s.GetClientSessions(context.Background(), "alice")
				if err != nil || len(bindings) != 1 {
					t.Fatal("successful exchange did not register callback")
				}
				before := calls
				if out := protocolReviewPost(h.HandleToken, values); out.Code == 200 || calls != before {
					t.Fatal("consumed credential evaluated policy")
				}
				if err := s.RevokeLoginCredentials(context.Background(), "alice"); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

// TestAuthorizationExchangeWithoutSLO enforces policy even when no logout callback is requested. TestAuthorizationExchangeWithoutSLO 在不请求注销回调时仍执行授权策略。
func TestAuthorizationExchangeWithoutSLO(t *testing.T) {
	s := NewServer()
	defer s.Close()
	client := newTestClient()
	if err := s.RegisterClient(client); err != nil {
		t.Fatal(err)
	}
	ticket, err := s.GenerateTicket(context.Background(), client.ClientID, "alice", client.RedirectURIs[0], []string{"profile"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	allowed := false
	h := NewHTTPServer(s, HTTPOptions{AuthorizationPolicy: func(context.Context, AuthorizationRequest) (AuthorizationDecision, error) {
		return AuthorizationDecision{Allowed: allowed, Scopes: []string{}}, nil
	}})
	values := url.Values{"ticket": {ticket.Ticket}, "client": {client.ClientID}, "clientSecret": {client.ClientSecret}, "redirect": {client.RedirectURIs[0]}}
	if out := protocolReviewPost(h.HandleToken, values); out.Code != 403 {
		t.Fatalf("policy bypass without callback: %d", out.Code)
	}
	allowed = true
	out := protocolReviewPost(h.HandleToken, values)
	if out.Code != 200 || strings.Contains(out.Body.String(), "profile") {
		t.Fatalf("empty grant not retained: %d %s", out.Code, out.Body.String())
	}
}
