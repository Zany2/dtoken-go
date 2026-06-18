package context

import (
	stdctx "context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/manager"
	"github.com/Zany2/dtoken-go/core/nonce"
	"github.com/Zany2/dtoken-go/core/oauth2"
	"github.com/Zany2/dtoken-go/core/shortkey"
	"github.com/Zany2/dtoken-go/core/ticket"
)

// TestGetTokenValuePrecedence verifies header, bearer, cookie, query, and body lookup order. TestGetTokenValuePrecedence 验证 Header、Bearer、Cookie、Query 和 Body 的读取顺序。
func TestGetTokenValuePrecedence(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.TokenName = "X-Token"
	cfg.IsReadHeader = true
	cfg.IsReadCookie = true
	cfg.IsReadQuery = true
	cfg.IsReadBody = true
	mgr := manager.NewManager(cfg, nil, nil, nil, nil, nil, nil)

	req := &testRequestContext{
		headers: map[string]string{
			"X-Token":       " header-token ",
			"Authorization": "Bearer bearer-token",
		},
		cookies: map[string]string{"X-Token": "cookie-token"},
		queries: map[string]string{"X-Token": "query-token"},
		forms:   map[string]string{"X-Token": "body-token"},
	}
	if token := NewContext(req, mgr).GetTokenValue(); token != "header-token" {
		t.Fatalf("GetTokenValue() = %q, want header-token", token)
	}

	delete(req.headers, "X-Token")
	if token := NewContext(req, mgr).GetTokenValue(); token != "bearer-token" {
		t.Fatalf("GetTokenValue() = %q, want bearer-token", token)
	}

	delete(req.headers, "Authorization")
	if token := NewContext(req, mgr).GetTokenValue(); token != "cookie-token" {
		t.Fatalf("GetTokenValue() = %q, want cookie-token", token)
	}

	delete(req.cookies, "X-Token")
	if token := NewContext(req, mgr).GetTokenValue(); token != "query-token" {
		t.Fatalf("GetTokenValue() = %q, want query-token", token)
	}

	delete(req.queries, "X-Token")
	if token := NewContext(req, mgr).GetTokenValue(); token != "body-token" {
		t.Fatalf("GetTokenValue() = %q, want body-token", token)
	}
}

// TestGetTokenValueParsesConfiguredAuthorizationHeader verifies TokenName=Authorization still parses bearer. TestGetTokenValueParsesConfiguredAuthorizationHeader 验证 TokenName=Authorization 时仍解析 Bearer。
func TestGetTokenValueParsesConfiguredAuthorizationHeader(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.TokenName = "Authorization"
	cfg.IsReadHeader = true
	mgr := manager.NewManager(cfg, nil, nil, nil, nil, nil, nil)

	req := &testRequestContext{
		headers: map[string]string{
			"Authorization": "Bearer auth-token",
		},
	}

	if token := NewContext(req, mgr).GetTokenValue(); token != "auth-token" {
		t.Fatalf("GetTokenValue() = %q, want auth-token", token)
	}
}

// TestExtractBearerToken verifies bearer parsing compatibility. TestExtractBearerToken 验证 Bearer 解析兼容性。
func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name string
		auth string
		want string
	}{
		{name: "bearer", auth: "Bearer abc", want: "abc"},
		{name: "case insensitive", auth: "bearer abc", want: "abc"},
		{name: "empty bearer", auth: "Bearer", want: ""},
		{name: "empty bearer with spaces", auth: "Bearer   ", want: ""},
		{name: "raw compatibility", auth: "raw-token", want: "raw-token"},
		{name: "empty", auth: "  ", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractBearerToken(tt.auth); got != tt.want {
				t.Fatalf("extractBearerToken(%q) = %q, want %q", tt.auth, got, tt.want)
			}
		})
	}
}

