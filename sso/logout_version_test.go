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

	adapter "github.com/Zany2/dtoken-go/contract"
)

// logoutCredentials groups the four credential modes for lifecycle checks. logoutCredentials 汇总四种凭证以检查生命周期。
type logoutCredentials struct {
	ticket  *Ticket
	token   *SharedToken
	session *RemoteSession
	code    *OAuth2Code
}

// issueLogoutCredentials issues all supported modes for one account and client. issueLogoutCredentials 为一个账号与客户端签发所有支持的凭证模式。
func issueLogoutCredentials(t *testing.T, s *Server, client *Client, loginID string) logoutCredentials {
	t.Helper()
	ctx := context.Background()
	ticket, err := s.GenerateTicket(ctx, client.ClientID, loginID, client.RedirectURIs[0], nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	token, err := s.GenerateSharedToken(ctx, client.ClientID, loginID, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	session, err := s.CreateRemoteSession(ctx, client.ClientID, loginID, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	code, err := s.GenerateOAuth2Code(ctx, client.ClientID, loginID, client.RedirectURIs[0], nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	return logoutCredentials{ticket, token, session, code}
}

// assertLogoutCredentials checks read validity without consuming one-time credentials. assertLogoutCredentials 检查读取有效性，不消费一次性凭证。
func assertLogoutCredentials(t *testing.T, s *Server, c logoutCredentials, active bool) {
	t.Helper()
	ctx := context.Background()
	_, ticketErr := s.ValidateTicket(ctx, c.ticket.Ticket)
	_, tokenErr := s.ValidateSharedToken(ctx, c.token.Token, c.token.ClientID)
	_, sessionErr := s.ValidateRemoteSession(ctx, c.session.SessionID, c.session.ClientID)
	_, codeErr := s.getOAuth2Code(ctx, c.code.Code)
	for i, err := range []error{ticketErr, tokenErr, sessionErr, codeErr} {
		invalid := []error{ErrInvalidTicket, ErrInvalidSharedToken, ErrInvalidRemoteSession, ErrInvalidOAuth2Code}[i]
		if active && err != nil || !active && !errors.Is(err, invalid) {
			t.Fatalf("mode %d: error=%v, active=%v", i, err, active)
		}
	}
}

// TestLogoutVersionAcrossServers verifies account-wide revocation, namespace isolation, and ordinary storage support. TestLogoutVersionAcrossServers 验证跨实例账号撤销、命名空间隔离与普通存储支持。
func TestLogoutVersionAcrossServers(t *testing.T) {
	for _, basic := range []bool{false, true} {
		memory := NewMemoryStorage()
		var storage adapter.Storage = memory
		if basic {
			storage = &basicSSOStorage{inner: memory}
		}
		s := NewServer(WithStorage(storage))
		peer := NewServer(WithStorage(storage))
		isolated := NewServer(WithStorage(storage), WithAuthType("isolated:"))
		client := newTestClient()
		client.Modes = []Mode{ModeTicket, ModeSharedToken, ModeRemoteSession, ModeOAuth2}
		for _, server := range []*Server{s, isolated} {
			if err := server.RegisterClient(client); err != nil {
				t.Fatal(err)
			}
		}
		otherClient := *client
		otherClient.ClientID = "app-b"
		if err := s.RegisterClient(&otherClient); err != nil {
			t.Fatal(err)
		}
		old := issueLogoutCredentials(t, s, client, "user")
		otherApp := issueLogoutCredentials(t, s, &otherClient, "user")
		otherUser := issueLogoutCredentials(t, s, client, "other-user")
		otherNamespace := issueLogoutCredentials(t, isolated, client, "user")
		assertLogoutCredentials(t, peer, old, true)
		if err := peer.RevokeLoginCredentials(context.Background(), "user"); err != nil {
			t.Fatal(err)
		}
		assertLogoutCredentials(t, s, old, false)
		assertLogoutCredentials(t, s, otherApp, false)
		assertLogoutCredentials(t, s, otherUser, true)
		assertLogoutCredentials(t, isolated, otherNamespace, true)
		if _, err := s.ConsumeTicket(context.Background(), old.ticket.Ticket, client.ClientID, client.ClientSecret, client.RedirectURIs[0]); !errors.Is(err, ErrInvalidTicket) {
			t.Fatalf("revoked ticket exchange: %v", err)
		}
		if _, err := s.ConsumeOAuth2Code(context.Background(), old.code.Code, client.ClientID, client.ClientSecret, client.RedirectURIs[0]); !errors.Is(err, ErrInvalidOAuth2Code) {
			t.Fatalf("revoked code exchange: %v", err)
		}
		if err := s.RenewRemoteSession(context.Background(), old.session.SessionID, time.Hour); !errors.Is(err, ErrInvalidRemoteSession) {
			t.Fatalf("revoked session renewal: %v", err)
		}
		fresh := issueLogoutCredentials(t, s, client, "user")
		assertLogoutCredentials(t, peer, fresh, true)
		assertLogoutCredentials(t, peer, old, false)
		if err := s.RevokeLoginCredentials(context.Background(), "user"); err != nil {
			t.Fatal(err)
		}
		assertLogoutCredentials(t, peer, fresh, false)
	}
}

// TestLogoutVersionLegacyRecords verifies pre-upgrade credentials retain their original lifetime. TestLogoutVersionLegacyRecords 验证升级前凭证保留原有效期语义。
func TestLogoutVersionLegacyRecords(t *testing.T) {
	s := NewServer()
	defer s.Close()
	client := newTestClient()
	client.Modes = []Mode{ModeTicket, ModeSharedToken, ModeRemoteSession, ModeOAuth2}
	if err := s.RegisterClient(client); err != nil {
		t.Fatal(err)
	}
	c := issueLogoutCredentials(t, s, client, "user")
	c.ticket.LogoutVersion, c.token.LogoutVersion, c.session.LogoutVersion, c.code.LogoutVersion = "", "", "", ""
	ctx := context.Background()
	for _, err := range []error{s.saveTicket(ctx, c.ticket), s.saveSharedToken(ctx, c.token), s.saveRemoteSession(ctx, c.session), s.saveOAuth2Code(ctx, c.code), s.RevokeLoginCredentials(ctx, "user")} {
		if err != nil {
			t.Fatal(err)
		}
	}
	assertLogoutCredentials(t, s, c, true)
}

// TestHTTPAuthorizationBindingAndLogout verifies trusted metadata travels through issuance and logout invalidates every mode. TestHTTPAuthorizationBindingAndLogout 验证可信元数据经过签发传递，注销使所有模式失效。
func TestHTTPAuthorizationBindingAndLogout(t *testing.T) {
	for _, mode := range []Mode{ModeTicket, ModeSharedToken, ModeRemoteSession, ModeOAuth2} {
		t.Run(string(mode), func(t *testing.T) {
			s := NewServer()
			defer s.Close()
			client := newTestClient()
			client.Modes = []Mode{mode}
			if err := s.RegisterClient(client); err != nil {
				t.Fatal(err)
			}
			centerFails := true
			h := NewHTTPServer(s, HTTPOptions{
				LoginIDResolver: func(*http.Request) (string, bool) { return "user", true },
				AuthorizationExtra: func(_ *http.Request, clientID, loginID string) (map[string]any, error) {
					if clientID != client.ClientID || loginID != "user" {
						t.Fatal("untrusted authorization identity")
					}
					return map[string]any{"centerSessionID": "trusted-session", "deviceId": "browser"}, nil
				},
				EnrichExchangeResult: func(_ *http.Request, _ string, result *TicketExchangeResult) error {
					if result.Extra["centerSessionID"] != "trusted-session" {
						t.Fatal("lost center-session binding")
					}
					result.DeviceID = "browser"
					return nil
				},
				CenterLogout: func(*http.Request, string) error {
					if centerFails {
						return errors.New("center logout unavailable")
					}
					return nil
				},
			})
			issue := func() CredentialRequest {
				values := url.Values{"client": {client.ClientID}, "redirect": {client.RedirectURIs[0]}, "mode": {string(mode)}, "loginId": {"forged"}, "centerSessionID": {"forged"}}
				w := httptest.NewRecorder()
				h.HandleAuthorize(w, httptest.NewRequest(http.MethodGet, "/sso/authorize?"+values.Encode(), nil))
				if w.Code != http.StatusFound {
					t.Fatalf("authorize: %d %s", w.Code, w.Body.String())
				}
				location, err := url.Parse(w.Header().Get("Location"))
				if err != nil {
					t.Fatal(err)
				}
				q := location.Query()
				return CredentialRequest{Mode: mode, RedirectURI: client.RedirectURIs[0], Ticket: q.Get("ticket"), Code: q.Get("code"), TokenValue: q.Get("tokenValue"), SessionID: q.Get("sessionId")}
			}
			app := NewClientApp(ClientConfig{ClientID: client.ClientID, ClientSecret: client.ClientSecret, ServerURL: "https://center.example.com", HTTPClient: &http.Client{Transport: logoutReviewTransport(func(r *http.Request) (*http.Response, error) {
				w := httptest.NewRecorder()
				h.Handler().ServeHTTP(w, r)
				return w.Result(), nil
			})}})
			credential := issue()
			info, err := app.Introspect(context.Background(), credential)
			if err != nil || !info.Active || info.Extra["centerSessionID"] != "trusted-session" {
				t.Fatalf("initial info: %+v %v", info, err)
			}
			if mode == ModeTicket || mode == ModeOAuth2 {
				result, err := app.ExchangeCredential(context.Background(), issue())
				if err != nil || result.DeviceID != "browser" {
					t.Fatalf("exchange enrichment: %+v %v", result, err)
				}
			}
			if w := protocolReviewPost(h.HandleLogout, nil); w.Code != http.StatusInternalServerError {
				t.Fatalf("failed center logout: %d", w.Code)
			}
			info, err = app.Introspect(context.Background(), credential)
			if err != nil || !info.Active {
				t.Fatalf("failed logout revoked credential: %+v %v", info, err)
			}
			centerFails = false
			if w := protocolReviewPost(h.HandleLogout, nil); w.Code != http.StatusOK {
				t.Fatalf("logout: %d %s", w.Code, w.Body.String())
			}
			info, err = app.Introspect(context.Background(), credential)
			if err != nil || info.Active {
				t.Fatalf("revoked info: %+v %v", info, err)
			}
			if _, err := app.UserInfo(context.Background(), credential); err == nil {
				t.Fatal("revoked credential exposed user info")
			}
			if mode == ModeTicket || mode == ModeOAuth2 {
				if _, err := app.ExchangeCredential(context.Background(), credential); err == nil {
					t.Fatal("revoked credential exchanged")
				}
			}
		})
	}
}

// logoutVersionFailureStorage injects failures only for account revocation markers. logoutVersionFailureStorage 仅为账号撤销标记注入故障。
type logoutVersionFailureStorage struct {
	*MemoryStorage
	failRead, failWrite bool
}

// Get injects marker lookup errors. Get 注入标记读取错误。
func (s *logoutVersionFailureStorage) Get(ctx context.Context, key string) (any, error) {
	if s.failRead && strings.Contains(key, "sso:logout-version:") {
		return nil, errors.New("marker unavailable")
	}
	return s.MemoryStorage.Get(ctx, key)
}

// Set injects marker write errors. Set 注入标记写入错误。
func (s *logoutVersionFailureStorage) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	if s.failWrite && strings.Contains(key, "sso:logout-version:") {
		return errors.New("marker unavailable")
	}
	return s.MemoryStorage.Set(ctx, key, value, ttl)
}

// TestLogoutVersionStorageFailures verifies storage errors never masquerade as valid or inactive credentials. TestLogoutVersionStorageFailures 验证存储错误不会被当作有效或非活动凭证。
func TestLogoutVersionStorageFailures(t *testing.T) {
	storage := &logoutVersionFailureStorage{MemoryStorage: NewMemoryStorage()}
	s := NewServer(WithStorage(storage))
	client := newTestClient()
	client.Modes = []Mode{ModeTicket, ModeSharedToken, ModeRemoteSession, ModeOAuth2}
	if err := s.RegisterClient(client); err != nil {
		t.Fatal(err)
	}
	c := issueLogoutCredentials(t, s, client, "user")
	ctx := context.Background()
	storage.failRead = true
	if _, err := s.ValidateSharedToken(ctx, c.token.Token, client.ClientID); !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("validation: %v", err)
	}
	if _, err := s.GenerateTicket(ctx, client.ClientID, "user", client.RedirectURIs[0], nil, nil); !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("issue: %v", err)
	}
	h := NewHTTPServer(s, HTTPOptions{LoginIDResolver: func(*http.Request) (string, bool) { return "user", true }})
	w := protocolReviewPost(h.HandleIntrospect, url.Values{"mode": {string(ModeSharedToken)}, "client": {client.ClientID}, "clientSecret": {client.ClientSecret}, "tokenValue": {c.token.Token}})
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("introspection hid storage error: %d", w.Code)
	}
	storage.failRead, storage.failWrite = false, true
	w = protocolReviewPost(h.HandleLogout, nil)
	if w.Code != http.StatusInternalServerError || len(w.Result().Cookies()) != 0 {
		t.Fatalf("failed revocation reported success: %d", w.Code)
	}
	storage.failWrite = false
	assertLogoutCredentials(t, s, c, true)
	if err := s.RevokeLoginCredentials(ctx, ""); !errors.Is(err, ErrUserIDEmpty) {
		t.Fatalf("empty subject: %v", err)
	}
}

