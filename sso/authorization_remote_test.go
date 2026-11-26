package sso

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestAuthorizationRemoteLifecycle propagates current grants and denies to every protected request without rewriting stored credentials. TestAuthorizationRemoteLifecycle 将当前授权与拒绝同步到每个受保护请求且不重写存储凭证。
func TestAuthorizationRemoteLifecycle(t *testing.T) {
	for _, mode := range []Mode{ModeRemoteSession, ModeSharedToken} {
		t.Run(string(mode), func(t *testing.T) {
			s := NewServer()
			defer s.Close()
			client := newTestClient()
			client.Modes = []Mode{mode}
			if err := s.RegisterClient(client); err != nil {
				t.Fatal(err)
			}
			var credential string
			if mode == ModeRemoteSession {
				session, err := s.CreateRemoteSession(context.Background(), client.ClientID, "alice", []string{"profile", "email"}, nil)
				if err != nil {
					t.Fatal(err)
				}
				credential = session.SessionID
			} else {
				token, err := s.GenerateSharedToken(context.Background(), client.ClientID, "alice", []string{"profile", "email"}, nil)
				if err != nil {
					t.Fatal(err)
				}
				credential = token.Token
			}
			decision := AuthorizationDecision{Allowed: true}
			var failure error
			policyCalls := 0
			h := NewHTTPServer(s, HTTPOptions{AuthorizationPolicy: func(_ context.Context, request AuthorizationRequest) (AuthorizationDecision, error) {
				policyCalls++
				if request.Phase != AuthorizationInspect || request.LoginID != "alice" || len(request.Scopes) != 2 {
					t.Fatalf("policy saw modified stored grant: %+v", request)
				}
				return decision, failure
			}})
			app := NewClientApp(ClientConfig{Mode: mode, ClientID: client.ClientID, ClientSecret: client.ClientSecret, ServerURL: "https://center.example", HTTPClient: &http.Client{Transport: logoutReviewTransport(func(r *http.Request) (*http.Response, error) {
				out := httptest.NewRecorder()
				h.Handler().ServeHTTP(out, r)
				return out.Result(), nil
			})}})
			guard, err := NewRemoteAuth(app, RemoteAuthOptions{RequiredScopes: []string{"email"}})
			if err != nil {
				t.Fatal(err)
			}
			calls := 0
			protected := guard.Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls++ }))
			check := func(status int) {
				t.Helper()
				before := calls
				r := httptest.NewRequest("GET", "/profile", nil)
				r.Header.Set("Authorization", "Bearer "+credential)
				out := httptest.NewRecorder()
				protected.ServeHTTP(out, r)
				if out.Code != status || (status != 200 && calls != before) || strings.Contains(out.Body.String(), "private-policy-error") {
					t.Fatalf("remote decision: %d %s", out.Code, out.Body.String())
				}
			}
			check(200)
			decision.Scopes = []string{"profile"}
			check(403)
			decision = AuthorizationDecision{}
			check(401)
			failure = errors.New("private-policy-error")
			check(503)
			failure = nil
			decision = AuthorizationDecision{Allowed: true}
			check(200)
			before := policyCalls
			if err := s.RevokeLoginCredentials(context.Background(), "alice"); err != nil {
				t.Fatal(err)
			}
			check(401)
			if policyCalls != before {
				t.Fatal("policy reactivated revoked credential")
			}
		})
	}
}