// TestContextAuthAccessSessionTerminalFacades verifies core context helpers use the current token. TestContextAuthAccessSessionTerminalFacades 验证核心上下文快捷方法使用当前 Token。
func TestContextAuthAccessSessionTerminalFacades(t *testing.T) {
	ctx := stdctx.Background()
	dctx, req, mgr := newTestDTokenContext(t)

	token, err := dctx.Auth().Login(ctx, "ctx-user", "web", "browser-1")
	if err != nil {
		t.Fatalf("Auth.Login() error = %v", err)
	}
	req.headers[mgr.GetConfig().TokenName] = token

	if dctx.Auth().Value() != token {
		t.Fatalf("Auth.Value() = %q, want token", dctx.Auth().Value())
	}
	if !dctx.Auth().IsLogin(ctx) {
		t.Fatal("Auth.IsLogin() = false, want true")
	}
	loginID, err := dctx.Auth().GetLoginID(ctx)
	if err != nil {
		t.Fatalf("Auth.GetLoginID() error = %v", err)
	}
	if loginID != "ctx-user" {
		t.Fatalf("Auth.GetLoginID() = %q, want ctx-user", loginID)
	}
	device, err := dctx.Auth().GetDevice(ctx)
	if err != nil {
		t.Fatalf("Auth.GetDevice() error = %v", err)
	}
	if device != "web" {
		t.Fatalf("Auth.GetDevice() = %q, want web", device)
	}
	if err = dctx.Auth().RenewTimeout(ctx, time.Minute); err != nil {
		t.Fatalf("Auth.RenewTimeout() error = %v", err)
	}

	if err = dctx.Access().AddPermissions(ctx, []string{"ctx:read", "ctx:write"}); err != nil {
		t.Fatalf("Access.AddPermissions() error = %v", err)
	}
	if err = dctx.Access().AddRoles(ctx, []string{"ctx-admin", "ctx-editor"}); err != nil {
		t.Fatalf("Access.AddRoles() error = %v", err)
	}
	if !dctx.Access().HasPermission(ctx, "ctx:read") {
		t.Fatal("Access.HasPermission(ctx:read) = false, want true")
	}
	if !dctx.Access().HasPermissionsAnd(ctx, []string{"ctx:read", "ctx:write"}) {
		t.Fatal("Access.HasPermissionsAnd() = false, want true")
	}
	if err = dctx.Access().CheckRolesOr(ctx, []string{"missing", "ctx-admin"}); err != nil {
		t.Fatalf("Access.CheckRolesOr() error = %v", err)
	}
	if err = dctx.Access().RemovePermissions(ctx, []string{"ctx:write"}); err != nil {
		t.Fatalf("Access.RemovePermissions() error = %v", err)
	}
	if dctx.Access().HasPermission(ctx, "ctx:write") {
		t.Fatal("Access.HasPermission(ctx:write) = true after remove, want false")
	}

	if err = dctx.Session().SetValue(ctx, "theme", "dark"); err != nil {
		t.Fatalf("Session.SetValue() error = %v", err)
	}
	value, ok, err := dctx.Session().GetValue(ctx, "theme")
	if err != nil {
		t.Fatalf("Session.GetValue() error = %v", err)
	}
	if !ok || value != "dark" {
		t.Fatalf("Session.GetValue() = %v, %v, want dark, true", value, ok)
	}
	if err = dctx.Session().DeleteValue(ctx, "theme"); err != nil {
		t.Fatalf("Session.DeleteValue() error = %v", err)
	}

	tokens, err := dctx.Terminal().GetTokenValueList(ctx, true)
	if err != nil {
		t.Fatalf("Terminal.GetTokenValueList() error = %v", err)
	}
	if !sameContextStrings(tokens, []string{token}) {
		t.Fatalf("Terminal.GetTokenValueList() = %v, want [%s]", tokens, token)
	}
	count, err := dctx.Terminal().GetOnlineTerminalCountByDevice(ctx, "web")
	if err != nil {
		t.Fatalf("Terminal.GetOnlineTerminalCountByDevice() error = %v", err)
	}
	if count != 1 {
		t.Fatalf("Terminal.GetOnlineTerminalCountByDevice() = %d, want 1", count)
	}
	terminalInfo, err := dctx.Terminal().GetTerminalInfo(ctx)
	if err != nil {
		t.Fatalf("Terminal.GetTerminalInfo() error = %v", err)
	}
	if terminalInfo.Token != token || terminalInfo.Device != "web" {
		t.Fatalf("Terminal.GetTerminalInfo() = %+v, want current web terminal", terminalInfo)
	}

	info, err := dctx.Auth().IntrospectToken(ctx)
	if err != nil {
		t.Fatalf("Auth.IntrospectToken() error = %v", err)
	}
	if !info.Active || info.LoginID != "ctx-user" {
		t.Fatalf("Auth.IntrospectToken() = %+v, want active ctx-user", info)
	}
	if err = dctx.Terminal().Logout(ctx); err != nil {
		t.Fatalf("Terminal.Logout() error = %v", err)
	}
	if dctx.Auth().IsLogin(ctx) {
		t.Fatal("Auth.IsLogin() after logout = true, want false")
	}
}

// TestContextCookieAndOptionalFacades verifies cookie and optional module facades. TestContextCookieAndOptionalFacades 验证 Cookie 与可选模块快捷入口。
func TestContextCookieAndOptionalFacades(t *testing.T) {
	ctx := stdctx.Background()
	dctx, req, mgr := newTestDTokenContext(t)
	enableContextOptionalManagers(mgr)

	token, err := dctx.Cookie().Login(ctx, "cookie-user", "app")
	if err != nil {
		t.Fatalf("Cookie.Login() error = %v", err)
	}
	if req.cookie == nil || req.cookie.Name != mgr.GetConfig().TokenName || req.cookie.Value != token {
		t.Fatalf("Cookie.Login() cookie = %+v, want token cookie", req.cookie)
	}
	req.headers[mgr.GetConfig().TokenName] = token

	nonceValue, err := dctx.Nonce().Generate(ctx)
	if err != nil {
		t.Fatalf("Nonce.Generate() error = %v", err)
	}
	if !dctx.Nonce().IsValid(ctx, nonceValue) {
		t.Fatal("Nonce.IsValid() = false, want true")
	}
	if err = dctx.Nonce().VerifyAndConsume(ctx, nonceValue); err != nil {
		t.Fatalf("Nonce.VerifyAndConsume() error = %v", err)
	}

	createdTicket, err := dctx.Ticket().CreateForCurrentLogin(ctx, ticket.CreateOptions{TargetApp: "admin"})
	if err != nil {
		t.Fatalf("Ticket.CreateForCurrentLogin() error = %v", err)
	}
	if createdTicket.LoginID != "cookie-user" {
		t.Fatalf("Ticket loginID = %q, want cookie-user", createdTicket.LoginID)
	}
	if _, err = dctx.Ticket().Consume(ctx, createdTicket.Ticket, ticket.ValidateOptions{LoginID: "cookie-user"}); err != nil {
		t.Fatalf("Ticket.Consume() error = %v", err)
	}

	createdShortKey, err := dctx.ShortKey().Create(ctx, shortkey.CreateOptions{TargetApp: "admin"})
	if err != nil {
		t.Fatalf("ShortKey.Create() error = %v", err)
	}
	if _, err = dctx.ShortKey().ConfirmForCurrentLogin(ctx, createdShortKey.Key, shortkey.ConfirmOptions{Device: "app"}); err != nil {
		t.Fatalf("ShortKey.ConfirmForCurrentLogin() error = %v", err)
	}
	if _, err = dctx.ShortKey().Consume(ctx, createdShortKey.Key, shortkey.ValidateOptions{LoginID: "cookie-user"}); err != nil {
		t.Fatalf("ShortKey.Consume() error = %v", err)
	}

	refreshPair, err := dctx.Refresh().Login(ctx, "refresh-context-user", "web")
	if err != nil {
		t.Fatalf("Refresh.Login() error = %v", err)
	}
	if refreshPair.AccessToken == "" || refreshPair.RefreshToken == "" {
		t.Fatalf("Refresh.Login() pair = %+v, want non-empty tokens", refreshPair)
	}
	if _, err = dctx.Refresh().Refresh(ctx, refreshPair.RefreshToken); err != nil {
		t.Fatalf("Refresh.Refresh() error = %v", err)
	}

	client := &oauth2.Client{
		ClientID:     "ctx-client",
		ClientSecret: "secret",
		GrantTypes:   []oauth2.GrantType{oauth2.GrantTypeClientCredentials},
		Scopes:       []string{"read"},
	}
	if err = dctx.OAuth2().RegisterClient(client); err != nil {
		t.Fatalf("OAuth2.RegisterClient() error = %v", err)
	}
	oauthToken, err := dctx.OAuth2().ClientCredentialsToken(ctx, client.ClientID, client.ClientSecret, []string{"read"})
	if err != nil {
		t.Fatalf("OAuth2.ClientCredentialsToken() error = %v", err)
	}
	if !dctx.OAuth2().ValidateAccessToken(ctx, oauthToken.Token) {
		t.Fatal("OAuth2.ValidateAccessToken() = false, want true")
	}

	if err = dctx.Cookie().Logout(ctx); err != nil {
		t.Fatalf("Cookie.Logout() error = %v", err)
	}
	if req.cookie == nil || req.cookie.Value != "" || req.cookie.MaxAge != -1 {
		t.Fatalf("Cookie.Logout() cookie = %+v, want cleared cookie", req.cookie)
	}
}

