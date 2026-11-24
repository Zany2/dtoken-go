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

// TestRequestValidationScalarFields checks every configured field and retains multi-valued application extensions. TestRequestValidationScalarFields 检查全部配置字段并保留应用扩展参数的多值能力。
func TestRequestValidationScalarFields(t *testing.T) {
	for _, names := range []ParamNames{DefaultParamNames(), normalizeParamNames(ParamNames{Client: "application", Ticket: "credential", Scope: "grants", Sign: "signature"})} {
		for _, name := range protocolParamNames(names) {
			for _, items := range [][]string{nil, {}, {"one", "two"}, {"same", "same"}} {
				if err := validateScalarParameters(url.Values{name: items}, names); !errors.Is(err, ErrInvalidRequest) {
					t.Fatalf("field %s cardinality %d: %v", name, len(items), err)
				}
			}
			if err := validateScalarParameters(url.Values{name: {""}, "extension": {"one", "two"}}, names); err != nil {
				t.Fatalf("single field %s: %v", name, err)
			}
		}
	}
}

// TestRequestValidationCredentialModes rejects mismatched credentials without changing missing-credential semantics. TestRequestValidationCredentialModes 拒绝模式不匹配的凭证，不改变缺失凭证的原有语义。
func TestRequestValidationCredentialModes(t *testing.T) {
	names := normalizeParamNames(ParamNames{Ticket: "ticket_value", Code: "code_value", TokenValue: "token_value", SessionID: "session_value", Mode: "flow"})
	for _, action := range []string{ActionToken, ActionIntrospect, ActionUserInfo, ActionRevoke} {
		for _, mode := range []Mode{ModeTicket, ModeOAuth2, ModeSharedToken, ModeRemoteSession} {
			expected := map[Mode]string{ModeTicket: names.Ticket, ModeOAuth2: names.Code, ModeSharedToken: names.TokenValue, ModeRemoteSession: names.SessionID}[mode]
			valid := url.Values{names.Mode: {string(mode)}, expected: {"value"}}
			if action == ActionToken && (mode == ModeSharedToken || mode == ModeRemoteSession) {
				if !errors.Is(validateCredentialParameters(valid, names, action), ErrModeUnsupported) {
					t.Fatal("reusable mode accepted for exchange")
				}
				continue
			}
			if err := validateCredentialParameters(valid, names, action); err != nil {
				t.Fatal(err)
			}
			delete(valid, expected)
			if err := validateCredentialParameters(valid, names, action); err != nil {
				t.Fatal("missing credential no longer delegated to endpoint")
			}
			valid.Set(expected, "value")
			for _, foreign := range []string{names.Ticket, names.Code, names.TokenValue, names.SessionID} {
				if foreign == expected {
					continue
				}
				for _, value := range []string{"", "other"} {
					bad := cloneValues(valid)
					bad.Set(foreign, value)
					if !errors.Is(validateCredentialParameters(bad, names, action), ErrInvalidRequest) {
						t.Fatalf("mixed %s/%s accepted", action, mode)
					}
				}
			}
		}
	}
	if err := validateCredentialParameters(url.Values{names.Code: {"code"}}, names, ActionToken); err != nil {
		t.Fatal("code inference changed")
	}
	if err := validateCredentialParameters(url.Values{names.Code: {"code"}}, names, ActionIntrospect); !errors.Is(err, ErrInvalidRequest) {
		t.Fatal("inspection inferred an unintended mode")
	}
}

