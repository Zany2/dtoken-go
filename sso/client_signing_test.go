package sso

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
)

// TestClientSigningKeySelection verifies explicit keys, legacy fallback, and ambiguous client rejection. TestClientSigningKeySelection 验证独立密钥、旧配置回退和歧义客户端拒绝。
func TestClientSigningKeySelection(t *testing.T) {
	for _, tc := range []struct {
		name, clientID, clientKey, serverKey, requestKey string
		anonymous, duplicate, disabled                   bool
		want                                             error
	}{
		{name: "dedicated key", clientID: "app-a", clientKey: "dedicated", serverKey: "default", requestKey: "dedicated"},
		{name: "no global key", clientID: "app-a", clientKey: "dedicated", requestKey: "dedicated"},
		{name: "global cannot override", clientID: "app-a", clientKey: "dedicated", serverKey: "default", requestKey: "default", want: ErrInvalidSign},
		{name: "legacy fallback", clientID: "app-a", serverKey: "default", requestKey: "default"},
		{name: "missing secret", clientID: "app-a", requestKey: "default", want: ErrSignSecretRequired},
		{name: "unknown client", clientID: "unknown", serverKey: "default", requestKey: "default", want: ErrClientNotFound},
		{name: "center request", serverKey: "default", requestKey: "default"},
		{name: "duplicate client", clientID: "app-a", clientKey: "dedicated", requestKey: "dedicated", duplicate: true, want: ErrInvalidRequest},
		{name: "anonymous override", clientKey: "anonymous-key", serverKey: "default", requestKey: "anonymous-key", anonymous: true},
		{name: "anonymous fallback", serverKey: "default", requestKey: "default", anonymous: true},
		{name: "anonymous rejects global", clientKey: "anonymous-key", serverKey: "default", requestKey: "default", anonymous: true, want: ErrInvalidSign},
		{name: "signing disabled", clientID: "unknown", disabled: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := NewServer()
			defer s.Close()
			registeredID := "app-a"
			if tc.anonymous {
				registeredID = ClientAnonymous
			}
			params := normalizeParamNames(ParamNames{Client: "application", Sign: "signature"})
			h := NewHTTPServer(s, HTTPOptions{ServerOptions: ServerOptions{
				CheckSign: !tc.disabled, SecretKey: tc.serverKey, Params: params, AllowAnonymousClient: tc.anonymous,
				Clients: map[string]Client{registeredID: {SecretKey: tc.clientKey}},
			}})
			if err := h.initializationError(); err != nil {
				t.Fatal(err)
			}
			values := url.Values{}
			if tc.clientID != "" {
				values.Set(params.Client, tc.clientID)
			}
			if tc.duplicate {
				values.Add(params.Client, tc.clientID)
			}
			values = NewSignerWithParams(tc.requestKey, params).AttachSign(values)
			if err := h.verifySign(context.Background(), values); !errors.Is(err, tc.want) {
				t.Fatalf("verifySign() = %v, want %v", err, tc.want)
			}
		})
	}
}

// TestClientSigningKeyRefresh verifies rotations, removal, and storage failures never reuse stale keys. TestClientSigningKeyRefresh 验证轮换、注销和存储故障不会继续使用旧密钥。
func TestClientSigningKeyRefresh(t *testing.T) {
	storage := &registrationFailureStorage{MemoryStorage: NewMemoryStorage()}
	s := NewServer(WithStorage(storage))
	defer s.Close()
	client := newTestClient()
	client.SecretKey = "old-key"
	if err := s.RegisterClient(client); err != nil {
		t.Fatal(err)
	}
	h := NewHTTPServer(s, HTTPOptions{ServerOptions: ServerOptions{CheckSign: true, SecretKey: "default-key"}})
	values := url.Values{"client": {client.ClientID}}
	oldSigned := NewSigner(client.SecretKey).AttachSign(values)
	if err := h.verifySign(context.Background(), oldSigned); err != nil {
		t.Fatal(err)
	}
	client.SecretKey = "new-key"
	if err := s.RegisterClient(client); err != nil {
		t.Fatal(err)
	}
	if err := h.verifySign(context.Background(), oldSigned); !errors.Is(err, ErrInvalidSign) {
		t.Fatalf("old key after rotation = %v", err)
	}
	newSigned := NewSigner(client.SecretKey).AttachSign(values)
	if err := h.verifySign(context.Background(), newSigned); err != nil {
		t.Fatal(err)
	}
	storage.failRead = true
	if err := h.verifySign(context.Background(), NewSigner("default-key").AttachSign(values)); !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("lookup failure = %v", err)
	}
	storage.failRead = false
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := h.verifySign(ctx, newSigned); !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("canceled lookup = %v", err)
	}
	if err := s.UnregisterClient(client.ClientID); err != nil {
		t.Fatal(err)
	}
	if err := h.verifySign(context.Background(), newSigned); !errors.Is(err, ErrClientNotFound) {
		t.Fatalf("unregistered client = %v", err)
	}
}