// TestContextNoTokenErrors verifies current-token helpers fail consistently without a token. TestContextNoTokenErrors 验证无 Token 时快捷方法一致返回未登录。
func TestContextNoTokenErrors(t *testing.T) {
	ctx := stdctx.Background()
	dctx, _, _ := newTestDTokenContext(t)

	if dctx.Auth().IsLogin(ctx) {
		t.Fatal("Auth.IsLogin() without token = true, want false")
	}
	if err := dctx.Auth().CheckLogin(ctx); !errors.Is(err, derror.ErrNotLogin) {
		t.Fatalf("Auth.CheckLogin() error = %v, want ErrNotLogin", err)
	}
	if _, err := dctx.Auth().GetLoginID(ctx); !errors.Is(err, derror.ErrNotLogin) {
		t.Fatalf("Auth.GetLoginID() error = %v, want ErrNotLogin", err)
	}
	if err := dctx.Access().CheckPermission(ctx, "ctx:read"); !errors.Is(err, derror.ErrNotLogin) {
		t.Fatalf("Access.CheckPermission() error = %v, want ErrNotLogin", err)
	}
	if dctx.Access().HasRole(ctx, "admin") {
		t.Fatal("Access.HasRole() without token = true, want false")
	}
	if _, err := dctx.Session().Get(ctx); !errors.Is(err, derror.ErrNotLogin) {
		t.Fatalf("Session.Get() error = %v, want ErrNotLogin", err)
	}
	if err := dctx.Terminal().LogoutAll(ctx); !errors.Is(err, derror.ErrNotLogin) {
		t.Fatalf("Terminal.LogoutAll() error = %v, want ErrNotLogin", err)
	}
	if err := dctx.Disable().Account(ctx, time.Minute); !errors.Is(err, derror.ErrNotLogin) {
		t.Fatalf("Disable.Account() error = %v, want ErrNotLogin", err)
	}
	if err := dctx.Disable().Service(ctx, "billing", time.Minute); !errors.Is(err, derror.ErrNotLogin) {
		t.Fatalf("Disable.Service() error = %v, want ErrNotLogin", err)
	}
	if err := dctx.Disable().DeviceAndDeviceId(ctx, "web", "browser-1", time.Minute); !errors.Is(err, derror.ErrNotLogin) {
		t.Fatalf("Disable.DeviceAndDeviceId() error = %v, want ErrNotLogin", err)
	}
	if _, err := dctx.Ticket().CreateForCurrentLogin(ctx, ticket.CreateOptions{}); !errors.Is(err, derror.ErrNotLogin) {
		t.Fatalf("Ticket.CreateForCurrentLogin() error = %v, want ErrNotLogin", err)
	}
	if _, err := dctx.ShortKey().ConfirmForCurrentLogin(ctx, "short-key", shortkey.ConfirmOptions{}); !errors.Is(err, derror.ErrNotLogin) {
		t.Fatalf("ShortKey.ConfirmForCurrentLogin() error = %v, want ErrNotLogin", err)
	}
}

// TestGetTokenValueReadSwitches verifies token sources respect config switches. TestGetTokenValueReadSwitches 验证 Token 来源读取开关生效。
func TestGetTokenValueReadSwitches(t *testing.T) {
	tests := []struct {
		name   string
		config func(*config.Config)
		want   string
	}{
		{
			name: "header disabled falls back to cookie",
			config: func(cfg *config.Config) {
				cfg.IsReadHeader = false
				cfg.IsReadCookie = true
			},
			want: "cookie-token",
		},
		{
			name: "cookie disabled falls back to query",
			config: func(cfg *config.Config) {
				cfg.IsReadHeader = false
				cfg.IsReadCookie = false
				cfg.IsReadQuery = true
			},
			want: "query-token",
		},
		{
			name: "query disabled falls back to body",
			config: func(cfg *config.Config) {
				cfg.IsReadHeader = false
				cfg.IsReadCookie = false
				cfg.IsReadQuery = false
				cfg.IsReadBody = true
			},
			want: "body-token",
		},
		{
			name: "all disabled returns empty",
			config: func(cfg *config.Config) {
				cfg.IsReadHeader = false
				cfg.IsReadCookie = false
				cfg.IsReadQuery = false
				cfg.IsReadBody = false
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.TokenName = "X-Token"
			tt.config(cfg)
			mgr := manager.NewManager(cfg, nil, nil, nil, nil, nil, nil)

			req := &testRequestContext{
				headers: map[string]string{"X-Token": "header-token"},
				cookies: map[string]string{"X-Token": "cookie-token"},
				queries: map[string]string{"X-Token": "query-token"},
				forms:   map[string]string{"X-Token": "body-token"},
			}
			if token := NewContext(req, mgr).GetTokenValue(); token != tt.want {
				t.Fatalf("GetTokenValue() = %q, want %q", token, tt.want)
			}
		})
	}
}