// TestRequestValidationBeforeReplay rejects ambiguous signed input before nonce claims for all center actions. TestRequestValidationBeforeReplay 为全部中心动作在随机值消费前拒绝歧义签名输入。
func TestRequestValidationBeforeReplay(t *testing.T) {
	s := NewServer()
	defer s.Close()
	client := newTestClient()
	client.SecretKey = "request-signing-key"
	if err := s.RegisterClient(client); err != nil {
		t.Fatal(err)
	}
	names := normalizeParamNames(ParamNames{Client: "application", Scope: "grants", Nonce: "request_id", Sign: "signature"})
	protection := newReplayTestProtection(t, nil)
	h := NewHTTPServer(s, HTTPOptions{ServerOptions: ServerOptions{CheckSign: true, Params: names}, ReplayProtection: protection})
	for _, action := range []string{ActionAuthorize, ActionToken, ActionIntrospect, ActionUserInfo, ActionRevoke, ActionLogout} {
		values := protection.stamp(url.Values{names.Client: {client.ClientID}, names.Scope: {"profile"}, "extension": {"a", "b"}}, names, action)
		bad := cloneValues(values)
		bad.Add(names.Scope, "email")
		bad = NewSignerWithParams(client.SecretKey, names).AttachSign(bad)
		if err := h.verifyRequest(context.Background(), bad, action, true); !errors.Is(err, ErrInvalidRequest) {
			t.Fatalf("duplicate %s: %v", action, err)
		}
		if action == ActionToken || action == ActionIntrospect || action == ActionUserInfo || action == ActionRevoke {
			mixed := cloneValues(values)
			mixed.Set(names.Ticket, "ticket")
			mixed.Set(names.TokenValue, "other")
			mixed = NewSignerWithParams(client.SecretKey, names).AttachSign(mixed)
			if err := h.verifyRequest(context.Background(), mixed, action, true); !errors.Is(err, ErrInvalidRequest) {
				t.Fatalf("mixed %s: %v", action, err)
			}
		}
		valid := NewSignerWithParams(client.SecretKey, names).AttachSign(values)
		if err := h.verifyRequest(context.Background(), valid, action, true); err != nil {
			t.Fatalf("rejected request consumed nonce for %s: %v", action, err)
		}
		if err := h.verifyRequest(context.Background(), valid, action, true); !errors.Is(err, ErrRequestReplayed) {
			t.Fatalf("accepted request did not consume nonce: %v", err)
		}
	}
}

