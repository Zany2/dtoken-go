package sso

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"
)

// TestAuthorizationDecisionBounds checks deny-by-default, immutable inputs, subset grants, and safe failures. TestAuthorizationDecisionBounds 检查默认拒绝、输入隔离、子集授权与安全失败。
func TestAuthorizationDecisionBounds(t *testing.T) {
	for _, test := range []struct {
		name     string
		decision AuthorizationDecision
		want     []string
		err      error
	}{
		{"zero denies", AuthorizationDecision{}, nil, ErrAuthorizationDenied},
		{"nil preserves", AuthorizationDecision{Allowed: true}, []string{"profile", "email"}, nil},
		{"empty clears", AuthorizationDecision{Allowed: true, Scopes: []string{}}, []string{}, nil},
		{"subset", AuthorizationDecision{Allowed: true, Scopes: []string{"email"}}, []string{"email"}, nil},
		{"ordered deduplication", AuthorizationDecision{Allowed: true, Scopes: []string{"email", "profile", "email"}}, []string{"profile", "email"}, nil},
		{"expansion", AuthorizationDecision{Allowed: true, Scopes: []string{"admin"}}, nil, ErrAuthorizationDecision},
		{"case", AuthorizationDecision{Allowed: true, Scopes: []string{"Profile"}}, nil, ErrAuthorizationDecision},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := AuthorizationRequest{Scopes: []string{"profile", "email"}}
			h := &HTTPServer{options: HTTPOptions{AuthorizationPolicy: func(_ context.Context, request AuthorizationRequest) (AuthorizationDecision, error) {
				request.Scopes[0] = "admin"
				return test.decision, nil
			}}}
			got, err := h.evaluateAuthorization(context.Background(), input)
			if !errors.Is(err, test.err) || !slices.Equal(got, test.want) || input.Scopes[0] != "profile" {
				t.Fatalf("decision: %v %v input=%v", got, err, input.Scopes)
			}
			if len(test.decision.Scopes) > 0 {
				test.decision.Scopes[0] = "changed-after-return"
				if slices.Contains(got, "changed-after-return") {
					t.Fatal("output aliases policy memory")
				}
			}
		})
	}
	private := errors.New("private-policy-store-secret")
	h := &HTTPServer{options: HTTPOptions{AuthorizationPolicy: func(context.Context, AuthorizationRequest) (AuthorizationDecision, error) {
		return AuthorizationDecision{Allowed: true}, private
	}}}
	_, err := h.evaluateAuthorization(context.Background(), AuthorizationRequest{})
	if !errors.Is(err, ErrAuthorizationPolicyUnavailable) || !errors.Is(err, private) || strings.Contains(err.Error(), private.Error()) || statusFromError(err) != 503 {
		t.Fatalf("unsafe policy failure: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	h.options.AuthorizationPolicy = func(context.Context, AuthorizationRequest) (AuthorizationDecision, error) {
		cancel()
		return AuthorizationDecision{Allowed: true}, nil
	}
	if _, err := h.evaluateAuthorization(ctx, AuthorizationRequest{}); !errors.Is(err, context.Canceled) || !errors.Is(err, ErrAuthorizationPolicyUnavailable) {
		t.Fatalf("late success accepted: %v", err)
	}
	h.options.AuthorizationPolicy = func(context.Context, AuthorizationRequest) (AuthorizationDecision, error) {
		t.Fatal("already canceled request evaluated policy")
		return AuthorizationDecision{}, nil
	}
	_, _ = h.evaluateAuthorization(ctx, AuthorizationRequest{})
}

