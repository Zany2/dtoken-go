package sso

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestSSOConfigValidationAndConstruction verifies config validation, cloning, constructor options, and client registration. TestSSOConfigValidationAndConstruction 验证配置校验、克隆、构造选项和客户端注册。
func TestSSOConfigValidationAndConstruction(t *testing.T) {
	if err := (*Config)(nil).Validate(); err != nil {
		t.Fatalf("nil Config.Validate() error = %v", err)
	}
	defaults := DefaultConfig()
	if err := defaults.Validate(); err != nil {
		t.Fatalf("DefaultConfig().Validate() error = %v", err)
	}
	clone := defaults.Clone()
	clone.TicketExpiration = 2 * defaults.TicketExpiration
	if defaults.TicketExpiration == clone.TicketExpiration {
		t.Fatal("Config.Clone() should return an independent copy")
	}

	invalidConfigs := map[string]*Config{}
	for _, name := range []string{"ticket", "shared-token", "remote-session", "oauth2-code"} {
		cfg := defaults.Clone()
		switch name {
		case "ticket":
			cfg.TicketExpiration = 0
		case "shared-token":
			cfg.SharedTokenExpiration = 0
		case "remote-session":
			cfg.RemoteSessionExpiration = 0
		case "oauth2-code":
			cfg.OAuth2CodeExpiration = 0
		}
		invalidConfigs[name] = cfg
	}
	for name, cfg := range invalidConfigs {
		if err := cfg.Validate(); err == nil {
			t.Fatalf("%s Config.Validate() error = nil, want validation error", name)
		}
	}

	storage := NewMemoryStorage()
	server := NewServer(
		nil,
		WithAuthType("review:"),
		WithKeyPrefix("tests:"),
		WithStorage(storage),
		WithCodec(JSONCodec{}),
		WithConfig(clone),
	)
	if server.authType != "review:" || server.keyPrefix != "tests:" || server.storage != storage || server.serializer.Name() != "json" {
		t.Fatalf("NewServer() dependencies = auth:%q prefix:%q storage:%T codec:%q", server.authType, server.keyPrefix, server.storage, server.serializer.Name())
	}
	if server.ticketExpiration != clone.TicketExpiration {
		t.Fatalf("NewServer() ticket expiration = %v, want %v", server.ticketExpiration, clone.TicketExpiration)
	}
	if err := server.RegisterClients(map[string]Client{
		"mapped-client": {
			RedirectURIs: []string{"https://mapped.example.com/callback"},
			Modes:        []Mode{ModeTicket},
		},
	}); err != nil {
		t.Fatalf("RegisterClients() error = %v", err)
	}
	client, err := server.GetClient("mapped-client")
	if err != nil || client.ClientID != "mapped-client" {
		t.Fatalf("GetClient(mapped-client) = %+v, %v", client, err)
	}

	direct := NewDefaultServer("direct:", "prefix:", nil, nil)
	defer direct.Close()
	if direct.storage == nil || direct.serializer == nil || !direct.storageOwned {
		t.Fatal("NewDefaultServer() should install owned default dependencies")
	}
}