// TestContextCookieOptionsAndLoginVariants verifies cookie options and login shortcuts. TestContextCookieOptionsAndLoginVariants 验证 Cookie 配置和登录快捷方法。
func TestContextCookieOptionsAndLoginVariants(t *testing.T) {
	ctx := stdctx.Background()
	dctx, req, mgr := newTestDTokenContext(t)
	cfg := mgr.GetConfig()
	cfg.TokenName = "CtxCookie"
	cfg.CookieConfig = &config.CookieConfig{
		Domain:   "example.com",
		Path:     "/api",
		Secure:   true,
		HttpOnly: false,
		SameSite: config.SameSiteNone,
		MaxAge:   3600,
	}

	dctx.Cookie().SetToken("manual-token")
	assertContextCookie(t, req.cookie, "CtxCookie", "manual-token", 3600, "/api", "example.com", true, false, string(config.SameSiteNone))

	dctx.Cookie().ClearToken()
	assertContextCookie(t, req.cookie, "CtxCookie", "", -1, "/api", "example.com", true, false, string(config.SameSiteNone))

	token, err := dctx.Cookie().LoginWithTimeout(ctx, "cookie-timeout", time.Minute, "mobile", "phone-1")
	if err != nil {
		t.Fatalf("Cookie.LoginWithTimeout() error = %v", err)
	}
	assertContextCookie(t, req.cookie, "CtxCookie", token, 3600, "/api", "example.com", true, false, string(config.SameSiteNone))
	req.headers[cfg.TokenName] = token
	if deviceID, err := dctx.Auth().GetDeviceId(ctx); err != nil || deviceID != "phone-1" {
		t.Fatalf("Auth.GetDeviceId() = %q, %v, want phone-1, nil", deviceID, err)
	}
	if ttl, err := dctx.Auth().GetTokenTTL(ctx); err != nil || ttl <= 0 || ttl > 60 {
		t.Fatalf("Auth.GetTokenTTL() = %d, %v, want 1..60 seconds", ttl, err)
	}

	optionToken, err := dctx.Cookie().LoginWithOptions(ctx, manager.LoginOptions{
		LoginID:  "cookie-options",
		Device:   "api",
		DeviceID: "api-1",
		Timeout:  2 * time.Minute,
		Extra:    map[string]any{"source": "context-test"},
	})
	if err != nil {
		t.Fatalf("Cookie.LoginWithOptions() error = %v", err)
	}
	assertContextCookie(t, req.cookie, "CtxCookie", optionToken, 3600, "/api", "example.com", true, false, string(config.SameSiteNone))
	req.headers[cfg.TokenName] = optionToken
	info, err := dctx.Auth().GetTokenInfo(ctx)
	if err != nil {
		t.Fatalf("Auth.GetTokenInfo() error = %v", err)
	}
	if info.LoginID != "cookie-options" || info.Device != "api" || info.DeviceId != "api-1" {
		t.Fatalf("Auth.GetTokenInfo() = %+v, want cookie-options/api/api-1", info)
	}
	if err = dctx.Auth().LoginByToken(ctx); err != nil {
		t.Fatalf("Auth.LoginByToken() error = %v", err)
	}
}

// TestContextDisableFacades verifies account, service, and device disable helpers. TestContextDisableFacades 验证账号、服务和设备封禁快捷方法。
func TestContextDisableFacades(t *testing.T) {
	ctx := stdctx.Background()
	dctx, req, mgr := newTestDTokenContext(t)

	token, err := dctx.Auth().Login(ctx, "disable-user", "web", "browser-1")
	if err != nil {
		t.Fatalf("Auth.Login() error = %v", err)
	}
	req.headers[mgr.GetConfig().TokenName] = token

	if err = dctx.Disable().Account(ctx, time.Minute, "risk"); err != nil {
		t.Fatalf("Disable.Account() error = %v", err)
	}
	if !mgr.IsDisable(ctx, "disable-user") {
		t.Fatal("manager.IsDisable() = false, want true")
	}
	if err = mgr.CheckDisable(ctx, "disable-user"); !errors.Is(err, derror.ErrAccountDisabled) {
		t.Fatalf("manager.CheckDisable() error = %v, want ErrAccountDisabled", err)
	}
	accountInfo, err := mgr.GetDisableInfo(ctx, "disable-user")
	if err != nil {
		t.Fatalf("manager.GetDisableInfo() error = %v", err)
	}
	if accountInfo.DisableReason != "risk" {
		t.Fatalf("manager.GetDisableInfo().DisableReason = %q, want risk", accountInfo.DisableReason)
	}
	if ttl, err := mgr.GetDisableTTL(ctx, "disable-user"); err != nil || ttl <= 0 {
		t.Fatalf("manager.GetDisableTTL() = %d, %v, want positive ttl", ttl, err)
	}
	if err = mgr.Untie(ctx, "disable-user"); err != nil {
		t.Fatalf("manager.Untie() error = %v", err)
	}

	token, err = dctx.Auth().Login(ctx, "disable-user", "web", "browser-1")
	if err != nil {
		t.Fatalf("Auth.Login(after account untie) error = %v", err)
	}
	req.headers[mgr.GetConfig().TokenName] = token
	if !dctx.Auth().IsLogin(ctx) {
		t.Fatal("Auth.IsLogin() after re-login = false, want true")
	}

	if err = dctx.Disable().ServiceLevel(ctx, "billing", 3, time.Minute, "quota"); err != nil {
		t.Fatalf("Disable.ServiceLevel() error = %v", err)
	}
	if !dctx.Disable().IsService(ctx, "billing") || !dctx.Disable().IsServiceLevel(ctx, "billing", 2) {
		t.Fatal("Disable service checks = false, want true")
	}
	if err = dctx.Disable().CheckService(ctx, "billing"); !errors.Is(err, derror.ErrServiceDisabled) {
		t.Fatalf("Disable.CheckService() error = %v, want ErrServiceDisabled", err)
	}
	serviceInfo, err := dctx.Disable().GetServiceInfo(ctx, "billing")
	if err != nil {
		t.Fatalf("Disable.GetServiceInfo() error = %v", err)
	}
	if serviceInfo.Service != "billing" || serviceInfo.Level != 3 || serviceInfo.DisableReason != "quota" {
		t.Fatalf("Disable.GetServiceInfo() = %+v, want billing level 3 quota", serviceInfo)
	}
	if ttl, err := dctx.Disable().GetServiceTTL(ctx, "billing"); err != nil || ttl <= 0 {
		t.Fatalf("Disable.GetServiceTTL() = %d, %v, want positive ttl", ttl, err)
	}
	if err = dctx.Disable().UntieService(ctx, "billing"); err != nil {
		t.Fatalf("Disable.UntieService() error = %v", err)
	}

	if err = dctx.Disable().DeviceAndDeviceId(ctx, "web", "browser-1", time.Minute, "lost"); err != nil {
		t.Fatalf("Disable.DeviceAndDeviceId() error = %v", err)
	}
	if !mgr.IsDisableDeviceAndDeviceId(ctx, "disable-user", "web", "browser-1") {
		t.Fatal("manager.IsDisableDeviceAndDeviceId() = false, want true")
	}
	if err = mgr.CheckDisableDeviceAndDeviceId(ctx, "disable-user", "web", "browser-1"); !errors.Is(err, derror.ErrDeviceDisabled) {
		t.Fatalf("manager.CheckDisableDeviceAndDeviceId() error = %v, want ErrDeviceDisabled", err)
	}
	deviceInfo, err := mgr.GetDisableDeviceAndDeviceIdInfo(ctx, "disable-user", "web", "browser-1")
	if err != nil {
		t.Fatalf("manager.GetDisableDeviceAndDeviceIdInfo() error = %v", err)
	}
	if deviceInfo.Device != "web" || deviceInfo.DeviceId != "browser-1" || deviceInfo.DisableReason != "lost" {
		t.Fatalf("manager.GetDisableDeviceAndDeviceIdInfo() = %+v, want web/browser-1 lost", deviceInfo)
	}
	if ttl, err := mgr.GetDisableDeviceAndDeviceIdTTL(ctx, "disable-user", "web", "browser-1"); err != nil || ttl <= 0 {
		t.Fatalf("manager.GetDisableDeviceAndDeviceIdTTL() = %d, %v, want positive ttl", ttl, err)
	}
	if err = mgr.UntieDeviceAndDeviceId(ctx, "disable-user", "web", "browser-1"); err != nil {
		t.Fatalf("manager.UntieDeviceAndDeviceId() error = %v", err)
	}
}