// logoutDuringTakeStorage revokes the account between preflight and atomic consumption. logoutDuringTakeStorage 在预检查与原子消费之间撤销账号。
type logoutDuringTakeStorage struct {
	*MemoryStorage
	beforeTake func() error
}

// GetAndDelete runs the interleaved logout before taking the stored credential. GetAndDelete 在获取存储凭证前执行交错的注销操作。
func (s *logoutDuringTakeStorage) GetAndDelete(ctx context.Context, key string) (any, error) {
	if err := s.beforeTake(); err != nil {
		return nil, err
	}
	return s.MemoryStorage.GetAndDelete(ctx, key)
}

// TestLogoutDuringAtomicConsumption verifies the actually consumed payload is checked against logout state. TestLogoutDuringAtomicConsumption 验证实际消费载荷仍会检查注销状态。
func TestLogoutDuringAtomicConsumption(t *testing.T) {
	storage := &logoutDuringTakeStorage{MemoryStorage: NewMemoryStorage()}
	s := NewServer(WithStorage(storage))
	client := newTestClient()
	client.Modes = []Mode{ModeTicket, ModeSharedToken, ModeRemoteSession, ModeOAuth2}
	if err := s.RegisterClient(client); err != nil {
		t.Fatal(err)
	}
	storage.beforeTake = func() error { return s.RevokeLoginCredentials(context.Background(), "user") }
	c := issueLogoutCredentials(t, s, client, "user")
	if _, err := s.ConsumeTicket(context.Background(), c.ticket.Ticket, client.ClientID, client.ClientSecret, client.RedirectURIs[0]); !errors.Is(err, ErrInvalidTicket) {
		t.Fatalf("ticket escaped interleaved logout: %v", err)
	}
	c = issueLogoutCredentials(t, s, client, "user")
	if _, err := s.ConsumeOAuth2Code(context.Background(), c.code.Code, client.ClientID, client.ClientSecret, client.RedirectURIs[0]); !errors.Is(err, ErrInvalidOAuth2Code) {
		t.Fatalf("code escaped interleaved logout: %v", err)
	}
}