// requestValidationPost creates an actual parsed form with optional query/body duplication. requestValidationPost 创建实际解析的表单，可包含查询串与请求体重复参数。
func requestValidationPost(handler http.Handler, path string, values url.Values, duplicate string) *httptest.ResponseRecorder {
	if duplicate != "" {
		path += "?" + url.Values{duplicate: {values.Get(duplicate)}}.Encode()
	}
	r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(values.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	return w
}

// TestRequestValidationCenterLifecycle preserves all credentials and skips hooks for ambiguous exchange, inspection and revocation. TestRequestValidationCenterLifecycle 对歧义兑换、检查及撤销保留全部凭证并跳过钩子。
func TestRequestValidationCenterLifecycle(t *testing.T) {
	for _, custom := range []bool{false, true} {
		s := NewServer()
		t.Cleanup(func() { _ = s.Close() })
		client := newTestClient()
		client.Modes = []Mode{ModeTicket, ModeOAuth2, ModeSharedToken, ModeRemoteSession}
		if err := s.RegisterClient(client); err != nil {
			t.Fatal(err)
		}
		issued := issueLogoutCredentials(t, s, client, "alice")
		names := DefaultParamNames()
		if custom {
			names = normalizeParamNames(ParamNames{Client: "application", Ticket: "proof", Mode: "flow", ClientSecret: "secret", TokenValue: "token"})
		}
		calls := 0
		h := NewHTTPServer(s, HTTPOptions{ServerOptions: ServerOptions{EnableSLO: true, Params: names},
			AuthorizationPolicy: func(context.Context, AuthorizationRequest) (AuthorizationDecision, error) {
				calls++
				return AuthorizationDecision{Allowed: true}, nil
			},
			CheckCredentialActive: func(*http.Request, CredentialInfo) (bool, error) { calls++; return true, nil },
			EnrichExchangeResult:  func(*http.Request, string, *TicketExchangeResult) error { calls++; return nil },
		})
		for _, tc := range []struct {
			mode       Mode
			key, value string
		}{
			{ModeTicket, names.Ticket, issued.ticket.Ticket}, {ModeOAuth2, names.Code, issued.code.Code},
			{ModeSharedToken, names.TokenValue, issued.token.Token}, {ModeRemoteSession, names.SessionID, issued.session.SessionID},
		} {
			for _, path := range []string{"/sso/token", "/sso/introspect", "/sso/userinfo", "/sso/revoke"} {
				if path == "/sso/token" && tc.mode != ModeTicket && tc.mode != ModeOAuth2 {
					continue
				}
				valid := url.Values{names.Client: {client.ClientID}, names.ClientSecret: {client.ClientSecret}, names.Mode: {string(tc.mode)}, names.Redirect: {client.RedirectURIs[0]}, tc.key: {tc.value}}
				for _, duplicate := range []string{names.Client, names.ClientSecret, names.Mode, names.Redirect, tc.key} {
					for _, split := range []bool{true, false} {
						values := cloneValues(valid)
						queryKey := duplicate
						if !split {
							values.Add(duplicate, values.Get(duplicate))
							queryKey = ""
						}
						if w := requestValidationPost(h.Handler(), path, values, queryKey); w.Code != http.StatusBadRequest {
							t.Fatalf("duplicate %s at %s: %d %s", duplicate, path, w.Code, w.Body.String())
						}
					}
				}
				foreign := names.Ticket
				if tc.key == foreign {
					foreign = names.TokenValue
				}
				valid.Set(foreign, "")
				if w := requestValidationPost(h.Handler(), path, valid, ""); w.Code != http.StatusBadRequest {
					t.Fatalf("mixed credential %s/%s accepted: %d", tc.mode, path, w.Code)
				}
			}
		}
		if calls != 0 {
			t.Fatalf("invalid request reached hooks %d times", calls)
		}
		assertLogoutCredentials(t, s, issued, true)
		if bindings, err := s.GetClientSessions(context.Background(), "alice"); err != nil || len(bindings) != 0 {
			t.Fatal("invalid exchange registered callbacks")
		}
	}
}

// TestRequestValidationClientAndCallback prevents ambiguous outbound requests and callback effects without changing signer canonicalization. TestRequestValidationClientAndCallback 阻止歧义出站请求及回调副作用，不改变签名规范化规则。
func TestRequestValidationClientAndCallback(t *testing.T) {
	for _, signed := range []bool{false, true} {
		requests, callbacks := 0, 0
		app := NewClientApp(ClientConfig{ClientID: "app", ServerURL: "https://center.example", CheckSign: signed, SecretKey: "key", HTTPClient: &http.Client{Transport: logoutReviewTransport(func(*http.Request) (*http.Response, error) {
			requests++
			return notificationResponse(200, `{"code":0}`), nil
		})}})
		req := CredentialRequest{Mode: ModeTicket, Ticket: "ticket", TokenValue: "other"}
		if _, err := app.ExchangeCredential(context.Background(), req); !errors.Is(err, ErrInvalidRequest) {
			t.Fatalf("mixed exchange: %v", err)
		}
		if _, err := app.Introspect(context.Background(), req); !errors.Is(err, ErrInvalidRequest) {
			t.Fatalf("mixed introspect: %v", err)
		}
		if _, err := app.UserInfo(context.Background(), req); !errors.Is(err, ErrInvalidRequest) {
			t.Fatalf("mixed userinfo: %v", err)
		}
		if err := app.Revoke(context.Background(), req); !errors.Is(err, ErrInvalidRequest) {
			t.Fatalf("mixed revoke: %v", err)
		}
		if requests != 0 {
			t.Fatal("mixed request reached transport")
		}
		if _, err := app.AuthURL("https://app.example/callback", url.Values{"scope": {"profile", "email"}}); !errors.Is(err, ErrInvalidRequest) {
			t.Fatalf("duplicate auth scope: %v", err)
		}
		if _, err := app.AuthURL("https://app.example/callback", url.Values{"scope": {"profile email"}, "tag": {"one", "two"}}); err != nil {
			t.Fatal(err)
		}
		handler := app.LogoutCallbackHandler(func(*http.Request, LogoutCallback) error { callbacks++; return nil })
		for _, key := range []string{"client", "loginId", "timestamp"} {
			values := url.Values{"client": {"app"}, "loginId": {"alice"}, "timestamp": {time.Now().Format(time.RFC3339)}}
			values.Add(key, values.Get(key))
			if signed {
				values = NewSigner("key").AttachSign(values)
				if !NewSigner("key").Verify(values) {
					t.Fatal("signer multi-value contract changed")
				}
			}
			if w := requestValidationPost(handler, "/callback", values, ""); w.Code != http.StatusBadRequest {
				t.Fatalf("callback duplicate %s: %d", key, w.Code)
			}
		}
		if callbacks != 0 {
			t.Fatal("ambiguous callback reached application")
		}
	}
}

// TestRequestValidationBrowserPreservesState rejects duplicate protocol metadata before consuming browser state. TestRequestValidationBrowserPreservesState 在消费浏览器状态前拒绝重复协议元数据。
func TestRequestValidationBrowserPreservesState(t *testing.T) {
	browser := newBrowserTestClient(t, nil, nil)
	authorize, cookie := beginBrowserTestLogin(t, browser, "/protected")
	r := browserTestCallback(browser, authorize, cookie)
	r.URL.RawQuery += "&scope=profile&scope=email"
	if _, err := browser.CompleteLogin(httptest.NewRecorder(), r); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("ambiguous browser callback: %v", err)
	}
	if _, err := browser.CompleteLogin(httptest.NewRecorder(), browserTestCallback(browser, authorize, cookie)); err != nil {
		t.Fatalf("rejected callback consumed state: %v", err)
	}
}