// TestContextTerminalFacades verifies terminal list, search, visit, and terminate helpers. TestContextTerminalFacades 验证终端列表、搜索、遍历和下线快捷方法。
func TestContextTerminalFacades(t *testing.T) {
	ctx := stdctx.Background()
	dctx, req, mgr := newTestDTokenContext(t)

	token1, err := dctx.Auth().Login(ctx, "terminal-user", "web", "browser-1")
	if err != nil {
		t.Fatalf("Auth.Login(token1) error = %v", err)
	}
	token2, err := dctx.Auth().Login(ctx, "terminal-user", "web", "browser-2")
	if err != nil {
		t.Fatalf("Auth.Login(token2) error = %v", err)
	}
	token3, err := dctx.Auth().Login(ctx, "terminal-user", "app", "phone-1")
	if err != nil {
		t.Fatalf("Auth.Login(token3) error = %v", err)
	}
	req.headers[mgr.GetConfig().TokenName] = token1

	tokens, err := dctx.Terminal().GetTokenValueList(ctx, true)
	if err != nil {
		t.Fatalf("Terminal.GetTokenValueList() error = %v", err)
	}
	if !sameContextStrings(tokens, []string{token1, token2, token3}) {
		t.Fatalf("Terminal.GetTokenValueList() = %v, want all tokens", tokens)
	}
	webTokens, err := dctx.Terminal().GetTokenValueListByDevice(ctx, "web", true)
	if err != nil {
		t.Fatalf("Terminal.GetTokenValueListByDevice() error = %v", err)
	}
	if !sameContextStrings(webTokens, []string{token1, token2}) {
		t.Fatalf("Terminal.GetTokenValueListByDevice() = %v, want web tokens", webTokens)
	}
	oneToken, err := dctx.Terminal().GetTokenValueListByDeviceAndDeviceId(ctx, "web", "browser-1", true)
	if err != nil {
		t.Fatalf("Terminal.GetTokenValueListByDeviceAndDeviceId() error = %v", err)
	}
	if !sameContextStrings(oneToken, []string{token1}) {
		t.Fatalf("Terminal.GetTokenValueListByDeviceAndDeviceId() = %v, want token1", oneToken)
	}

	count, err := dctx.Terminal().GetOnlineTerminalCount(ctx)
	if err != nil || count != 3 {
		t.Fatalf("Terminal.GetOnlineTerminalCount() = %d, %v, want 3, nil", count, err)
	}
	if count, err = dctx.Terminal().GetOnlineTerminalCountByDeviceAndDeviceId(ctx, "web", "browser-2"); err != nil || count != 1 {
		t.Fatalf("Terminal.GetOnlineTerminalCountByDeviceAndDeviceId() = %d, %v, want 1, nil", count, err)
	}

	list, err := dctx.Terminal().GetTerminalList(ctx, "web")
	if err != nil {
		t.Fatalf("Terminal.GetTerminalList() error = %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("Terminal.GetTerminalList(web) len = %d, want 2", len(list))
	}
	latest, err := dctx.Terminal().GetLatestTokenValue(ctx, "web")
	if err != nil {
		t.Fatalf("Terminal.GetLatestTokenValue() error = %v", err)
	}
	if latest == "" {
		t.Fatal("Terminal.GetLatestTokenValue() = empty, want token")
	}
	found, err := dctx.Terminal().SearchTokenValue(ctx, "terminal-user", 0, 10)
	if err != nil {
		t.Fatalf("Terminal.SearchTokenValue() error = %v", err)
	}
	if len(found) == 0 {
		t.Fatal("Terminal.SearchTokenValue() returned no tokens, want at least one")
	}

	visited := 0
	if err = dctx.Terminal().ForEachTerminal(ctx, func(terminal manager.TerminalInfo) bool {
		visited++
		return true
	}); err != nil {
		t.Fatalf("Terminal.ForEachTerminal() error = %v", err)
	}
	if visited != 3 {
		t.Fatalf("Terminal.ForEachTerminal() visited = %d, want 3", visited)
	}
	visitedWeb := 0
	if err = dctx.Terminal().ForEachTerminalByDevice(ctx, "web", func(terminal manager.TerminalInfo) bool {
		visitedWeb++
		return true
	}); err != nil {
		t.Fatalf("Terminal.ForEachTerminalByDevice() error = %v", err)
	}
	if visitedWeb != 2 {
		t.Fatalf("Terminal.ForEachTerminalByDevice() visited = %d, want 2", visitedWeb)
	}

	if err = dctx.Terminal().Terminate(ctx, manager.TerminateOptions{Token: token2, Action: manager.TerminateActionLogout}); err != nil {
		t.Fatalf("Terminal.Terminate(token2 logout) error = %v", err)
	}
	if mgr.IsLogin(ctx, token2) {
		t.Fatal("manager.IsLogin(token2) after terminate = true, want false")
	}
}

// TestContextOptionalFacadeVariants verifies optional module variant helpers. TestContextOptionalFacadeVariants 验证可选模块更多快捷方法。
func TestContextOptionalFacadeVariants(t *testing.T) {
	ctx := stdctx.Background()
	dctx, req, mgr := newTestDTokenContext(t)
	enableContextOptionalManagers(mgr)

	token, err := dctx.Auth().Login(ctx, "optional-user", "web", "browser-1")
	if err != nil {
		t.Fatalf("Auth.Login() error = %v", err)
	}
	req.headers[mgr.GetConfig().TokenName] = token

	nonceValue, err := dctx.Nonce().GenerateWithTimeout(ctx, time.Minute)
	if err != nil {
		t.Fatalf("Nonce.GenerateWithTimeout() error = %v", err)
	}
	if ttl, err := dctx.Nonce().GetTTL(ctx, nonceValue); err != nil || ttl <= 0 {
		t.Fatalf("Nonce.GetTTL() = %d, %v, want positive ttl", ttl, err)
	}
	if !dctx.Nonce().Verify(ctx, nonceValue) {
		t.Fatal("Nonce.Verify() = false, want true")
	}

	createdTicket, err := dctx.Ticket().CreateWithTimeout(ctx, ticket.CreateOptions{LoginID: "optional-user", TargetApp: "console"}, time.Minute)
	if err != nil {
		t.Fatalf("Ticket.CreateWithTimeout() error = %v", err)
	}
	if status, err := dctx.Ticket().GetStatus(ctx, createdTicket.Ticket); err != nil || status != ticket.StatusValid {
		t.Fatalf("Ticket.GetStatus() = %s, %v, want valid, nil", status, err)
	}
	if ttl, err := dctx.Ticket().GetTTL(ctx, createdTicket.Ticket); err != nil || ttl <= 0 {
		t.Fatalf("Ticket.GetTTL() = %d, %v, want positive ttl", ttl, err)
	}
	if err = dctx.Ticket().Revoke(ctx, createdTicket.Ticket); err != nil {
		t.Fatalf("Ticket.Revoke() error = %v", err)
	}

	createdShortKey, err := dctx.ShortKey().CreateWithTimeout(ctx, shortkey.CreateOptions{TargetApp: "console"}, time.Minute)
	if err != nil {
		t.Fatalf("ShortKey.CreateWithTimeout() error = %v", err)
	}
	if status, err := dctx.ShortKey().GetStatus(ctx, createdShortKey.Key); err != nil || status != shortkey.StatusPending {
		t.Fatalf("ShortKey.GetStatus() = %s, %v, want pending, nil", status, err)
	}
	if ttl, err := dctx.ShortKey().GetTTL(ctx, createdShortKey.Key); err != nil || ttl <= 0 {
		t.Fatalf("ShortKey.GetTTL() = %d, %v, want positive ttl", ttl, err)
	}
	if err = dctx.ShortKey().Revoke(ctx, createdShortKey.Key); err != nil {
		t.Fatalf("ShortKey.Revoke() error = %v", err)
	}

	refreshPair, err := dctx.Refresh().LoginWithOptions(ctx, manager.RefreshTokenOptions{
		LoginOptions: manager.LoginOptions{
			LoginID:  "refresh-options",
			Device:   "web",
			DeviceID: "browser-1",
			Timeout:  time.Minute,
		},
		RefreshTimeout: 2 * time.Minute,
	})
	if err != nil {
		t.Fatalf("Refresh.LoginWithOptions() error = %v", err)
	}
	if ttl, err := dctx.Refresh().GetTTL(ctx, refreshPair.RefreshToken); err != nil || ttl <= 0 {
		t.Fatalf("Refresh.GetTTL() = %d, %v, want positive ttl", ttl, err)
	}
	if err = dctx.Refresh().Revoke(ctx, refreshPair.RefreshToken); err != nil {
		t.Fatalf("Refresh.Revoke() error = %v", err)
	}

	client := &oauth2.Client{
		ClientID:     "ctx-oauth-client",
		ClientSecret: "secret",
		RedirectURIs: []string{
			"https://example.com/callback",
		},
		GrantTypes: []oauth2.GrantType{
			oauth2.GrantTypeAuthorizationCode,
			oauth2.GrantTypeRefreshToken,
			oauth2.GrantTypePassword,
			oauth2.GrantTypeClientCredentials,
		},
		Scopes: []string{"read", "write"},
	}
	if err = dctx.OAuth2().RegisterClient(client); err != nil {
		t.Fatalf("OAuth2.RegisterClient() error = %v", err)
	}
	gotClient, err := dctx.OAuth2().GetClient(client.ClientID)
	if err != nil {
		t.Fatalf("OAuth2.GetClient() error = %v", err)
	}
	if gotClient.ClientID != client.ClientID {
		t.Fatalf("OAuth2.GetClient().ClientID = %q, want %q", gotClient.ClientID, client.ClientID)
	}
	code, err := dctx.OAuth2().GenerateAuthorizationCode(ctx, client.ClientID, "oauth-user", "https://example.com/callback", []string{"read"})
	if err != nil {
		t.Fatalf("OAuth2.GenerateAuthorizationCode() error = %v", err)
	}
	accessToken, err := dctx.OAuth2().ExchangeCodeForToken(ctx, code.Code, client.ClientID, client.ClientSecret, "https://example.com/callback")
	if err != nil {
		t.Fatalf("OAuth2.ExchangeCodeForToken() error = %v", err)
	}
	info, err := dctx.OAuth2().ValidateAccessTokenAndGetInfo(ctx, accessToken.Token)
	if err != nil {
		t.Fatalf("OAuth2.ValidateAccessTokenAndGetInfo() error = %v", err)
	}
	if info.UserID != "oauth-user" || info.ClientID != client.ClientID {
		t.Fatalf("OAuth2 token info = %+v, want oauth-user/%s", info, client.ClientID)
	}
	rotated, err := dctx.OAuth2().RefreshAccessToken(ctx, client.ClientID, accessToken.RefreshToken, client.ClientSecret)
	if err != nil {
		t.Fatalf("OAuth2.RefreshAccessToken() error = %v", err)
	}
	if rotated.Token == "" || rotated.Token == accessToken.Token {
		t.Fatalf("OAuth2.RefreshAccessToken() token = %q, old = %q, want new token", rotated.Token, accessToken.Token)
	}
	passwordToken, err := dctx.OAuth2().PasswordGrantToken(ctx, client.ClientID, client.ClientSecret, "demo", "pass", []string{"read"}, func(username, password string) (string, error) {
		if username == "demo" && password == "pass" {
			return "password-user", nil
		}
		return "", derror.ErrInvalidUserCredentials
	})
	if err != nil {
		t.Fatalf("OAuth2.PasswordGrantToken() error = %v", err)
	}
	tokenByEndpoint, err := dctx.OAuth2().Token(ctx, &oauth2.TokenRequest{
		GrantType:    oauth2.GrantTypeClientCredentials,
		ClientID:     client.ClientID,
		ClientSecret: client.ClientSecret,
		Scopes:       []string{"read"},
	}, nil)
	if err != nil {
		t.Fatalf("OAuth2.Token(client_credentials) error = %v", err)
	}
	if err = dctx.OAuth2().RevokeToken(ctx, rotated.Token); err != nil {
		t.Fatalf("OAuth2.RevokeToken(rotated) error = %v", err)
	}
	if dctx.OAuth2().ValidateAccessToken(ctx, rotated.Token) {
		t.Fatal("OAuth2.ValidateAccessToken(rotated after revoke) = true, want false")
	}
	_ = dctx.OAuth2().RevokeToken(ctx, passwordToken.Token)
	_ = dctx.OAuth2().RevokeToken(ctx, tokenByEndpoint.Token)
	if err = dctx.OAuth2().UnregisterClient(client.ClientID); err != nil {
		t.Fatalf("OAuth2.UnregisterClient() error = %v", err)
	}
	if _, err = dctx.OAuth2().GetClient(client.ClientID); !errors.Is(err, derror.ErrClientNotFound) {
		t.Fatalf("OAuth2.GetClient(after unregister) error = %v, want ErrClientNotFound", err)
	}
}

type testRequestContext struct {
	headers map[string]string
	cookies map[string]string
	queries map[string]string
	forms   map[string]string
	values  map[string]any
	cookie  *adapter.CookieOptions
}

func (c *testRequestContext) GetHeader(key string) string { return c.headers[key] }
func (c *testRequestContext) GetHeaders() map[string][]string {
	result := map[string][]string{}
	for key, value := range c.headers {
		result[key] = []string{value}
	}
	return result
}
func (c *testRequestContext) GetQuery(key string) string       { return c.queries[key] }
func (c *testRequestContext) GetQueryAll() map[string][]string { return nil }
func (c *testRequestContext) GetPostForm(key string) string    { return c.forms[key] }
func (c *testRequestContext) GetCookie(key string) string      { return c.cookies[key] }
func (c *testRequestContext) GetBody() ([]byte, error)         { return nil, nil }
func (c *testRequestContext) GetClientIP() string              { return "" }
func (c *testRequestContext) GetMethod() string                { return "" }
func (c *testRequestContext) GetPath() string                  { return "" }
func (c *testRequestContext) GetURL() string                   { return "" }
func (c *testRequestContext) GetUserAgent() string             { return "" }
func (c *testRequestContext) IsTLS() bool                      { return false }
func (c *testRequestContext) SetStatusCode(int)                {}
func (c *testRequestContext) SetHeader(string, string)         {}
func (c *testRequestContext) Write(data []byte) (int, error)   { return len(data), nil }
func (c *testRequestContext) SetCookie(string, string, int, string, string, bool, bool) {
}
func (c *testRequestContext) SetCookieWithOptions(options *adapter.CookieOptions) {
	c.cookie = options
}
func (c *testRequestContext) Set(key string, value any) {
	if c.values == nil {
		c.values = map[string]any{}
	}
	c.values[key] = value
}
func (c *testRequestContext) Get(key string) (any, bool) {
	value, ok := c.values[key]
	return value, ok
}
func (c *testRequestContext) GetString(key string) string {
	value, _ := c.values[key].(string)
	return value
}
func (c *testRequestContext) MustGet(key string) any { return c.values[key] }
func (c *testRequestContext) Abort()                 {}
func (c *testRequestContext) IsAborted() bool        { return false }

func newTestDTokenContext(t *testing.T) (*DTokenContext, *testRequestContext, *manager.Manager) {
	t.Helper()

	cfg := config.DefaultConfig()
	cfg.TokenName = "X-Context-Token"
	cfg.Timeout = 120
	cfg.RefreshTokenTimeout = 180
	cfg.AutoRenew = false
	cfg.AsyncEvent = false
	cfg.IsPrintBanner = false
	cfg.IsLog = false
	cfg.IsReadHeader = true
	cfg.IsReadCookie = true
	cfg.CookieConfig = config.DefaultCookieConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("test config invalid: %v", err)
	}

	mgr := manager.NewManager(
		cfg,
		&contextTestGenerator{},
		newContextTestStorage(),
		contextTestCodec{},
		adapter.NewNopLogger(),
		nil,
		nil,
	)
	t.Cleanup(mgr.CloseManager)

	req := &testRequestContext{
		headers: map[string]string{},
		cookies: map[string]string{},
		queries: map[string]string{},
		forms:   map[string]string{},
	}
	return NewContext(req, mgr), req, mgr
}

