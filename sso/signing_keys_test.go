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

// TestSigningRotationScopes keeps each verification companion within its selected client's trust scope. TestSigningRotationScopes 将额外验签密钥限制在所选客户端的信任范围。
func TestSigningRotationScopes(t *testing.T) {
	storage := &registrationFailureStorage{MemoryStorage: NewMemoryStorage()}
	s := NewServer(WithStorage(storage))
	defer s.Close()
	for _, client := range []*Client{
		{ClientID: "dedicated", SecretKey: "app-new", VerificationKey: "app-old"},
		{ClientID: "fallback"},
		{ClientID: ClientAnonymous, SecretKey: "anon-new", VerificationKey: "anon-old"},
		{ClientID: "invalid", VerificationKey: "orphan"},
	} {
		if err := s.RegisterClient(client); err != nil {
			t.Fatal(err)
		}
	}
	names := normalizeParamNames(ParamNames{Client: "application", Sign: "signature"})
	h := NewHTTPServer(s, HTTPOptions{ServerOptions: ServerOptions{CheckSign: true, SecretKey: "global-new", VerificationKey: "global-old", Params: names}})
	for _, tc := range []struct {
		client, key string
		want        error
	}{
		{"dedicated", "app-new", nil}, {"dedicated", "app-old", nil},
		{"dedicated", "global-new", ErrInvalidSign}, {"dedicated", "global-old", ErrInvalidSign},
		{"dedicated", "anon-old", ErrInvalidSign}, {"dedicated", "", ErrInvalidSign},
		{"fallback", "global-new", nil}, {"fallback", "global-old", nil}, {"fallback", "app-old", ErrInvalidSign},
		{"", "global-old", nil}, {ClientAnonymous, "anon-old", nil}, {ClientAnonymous, "global-old", ErrInvalidSign},
		{"missing", "global-old", ErrClientNotFound}, {"invalid", "orphan", ErrSignSecretRequired}, {"invalid", "global-new", ErrSignSecretRequired},
	} {
		values := NewSignerWithParams(tc.key, names).AttachSign(url.Values{names.Client: {tc.client}})
		if err := h.verifySign(context.Background(), values); !errors.Is(err, tc.want) {
			t.Fatalf("client %q: got %v, want %v", tc.client, err, tc.want)
		}
	}
	h.options.AllowAnonymousClient = true
	for _, key := range []string{"anon-new", "anon-old"} {
		if err := h.verifySign(context.Background(), NewSignerWithParams(key, names).AttachSign(url.Values{})); err != nil {
			t.Fatal(err)
		}
	}
	storage.failRead = true
	values := NewSignerWithParams("app-old", names).AttachSign(url.Values{names.Client: {"dedicated"}})
	if err := h.verifySign(context.Background(), values); !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("lookup failure reused verification key: %v", err)
	}
	storage.failRead = false
	if err := s.UnregisterClient("dedicated"); err != nil {
		t.Fatal(err)
	}
	if err := h.verifySign(context.Background(), values); !errors.Is(err, ErrClientNotFound) {
		t.Fatalf("removed client retained verification key: %v", err)
	}
}

// TestSigningRotationReplay spans preparation, switch and retirement without resetting nonce history. TestSigningRotationReplay 跨准备、切换及退役阶段保持随机值历史。
func TestSigningRotationReplay(t *testing.T) {
	s := NewServer()
	defer s.Close()
	names := normalizeParamNames(ParamNames{Client: "application", Sign: "signature", Nonce: "request_id"})
	h := NewHTTPServer(s, HTTPOptions{ServerOptions: ServerOptions{CheckSign: true, Params: names}, ReplayProtection: newReplayTestProtection(t, nil)})
	client := &Client{ClientID: "app", SecretKey: "old", VerificationKey: "new"}
	for _, action := range []string{ActionAuthorize, ActionToken, ActionIntrospect, ActionUserInfo, ActionRevoke, ActionLogout} {
		client.SecretKey, client.VerificationKey = "old", "new"
		if err := s.RegisterClientChecked(client); err != nil {
			t.Fatal(err)
		}
		values := h.options.ReplayProtection.stamp(url.Values{names.Client: {client.ClientID}}, names, action)
		old := NewSignerWithParams("old", names).AttachSign(values)
		if err := h.verifyRequest(context.Background(), old, action, true); err != nil {
			t.Fatal(err)
		}
		client.SecretKey, client.VerificationKey = "new", "old"
		if err := s.RegisterClientChecked(client); err != nil {
			t.Fatal(err)
		}
		if err := h.verifyRequest(context.Background(), NewSignerWithParams("new", names).AttachSign(values), action, true); !errors.Is(err, ErrRequestReplayed) {
			t.Fatalf("rotation reset nonce for %s: %v", action, err)
		}
		fresh := h.options.ReplayProtection.stamp(url.Values{names.Client: {client.ClientID}}, names, action)
		if err := h.verifyRequest(context.Background(), NewSignerWithParams("old", names).AttachSign(fresh), action, true); err != nil {
			t.Fatalf("old key unavailable during overlap: %v", err)
		}
		client.VerificationKey = ""
		if err := s.RegisterClientChecked(client); err != nil {
			t.Fatal(err)
		}
		fresh = h.options.ReplayProtection.stamp(url.Values{names.Client: {client.ClientID}}, names, action)
		if err := h.verifyRequest(context.Background(), NewSignerWithParams("old", names).AttachSign(fresh), action, true); !errors.Is(err, ErrInvalidSign) {
			t.Fatalf("retired key accepted: %v", err)
		}
		if err := h.verifyRequest(context.Background(), NewSignerWithParams("new", names).AttachSign(fresh), action, true); err != nil {
			t.Fatalf("retired key consumed nonce: %v", err)
		}
	}
}

