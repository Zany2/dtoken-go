package sso

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// TestPKCEBrowserFlowIsolation rejects exchanged credentials from another tab while preserving that tab's legitimate flow. TestPKCEBrowserFlowIsolation 拒绝其他标签页的凭证，同时保留该标签页的合法流程。
func TestPKCEBrowserFlowIsolation(t *testing.T) {
	for _, mode := range []Mode{ModeTicket, ModeOAuth2} {
		t.Run(string(mode), func(t *testing.T) {
			s := NewServer()
			defer s.Close()
			client := newTestClient()
			client.Modes, client.RequirePKCE = []Mode{mode}, true
			if err := s.RegisterClient(client); err != nil {
				t.Fatal(err)
			}
			h := NewHTTPServer(s, HTTPOptions{LoginIDResolver: func(*http.Request) (string, bool) { return "alice", true }})
			app := NewClientApp(ClientConfig{Mode: mode, ClientID: client.ClientID, ClientSecret: client.ClientSecret, ServerURL: "https://center.example",
				HTTPClient: &http.Client{Transport: logoutReviewTransport(func(r *http.Request) (*http.Response, error) {
					out := httptest.NewRecorder()
					h.HandleToken(out, r)
					return out.Result(), nil
				})},
			})
			memory := NewMemoryStorage()
			b, err := NewBrowserClient(app, BrowserClientOptions{PKCE: true, Storage: memory, CallbackURL: client.RedirectURIs[0], StateCookie: CookieOptions{SecretKey: "cookie-key"}})
			if err != nil {
				t.Fatal(err)
			}
			var callbacks []*url.URL
			var cookies []*http.Cookie
			var proofs []string
			for range 2 {
				auth, cookie := beginBrowserTestLogin(t, b, "/orders")
				cookies = append(cookies, cookie)
				raw, err := memory.Get(context.Background(), b.options.KeyPrefix+auth.Query().Get("back"))
				if err != nil {
					t.Fatal(err)
				}
				var state browserLoginState
				if err := json.Unmarshal([]byte(raw.(string)), &state); err != nil {
					t.Fatal(err)
				}
				proofs = append(proofs, state.CodeVerifier)
				challenge, err := PKCEChallenge(state.CodeVerifier)
				if err != nil || challenge != auth.Query().Get("code_challenge") || strings.Contains(auth.String(), state.CodeVerifier) || strings.Contains(cookie.Value, state.CodeVerifier) {
					t.Fatal("flow proof was missing or exposed to the browser")
				}
				out := httptest.NewRecorder()
				h.HandleAuthorize(out, httptest.NewRequest("GET", auth.String(), nil))
				if out.Code != http.StatusFound {
					t.Fatalf("authorization: %d %s", out.Code, out.Body.String())
				}
				callback, err := url.Parse(out.Header().Get("Location"))
				if err != nil {
					t.Fatal(err)
				}
				callbacks = append(callbacks, callback)
			}
			if proofs[0] == proofs[1] {
				t.Fatal("parallel flows reused proof")
			}
			mixed := *callbacks[0]
			query := mixed.Query()
			key := map[Mode]string{ModeTicket: "ticket", ModeOAuth2: "code"}[mode]
			query.Set(key, callbacks[1].Query().Get(key))
			mixed.RawQuery = query.Encode()
			request := httptest.NewRequest("GET", mixed.String(), nil)
			request.AddCookie(cookies[0])
			if _, err := b.CompleteLogin(httptest.NewRecorder(), request); !errors.Is(err, ErrBrowserExchange) {
				t.Fatalf("accepted cross-flow credential: %v", err)
			}
			request = httptest.NewRequest("GET", callbacks[1].String(), nil)
			request.AddCookie(cookies[1])
			result, err := b.CompleteLogin(httptest.NewRecorder(), request)
			if err != nil || result.Identity.LoginID != "alice" {
				t.Fatalf("legitimate flow failed after interception: %v", err)
			}
			request = httptest.NewRequest("GET", callbacks[0].String(), nil)
			request.AddCookie(cookies[0])
			if _, err := b.CompleteLogin(httptest.NewRecorder(), request); !errors.Is(err, ErrInvalidLoginState) {
				t.Fatalf("failed exchange reused consumed browser state: %v", err)
			}
		})
	}
}

// TestPKCEBrowserRejectsStateDowngrade requires a new flow when enabling PKCE and disallows caller-owned proof parameters. TestPKCEBrowserRejectsStateDowngrade 启用 PKCE 后要求新流程，并禁止调用方注入证明参数。
func TestPKCEBrowserRejectsStateDowngrade(t *testing.T) {
	b := newBrowserTestClient(t, nil, nil)
	auth, cookie := beginBrowserTestLogin(t, b, "/")
	b.options.PKCE = true
	if _, err := b.CompleteLogin(httptest.NewRecorder(), browserTestCallback(b, auth, cookie)); !errors.Is(err, ErrInvalidLoginState) {
		t.Fatalf("unbound pending flow accepted after enabling PKCE: %v", err)
	}
	for _, key := range []string{"code_challenge", "code_challenge_method", "code_verifier"} {
		out := httptest.NewRecorder()
		if err := b.BeginLogin(out, httptest.NewRequest("GET", "/login", nil), "/", url.Values{key: {"injected"}}); !errors.Is(err, ErrInvalidPKCE) || out.Header().Get("Location") != "" {
			t.Fatalf("caller proof injection: %v", err)
		}
	}
	for _, mode := range []Mode{ModeSharedToken, ModeRemoteSession} {
		app := NewClientApp(ClientConfig{ClientID: "app", ServerURL: "https://center.example", Mode: mode})
		if _, err := NewBrowserClient(app, b.options); !errors.Is(err, ErrModeUnsupported) {
			t.Fatalf("unsupported PKCE mode %s: %v", mode, err)
		}
	}
}