// TestSSOClientBoundOperationsRejectEmptyClientID verifies client-bound operations fail before storage lookup. TestSSOClientBoundOperationsRejectEmptyClientID 验证客户端相关操作在存储查询前拒绝空客户端 ID。
func TestSSOClientBoundOperationsRejectEmptyClientID(t *testing.T) {
	server := newTestServer()
	ctx := context.Background()
	redirectURI := "https://app.example.com/sso/callback"

	checks := []struct {
		name string
		call func() error
	}{
		{name: "generate ticket", call: func() error {
			_, err := server.GenerateTicket(ctx, "", "user-1", redirectURI, nil, nil)
			return err
		}},
		{name: "generate shared token", call: func() error {
			_, err := server.GenerateSharedToken(ctx, "", "user-1", nil, nil)
			return err
		}},
		{name: "create remote session", call: func() error {
			_, err := server.CreateRemoteSession(ctx, "", "user-1", nil, nil)
			return err
		}},
		{name: "generate oauth2 code", call: func() error {
			_, err := server.GenerateOAuth2Code(ctx, "", "user-1", redirectURI, nil, nil)
			return err
		}},
		{name: "consume ticket", call: func() error {
			_, err := server.ConsumeTicket(ctx, "ticket", "", "", redirectURI)
			return err
		}},
		{name: "consume oauth2 code", call: func() error {
			_, err := server.ConsumeOAuth2Code(ctx, "code", "", "", redirectURI)
			return err
		}},
		{name: "validate shared token", call: func() error {
			_, err := server.ValidateSharedToken(ctx, "token", "")
			return err
		}},
		{name: "validate remote session", call: func() error {
			_, err := server.ValidateRemoteSession(ctx, "session", "")
			return err
		}},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			if err := check.call(); !errors.Is(err, ErrClientOrClientIDEmpty) {
				t.Fatalf("%s error = %v, want ErrClientOrClientIDEmpty", check.name, err)
			}
		})
	}
}

// TestSSOResultHelpersCloneCredentialData verifies result helpers preserve modes and isolate mutable metadata. TestSSOResultHelpersCloneCredentialData 验证结果辅助方法保留模式并隔离可变元数据。
func TestSSOResultHelpersCloneCredentialData(t *testing.T) {
	ticket := &Ticket{
		LoginID:  "user-1",
		ClientID: "app-a",
		Scopes:   []string{"profile"},
		Extra:    map[string]any{"scene": "web"},
	}
	result := TicketResult(ticket)
	info := TicketCredentialInfo(ticket, 30)
	result.Scopes[0] = "changed"
	result.Extra["scene"] = "changed"
	if ticket.Scopes[0] != "profile" || ticket.Extra["scene"] != "web" {
		t.Fatal("TicketResult() should clone slices and metadata")
	}
	if !info.Active || info.Mode != ModeTicket || info.ExpiresIn != 30 {
		t.Fatalf("TicketCredentialInfo() = %+v", info)
	}

	code := &OAuth2Code{LoginID: "user-2", ClientID: "app-b", Scopes: []string{"email"}}
	if got := OAuth2CodeResult(code); got == nil || got.LoginID != "user-2" {
		t.Fatalf("OAuth2CodeResult() = %+v", got)
	}
	credentialCases := []struct {
		name string
		mode Mode
		info *CredentialInfo
	}{
		{name: "shared token", mode: ModeSharedToken, info: SharedTokenCredentialInfo(&SharedToken{LoginID: "user", ClientID: "app"}, 20)},
		{name: "remote session", mode: ModeRemoteSession, info: RemoteSessionCredentialInfo(&RemoteSession{LoginID: "user", ClientID: "app"}, 20)},
		{name: "oauth2 code", mode: ModeOAuth2, info: OAuth2CodeCredentialInfo(code, 20)},
	}
	for _, test := range credentialCases {
		if !test.info.Active || test.info.Mode != test.mode || test.info.ExpiresIn != 20 {
			t.Fatalf("%s credential info = %+v", test.name, test.info)
		}
	}
	if TicketResult(nil) != nil || OAuth2CodeResult(nil) != nil || TicketCredentialInfo(nil, 0).Active || SharedTokenCredentialInfo(nil, 0).Active || RemoteSessionCredentialInfo(nil, 0).Active || OAuth2CodeCredentialInfo(nil, 0).Active {
		t.Fatal("nil result helpers should return nil or inactive credentials")
	}
}