func enableContextOptionalManagers(mgr *manager.Manager) {
	cfg := mgr.GetConfig()
	manager.WithNonceManager(nonce.NewDefaultNonceManager(
		cfg.AuthType,
		cfg.KeyPrefix,
		mgr.GetStorage(),
	))(mgr)
	manager.WithTicketManager(ticket.NewDefaultManager(
		cfg.AuthType,
		cfg.KeyPrefix,
		mgr.GetStorage(),
		mgr.GetSerializer(),
	))(mgr)
	manager.WithShortKeyManager(shortkey.NewDefaultManager(
		cfg.AuthType,
		cfg.KeyPrefix,
		mgr.GetStorage(),
		mgr.GetSerializer(),
	))(mgr)
	manager.WithOAuth2Manager(oauth2.NewOAuth2Server(
		cfg.AuthType,
		cfg.KeyPrefix,
		mgr.GetStorage(),
		mgr.GetSerializer(),
	))(mgr)
}

func sameContextStrings(got, want []string) bool {
	got = append([]string(nil), got...)
	want = append([]string(nil), want...)
	sort.Strings(got)
	sort.Strings(want)
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func assertContextCookie(t *testing.T, cookie *adapter.CookieOptions, name, value string, maxAge int, path, domain string, secure, httpOnly bool, sameSite string) {
	t.Helper()

	if cookie == nil {
		t.Fatal("cookie = nil, want configured cookie")
	}
	if cookie.Name != name || cookie.Value != value || cookie.MaxAge != maxAge || cookie.Path != path || cookie.Domain != domain || cookie.Secure != secure || cookie.HttpOnly != httpOnly || cookie.SameSite != sameSite {
		t.Fatalf("cookie = %+v, want name=%s value=%s maxAge=%d path=%s domain=%s secure=%v httpOnly=%v sameSite=%s", cookie, name, value, maxAge, path, domain, secure, httpOnly, sameSite)
	}
}

type contextTestGenerator struct {
	mu  sync.Mutex
	seq int
}

func (g *contextTestGenerator) Generate(loginID, device, deviceID string) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.seq++
	return fmt.Sprintf("ctx-token-%s-%s-%s-%d", loginID, device, deviceID, g.seq), nil
}