// TestSigningRotationCallback preserves callback replay state across immutable client replacements. TestSigningRotationCallback 在替换不可变客户端配置时保留注销回调重放状态。
func TestSigningRotationCallback(t *testing.T) {
	names := normalizeParamNames(ParamNames{Client: "application", Sign: "signature"})
	protection := newReplayTestProtection(t, nil)
	config := ClientConfig{ClientID: "app", CheckSign: true, SecretKey: "new", VerificationKey: "old", Params: names, ReplayProtection: protection}
	calls := 0
	hook := func(*http.Request, LogoutCallback) error { calls++; return nil }
	values := protection.stamp(url.Values{names.Client: {"app"}, names.LoginID: {"alice"}}, names, ActionLogoutCallback)
	app := NewClientApp(config)
	if w := requestValidationPost(app.LogoutCallbackHandler(hook), "/callback", NewSignerWithParams("old", names).AttachSign(values), ""); w.Code != http.StatusOK || calls != 1 {
		t.Fatalf("old callback: %d calls=%d", w.Code, calls)
	}
	if w := requestValidationPost(app.LogoutCallbackHandler(hook), "/callback", NewSignerWithParams("new", names).AttachSign(values), ""); w.Code != http.StatusUnauthorized || calls != 1 {
		t.Fatal("same callback with different key bypassed replay detection")
	}
	config.VerificationKey = ""
	app = NewClientApp(config)
	values = protection.stamp(url.Values{names.Client: {"app"}, names.LoginID: {"alice"}}, names, ActionLogoutCallback)
	if w := requestValidationPost(app.LogoutCallbackHandler(hook), "/callback", NewSignerWithParams("old", names).AttachSign(values), ""); w.Code != http.StatusUnauthorized || calls != 1 {
		t.Fatal("retired callback key accepted")
	}
	if w := requestValidationPost(app.LogoutCallbackHandler(hook), "/callback", NewSignerWithParams("new", names).AttachSign(values), ""); w.Code != http.StatusOK || calls != 2 {
		t.Fatal("invalid signature consumed callback nonce")
	}
}

// TestSigningRotationLifecycle exchanges all credential modes across a rolling center/client key switch. TestSigningRotationLifecycle 在中心与客户端滚动切换密钥期间验证四种凭证模式。
func TestSigningRotationLifecycle(t *testing.T) {
	for _, mode := range []Mode{ModeTicket, ModeOAuth2, ModeSharedToken, ModeRemoteSession} {
		t.Run(string(mode), func(t *testing.T) {
			s := NewServer()
			defer s.Close()
			client := newTestClient()
			client.SecretKey, client.VerificationKey = "old", "new"
			client.Modes = []Mode{mode}
			h := NewHTTPServer(s, HTTPOptions{ServerOptions: ServerOptions{CheckSign: true, Clients: map[string]Client{client.ClientID: *client}}, LoginIDResolver: func(*http.Request) (string, bool) { return "alice", true }, ReplayProtection: newReplayTestProtection(t, nil)})
			app := NewClientApp(ClientConfig{ClientID: client.ClientID, ClientSecret: client.ClientSecret, ServerURL: "https://center.example", Mode: mode, CheckSign: true, SecretKey: "new", VerificationKey: "old", ReplayProtection: newReplayTestProtection(t, nil), HTTPClient: &http.Client{Transport: logoutReviewTransport(func(r *http.Request) (*http.Response, error) {
				w := httptest.NewRecorder()
				h.Handler().ServeHTTP(w, r)
				return w.Result(), nil
			})}})
			verifier := ""
			var target string
			var err error
			if mode == ModeTicket || mode == ModeOAuth2 {
				verifier = NewPKCEVerifier()
				target, err = app.AuthURLWithPKCE(client.RedirectURIs[0], verifier, nil)
			} else {
				target, err = app.AuthURL(client.RedirectURIs[0], nil)
			}
			if err != nil {
				t.Fatal(err)
			}
			u, err := url.Parse(target)
			if err != nil {
				t.Fatal(err)
			}
			if !NewSigner("new").Verify(u.Query()) || NewSigner("old").Verify(u.Query()) {
				t.Fatal("SDK did not use the active signing key")
			}
			w := httptest.NewRecorder()
			h.Handler().ServeHTTP(w, httptest.NewRequest("GET", target, nil))
			if w.Code != http.StatusFound {
				t.Fatalf("authorize: %d %s", w.Code, w.Body.String())
			}
			callback, err := url.Parse(w.Header().Get("Location"))
			if err != nil {
				t.Fatal(err)
			}
			values := callback.Query()
			req := CredentialRequest{Mode: mode, RedirectURI: client.RedirectURIs[0], CodeVerifier: verifier, Ticket: values.Get("ticket"), Code: values.Get("code"), TokenValue: values.Get("tokenValue"), SessionID: values.Get("sessionId")}
			client.SecretKey, client.VerificationKey = "new", "old"
			if err := s.RegisterClientChecked(client); err != nil {
				t.Fatal(err)
			}
			if mode == ModeTicket || mode == ModeOAuth2 {
				result, err := app.ExchangeCredential(context.Background(), req)
				if err != nil || result == nil || result.LoginID != "alice" {
					t.Fatalf("exchange: %+v %v", result, err)
				}
			} else {
				result, err := app.UserInfo(context.Background(), req)
				if err != nil || result == nil || !result.Active || result.LoginID != "alice" {
					t.Fatalf("userinfo: %+v %v", result, err)
				}
			}
		})
	}
}