// TestSSOAtomicOneTimeCredentialsHaveSingleWinner verifies concurrent Ticket and Code exchanges cannot replay. TestSSOAtomicOneTimeCredentialsHaveSingleWinner 验证并发交换 Ticket 和 Code 时只能成功一次。
func TestSSOAtomicOneTimeCredentialsHaveSingleWinner(t *testing.T) {
	ctx := context.Background()
	server := newTestServer()
	client := newTestClient()
	client.Modes = []Mode{ModeTicket, ModeOAuth2}
	if err := server.RegisterClient(client); err != nil {
		t.Fatalf("RegisterClient() error = %v", err)
	}

	ticket, err := server.GenerateTicket(ctx, client.ClientID, "ticket-user", client.RedirectURIs[0], nil, nil)
	if err != nil {
		t.Fatalf("GenerateTicket() error = %v", err)
	}
	assertSingleCredentialWinner(t, ErrInvalidTicket, func() error {
		consumed, err := server.ConsumeTicket(ctx, ticket.Ticket, client.ClientID, client.ClientSecret, client.RedirectURIs[0])
		if err == nil && (consumed == nil || !consumed.Used) {
			return errors.New("ticket result was not marked used")
		}
		return err
	})

	code, err := server.GenerateOAuth2Code(ctx, client.ClientID, "code-user", client.RedirectURIs[0], nil, nil)
	if err != nil {
		t.Fatalf("GenerateOAuth2Code() error = %v", err)
	}
	assertSingleCredentialWinner(t, ErrInvalidOAuth2Code, func() error {
		consumed, err := server.ConsumeOAuth2Code(ctx, code.Code, client.ClientID, client.ClientSecret, client.RedirectURIs[0])
		if err == nil && (consumed == nil || !consumed.Used) {
			return errors.New("OAuth2 code result was not marked used")
		}
		return err
	})
}

// TestSSOClientSessionRegistrationPropagatesIndexReadFailure verifies session registration does not overwrite an unreadable index. TestSSOClientSessionRegistrationPropagatesIndexReadFailure 验证客户端会话索引不可读时不会覆盖写入。
func TestSSOClientSessionRegistrationPropagatesIndexReadFailure(t *testing.T) {
	storageErr := errors.New("client session index unavailable")
	storage := &failingSessionIndexStorage{
		basicSSOStorage: &basicSSOStorage{inner: NewMemoryStorage()},
		err:             storageErr,
	}
	server := NewServer(WithStorage(storage))
	client := newTestClient()
	if err := server.RegisterClient(client); err != nil {
		t.Fatalf("RegisterClient() error = %v", err)
	}
	if _, err := server.RegisterClientSession(context.Background(), "user-1", client.ClientID, client.RedirectURIs[0]); !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("RegisterClientSession() error = %v, want ErrStorageUnavailable", err)
	}
}

// TestSSOBasicStorageConsumptionFallback verifies ordinary Storage implementations can consume one-time credentials sequentially. TestSSOBasicStorageConsumptionFallback 验证普通 Storage 实现可以顺序消费一次性凭证。
func TestSSOBasicStorageConsumptionFallback(t *testing.T) {
	ctx := context.Background()
	storage := &basicSSOStorage{inner: NewMemoryStorage()}
	server := NewServer(WithStorage(storage))
	client := newTestClient()
	client.Modes = []Mode{ModeTicket, ModeOAuth2}
	if err := server.RegisterClient(client); err != nil {
		t.Fatalf("RegisterClient() error = %v", err)
	}

	ticket, err := server.GenerateTicket(ctx, client.ClientID, "ticket-user", client.RedirectURIs[0], nil, nil)
	if err != nil {
		t.Fatalf("GenerateTicket() error = %v", err)
	}
	if consumed, err := server.ConsumeTicket(ctx, ticket.Ticket, client.ClientID, client.ClientSecret, client.RedirectURIs[0]); err != nil || consumed == nil || !consumed.Used {
		t.Fatalf("ConsumeTicket(basic storage) = %+v, %v", consumed, err)
	}
	if _, err = server.ConsumeTicket(ctx, ticket.Ticket, client.ClientID, client.ClientSecret, client.RedirectURIs[0]); !errors.Is(err, ErrInvalidTicket) {
		t.Fatalf("ConsumeTicket(basic storage, second) error = %v, want ErrInvalidTicket", err)
	}

	code, err := server.GenerateOAuth2Code(ctx, client.ClientID, "code-user", client.RedirectURIs[0], nil, nil)
	if err != nil {
		t.Fatalf("GenerateOAuth2Code() error = %v", err)
	}
	if consumed, err := server.ConsumeOAuth2Code(ctx, code.Code, client.ClientID, client.ClientSecret, client.RedirectURIs[0]); err != nil || consumed == nil || !consumed.Used {
		t.Fatalf("ConsumeOAuth2Code(basic storage) = %+v, %v", consumed, err)
	}
}

