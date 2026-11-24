package sso

import (
	"context"
	"errors"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"testing/iotest"
)

// requestLimitBody records actual read volume for unknown-length streams. requestLimitBody 记录未知长度流的实际读取量。
type requestLimitBody struct {
	reader io.Reader
	read   int
}

// Read counts delivered bytes. Read 统计已交付字节。
func (b *requestLimitBody) Read(p []byte) (int, error) {
	n, err := b.reader.Read(p)
	b.read += n
	return n, err
}

// Close satisfies the request body contract. Close 满足请求体接口。
func (b *requestLimitBody) Close() error { return nil }

// TestRequestLimitsParsing covers exact boundaries, streaming bodies and safe parser failures. TestRequestLimitsParsing 覆盖精确边界、流式请求体及安全解析失败。
func TestRequestLimitsParsing(t *testing.T) {
	limits := RequestLimits{MaxQueryBytes: 32, MaxBodyBytes: 16}
	for _, tc := range []struct {
		name, query, body string
		length            int64
		want              error
	}{
		{"body below", "", "x=" + strings.Repeat("a", 13), -1, nil},
		{"body exact", "", "x=" + strings.Repeat("a", 14), -1, nil},
		{"body over", "", "x=" + strings.Repeat("a", 15), -1, ErrRequestTooLarge},
		{"stream", "", "x=" + strings.Repeat("a", 8192), -1, ErrRequestTooLarge},
		{"understated length", "", "x=" + strings.Repeat("a", 20), 1, ErrRequestTooLarge},
		{"zero length", "", "x=" + strings.Repeat("a", 20), 0, ErrRequestTooLarge},
		{"known oversized", "", "x=a", 17, ErrRequestTooLarge},
		{"query exact", "x=" + strings.Repeat("a", 30), "y=b", -1, nil},
		{"query over", "x=" + strings.Repeat("a", 31), "y=b", -1, ErrRequestTooLarge},
		{"encoded query", "x=" + strings.Repeat("%61", 11), "", -1, ErrRequestTooLarge},
		{"encoded body", "", "x=" + strings.Repeat("%61", 5), -1, ErrRequestTooLarge},
		{"bad query", "x=ok&secret=%GG", "y=b", -1, ErrInvalidRequest},
		{"bad body", "x=ok", "secret=%GG", -1, ErrInvalidRequest},
		{"semicolon", "", "x=ok;y=bad", -1, ErrInvalidRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := &requestLimitBody{reader: strings.NewReader(tc.body)}
			r := httptest.NewRequest(http.MethodPost, "/?"+tc.query, nil)
			r.Body, r.ContentLength = body, tc.length
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			err := parseRequestForm(r, limits)
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
			if body.read > int(limits.MaxBodyBytes)+1 {
				t.Fatalf("read beyond limit plus probe: %d", body.read)
			}
			if tc.want != nil && (r.Form != nil || r.PostForm != nil) {
				t.Fatal("failed parse published partial parameters")
			}
			if (int64(len(tc.query)) > limits.MaxQueryBytes || tc.length > limits.MaxBodyBytes) && body.read != 0 {
				t.Fatal("known oversized request read body")
			}
		})
	}
	r := httptest.NewRequest(http.MethodPost, "/", io.NopCloser(iotest.ErrReader(errors.New("private upstream diagnostic"))))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err := parseRequestForm(r, limits); err != ErrInvalidRequest {
		t.Fatalf("reader diagnostic exposed: %v", err)
	}
	r = httptest.NewRequest(http.MethodPost, "/", strings.NewReader("x="+strings.Repeat("a", 10)))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Body = http.MaxBytesReader(nil, r.Body, 4)
	if err := parseRequestForm(r, limits); err != ErrRequestTooLarge {
		t.Fatalf("outer middleware limit lost: %v", err)
	}
}