// TestSigningRotationDelivery signs each retry with the current primary key while a prepared client accepts both. TestSigningRotationDelivery 每次重试使用当前主密钥签名，已准备客户端同时接受两把密钥。
func TestSigningRotationDelivery(t *testing.T) {
	s := NewServer()
	defer s.Close()
	registerNotificationClients(t, s, "app-a")
	client, err := s.GetClient("app-a")
	if err != nil {
		t.Fatal(err)
	}
	client.SecretKey, client.VerificationKey = "old", "new"
	if err := s.RegisterClientChecked(client); err != nil {
		t.Fatal(err)
	}
	app := NewClientApp(ClientConfig{ClientID: client.ClientID, CheckSign: true, SecretKey: "old", VerificationKey: "new", ReplayProtection: newReplayTestProtection(t, nil)})
	calls := 0
	h := NewHTTPServer(s, HTTPOptions{ReplayProtection: newReplayTestProtection(t, nil), ServerOptions: ServerOptions{CheckSign: true, LogoutCallbackMaxAttempts: 2, LogoutCallbackRetryInterval: time.Nanosecond, LogoutHTTPClient: &http.Client{Transport: logoutReviewTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		callback, err := app.VerifyLogoutCallback(r)
		if err != nil {
			return nil, err
		}
		key := "old"
		if calls > 1 {
			key = "new"
		}
		if !NewSigner(key).Verify(callback.Values) {
			t.Error("delivery used a verification-only key")
		}
		if calls == 1 {
			client.SecretKey, client.VerificationKey = "new", "old"
			if err := s.RegisterClientChecked(client); err != nil {
				return nil, err
			}
			return notificationResponse(503, ""), nil
		}
		return notificationResponse(200, `{"code":0}`), nil
	})}}})
	report, err := h.NotifyLogoutClients(httptest.NewRequest("POST", "/logout", nil), "alice")
	if err != nil || report == nil || !report.Complete || calls != 2 {
		t.Fatalf("delivery: %+v %v calls=%d", report, err, calls)
	}
}

// TestSigningRotationConfiguration rejects orphan or disabled verification settings without leaking secrets. TestSigningRotationConfiguration 拒绝无主密钥或关闭验签的额外密钥配置，且不泄露密钥。
func TestSigningRotationConfiguration(t *testing.T) {
	for _, check := range []func() error{
		func() error { return (&Client{ClientID: "app", VerificationKey: "private-companion"}).Validate() },
		func() error { return (ClientConfig{VerificationKey: "private-companion"}).Validate() },
		func() error {
			return (ClientConfig{SecretKey: "active", VerificationKey: "private-companion"}).Validate()
		},
		func() error {
			return (HTTPOptions{ServerOptions: ServerOptions{VerificationKey: "private-companion"}}).Validate()
		},
		func() error {
			return (HTTPOptions{ServerOptions: ServerOptions{SecretKey: "active", VerificationKey: "private-companion"}}).Validate()
		},
	} {
		err := check()
		if !errors.Is(err, ErrSSOConfig) || strings.Contains(err.Error(), "private-companion") {
			t.Fatalf("configuration diagnostic: %v", err)
		}
	}
	storage := &configurationStorage{MemoryStorage: NewMemoryStorage()}
	s := NewServer(WithStorage(storage))
	defer s.Close()
	options := configurationOptions()
	options.CheckSign = true
	options.Clients = map[string]Client{"app": {VerificationKey: "private-companion"}}
	if _, err := NewHTTPServerChecked(s, options); !errors.Is(err, ErrSignSecretRequired) || storage.writes != 0 {
		t.Fatal("invalid registration reached storage")
	}
	app := NewClientApp(ClientConfig{CheckSign: true, VerificationKey: "private-companion", ServerURL: "https://center.example"})
	if _, err := app.AuthURL("https://app.example/callback", nil); !errors.Is(err, ErrSignSecretRequired) {
		t.Fatal("verification-only key used for signing")
	}
}
