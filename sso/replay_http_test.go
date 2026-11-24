package sso

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// replayTestRequest sends a captured form through the real HTTP handlers. replayTestRequest 将捕获的表单送入真实 HTTP 处理器。
func replayTestRequest(handler http.Handler, method, path string, values url.Values) *httptest.ResponseRecorder {
	var r *http.Request
	if method == http.MethodGet {
		r = httptest.NewRequest(method, path+"?"+values.Encode(), nil)
	} else {
		r = httptest.NewRequest(method, path, strings.NewReader(values.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	return w
}

// TestReplayProtocolLifecycle covers all credential modes, custom protocol names, login bounce, and protected routes. TestReplayProtocolLifecycle 覆盖全部凭证模式、自定义协议名、登录往返与受保护路由。
func TestReplayProtocolLifecycle(t *testing.T) {
	for _, mode := range []Mode{ModeTicket, ModeOAuth2, ModeSharedToken, ModeRemoteSession} {
		t.Run(string(mode), func(t *testing.T) {
			s := NewServer()
			defer s.Close()
			registered := newTestClient()
			registered.SecretKey = "dedicated-key"
			registered.Modes = []Mode{mode}
			if err := s.RegisterClient(registered); err != nil {
				t.Fatal(err)
			}
			params := normalizeParamNames(ParamNames{Action: "operation", Nonce: "requestID", Timestamp: "issuedAt", Client: "application", Sign: "signature"})
			paths := DefaultEndpoints().ReplacePrefix("/custom")
			loggedIn, hookCalls, logoutCalls := false, 0, 0
			h := NewHTTPServer(s, HTTPOptions{
				ServerOptions:    ServerOptions{CheckSign: true, SecretKey: "global-key", Params: params, Endpoints: paths},
				ReplayProtection: newReplayTestProtection(t, nil), LoginPageURL: "/login",
				LoginIDResolver:    func(*http.Request) (string, bool) { return "alice", loggedIn },
				AuthorizationExtra: func(*http.Request, string, string) (map[string]any, error) { hookCalls++; return nil, nil },
				CenterLogout:       func(*http.Request, string) error { logoutCalls++; return nil },
			})
			mux := h.Handler()
			captured := make(map[string]url.Values)
			app := NewClientApp(ClientConfig{
				Mode: mode, ClientID: registered.ClientID, ClientSecret: registered.ClientSecret,
				ServerURL: "https://center.example", CheckSign: true, SecretKey: registered.SecretKey,
				Params: params, Endpoints: paths, ReplayProtection: newReplayTestProtection(t, nil),
				HTTPClient: &http.Client{Transport: logoutReviewTransport(func(r *http.Request) (*http.Response, error) {
					if err := r.ParseForm(); err != nil {
						return nil, err
					}
					captured[r.URL.Path] = cloneValues(r.Form)
					w := httptest.NewRecorder()
					mux.ServeHTTP(w, r)
					return w.Result(), nil
				})},
			})
			issue := func(loginBounce bool) CredentialRequest {
				t.Helper()
				target, err := app.AuthURL(registered.RedirectURIs[0], url.Values{params.Action: {"untrusted"}, params.Nonce: {"untrusted"}})
				if err != nil {
					t.Fatal(err)
				}
				u, err := url.Parse(target)
				if err != nil {
					t.Fatal(err)
				}
				if loginBounce {
					w := replayTestRequest(mux, http.MethodGet, paths.Authorize, u.Query())
					if w.Code != http.StatusFound || !strings.HasPrefix(w.Header().Get("Location"), "/login?") || hookCalls != 0 {
						t.Fatalf("login bounce: %d %s", w.Code, w.Body.String())
					}
					loggedIn = true
				}
				w := replayTestRequest(mux, http.MethodGet, paths.Authorize, u.Query())
				if w.Code != http.StatusFound {
					t.Fatalf("authorize: %d %s", w.Code, w.Body.String())
				}
				before := hookCalls
				if replay := replayTestRequest(mux, http.MethodGet, paths.Authorize, u.Query()); replay.Code != http.StatusUnauthorized || hookCalls != before {
					t.Fatalf("authorization replay: %d %s", replay.Code, replay.Body.String())
				}
				location, err := url.Parse(w.Header().Get("Location"))
				if err != nil {
					t.Fatal(err)
				}
				query := location.Query()
				return CredentialRequest{Mode: mode, Ticket: query.Get(params.Ticket), Code: query.Get(params.Code), TokenValue: query.Get(params.TokenValue), SessionID: query.Get(params.SessionID), RedirectURI: registered.RedirectURIs[0]}
			}
			req := issue(true)
			info, err := app.Introspect(context.Background(), req)
			if err != nil || !info.Active {
				t.Fatalf("introspect: %+v %v", info, err)
			}
			if replay := replayTestRequest(mux, http.MethodPost, paths.Introspect, captured[paths.Introspect]); replay.Code != http.StatusUnauthorized {
				t.Fatalf("introspection replay: %d", replay.Code)
			}
			if crossed := replayTestRequest(mux, http.MethodPost, paths.Revoke, captured[paths.Introspect]); crossed.Code != http.StatusBadRequest {
				t.Fatalf("cross-operation replay: %d", crossed.Code)
			}
			if info, err := app.UserInfo(context.Background(), req); err != nil || info.LoginID != "alice" {
				t.Fatalf("userinfo: %+v %v", info, err)
			}
			if replay := replayTestRequest(mux, http.MethodPost, paths.UserInfo, captured[paths.UserInfo]); replay.Code != http.StatusUnauthorized {
				t.Fatalf("userinfo replay: %d", replay.Code)
			}
			if mode == ModeTicket || mode == ModeOAuth2 {
				if result, err := app.ExchangeCredential(context.Background(), req); err != nil || result.LoginID != "alice" {
					t.Fatalf("exchange: %+v %v", result, err)
				}
				if replay := replayTestRequest(mux, http.MethodPost, paths.Token, captured[paths.Token]); replay.Code != http.StatusUnauthorized {
					t.Fatalf("exchange replay: %d", replay.Code)
				}
				req = issue(false)
			}
			if err := app.Revoke(context.Background(), req); err != nil {
				t.Fatal(err)
			}
			if replay := replayTestRequest(mux, http.MethodPost, paths.Revoke, captured[paths.Revoke]); replay.Code != http.StatusUnauthorized {
				t.Fatalf("revoke replay: %d", replay.Code)
			}
			if info, err := app.Introspect(context.Background(), req); err != nil || info.Active {
				t.Fatalf("revoked: %+v %v", info, err)
			}
			target, err := app.SignoutURL("alice", nil)
			if err != nil {
				t.Fatal(err)
			}
			u, err := url.Parse(target)
			if err != nil {
				t.Fatal(err)
			}
			if w := replayTestRequest(mux, http.MethodGet, paths.Logout, u.Query()); w.Code != http.StatusOK {
				t.Fatalf("logout: %d %s", w.Code, w.Body.String())
			}
			if w := replayTestRequest(mux, http.MethodGet, paths.Logout, u.Query()); w.Code != http.StatusUnauthorized || logoutCalls != 1 {
				t.Fatalf("logout replay: %d calls=%d", w.Code, logoutCalls)
			}
		})
	}
}

// TestReplayVerificationOrder ensures invalid signatures and actions cannot reserve a valid nonce. TestReplayVerificationOrder 确保无效签名与操作不能占用有效随机值。
func TestReplayVerificationOrder(t *testing.T) {
	s := NewServer()
	defer s.Close()
	p := newReplayTestProtection(t, nil)
	client := newTestClient()
	if err := s.RegisterClient(client); err != nil {
		t.Fatal(err)
	}
	h := NewHTTPServer(s, HTTPOptions{ReplayProtection: p, ServerOptions: ServerOptions{CheckSign: true, SecretKey: "key"}})
	names := DefaultParamNames()
	values := p.stamp(url.Values{names.Client: {client.ClientID}}, names, ActionIntrospect)
	bad := NewSigner("wrong").AttachSign(values)
	if err := h.verifyRequest(context.Background(), bad, ActionIntrospect, true); !errors.Is(err, ErrInvalidSign) {
		t.Fatalf("signature: %v", err)
	}
	signed := NewSigner("key").AttachSign(values)
	if err := h.verifyRequest(context.Background(), signed, ActionRevoke, true); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("action: %v", err)
	}
	if err := h.verifyRequest(context.Background(), signed, ActionIntrospect, true); err != nil {
		t.Fatalf("poisoned nonce: %v", err)
	}
	values = p.stamp(values, names, ActionIntrospect)
	values.Set(names.Timestamp, time.Now().Add(-time.Hour).Format(time.RFC3339Nano))
	if w := replayTestRequest(h.Handler(), http.MethodPost, "/sso/introspect", NewSigner("key").AttachSign(values)); w.Code != http.StatusUnauthorized {
		t.Fatalf("expired request: %d", w.Code)
	}
}

// TestReplayBusinessFailureRetry requires fresh requests after a failed business operation. TestReplayBusinessFailureRetry 要求业务失败后使用新请求重试。
func TestReplayBusinessFailureRetry(t *testing.T) {
	s := NewServer()
	defer s.Close()
	client := newTestClient()
	if err := s.RegisterClient(client); err != nil {
		t.Fatal(err)
	}
	calls := 0
	h := NewHTTPServer(s, HTTPOptions{
		ServerOptions:    ServerOptions{CheckSign: true, SecretKey: "key"},
		ReplayProtection: newReplayTestProtection(t, nil),
		LoginIDResolver:  func(*http.Request) (string, bool) { return "alice", true },
		CenterLogout: func(*http.Request, string) error {
			calls++
			if calls == 1 {
				return errors.New("center session unavailable")
			}
			return nil
		},
	})
	app := NewClientApp(ClientConfig{ServerURL: "https://center.example", ClientID: client.ClientID, CheckSign: true, SecretKey: "key", ReplayProtection: newReplayTestProtection(t, nil)})
	makeURL := func() *url.URL {
		t.Helper()
		target, err := app.SignoutURL("alice", nil)
		if err != nil {
			t.Fatal(err)
		}
		u, err := url.Parse(target)
		if err != nil {
			t.Fatal(err)
		}
		return u
	}
	first := makeURL()
	if w := replayTestRequest(h.Handler(), http.MethodGet, first.Path, first.Query()); w.Code != http.StatusInternalServerError || calls != 1 {
		t.Fatalf("business failure: %d calls=%d", w.Code, calls)
	}
	if w := replayTestRequest(h.Handler(), http.MethodGet, first.Path, first.Query()); w.Code != http.StatusUnauthorized || calls != 1 {
		t.Fatalf("business replay: %d calls=%d", w.Code, calls)
	}
	second := makeURL()
	if w := replayTestRequest(h.Handler(), http.MethodGet, second.Path, second.Query()); w.Code != http.StatusOK || calls != 2 {
		t.Fatalf("business retry: %d calls=%d", w.Code, calls)
	}
}

// TestReplayLogoutCallbackRetry checks one-time delivery, callback failure, fresh retries, and target binding. TestReplayLogoutCallbackRetry 检查一次性投递、回调失败、新请求重试与目标绑定。
func TestReplayLogoutCallbackRetry(t *testing.T) {
	s := NewServer()
	defer s.Close()
	client := newTestClient()
	client.SecretKey = "dedicated-key"
	if err := s.RegisterClient(client); err != nil {
		t.Fatal(err)
	}
	p := newReplayTestProtection(t, nil)
	app := NewClientApp(ClientConfig{ClientID: client.ClientID, CheckSign: true, SecretKey: client.SecretKey, ReplayProtection: p})
	var captured url.Values
	calls := 0
	callback := app.LogoutCallbackHandler(func(*http.Request, LogoutCallback) error {
		calls++
		if calls == 1 {
			return errors.New("temporary application failure")
		}
		return nil
	})
	h := NewHTTPServer(s, HTTPOptions{
		ReplayProtection: newReplayTestProtection(t, nil),
		ServerOptions: ServerOptions{CheckSign: true, SecretKey: "global-key", LogoutHTTPClient: &http.Client{Transport: logoutReviewTransport(func(r *http.Request) (*http.Response, error) {
			if err := r.ParseForm(); err != nil {
				return nil, err
			}
			captured = cloneValues(r.Form)
			w := httptest.NewRecorder()
			callback(w, r)
			return w.Result(), nil
		})}},
	})
	session := ClientSession{ClientID: client.ClientID, LoginID: "alice", LogoutCallbackURL: client.RedirectURIs[0]}
	r := httptest.NewRequest(http.MethodPost, "/sso/logout", nil)
	if err := h.postLogoutCallback(r, session); err == nil {
		t.Fatal("callback failure lost")
	}
	firstNonce := captured.Get("nonce")
	if w := replayTestRequest(callback, http.MethodPost, "/callback", captured); w.Code != http.StatusUnauthorized || calls != 1 {
		t.Fatalf("failed callback replay: %d calls=%d", w.Code, calls)
	}
	if err := h.postLogoutCallback(r, session); err != nil {
		t.Fatal(err)
	}
	if captured.Get("nonce") == firstNonce || calls != 2 {
		t.Fatal("retry did not use a fresh request")
	}
	if w := replayTestRequest(callback, http.MethodPost, "/callback", captured); w.Code != http.StatusUnauthorized || calls != 2 {
		t.Fatalf("callback replay: %d calls=%d", w.Code, calls)
	}
	values := p.stamp(url.Values{"client": {"wrong-client"}, "loginId": {"alice"}}, DefaultParamNames(), ActionLogoutCallback)
	if w := replayTestRequest(callback, http.MethodPost, "/callback", NewSigner(client.SecretKey).AttachSign(values)); w.Code != http.StatusBadRequest {
		t.Fatalf("target binding: %d", w.Code)
	}
	values.Set("client", client.ClientID)
	if w := replayTestRequest(callback, http.MethodPost, "/callback", NewSigner("wrong").AttachSign(values)); w.Code != http.StatusUnauthorized {
		t.Fatalf("callback signature: %d", w.Code)
	}
	if w := replayTestRequest(callback, http.MethodPost, "/callback", NewSigner(client.SecretKey).AttachSign(values)); w.Code != http.StatusOK || calls != 3 {
		t.Fatalf("callback nonce poisoned: %d calls=%d", w.Code, calls)
	}
}