// TestRequestLimitsDefaultsAndCachedForms retains middleware compatibility while bounding injected values. TestRequestLimitsDefaultsAndCachedForms 保留中间件兼容性，同时限制注入的参数大小。
func TestRequestLimitsDefaultsAndCachedForms(t *testing.T) {
	defaults := DefaultRequestLimits()
	if DefaultHTTPOptions().RequestLimits != defaults || DefaultClientConfig().RequestLimits != defaults {
		t.Fatal("public defaults differ")
	}
	for _, limits := range []RequestLimits{{}, {MaxQueryBytes: -1, MaxBodyBytes: -2}} {
		if NewClientApp(ClientConfig{RequestLimits: limits}).Config().RequestLimits != defaults || NewHTTPServer(nil, HTTPOptions{RequestLimits: limits}).options.RequestLimits != defaults {
			t.Fatal("constructor defaults differ")
		}
	}
	for _, tc := range []struct {
		name, query string
		post, form  url.Values
		want        error
	}{
		{"valid", "tag=a", url.Values{"tag": {"b"}}, url.Values{"tag": {"b", "a"}}, nil},
		{"cached body", "", url.Values{"x": {strings.Repeat("a", 16)}}, url.Values{}, ErrRequestTooLarge},
		{"injected merged form", "", url.Values{}, url.Values{"x": {strings.Repeat("a", 48)}}, ErrRequestTooLarge},
		{"cached malformed query", "secret=%GG", url.Values{}, url.Values{"x": {"ok"}}, ErrInvalidRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/?"+tc.query, nil)
			r.Form, r.PostForm = tc.form, tc.post
			if err := parseRequestForm(r, RequestLimits{MaxQueryBytes: 32, MaxBodyBytes: 16}); !errors.Is(err, tc.want) {
				t.Fatalf("cached form: %v", err)
			}
		})
	}
	r := httptest.NewRequest(http.MethodPost, "/?x=a", strings.NewReader("y=b"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err := parseRequestForm(r, RequestLimits{MaxQueryBytes: math.MaxInt64, MaxBodyBytes: math.MaxInt64}); err != nil {
		t.Fatalf("limit addition overflow: %v", err)
	}
	if r.Form.Get("x") != "a" || r.Form.Get("y") != "b" {
		t.Fatal("query/body merge changed")
	}
}

// TestRequestLimitsCenterEntrypoints checks every direct handler and rejects before application hooks. TestRequestLimitsCenterEntrypoints 检查所有直接处理器，并在业务钩子前拒绝输入。
func TestRequestLimitsCenterEntrypoints(t *testing.T) {
	s := NewServer()
	defer s.Close()
	calls := 0
	h := NewHTTPServer(s, HTTPOptions{RequestLimits: RequestLimits{MaxQueryBytes: 32, MaxBodyBytes: 32},
		LoginIDResolver: func(*http.Request) (string, bool) { calls++; return "alice", true },
		CenterLogout:    func(*http.Request, string) error { calls++; return nil },
		AuthorizationPolicy: func(context.Context, AuthorizationRequest) (AuthorizationDecision, error) {
			calls++
			return AuthorizationDecision{Allowed: true}, nil
		},
	})
	for _, endpoint := range []struct {
		method string
		handle http.HandlerFunc
	}{
		{http.MethodGet, h.HandleAuthorize}, {http.MethodGet, h.HandleLogout},
		{http.MethodPost, h.HandleToken}, {http.MethodPost, h.HandleIntrospect},
		{http.MethodPost, h.HandleUserInfo}, {http.MethodPost, h.HandleRevoke}, {http.MethodPost, h.HandleLogout},
	} {
		for _, input := range []struct {
			query, body string
			status      int
		}{
			{"x=" + strings.Repeat("a", 32), "", http.StatusRequestEntityTooLarge},
			{"secret=%GG", "", http.StatusBadRequest},
			{"", "x=" + strings.Repeat("a", 32), http.StatusRequestEntityTooLarge},
			{"", "secret=%GG", http.StatusBadRequest},
		} {
			if endpoint.method == http.MethodGet && input.body != "" {
				continue
			}
			r := httptest.NewRequest(endpoint.method, "/?"+input.query, strings.NewReader(input.body))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			r.ContentLength = -1
			w := httptest.NewRecorder()
			endpoint.handle(w, r)
			if w.Code != input.status || strings.Contains(w.Body.String(), "%GG") || w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Location") != "" || len(w.Result().Cookies()) != 0 {
				t.Fatalf("response: %d %s", w.Code, w.Body.String())
			}
		}
	}
	if calls != 0 {
		t.Fatal("rejected input reached application")
	}
}