// TestHTTPRejectsOtherClientSigningKeys verifies all protocol routes reject global and other-client signatures. TestHTTPRejectsOtherClientSigningKeys 验证所有协议路由拒绝默认密钥及其他客户端密钥冒用。
func TestHTTPRejectsOtherClientSigningKeys(t *testing.T) {
	s := NewServer()
	defer s.Close()
	client := newTestClient()
	client.SecretKey = "app-a-key"
	if err := s.RegisterClient(client); err != nil {
		t.Fatal(err)
	}
	other := *client
	other.ClientID, other.SecretKey = "app-b", "app-b-key"
	if err := s.RegisterClient(&other); err != nil {
		t.Fatal(err)
	}
	ticket, err := s.GenerateTicket(context.Background(), client.ClientID, "user", client.RedirectURIs[0], nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	centerCalls := 0
	h := NewHTTPServer(s, HTTPOptions{
		ServerOptions:   ServerOptions{CheckSign: true, SecretKey: "default-key"},
		LoginIDResolver: func(*http.Request) (string, bool) { return "user", true },
		CenterLogout:    func(*http.Request, string) error { centerCalls++; return nil },
	})
	values := url.Values{"client": {client.ClientID}, "clientSecret": {client.ClientSecret}, "ticket": {ticket.Ticket}, "redirect": {client.RedirectURIs[0]}}
	for _, wrongKey := range []string{"default-key", other.SecretKey} {
		signed := NewSigner(wrongKey).AttachSign(values)
		response := httptest.NewRecorder()
		h.HandleAuthorize(response, httptest.NewRequest(http.MethodGet, "/sso/authorize?"+signed.Encode(), nil))
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("authorize accepted wrong key: %d", response.Code)
		}
		for _, handler := range []http.HandlerFunc{h.HandleToken, h.HandleIntrospect, h.HandleUserInfo, h.HandleRevoke, h.HandleLogout} {
			if response := protocolReviewPost(handler, signed); response.Code != http.StatusUnauthorized {
				t.Fatalf("protocol accepted wrong key: %d, %s", response.Code, response.Body.String())
			}
		}
	}
	if centerCalls != 0 {
		t.Fatal("rejected signature invoked center logout")
	}
	if _, err := s.ConsumeTicket(context.Background(), ticket.Ticket, client.ClientID, client.ClientSecret, client.RedirectURIs[0]); err != nil {
		t.Fatalf("rejected signatures changed ticket state: %v", err)
	}
}

// TestLogoutUsesEachClientSigningKey verifies mixed dedicated and fallback keys in one logout fan-out. TestLogoutUsesEachClientSigningKey 验证一次注销同时使用各客户端独立密钥及默认密钥。
func TestLogoutUsesEachClientSigningKey(t *testing.T) {
	s := NewServer()
	defer s.Close()
	params := normalizeParamNames(ParamNames{Client: "application", Sign: "signature"})
	apps := make(map[string]*ClientApp)
	for _, clientID := range []string{"app-a", "app-b", "legacy"} {
		client := newTestClient()
		client.ClientID = clientID
		key := "default-key"
		if clientID != "legacy" {
			key = clientID + "-key"
			client.SecretKey = key
		}
		if err := s.RegisterClient(client); err != nil {
			t.Fatal(err)
		}
		if _, err := s.RegisterClientSession(context.Background(), "user", clientID, client.RedirectURIs[0]); err != nil {
			t.Fatal(err)
		}
		apps[clientID] = NewClientApp(ClientConfig{ClientID: clientID, CheckSign: true, SecretKey: key, Params: params})
	}
	var callbacks atomic.Int32
	h := NewHTTPServer(s, HTTPOptions{
		ServerOptions: ServerOptions{
			EnableSLO: true, CheckSign: true, SecretKey: "default-key", Params: params,
			LogoutHTTPClient: &http.Client{Transport: logoutReviewTransport(func(r *http.Request) (*http.Response, error) {
				if err := r.ParseForm(); err != nil {
					return nil, err
				}
				app := apps[r.Form.Get(params.Client)]
				if app == nil {
					return nil, errors.New("unknown callback client")
				}
				w := httptest.NewRecorder()
				app.LogoutCallbackHandler(func(_ *http.Request, callback LogoutCallback) error {
					if callback.LoginID != "user" {
						return errors.New("incorrect logout subject")
					}
					callbacks.Add(1)
					return nil
				})(w, r)
				return w.Result(), nil
			})},
		},
		LoginIDResolver: func(*http.Request) (string, bool) { return "user", true },
	})
	values := NewSignerWithParams("app-a-key", params).AttachSign(url.Values{params.Client: {"app-a"}})
	response := protocolReviewPost(h.HandleLogout, values)
	if response.Code != http.StatusOK || callbacks.Load() != 3 {
		t.Fatalf("logout = %d, callbacks = %d: %s", response.Code, callbacks.Load(), response.Body.String())
	}
}