// TestSSOOAuth2CodeTTLAndRevoke verifies authorization-code TTL normalization and idempotent revocation. TestSSOOAuth2CodeTTLAndRevoke 验证授权码 TTL 归一化和幂等撤销。
func TestSSOOAuth2CodeTTLAndRevoke(t *testing.T) {
	ctx := context.Background()
	server := newTestServer()
	client := newTestClient()
	client.Modes = []Mode{ModeOAuth2}
	if err := server.RegisterClient(client); err != nil {
		t.Fatalf("RegisterClient() error = %v", err)
	}
	code, err := server.GenerateOAuth2Code(ctx, client.ClientID, "user-1", client.RedirectURIs[0], nil, nil)
	if err != nil {
		t.Fatalf("GenerateOAuth2Code() error = %v", err)
	}
	ttl, err := server.GetOAuth2CodeTTL(ctx, code.Code)
	if err != nil || ttl < 0 || ttl > 60 {
		t.Fatalf("GetOAuth2CodeTTL() = %d, %v, want 0..60 nil", ttl, err)
	}
	if err = server.RevokeOAuth2Code(ctx, code.Code); err != nil {
		t.Fatalf("RevokeOAuth2Code() error = %v", err)
	}
	if err = server.RevokeOAuth2Code(ctx, code.Code); err != nil {
		t.Fatalf("RevokeOAuth2Code() second error = %v", err)
	}
	if ttl, err = server.GetOAuth2CodeTTL(ctx, code.Code); err != nil || ttl != -2 {
		t.Fatalf("GetOAuth2CodeTTL(revoked) = %d, %v, want -2 nil", ttl, err)
	}
	if ttl, err = server.GetOAuth2CodeTTL(ctx, ""); err != nil || ttl != -2 {
		t.Fatalf("GetOAuth2CodeTTL(empty) = %d, %v, want -2 nil", ttl, err)
	}
}

// TestSSOSubsecondCredentialDurationRoundsUp verifies positive subsecond lifetimes remain representable in payload seconds. TestSSOSubsecondCredentialDurationRoundsUp 验证正数亚秒有效期在载荷秒数中向上取整。
func TestSSOSubsecondCredentialDurationRoundsUp(t *testing.T) {
	ctx := context.Background()
	server := newTestServer()
	registerTestClient(t, server)
	ticket, err := server.GenerateTicketWithTimeout(ctx, "app-a", "user-1", "https://app.example.com/sso/callback", nil, nil, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("GenerateTicketWithTimeout() error = %v", err)
	}
	if ticket.ExpiresIn != 1 {
		t.Fatalf("Ticket.ExpiresIn = %d, want 1", ticket.ExpiresIn)
	}
}