// TestRequestValidationAuthorizeAndLogout rejects ambiguity before resolving identity or changing account state. TestRequestValidationAuthorizeAndLogout 在解析身份或改变账号状态前拒绝授权与注销歧义。
func TestRequestValidationAuthorizeAndLogout(t *testing.T) {
	s := NewServer()
	defer s.Close()
	registerNotificationClients(t, s, "app-a")
	client, err := s.GetClient("app-a")
	if err != nil {
		t.Fatal(err)
	}
	ticket, err := s.GenerateTicket(context.Background(), client.ClientID, "alice", client.RedirectURIs[0], nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	resolverCalls, hookCalls := 0, 0
	h := NewHTTPServer(s, HTTPOptions{
		ServerOptions:      ServerOptions{EnableSLO: true, LogoutHTTPClient: &http.Client{Transport: logoutReviewTransport(func(*http.Request) (*http.Response, error) { hookCalls++; return notificationResponse(200, "ok"), nil })}},
		LoginIDResolver:    func(*http.Request) (string, bool) { resolverCalls++; return "alice", true },
		AuthorizationExtra: func(*http.Request, string, string) (map[string]any, error) { hookCalls++; return nil, nil },
		CenterLogout:       func(*http.Request, string) error { hookCalls++; return nil },
	})
	for _, action := range []string{"authorize", "logout"} {
		for _, duplicate := range []string{"client", "redirect", "scope", "back", "callback", "loginId"} {
			values := url.Values{"client": {client.ClientID}, "redirect": {client.RedirectURIs[0]}, "scope": {"profile"}, "back": {"state"}, "callback": {client.RedirectURIs[0]}, "loginId": {"alice"}}
			values.Add(duplicate, values.Get(duplicate))
			w := httptest.NewRecorder()
			h.Handler().ServeHTTP(w, httptest.NewRequest("GET", "/sso/"+action+"?"+values.Encode(), nil))
			if w.Code != http.StatusBadRequest || w.Header().Get("Location") != "" || len(w.Result().Cookies()) != 0 {
				t.Fatalf("duplicate %s %s: %d", action, duplicate, w.Code)
			}
		}
	}
	if resolverCalls != 0 || hookCalls != 0 {
		t.Fatal("ambiguous browser request invoked application or delivery")
	}
	if _, err := s.ValidateTicket(context.Background(), ticket.Ticket); err != nil {
		t.Fatalf("ambiguous logout revoked credentials: %v", err)
	}
	if bindings, err := s.GetClientSessions(context.Background(), "alice"); err != nil || len(bindings) != 1 {
		t.Fatal("ambiguous logout changed bindings")
	}
}

// TestRequestValidationCallbackReplay preserves nonce eligibility after signed query/body ambiguity, including custom names. TestRequestValidationCallbackReplay 在签名查询串与请求体歧义后保留随机值可用性，并覆盖自定义参数名。
func TestRequestValidationCallbackReplay(t *testing.T) {
	names := normalizeParamNames(ParamNames{Client: "application", LoginID: "subject", Timestamp: "at", Nonce: "request_id", Sign: "signature"})
	protection := newReplayTestProtection(t, nil)
	app := NewClientApp(ClientConfig{ClientID: "app", Params: names, CheckSign: true, SecretKey: "key", ReplayProtection: protection})
	calls := 0
	handler := app.LogoutCallbackHandler(func(*http.Request, LogoutCallback) error { calls++; return nil })
	values := protection.stamp(url.Values{names.Client: {"app"}, names.LoginID: {"alice"}}, names, ActionLogoutCallback)
	bad := cloneValues(values)
	bad.Add(names.LoginID, "alice")
	bad = NewSignerWithParams("key", names).AttachSign(bad)
	// Sign the merged form, then split duplicate values across query and body. 对合并表单签名，再将重复值拆到查询串与请求体。
	bad[names.LoginID] = []string{"alice"}
	if w := requestValidationPost(handler, "/callback", bad, names.LoginID); w.Code != http.StatusBadRequest || calls != 0 {
		t.Fatalf("ambiguous signed callback: %d calls=%d", w.Code, calls)
	}
	valid := NewSignerWithParams("key", names).AttachSign(values)
	if w := requestValidationPost(handler, "/callback", valid, ""); w.Code != http.StatusOK || calls != 1 {
		t.Fatalf("rejected callback consumed nonce: %d calls=%d", w.Code, calls)
	}
	if w := requestValidationPost(handler, "/callback", valid, ""); w.Code != http.StatusUnauthorized || calls != 1 {
		t.Fatal("accepted callback failed replay protection")
	}
}