// TestRequestLimitsCenterPreservesCredentials rejects signed oversized forms before nonce or credential consumption in all modes. TestRequestLimitsCenterPreservesCredentials 在全部模式下于消费随机值或凭证前拒绝已签名的超大表单。
func TestRequestLimitsCenterPreservesCredentials(t *testing.T) {
	s := NewServer()
	defer s.Close()
	client := newTestClient()
	client.SecretKey = "signing-key"
	client.Modes = []Mode{ModeTicket, ModeOAuth2, ModeSharedToken, ModeRemoteSession}
	if err := s.RegisterClient(client); err != nil {
		t.Fatal(err)
	}
	issued := issueLogoutCredentials(t, s, client, "alice")
	p := DefaultParamNames()
	protection := newReplayTestProtection(t, nil)
	h := NewHTTPServer(s, HTTPOptions{ServerOptions: ServerOptions{CheckSign: true}, ReplayProtection: protection,
		RequestLimits: RequestLimits{MaxBodyBytes: 1024},
	})
	for _, credential := range []struct {
		mode       Mode
		key, value string
	}{
		{ModeTicket, p.Ticket, issued.ticket.Ticket}, {ModeOAuth2, p.Code, issued.code.Code},
		{ModeSharedToken, p.TokenValue, issued.token.Token}, {ModeRemoteSession, p.SessionID, issued.session.SessionID},
	} {
		for _, action := range []string{ActionToken, ActionIntrospect, ActionUserInfo, ActionRevoke} {
			if action == ActionToken && credential.mode != ModeTicket && credential.mode != ModeOAuth2 {
				continue
			}
			values := protection.stamp(url.Values{p.Client: {client.ClientID}, p.ClientSecret: {client.ClientSecret}, p.Mode: {string(credential.mode)}, p.Redirect: {client.RedirectURIs[0]}, credential.key: {credential.value}}, p, action)
			bad := cloneValues(values)
			bad.Set("extension", strings.Repeat("a", 1024))
			bad = NewSigner(client.SecretKey).AttachSign(bad)
			path := map[string]string{ActionToken: "/sso/token", ActionIntrospect: "/sso/introspect", ActionUserInfo: "/sso/userinfo", ActionRevoke: "/sso/revoke"}[action]
			if w := requestValidationPost(h.Handler(), path, bad, ""); w.Code != http.StatusRequestEntityTooLarge {
				t.Fatalf("%s/%s: %d %s", credential.mode, action, w.Code, w.Body.String())
			}
			valid := NewSigner(client.SecretKey).AttachSign(values)
			if err := h.verifyRequest(context.Background(), valid, action, true); err != nil {
				t.Fatalf("rejected request consumed nonce: %v", err)
			}
			assertLogoutCredentials(t, s, issued, true)
		}
	}
	for _, endpoint := range []struct{ action, path string }{
		{ActionAuthorize, "/sso/authorize"}, {ActionLogout, "/sso/logout"},
	} {
		values := protection.stamp(url.Values{p.Client: {client.ClientID}, p.Redirect: {client.RedirectURIs[0]}}, p, endpoint.action)
		bad := cloneValues(values)
		bad.Set("extension", strings.Repeat("a", int(h.options.RequestLimits.MaxQueryBytes)))
		bad = NewSigner(client.SecretKey).AttachSign(bad)
		if w := replayTestRequest(h.Handler(), http.MethodGet, endpoint.path, bad); w.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("oversized %s: %d %s", endpoint.action, w.Code, w.Body.String())
		}
		valid := NewSigner(client.SecretKey).AttachSign(values)
		if err := h.verifyRequest(context.Background(), valid, endpoint.action, true); err != nil {
			t.Fatalf("rejected query consumed nonce: %v", err)
		}
		assertLogoutCredentials(t, s, issued, true)
	}
}