// TestClientAppAdditionalProtocolRoutes verifies remaining client helpers use the configured protocol routes. TestClientAppAdditionalProtocolRoutes 验证其余客户端辅助方法使用配置的协议路由。
func TestClientAppAdditionalProtocolRoutes(t *testing.T) {
	var requested []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested = append(requested, r.URL.Path)
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm() error = %v", err)
		}
		switch r.URL.Path {
		case "/sso/token":
			_ = json.NewEncoder(w).Encode(OKResponse(TicketExchangeResult{LoginID: "exchange-user"}))
		case "/sso/userinfo":
			_ = json.NewEncoder(w).Encode(OKResponse(CredentialInfo{Active: true, Mode: ModeSharedToken, LoginID: "info-user"}))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	defaults := DefaultClientConfig()
	if defaults.Mode != ModeTicket || defaults.Endpoints.Token == "" || defaults.Params.Ticket == "" {
		t.Fatalf("DefaultClientConfig() = %+v", defaults)
	}
	app := NewClientApp(ClientConfig{
		ClientID:     "app-a",
		ClientSecret: "secret-a",
		ServerURL:    server.URL,
		CheckSign:    false,
	})
	if cfg := app.Config(); cfg.Mode != ModeTicket || cfg.Endpoints.Token == "" || cfg.Params.Sign == "" {
		t.Fatalf("ClientApp.Config() = %+v", cfg)
	}

	ticketURL, err := app.ExchangeTicketURL("ticket-value", nil)
	if err != nil {
		t.Fatalf("ExchangeTicketURL() error = %v", err)
	}
	parsedTicketURL, err := url.Parse(ticketURL)
	if err != nil || parsedTicketURL.Query().Get("ticket") != "ticket-value" {
		t.Fatalf("ExchangeTicketURL() = %q, %v", ticketURL, err)
	}
	signoutURL, err := app.SignoutURL("user-1", nil)
	if err != nil {
		t.Fatalf("SignoutURL() error = %v", err)
	}
	parsedSignoutURL, err := url.Parse(signoutURL)
	if err != nil || parsedSignoutURL.Query().Get("loginId") != "user-1" {
		t.Fatalf("SignoutURL() = %q, %v", signoutURL, err)
	}

	exchanged, err := app.ExchangeOAuth2Code(context.Background(), "code-value", "https://app.example.com/callback")
	if err != nil || exchanged.LoginID != "exchange-user" {
		t.Fatalf("ExchangeOAuth2Code() = %+v, %v", exchanged, err)
	}
	exchanged, err = app.ExchangeCredential(context.Background(), CredentialRequest{Mode: ModeTicket, Ticket: "ticket-value"})
	if err != nil || exchanged.LoginID != "exchange-user" {
		t.Fatalf("ExchangeCredential() = %+v, %v", exchanged, err)
	}
	userInfo, err := app.UserInfo(context.Background(), CredentialRequest{Mode: ModeSharedToken, TokenValue: "token-value"})
	if err != nil || !userInfo.Active || userInfo.LoginID != "info-user" {
		t.Fatalf("UserInfo() = %+v, %v", userInfo, err)
	}
	if strings.Join(requested, ",") != "/sso/token,/sso/token,/sso/userinfo" {
		t.Fatalf("protocol request paths = %v", requested)
	}

	var nilApp *ClientApp
	if nilApp.Config().Mode != ModeTicket {
		t.Fatal("nil ClientApp.Config() should return defaults")
	}
	if _, err = nilApp.ExchangeCredential(context.Background(), CredentialRequest{}); !errors.Is(err, ErrServerNotInitialized) {
		t.Fatalf("nil ClientApp.ExchangeCredential() error = %v, want ErrServerNotInitialized", err)
	}
}