// TestAuthorizationPolicyIssueAndInspect exercises every mode with reduced issuance and live inspection decisions. TestAuthorizationPolicyIssueAndInspect 验证所有模式的缩减签发及动态检查决策。
func TestAuthorizationPolicyIssueAndInspect(t *testing.T) {
	for _, mode := range []Mode{ModeTicket, ModeOAuth2, ModeSharedToken, ModeRemoteSession} {
		t.Run(string(mode), func(t *testing.T) {
			s := NewServer()
			defer s.Close()
			client := newTestClient()
			client.Modes = []Mode{mode}
			if err := s.RegisterClient(client); err != nil {
				t.Fatal(err)
			}
			allow, issueCalls, inspectCalls, extraCalls := false, 0, 0, 0
			var failure error
			var inspectScopes []string
			h := NewHTTPServer(s, HTTPOptions{
				ServerOptions:      ServerOptions{EnableSLO: true},
				LoginIDResolver:    func(*http.Request) (string, bool) { return "alice", true },
				AuthorizationExtra: func(*http.Request, string, string) (map[string]any, error) { extraCalls++; return nil, nil },
				AuthorizationPolicy: func(_ context.Context, request AuthorizationRequest) (AuthorizationDecision, error) {
					if request.LoginID != "alice" || request.ClientID != client.ClientID || request.Mode != mode {
						t.Fatalf("untrusted policy input: %+v", request)
					}
					if request.Phase == AuthorizationIssue {
						issueCalls++
						if !slices.Equal(request.Scopes, []string{"profile", "email"}) {
							t.Fatalf("issue scopes: %v", request.Scopes)
						}
						return AuthorizationDecision{Allowed: allow, Scopes: []string{"profile"}}, failure
					}
					inspectCalls++
					if request.Phase != AuthorizationInspect || !slices.Equal(request.Scopes, []string{"profile"}) {
						t.Fatalf("inspect scopes: %+v", request)
					}
					return AuthorizationDecision{Allowed: allow, Scopes: inspectScopes}, failure
				},
			})
			values := url.Values{"client": {client.ClientID}, "redirect": {client.RedirectURIs[0]}, "mode": {string(mode)}, "scope": {"profile email"}, "loginId": {"forged"}, "callback": {"https://app.example.com/logout"}}
			issue := func() *httptest.ResponseRecorder {
				out := httptest.NewRecorder()
				h.HandleAuthorize(out, httptest.NewRequest("GET", "/sso/authorize?"+values.Encode(), nil))
				return out
			}
			if out := issue(); out.Code != 403 || out.Header().Get("Location") != "" || extraCalls != 0 {
				t.Fatalf("denied issuance: %d %s", out.Code, out.Body.String())
			}
			bindings, err := s.GetClientSessions(context.Background(), "alice")
			if err != nil || len(bindings) != 0 {
				t.Fatal("denial registered logout callback")
			}
			allow = true
			out := issue()
			if out.Code != 302 || issueCalls != 2 || extraCalls != 1 {
				t.Fatalf("issuance: %d %s", out.Code, out.Body.String())
			}
			location, err := url.Parse(out.Header().Get("Location"))
			if err != nil {
				t.Fatal(err)
			}
			inspect := location.Query()
			inspect.Set("mode", string(mode))
			inspect.Set("client", client.ClientID)
			inspect.Set("clientSecret", client.ClientSecret)
			inspect.Set("scope", "admin")
			inspect.Set("loginId", "forged")
			for _, test := range []struct {
				allowed bool
				scopes  []string
				failure error
				code    int
				active  bool
			}{
				{true, nil, nil, 200, true},
				{true, []string{}, nil, 200, true},
				{false, nil, nil, 200, false},
				{true, nil, errors.New("private-tenant-store-secret"), 503, false},
				{true, []string{"email"}, nil, 500, false},
				{true, nil, nil, 200, true},
			} {
				allow, inspectScopes, failure = test.allowed, test.scopes, test.failure
				result := protocolReviewPost(h.HandleIntrospect, inspect)
				if result.Code != test.code || strings.Contains(result.Body.String(), "private-tenant-store-secret") {
					t.Fatalf("inspection: %d %s", result.Code, result.Body.String())
				}
				if result.Code == 200 {
					var response struct {
						Data CredentialInfo `json:"data"`
					}
					if err := json.Unmarshal(result.Body.Bytes(), &response); err != nil {
						t.Fatal(err)
					}
					if response.Data.Active != test.active {
						t.Fatal("wrong active status")
					}
					if !test.active && (response.Data.LoginID != "" || len(response.Data.Scopes) != 0) {
						t.Fatal("denied inspection leaked identity")
					}
					if test.active && test.scopes != nil && len(response.Data.Scopes) != 0 {
						t.Fatal("explicit empty grant was ignored")
					}
				}
			}
			allow, failure = false, nil
			if out := protocolReviewPost(h.HandleUserInfo, inspect); out.Code != 401 || strings.Contains(out.Body.String(), "alice") {
				t.Fatal("userinfo bypassed policy")
			}
			before := inspectCalls
			h.options.CheckCredentialActive = func(*http.Request, CredentialInfo) (bool, error) { return false, nil }
			allow = true
			if out := protocolReviewPost(h.HandleIntrospect, inspect); out.Code != 200 || inspectCalls != before {
				t.Fatal("inactive application state reached policy")
			}
		})
	}
}
