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

// TestEndpointConfiguration checks literal routing and decoded collisions before mux registration. TestEndpointConfiguration 检查路由注册前的字面路径和解码后冲突。
func TestEndpointConfiguration(t *testing.T) {
	for _, route := range []string{"/center/auth", "/center/%61uth", "/中心/auth"} {
		if err := (Endpoints{Authorize: route}).Validate(); err != nil {
			t.Fatalf("valid route %q: %v", route, err)
		}
	}
	for _, route := range []string{"/", "relative", "GET /auth", "/auth/", "//auth", "/a/../auth", "/a/%2e%2e/auth", "/{name}", "/{$}", "/%7Bname%7D", "/a?b", "/a#b", "/a%2fb", "/a%5Cb", "/a%00b", "/a b", "/%zz", "/sso/%74oken"} {
		t.Run(route, func(t *testing.T) {
			options := HTTPOptions{ServerOptions: ServerOptions{Endpoints: Endpoints{Authorize: route}}}
			if err := options.Endpoints.Validate(); !errors.Is(err, ErrSSOConfig) {
				t.Fatalf("route error = %v", err)
			}
			s := NewServer()
			defer s.Close()
			h := NewHTTPServer(s, options)
			if !errors.Is(h.InitializationError(), ErrSSOConfig) {
				t.Fatalf("retained error = %v", h.InitializationError())
			}
			rec := httptest.NewRecorder()
			h.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sso/token", nil))
			if rec.Code != http.StatusInternalServerError {
				t.Fatalf("invalid handler status = %d", rec.Code)
			}
			mux := http.NewServeMux()
			h.Register(mux)
			rec = httptest.NewRecorder()
			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sso/token", nil))
			if rec.Code != http.StatusNotFound {
				t.Fatalf("invalid routes partially registered: %d", rec.Code)
			}
		})
	}
	if err := (Endpoints{Message: "/sso/token", ClientLogout: "/sso/token"}).Validate(); err != nil {
		t.Fatalf("reserved routes must not conflict with active routes: %v", err)
	}
}

// configurationStorage counts registration side effects and storage ownership changes. configurationStorage 统计注册副作用及存储所有权变化。
type configurationStorage struct {
	*MemoryStorage
	writes int
	closed bool
}

// Set records writes while preserving ordinary storage behavior. Set 记录写入并保留普通存储行为。
func (s *configurationStorage) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	s.writes++
	return s.MemoryStorage.Set(ctx, key, value, ttl)
}

// Close records explicit storage closure. Close 记录显式存储关闭。
func (s *configurationStorage) Close() error {
	s.closed = true
	return nil
}

// configurationOptions supplies application-owned center identity for static checks. configurationOptions 为静态检查提供应用管理的中心身份。
func configurationOptions() HTTPOptions {
	return HTTPOptions{LoginIDResolver: func(*http.Request) (string, bool) { return "user", true }}
}

// TestCheckedStartupRejectsBeforeWrites verifies all inline registrations are validated before persistence. TestCheckedStartupRejectsBeforeWrites 验证所有内联注册均在持久化前完成校验。
func TestCheckedStartupRejectsBeforeWrites(t *testing.T) {
	storage := &configurationStorage{MemoryStorage: NewMemoryStorage()}
	defer storage.Close()
	s := NewServer(WithStorage(storage))
	defer s.Close()
	options := configurationOptions()
	options.Clients = map[string]Client{"good": {}, "bad": {RedirectURIs: []string{"javascript:secret-value"}}}
	if h, err := NewHTTPServerChecked(s, options); h != nil || !errors.Is(err, ErrInvalidRedirectURI) || !errors.Is(err, ErrSSOConfig) {
		t.Fatalf("checked startup = %v, %v", h, err)
	}
	if storage.writes != 0 || storage.closed || options.Clients["good"].ClientID != "" {
		t.Fatal("invalid startup wrote storage, closed it, or mutated caller registrations")
	}
	delete(options.Clients, "bad")
	if _, err := NewHTTPServerChecked(s, options); err != nil {
		t.Fatal(err)
	}
	if storage.writes != 1 || options.Clients["good"].ClientID != "" {
		t.Fatal("valid registration did not preserve caller map or persist once")
	}
	options.Clients = map[string]Client{"a": {ClientID: "same"}, "b": {ClientID: "same"}}
	if _, err := NewHTTPServerChecked(s, options); !errors.Is(err, ErrSSOConfig) || storage.writes != 1 {
		t.Fatalf("duplicate effective IDs were not rejected before writes: %v", err)
	}
}