// TestHTTPServerHandlerRequiresConfiguredSign verifies signed UserInfo routing through the registered handler. TestHTTPServerHandlerRequiresConfiguredSign 验证已注册 UserInfo 路由会执行签名校验。
func TestHTTPServerHandlerRequiresConfiguredSign(t *testing.T) {
	serverDefaults := DefaultServerOptions()
	if serverDefaults.Mode != ModeTicket || !serverDefaults.CheckSign || serverDefaults.Endpoints.UserInfo == "" {
		t.Fatalf("DefaultServerOptions() = %+v", serverDefaults)
	}
	server := NewServer()
	client := newTestClient()
	client.Modes = []Mode{ModeSharedToken}
	if err := server.RegisterClient(client); err != nil {
		t.Fatalf("RegisterClient() error = %v", err)
	}
	token, err := server.GenerateSharedToken(context.Background(), client.ClientID, "user-1", nil, nil)
	if err != nil {
		t.Fatalf("GenerateSharedToken() error = %v", err)
	}

	options := DefaultHTTPOptions()
	options.ServerOptions.SecretKey = "sign-secret"
	handler := NewHTTPServer(server, options).Handler()
	values := url.Values{
		"mode":         {string(ModeSharedToken)},
		"client":       {client.ClientID},
		"clientSecret": {client.ClientSecret},
		"tokenValue":   {token.Token},
	}

	unsigned := httptest.NewRequest(http.MethodPost, options.Endpoints.UserInfo, strings.NewReader(values.Encode()))
	unsigned.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	unsignedRecorder := httptest.NewRecorder()
	handler.ServeHTTP(unsignedRecorder, unsigned)
	if unsignedRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("unsigned UserInfo status = %d, want 401", unsignedRecorder.Code)
	}

	signedValues := NewSigner("sign-secret").AttachSign(values)
	signed := httptest.NewRequest(http.MethodPost, options.Endpoints.UserInfo, strings.NewReader(signedValues.Encode()))
	signed.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	signedRecorder := httptest.NewRecorder()
	handler.ServeHTTP(signedRecorder, signed)
	if signedRecorder.Code != http.StatusOK {
		t.Fatalf("signed UserInfo status = %d, want 200, body=%s", signedRecorder.Code, signedRecorder.Body.String())
	}
}

// assertSingleCredentialWinner runs concurrent consume calls and requires exactly one success. assertSingleCredentialWinner 并发执行消费操作并要求仅一次成功。
func assertSingleCredentialWinner(t *testing.T, invalidErr error, consume func() error) {
	t.Helper()
	const workers = 16
	start := make(chan struct{})
	errorsByWorker := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errorsByWorker <- consume()
		}()
	}
	close(start)
	wg.Wait()
	close(errorsByWorker)

	successes := 0
	for err := range errorsByWorker {
		if err == nil {
			successes++
			continue
		}
		if !errors.Is(err, invalidErr) {
			t.Fatalf("concurrent consume error = %v, want %v", err, invalidErr)
		}
	}
	if successes != 1 {
		t.Fatalf("concurrent consume successes = %d, want 1", successes)
	}
}

// basicSSOStorage deliberately exposes only adapter.Storage behavior. basicSSOStorage 仅暴露 adapter.Storage 基础能力。
type basicSSOStorage struct {
	inner *MemoryStorage
}

// failingSessionIndexStorage injects a read failure for the client-session index. failingSessionIndexStorage 为客户端会话索引读取注入错误。
type failingSessionIndexStorage struct {
	*basicSSOStorage
	err error
}

func (s *failingSessionIndexStorage) Get(ctx context.Context, key string) (any, error) {
	if strings.Contains(key, ClientSessionKeySuffix) {
		return nil, s.err
	}
	return s.basicSSOStorage.Get(ctx, key)
}

func (s *basicSSOStorage) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	return s.inner.Set(ctx, key, value, expiration)
}

func (s *basicSSOStorage) Get(ctx context.Context, key string) (any, error) {
	return s.inner.Get(ctx, key)
}

func (s *basicSSOStorage) Delete(ctx context.Context, keys ...string) error {
	return s.inner.Delete(ctx, keys...)
}

func (s *basicSSOStorage) Exists(ctx context.Context, key string) bool {
	return s.inner.Exists(ctx, key)
}

func (s *basicSSOStorage) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return s.inner.Expire(ctx, key, expiration)
}

func (s *basicSSOStorage) TTL(ctx context.Context, key string) (time.Duration, error) {
	return s.inner.TTL(ctx, key)
}

func (s *basicSSOStorage) Ping(ctx context.Context) error {
	return s.inner.Ping(ctx)
}