// TestRequestLimitsLogoutCallback preserves replay eligibility and never acknowledges rejected input. TestRequestLimitsLogoutCallback 保留防重放随机值可用性，且不确认被拒绝的输入。
func TestRequestLimitsLogoutCallback(t *testing.T) {
	p := normalizeParamNames(ParamNames{Client: "application", LoginID: "subject", Nonce: "request_id"})
	protection := newReplayTestProtection(t, nil)
	app := NewClientApp(ClientConfig{ClientID: "app", Params: p, CheckSign: true, SecretKey: "key", ReplayProtection: protection,
		RequestLimits: RequestLimits{MaxBodyBytes: 512, MaxQueryBytes: 512},
	})
	calls := 0
	h := app.LogoutCallbackHandler(func(*http.Request, LogoutCallback) error { calls++; return nil })
	values := protection.stamp(url.Values{p.Client: {"app"}, p.LoginID: {"alice"}}, p, ActionLogoutCallback)
	bad := cloneValues(values)
	bad.Set("extension", strings.Repeat("a", 512))
	bad = NewSignerWithParams("key", p).AttachSign(bad)
	if w := requestValidationPost(h, "/callback", bad, ""); w.Code != http.StatusRequestEntityTooLarge || calls != 0 {
		t.Fatalf("oversized callback: %d calls=%d", w.Code, calls)
	}
	valid := NewSignerWithParams("key", p).AttachSign(values)
	for _, tc := range []struct {
		query, suffix string
		status        int
	}{
		{"x=" + strings.Repeat("a", 512), "", http.StatusRequestEntityTooLarge},
		{"secret=%GG", "", http.StatusBadRequest},
		{"", "&secret=%GG", http.StatusBadRequest},
	} {
		r := httptest.NewRequest(http.MethodPost, "/callback?"+tc.query, strings.NewReader(valid.Encode()+tc.suffix))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		h(w, r)
		if w.Code != tc.status || calls != 0 || strings.Contains(w.Body.String(), "%GG") {
			t.Fatalf("callback rejection: %d %s", w.Code, w.Body.String())
		}
	}
	if w := requestValidationPost(h, "/callback", valid, ""); w.Code != http.StatusOK || calls != 1 {
		t.Fatalf("valid callback: %d calls=%d", w.Code, calls)
	}
	if w := requestValidationPost(h, "/callback", valid, ""); w.Code != http.StatusUnauthorized || calls != 1 {
		t.Fatal("callback replay accepted")
	}
}

// TestRequestLimitsBrowserPreservesState checks all modes without consuming state or contacting the center on rejected queries. TestRequestLimitsBrowserPreservesState 检查全部模式，拒绝查询串时不消费状态或联系中心。
func TestRequestLimitsBrowserPreservesState(t *testing.T) {
	for _, mode := range []Mode{ModeTicket, ModeOAuth2, ModeSharedToken, ModeRemoteSession} {
		t.Run(string(mode), func(t *testing.T) {
			requests, sessions := 0, 0
			b := newBrowserTestClient(t, nil, logoutReviewTransport(func(*http.Request) (*http.Response, error) {
				requests++
				w := httptest.NewRecorder()
				if mode == ModeTicket || mode == ModeOAuth2 {
					writeJSON(w, http.StatusOK, OKResponse(TicketExchangeResult{LoginID: "alice", RemainTokenTimeout: 60}))
				} else {
					writeJSON(w, http.StatusOK, OKResponse(CredentialInfo{Active: true, LoginID: "alice", ClientID: "app", Mode: mode, ExpiresIn: 60}))
				}
				return w.Result(), nil
			}))
			config := b.app.Config()
			config.Mode = mode
			config.RequestLimits.MaxQueryBytes = 256
			var err error
			b, err = NewBrowserClient(NewClientApp(config), b.options)
			if err != nil {
				t.Fatal(err)
			}
			authorize, cookie := beginBrowserTestLogin(t, b, "/protected")
			h := b.CallbackHandler(func(http.ResponseWriter, *http.Request, *BrowserLoginResult) error { sessions++; return nil })
			for _, tc := range []struct {
				suffix string
				status int
			}{
				{"&extension=" + strings.Repeat("a", 256), http.StatusRequestEntityTooLarge},
				{"&secret=%GG", http.StatusBadRequest},
			} {
				r := browserTestCallback(b, authorize, cookie)
				r.URL.RawQuery += tc.suffix
				w := httptest.NewRecorder()
				h(w, r)
				if w.Code != tc.status || requests != 0 || sessions != 0 || len(w.Result().Cookies()) != 0 || w.Header().Get("Location") != "" {
					t.Fatalf("rejected callback changed state: %d requests=%d sessions=%d", w.Code, requests, sessions)
				}
			}
			w := httptest.NewRecorder()
			h(w, browserTestCallback(b, authorize, cookie))
			if w.Code != http.StatusSeeOther || requests != 1 || sessions != 1 {
				t.Fatalf("valid callback: %d %s", w.Code, w.Body.String())
			}
		})
	}
}