type contextTestCodec struct{}

func (contextTestCodec) Name() string { return "json-test" }

func (contextTestCodec) Encode(v any) ([]byte, error) { return json.Marshal(v) }

func (contextTestCodec) Decode(data []byte, v any) error { return json.Unmarshal(data, v) }

type contextTestStorage struct {
	mu    sync.RWMutex
	items map[string]contextTestStorageItem
}

type contextTestStorageItem struct {
	value    any
	expireAt time.Time
}

func newContextTestStorage() *contextTestStorage {
	return &contextTestStorage{items: map[string]contextTestStorageItem{}}
}

func (s *contextTestStorage) Set(_ stdctx.Context, key string, value any, expiration time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var expireAt time.Time
	if expiration > 0 {
		expireAt = time.Now().Add(expiration)
	}
	s.items[key] = contextTestStorageItem{value: value, expireAt: expireAt}
	return nil
}

func (s *contextTestStorage) Get(_ stdctx.Context, key string) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.items[key]
	if !ok {
		return nil, nil
	}
	if item.expired() {
		delete(s.items, key)
		return nil, nil
	}
	return item.value, nil
}

func (s *contextTestStorage) Delete(_ stdctx.Context, keys ...string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, key := range keys {
		delete(s.items, key)
	}
	return nil
}