// TestCheckedStartupFailures preserves storage errors and handles absent core objects. TestCheckedStartupFailures 保留存储错误并处理缺失的核心对象。
func TestCheckedStartupFailures(t *testing.T) {
	for _, s := range []*Server{nil, {}} {
		if _, err := NewHTTPServerChecked(s, configurationOptions()); !errors.Is(err, ErrServerNotInitialized) {
			t.Fatalf("uninitialized core: %v", err)
		}
		h := NewHTTPServer(s, HTTPOptions{ServerOptions: ServerOptions{Clients: map[string]Client{"app": {}}}})
		if !errors.Is(h.InitializationError(), ErrServerNotInitialized) {
			t.Fatalf("legacy uninitialized core: %v", h.InitializationError())
		}
		h.Register(http.NewServeMux())
	}
	var zero HTTPServer
	zero.Register(http.NewServeMux())
	storage := &registrationFailureStorage{MemoryStorage: NewMemoryStorage(), failWrite: true}
	s := NewServer(WithStorage(storage))
	defer s.Close()
	options := configurationOptions()
	options.Clients = map[string]Client{"app": {}}
	if h, err := NewHTTPServerChecked(s, options); h != nil || !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("write failure = %v, %v", h, err)
	}
	storage.failWrite = false
	if _, err := NewHTTPServerChecked(s, options); err != nil {
		t.Fatalf("caller-owned core must remain usable after failed startup: %v", err)
	}
}

// TestHTTPConfiguration checks keys, cookies, replay wiring, and normalized parameter conflicts. TestHTTPConfiguration 检查密钥、Cookie、防重放配置及补齐默认值后的参数冲突。
func TestHTTPConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*HTTPOptions)
	}{
		{"missing resolver", func(o *HTTPOptions) { o.LoginIDResolver = nil }},
		{"parameter collision", func(o *HTTPOptions) { o.Params.Client = DefaultParamNames().Ticket }},
		{"parameter control", func(o *HTTPOptions) { o.Params.Client = "client\n" }},
		{"missing signing key", func(o *HTTPOptions) { o.CheckSign = true; o.Clients = map[string]Client{"app": {}} }},
		{"anonymous signing", func(o *HTTPOptions) { o.CheckSign = true; o.AllowAnonymousClient = true }},
		{"anonymous callback", func(o *HTTPOptions) { o.AllowAnonymousClient = true; o.AllowURLs = []string{"//app.example/callback"} }},
		{"unsigned replay", func(o *HTTPOptions) { o.ReplayProtection = &ReplayProtection{} }},
		{"cookie none", func(o *HTTPOptions) { o.Cookie.SameSite = http.SameSiteNoneMode }},
		{"cookie host prefix", func(o *HTTPOptions) {
			o.Cookie.Name = "__Host-center"
			o.Cookie.Secure = true
			o.Cookie.Domain = "example.com"
		}},
		{"cookie path", func(o *HTTPOptions) { o.Cookie.Path = "relative" }},
		{"cookie domain", func(o *HTTPOptions) { o.Cookie.Domain = "invalid/domain" }},
		{"login redirect", func(o *HTTPOptions) { o.LoginPageURL = "//evil.example/login" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o := configurationOptions()
			tc.change(&o)
			if err := o.Validate(); !errors.Is(err, ErrSSOConfig) {
				t.Fatalf("configuration error = %v", err)
			}
		})
	}
	o := configurationOptions()
	o.CheckSign = true
	o.AllowAnonymousClient = true
	o.Clients = map[string]Client{"app": {SecretKey: "app-key"}, "alias": {ClientID: ClientAnonymous, SecretKey: "anon-key"}}
	o.Params.Client = "application"
	o.LoginPageURL = "/login?return=%2Fhome"
	if err := o.Validate(); err != nil {
		t.Fatalf("independent keys: %v", err)
	}
	o = HTTPOptions{Cookie: CookieOptions{SecretKey: "cookie-only"}}
	if err := o.Validate(); err != nil {
		t.Fatalf("signed center cookie: %v", err)
	}
}

// TestRegistrationConfiguration checks metadata before persistence while retaining the legacy API. TestRegistrationConfiguration 在持久化前检查元数据并保留旧 API。
func TestRegistrationConfiguration(t *testing.T) {
	storage := &configurationStorage{MemoryStorage: NewMemoryStorage()}
	defer storage.Close()
	s := NewServer(WithStorage(storage))
	defer s.Close()
	for _, client := range []*Client{
		nil, {}, {ClientID: "bad id"}, {ClientID: "app", Modes: []Mode{"unknown"}},
		{ClientID: "app", RedirectURIs: []string{"https://user:password@app.example/callback"}},
		{ClientID: "app", AllowOrigins: []string{"https://app.example/path"}},
		{ClientID: "app", Scopes: []string{"profile,email"}},
		{ClientID: "app", RequirePKCE: true, Modes: []Mode{ModeRemoteSession}},
	} {
		if err := s.RegisterClientChecked(client); !errors.Is(err, ErrSSOConfig) {
			t.Fatalf("invalid registration: %v", err)
		}
	}
	if storage.writes != 0 {
		t.Fatal("invalid registrations reached storage")
	}
	if err := s.RegisterClientChecked(&Client{ClientID: "app", RequirePKCE: true}); err != nil {
		t.Fatal(err)
	}
	if err := s.RegisterClient(&Client{ClientID: "legacy", Modes: []Mode{"future"}}); err != nil {
		t.Fatalf("legacy registration changed: %v", err)
	}
}