// TestAuthorizationMetadataRejectsInvalidRequests verifies untrusted requests cannot invoke credential metadata resolution. TestAuthorizationMetadataRejectsInvalidRequests 验证不可信请求不会触发凭证元数据解析。
func TestAuthorizationMetadataRejectsInvalidRequests(t *testing.T) {
	s := NewServer()
	defer s.Close()
	client := newTestClient()
	if err := s.RegisterClient(client); err != nil {
		t.Fatal(err)
	}
	calls := 0
	h := NewHTTPServer(s, HTTPOptions{
		ServerOptions:   ServerOptions{EnableSLO: true},
		LoginIDResolver: func(*http.Request) (string, bool) { return "user", true },
		AuthorizationExtra: func(*http.Request, string, string) (map[string]any, error) {
			calls++
			return nil, errors.New("metadata unavailable")
		},
	})
	for _, invalid := range []url.Values{
		{"redirect": {"https://unregistered.example.com"}},
		{"scope": {"admin"}},
		{"mode": {"unsupported"}},
		{"callback": {"https://unregistered.example.com/logout"}},
	} {
		values := url.Values{"client": {client.ClientID}, "redirect": {client.RedirectURIs[0]}}
		for k, v := range invalid {
			values[k] = v
		}
		w := httptest.NewRecorder()
		h.HandleAuthorize(w, httptest.NewRequest(http.MethodGet, "/sso/authorize?"+values.Encode(), nil))
		if w.Code != http.StatusBadRequest || calls != 0 {
			t.Fatalf("invalid request reached metadata hook: %d, %d", w.Code, calls)
		}
	}
	w := httptest.NewRecorder()
	values := url.Values{"client": {client.ClientID}, "redirect": {client.RedirectURIs[0]}, "callback": {client.RedirectURIs[0]}}
	h.HandleAuthorize(w, httptest.NewRequest(http.MethodGet, "/sso/authorize?"+values.Encode(), nil))
	sessions, err := s.GetClientSessions(context.Background(), "user")
	if w.Code != http.StatusInternalServerError || calls != 1 || w.Header().Get("Location") != "" || err != nil || len(sessions) != 0 {
		t.Fatalf("metadata failure: status=%d calls=%d sessions=%v error=%v", w.Code, calls, sessions, err)
	}
}