func (s *contextTestStorage) Exists(_ stdctx.Context, key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.items[key]
	if !ok {
		return false
	}
	if item.expired() {
		delete(s.items, key)
		return false
	}
	return true
}

func (s *contextTestStorage) Expire(_ stdctx.Context, key string, expiration time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.items[key]
	if !ok || item.expired() {
		delete(s.items, key)
		return derror.ErrInvalidToken
	}
	if expiration > 0 {
		item.expireAt = time.Now().Add(expiration)
	} else {
		item.expireAt = time.Time{}
	}
	s.items[key] = item
	return nil
}

func (s *contextTestStorage) TTL(_ stdctx.Context, key string) (time.Duration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.items[key]
	if !ok {
		return adapter.TTLNotFound, nil
	}
	if item.expired() {
		delete(s.items, key)
		return adapter.TTLNotFound, nil
	}
	if item.expireAt.IsZero() {
		return adapter.TTLNoExpire, nil
	}
	ttl := time.Until(item.expireAt)
	if ttl <= 0 {
		delete(s.items, key)
		return adapter.TTLNotFound, nil
	}
	return ttl, nil
}

func (s *contextTestStorage) Ping(stdctx.Context) error { return nil }

func (s *contextTestStorage) GetAndDelete(_ stdctx.Context, key string) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.items[key]
	if !ok {
		return nil, nil
	}
	delete(s.items, key)
	if item.expired() {
		return nil, nil
	}
	return item.value, nil
}

func (s *contextTestStorage) Keys(_ stdctx.Context, pattern string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	keys := make([]string, 0, len(s.items))
	for key, item := range s.items {
		if item.expired() {
			delete(s.items, key)
			continue
		}
		if matchContextTestPattern(pattern, key) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys, nil
}

func (s *contextTestStorage) Clear(stdctx.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = map[string]contextTestStorageItem{}
	return nil
}

func (item contextTestStorageItem) expired() bool {
	return !item.expireAt.IsZero() && time.Now().After(item.expireAt)
}

func matchContextTestPattern(pattern, value string) bool {
	parts := strings.Split(pattern, "*")
	if len(parts) == 1 {
		return pattern == value
	}
	if !strings.HasPrefix(value, parts[0]) {
		return false
	}
	value = strings.TrimPrefix(value, parts[0])
	for _, part := range parts[1 : len(parts)-1] {
		idx := strings.Index(value, part)
		if idx < 0 {
			return false
		}
		value = value[idx+len(part):]
	}
	last := parts[len(parts)-1]
	return last == "" || strings.HasSuffix(value, last)
}