// TestClientConfigurationDiagnostics checks normalized defaults and bounded error reporting. TestClientConfigurationDiagnostics 检查默认值补齐及不回显敏感值的错误报告。
func TestClientConfigurationDiagnostics(t *testing.T) {
	base := ClientConfig{ClientID: "app", ServerURL: "https://center.example/gateway/%61pi"}
	app, err := NewClientAppChecked(base)
	if err != nil {
		t.Fatal(err)
	}
	if app.Config().Mode != ModeTicket || app.Config().MaxResponseBytes != DefaultClientMaxResponseBytes || base.Mode != "" {
		t.Fatal("defaults or caller config changed")
	}
	for _, address := range []string{"relative", "https://user:private-value@center.example", "https://center.example?key=private-value", "https://center.example/#private-value", "https://center.example/a%20b", "https://center.example/a%5Cb"} {
		cfg := base
		cfg.ServerURL = address
		_, err := NewClientAppChecked(cfg)
		var detail *ConfigurationError
		if !errors.Is(err, ErrSSOConfig) || !errors.As(err, &detail) || detail.Field != "ServerURL" || strings.Contains(err.Error(), "private-value") {
			t.Fatalf("bounded URL diagnostic = %v", err)
		}
	}
	base.LogoutCallbackURL = "invalid"
	if err := base.Validate(); err != nil {
		t.Fatalf("unused callback: %v", err)
	}
	base.RegisterCallback = true
	if err := base.Validate(); !errors.Is(err, ErrInvalidCallbackURL) {
		t.Fatalf("required callback = %v", err)
	}
	base.RegisterCallback = false
	base.CheckSign = true
	if err := base.Validate(); !errors.Is(err, ErrSignSecretRequired) {
		t.Fatalf("signing key = %v", err)
	}
}

// TestIntegrationConstructorsValidateDormantRoutes rejects configuration before browser state or HTTP calls. TestIntegrationConstructorsValidateDormantRoutes 在浏览器状态或 HTTP 请求前拒绝配置中的未使用路由错误。
func TestIntegrationConstructorsValidateDormantRoutes(t *testing.T) {
	storage := &configurationStorage{MemoryStorage: NewMemoryStorage()}
	defer storage.Close()
	app := NewClientApp(ClientConfig{ClientID: "app", ServerURL: "https://center.example", Mode: ModeRemoteSession, Endpoints: Endpoints{Revoke: "/{bad}"}, HTTPClient: &http.Client{Transport: logoutReviewTransport(func(*http.Request) (*http.Response, error) {
		t.Fatal("constructor contacted center")
		return nil, nil
	})}})
	if _, err := NewRemoteAuth(app, RemoteAuthOptions{}); !errors.Is(err, ErrSSOConfig) || !errors.Is(err, ErrRemoteAuthConfig) {
		t.Fatalf("remote error = %v", err)
	}
	if _, err := NewBrowserClient(app, BrowserClientOptions{Storage: storage, CallbackURL: "https://app.example/callback", StateCookie: CookieOptions{SecretKey: "state-key"}}); !errors.Is(err, ErrSSOConfig) || !errors.Is(err, ErrBrowserConfig) {
		t.Fatalf("browser error = %v", err)
	}
	if storage.writes != 0 || storage.closed {
		t.Fatal("constructor changed borrowed storage")
	}
}

// TestCheckedConfigurationSignedLifecycle exercises checked constructors through authorization and ticket consumption. TestCheckedConfigurationSignedLifecycle 通过授权和 Ticket 消费验证带校验的构造入口。
func TestCheckedConfigurationSignedLifecycle(t *testing.T) {
	s := NewServer()
	defer s.Close()
	client := newTestClient()
	client.SecretKey = "dedicated-key"
	options := configurationOptions()
	options.CheckSign = true
	options.Clients = map[string]Client{client.ClientID: *client}
	options.Endpoints.Authorize = "/center/authorize"
	options.Params.Client = "application"
	h, err := NewHTTPServerChecked(s, options)
	if err != nil {
		t.Fatal(err)
	}
	app, err := NewClientAppChecked(ClientConfig{ClientID: client.ClientID, ClientSecret: client.ClientSecret, SecretKey: client.SecretKey, CheckSign: true, ServerURL: "https://center.example", Endpoints: options.Endpoints, Params: options.Params, HTTPClient: &http.Client{Transport: logoutReviewTransport(func(r *http.Request) (*http.Response, error) {
		rec := httptest.NewRecorder()
		h.Handler().ServeHTTP(rec, r)
		return rec.Result(), nil
	})}})
	if err != nil {
		t.Fatal(err)
	}
	target, err := app.AuthURL(client.RedirectURIs[0], nil)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	h.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	if rec.Code != http.StatusFound {
		t.Fatalf("authorize: %d %s", rec.Code, rec.Body.String())
	}
	callback, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	ticket := callback.Query().Get(app.Config().Params.Ticket)
	result, err := app.ExchangeTicket(context.Background(), ticket, client.RedirectURIs[0])
	if err != nil || result == nil || result.LoginID != "user" {
		t.Fatalf("exchange = %v, %v", result, err)
	}
	if _, err := app.ExchangeTicket(context.Background(), ticket, client.RedirectURIs[0]); err == nil {
		t.Fatal("ticket replay accepted")
	}
}
