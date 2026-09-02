package dtoken

import (
	"context"
	"errors"
	"testing"
	"time"

	djson "github.com/Zany2/dtoken-go/com/codec/json"
	"github.com/Zany2/dtoken-go/com/generator/dgenerator"
	"github.com/Zany2/dtoken-go/com/log/dlog"
	"github.com/Zany2/dtoken-go/com/pool/ants"
	"github.com/Zany2/dtoken-go/com/storage/memory"
	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/manager"
	"github.com/Zany2/dtoken-go/core/nonce"
	"github.com/Zany2/dtoken-go/core/oauth2"
	"github.com/Zany2/dtoken-go/core/shortkey"
	"github.com/Zany2/dtoken-go/core/ticket"
)

// TestBuilderSettersConfigureAllModules verifies every high-level builder setter writes the intended value. TestBuilderSettersConfigureAllModules 验证高层 Builder 配置器均写入预期值。
func TestBuilderSettersConfigureAllModules(t *testing.T) {
	presetPool := ants.DefaultRenewPoolConfig()
	presetLogger := dlog.DefaultLoggerConfig()
	presetNonce := nonce.DefaultConfig()
	presetOAuth2 := oauth2.DefaultConfig()
	presetTicket := ticket.DefaultConfig()
	presetShortKey := shortkey.DefaultConfig()
	presetCookie := &config.CookieConfig{Domain: "preset.example", Path: "/preset", HttpOnly: true}

	b := NewBuilder().
		AuthType("chain-auth").
		KeyPrefix("chain-prefix").
		TokenName("chain-token").
		Timeout(30).
		RefreshTokenTimeout(60).
		TimeoutDuration(45 * time.Second).
		RefreshTokenTimeoutDuration(90 * time.Second).
		AutoRenew(false).
		RenewMaxRefresh(10).
		RenewInterval(5).
		ActiveTimeout(20).
		ConcurrencyScope(config.ConcurrencyScopeDevice).
		IsConcurrent(false).
		IsShare(false).
		MaxLoginCount(3).
		ReplacedLoginExitMode(config.ReplacedLoginExitModeNewDevice).
		OverflowLogoutMode(config.LogoutModeLogout).
		IsReadBody(true).
		IsReadQuery(true).
		IsReadHeader(false).
		IsReadCookie(true).
		TokenStyle(adapter.TokenStyleJWT).
		JwtSecretKey("chain-jwt-key").
		IsLog(false).
		IsPrintBanner(false).
		AsyncEvent(false).
		CookieConfig(presetCookie).
		CookieDomain("example.com").
		CookiePath("/app").
		CookieSecure(true).
		CookieHttpOnly(false).
		CookieSameSite(config.SameSiteNone).
		CookieMaxAge(3600).
		RenewPoolConfig(presetPool).
		RenewPoolMinSize(2).
		RenewPoolMaxSize(4).
		RenewPoolScaleUpRate(0.8).
		RenewPoolScaleDownRate(0.2).
		RenewPoolCheckInterval(2 * time.Second).
		RenewPoolExpiry(3 * time.Second).
		RenewPoolPrintStatusInterval(4 * time.Second).
		RenewPoolPreAlloc(false).
		RenewPoolNonBlocking(false).
		LoggerConfig(presetLogger).
		LoggerPath("chain-logs").
		LoggerFileFormat("chain.log").
		LoggerPrefix("[CHAIN] ").
		LoggerLevel(dlog.LevelDebug).
		LoggerTimeFormat("15:04:05").
		LoggerStdout(false).
		LoggerStdoutOnly(true).
		LoggerQueueSize(128).
		LoggerRotateSize(2048).
		LoggerRotateExpire(5 * time.Minute).
		LoggerRotateBackupLimit(3).
		LoggerRotateBackupDays(5).
		NonceConfig(presetNonce).
		NonceTTL(3 * time.Minute).
		OAuth2Config(presetOAuth2).
		OAuth2CodeExpiration(2 * time.Minute).
		OAuth2TokenExpiration(4 * time.Minute).
		OAuth2RefreshExpiration(8 * time.Minute).
		TicketConfig(presetTicket).
		TicketTTL(3 * time.Minute).
		ShortKeyConfig(presetShortKey).
		ShortKeyTTL(3 * time.Minute).
		ShortKeyLength(7).
		JwtSecret("chain-jwt-secret")

	if b.GetRenewPoolConfig() == presetPool || b.GetLoggerConfig() == presetLogger || b.GetNonceConfig() == presetNonce || b.GetOAuth2Config() == presetOAuth2 || b.GetTicketConfig() == presetTicket || b.GetShortKeyConfig() == presetShortKey {
		t.Fatal("module config setters should clone caller-owned configs")
	}

	cfg := b.GetConfig()
	if cfg.AuthType != "chain-auth:" || cfg.KeyPrefix != "chain-prefix:" || cfg.TokenName != "chain-token" || cfg.Timeout != 45 || cfg.RefreshTokenTimeout != 90 {
		t.Fatalf("core config = %+v", cfg)
	}
	if cfg.AutoRenew || cfg.RenewMaxRefresh != 10 || cfg.RenewInterval != 5 || cfg.ActiveTimeout != 20 || cfg.ConcurrencyScope != config.ConcurrencyScopeDevice || cfg.IsConcurrent || cfg.IsShare || cfg.MaxLoginCount != 3 || cfg.ReplacedLoginExitMode != config.ReplacedLoginExitModeNewDevice || cfg.OverflowLogoutMode != config.LogoutModeLogout {
		t.Fatalf("login policy config = %+v", cfg)
	}
	if !cfg.IsReadBody || !cfg.IsReadQuery || cfg.IsReadHeader || !cfg.IsReadCookie || cfg.TokenStyle != adapter.TokenStyleJWT || cfg.JwtSecretKey != "chain-jwt-secret" || cfg.IsLog || cfg.IsPrintBanner || cfg.AsyncEvent {
		t.Fatalf("token source config = %+v", cfg)
	}
	if cfg.CookieConfig == nil || cfg.CookieConfig.Domain != "example.com" || cfg.CookieConfig.Path != "/app" || !cfg.CookieConfig.Secure || cfg.CookieConfig.HttpOnly || cfg.CookieConfig.SameSite != config.SameSiteNone || cfg.CookieConfig.MaxAge != 3600 {
		t.Fatalf("cookie config = %+v", cfg.CookieConfig)
	}

	poolCfg := b.GetRenewPoolConfig()
	if poolCfg.MinSize != 2 || poolCfg.MaxSize != 4 || poolCfg.ScaleUpRate != 0.8 || poolCfg.ScaleDownRate != 0.2 || poolCfg.CheckInterval != 2*time.Second || poolCfg.Expiry != 3*time.Second || poolCfg.PrintStatusInterval != 4*time.Second || poolCfg.PreAlloc || poolCfg.NonBlocking {
		t.Fatalf("renew pool config = %+v", poolCfg)
	}
	loggerCfg := b.GetLoggerConfig()
	if loggerCfg.Path != "chain-logs" || loggerCfg.FileFormat != "chain.log" || loggerCfg.Prefix != "[CHAIN] " || loggerCfg.Level != dlog.LevelDebug || loggerCfg.TimeFormat != "15:04:05" || !loggerCfg.Stdout || !loggerCfg.StdoutOnly || loggerCfg.QueueSize != 128 || loggerCfg.RotateSize != 2048 || loggerCfg.RotateExpire != 5*time.Minute || loggerCfg.RotateBackupLimit != 3 || loggerCfg.RotateBackupDays != 5 {
		t.Fatalf("logger config = %+v", loggerCfg)
	}
	if b.GetNonceConfig().TTL != 3*time.Minute || b.GetOAuth2Config().CodeExpiration != 2*time.Minute || b.GetOAuth2Config().TokenExpiration != 4*time.Minute || b.GetOAuth2Config().RefreshExpiration != 8*time.Minute || b.GetTicketConfig().TTL != 3*time.Minute || b.GetShortKeyConfig().TTL != 3*time.Minute || b.GetShortKeyConfig().Length != 7 {
		t.Fatal("optional module config setters did not write expected values")
	}

	mgr, err := b.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	mgr.CloseManager()
}

// TestBuilderInjectsExplicitComponents verifies direct component and optional-manager setters. TestBuilderInjectsExplicitComponents 验证显式核心组件和可选管理器注入。
func TestBuilderInjectsExplicitComponents(t *testing.T) {
	ctx := context.Background()
	storage := memory.NewStorage()
	codec := djson.NewJSONSerializer()
	generator := dgenerator.NewGenerator(60, "injected-secret", adapter.TokenStyleUUID)
	logger := &registryTestLogger{NopLogger: adapter.NewNopLogger()}
	pool := &registryTestPool{}
	nonceManager := nonce.NewNonceManager("injected:", "dtoken:", storage, time.Minute)
	oauthManager := oauth2.NewOAuth2Server("injected:", "dtoken:", storage, codec)
	ticketManager := ticket.NewManagerWithConfig("injected:", "dtoken:", storage, codec, ticket.DefaultConfig())
	shortKeyManager := shortkey.NewManagerWithConfig("injected:", "dtoken:", storage, codec, shortkey.DefaultConfig())

	mgr, err := NewBuilder().
		AuthType("injected").
		IsPrintBanner(false).
		IsLog(true).
		AutoRenew(false).
		SetGenerator(generator).
		SetStorage(storage).
		SetCodec(codec).
		SetLog(logger).
		SetPool(pool).
		SetNonceManager(nonceManager).
		SetOAuth2Manager(oauthManager).
		SetTicketManager(ticketManager).
		SetShortKeyManager(shortKeyManager).
		UseManagerOption(manager.WithComponentOwnership(manager.ComponentOwnership{})).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	defer mgr.CloseManager()

	if mgr.GetGenerator() != generator || mgr.GetStorage() != storage || mgr.GetSerializer() != codec || mgr.GetLogger() != logger || mgr.GetPool() != pool {
		t.Fatal("explicit core components were not injected")
	}
	if mgr.GetNonceManager() != nonceManager || mgr.GetOAuth2Manager() != oauthManager || mgr.GetTicketManager() != ticketManager || mgr.GetShortKeyManager() != shortKeyManager {
		t.Fatal("explicit optional managers were not injected")
	}
	if _, err := mgr.GenerateNonce(ctx); err != nil {
		t.Fatalf("injected nonce manager is not usable: %v", err)
	}
}

// TestBuilderConfigAndAccessProvider verifies config copy/reset behavior and access-provider wiring. TestBuilderConfigAndAccessProvider 验证 Config 的复制/重置行为和访问提供器装配。
func TestBuilderConfigAndAccessProvider(t *testing.T) {
	ctx := context.Background()
	provided := config.DefaultConfig()
	provided.AuthType = "provided"
	provided.KeyPrefix = "provided-prefix"
	provided.TokenName = "provided-token"
	provided.AutoRenew = false
	provided.IsPrintBanner = false

	provider := manager.AccessProviderFunc{
		PermissionFunc: func(_ context.Context, subject manager.AccessSubject) ([]string, error) {
			if subject.LoginID != "provider-user" {
				return nil, errors.New("unexpected provider login id")
			}
			return []string{"provider:read"}, nil
		},
		RoleFunc: func(_ context.Context, subject manager.AccessSubject) ([]string, error) {
			if subject.LoginID != "provider-user" {
				return nil, errors.New("unexpected provider login id")
			}
			return []string{"provider-user"}, nil
		},
	}

	b := NewBuilder().Config(provided).SetAccessProvider(provider)
	provided.AuthType = "mutated"
	provided.KeyPrefix = "mutated-prefix"

	if cfg := b.GetConfig(); cfg.AuthType != "provided" || cfg.KeyPrefix != "provided-prefix" || cfg.TokenName != "provided-token" {
		t.Fatalf("Config() did not isolate caller config: %+v", cfg)
	}

	mgr, err := b.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	defer mgr.CloseManager()
	if mgr.GetConfig().AuthType != "provided:" || mgr.GetConfig().KeyPrefix != "provided-prefix:" {
		t.Fatalf("built config = %+v, want normalized namespaces", mgr.GetConfig())
	}
	if mgr.GetAccessProvider() == nil {
		t.Fatal("SetAccessProvider() did not wire a provider")
	}
	permissions, err := mgr.GetAccessProvider().Permissions(ctx, manager.AccessSubject{LoginID: "provider-user"})
	if err != nil || len(permissions) != 1 || permissions[0] != "provider:read" {
		t.Fatalf("provider permissions = %v, %v", permissions, err)
	}
	roles, err := mgr.GetAccessProvider().Roles(ctx, manager.AccessSubject{LoginID: "provider-user"})
	if err != nil || len(roles) != 1 || roles[0] != "provider-user" {
		t.Fatalf("provider roles = %v, %v", roles, err)
	}

	reset := NewBuilder().AuthType("custom").Config(nil)
	if reset.GetConfig().AuthType != config.DefaultAuthType {
		t.Fatalf("Config(nil) auth type = %q, want %q", reset.GetConfig().AuthType, config.DefaultAuthType)
	}
}

// TestGlobalTerminalOperationFamilies verifies global logout, kickout, and replace routes. TestGlobalTerminalOperationFamilies 验证全局 logout、kickout 和 replace 路由。
func TestGlobalTerminalOperationFamilies(t *testing.T) {
	DeleteAllManager()
	t.Cleanup(DeleteAllManager)

	ctx := context.Background()
	mgr, err := NewBuilder().IsPrintBanner(false).AutoRenew(false).IsShare(false).AuthType("global-terminal").Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	SetManager(mgr)

	login := func(loginID, device, deviceID string) string {
		token, loginErr := Login(ctx, loginID, device, deviceID, "global-terminal")
		if loginErr != nil {
			t.Fatalf("Login(%s) error = %v", loginID, loginErr)
		}
		return token
	}

	logoutDeviceToken := login("logout-device", "web", "logout-1")
	login(ctx, "logout-device", "web", "logout-2", "global-terminal")
	if err = LogoutByDevice(ctx, "logout-device", "web", "global-terminal"); err != nil || IsLogin(ctx, logoutDeviceToken, "global-terminal") {
		t.Fatalf("LogoutByDevice() error = %v, token active = %v", err, IsLogin(ctx, logoutDeviceToken, "global-terminal"))
	}
	logoutLoginToken := login("logout-login", "mobile", "logout-login-1")
	if err = LogoutByLoginID(ctx, "logout-login", "global-terminal"); err != nil || IsLogin(ctx, logoutLoginToken, "global-terminal") {
		t.Fatalf("LogoutByLoginID() error = %v, token active = %v", err, IsLogin(ctx, logoutLoginToken, "global-terminal"))
	}
	directLogoutToken := login("logout-direct", "web", "logout-direct-1")
	if err = Logout(ctx, directLogoutToken, "global-terminal"); err != nil || IsLogin(ctx, directLogoutToken, "global-terminal") {
		t.Fatalf("Logout() error = %v, token active = %v", err, IsLogin(ctx, directLogoutToken, "global-terminal"))
	}

	kickToken := login("kick-token", "web", "kick-token-1")
	if err = Kickout(ctx, kickToken, "global-terminal"); err != nil || IsLogin(ctx, kickToken, "global-terminal") {
		t.Fatalf("Kickout() error = %v, token active = %v", err, IsLogin(ctx, kickToken, "global-terminal"))
	}
	kickConcreteToken := login("kick-concrete", "web", "kick-concrete-1")
	if err = KickoutByDeviceAndDeviceID(ctx, "kick-concrete", "web", "kick-concrete-1", "global-terminal"); err != nil || IsLogin(ctx, kickConcreteToken, "global-terminal") {
		t.Fatalf("KickoutByDeviceAndDeviceID() error = %v, token active = %v", err, IsLogin(ctx, kickConcreteToken, "global-terminal"))
	}
	kickDeviceToken := login("kick-device", "web", "kick-device-1")
	if err = KickoutByDevice(ctx, "kick-device", "web", "global-terminal"); err != nil || IsLogin(ctx, kickDeviceToken, "global-terminal") {
		t.Fatalf("KickoutByDevice() error = %v, token active = %v", err, IsLogin(ctx, kickDeviceToken, "global-terminal"))
	}
	kickLoginToken := login("kick-login", "mobile", "kick-login-1")
	if err = KickoutByLoginID(ctx, "kick-login", "global-terminal"); err != nil || IsLogin(ctx, kickLoginToken, "global-terminal") {
		t.Fatalf("KickoutByLoginID() error = %v, token active = %v", err, IsLogin(ctx, kickLoginToken, "global-terminal"))
	}

	replaceToken := login("replace-token", "web", "replace-token-1")
	if err = Replace(ctx, replaceToken, "global-terminal"); err != nil || IsLogin(ctx, replaceToken, "global-terminal") {
		t.Fatalf("Replace() error = %v, token active = %v", err, IsLogin(ctx, replaceToken, "global-terminal"))
	}
	replaceConcreteToken := login("replace-concrete", "web", "replace-concrete-1")
	if err = ReplaceByDeviceAndDeviceID(ctx, "replace-concrete", "web", "replace-concrete-1", "global-terminal"); err != nil || IsLogin(ctx, replaceConcreteToken, "global-terminal") {
		t.Fatalf("ReplaceByDeviceAndDeviceID() error = %v, token active = %v", err, IsLogin(ctx, replaceConcreteToken, "global-terminal"))
	}
	replaceDeviceToken := login("replace-device", "web", "replace-device-1")
	if err = ReplaceByDevice(ctx, "replace-device", "web", "global-terminal"); err != nil || IsLogin(ctx, replaceDeviceToken, "global-terminal") {
		t.Fatalf("ReplaceByDevice() error = %v, token active = %v", err, IsLogin(ctx, replaceDeviceToken, "global-terminal"))
	}
	replaceLoginToken := login("replace-login", "mobile", "replace-login-1")
	if err = ReplaceByLoginID(ctx, "replace-login", "global-terminal"); err != nil || IsLogin(ctx, replaceLoginToken, "global-terminal") {
		t.Fatalf("ReplaceByLoginID() error = %v, token active = %v", err, IsLogin(ctx, replaceLoginToken, "global-terminal"))
	}
}

// TestInstanceTerminalOperationFamilies verifies instance terminal shortcuts for all lifecycle actions. TestInstanceTerminalOperationFamilies 验证实例门面全部终端生命周期快捷方法。
func TestInstanceTerminalOperationFamilies(t *testing.T) {
	ctx := context.Background()
	mgr, err := NewBuilder().IsPrintBanner(false).AutoRenew(false).IsShare(false).AuthType("instance-terminal-complete").Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	auth := New(mgr)
	t.Cleanup(auth.Close)

	login := func(loginID, device, deviceID string) string {
		token, loginErr := auth.Login(ctx, LoginOptions{LoginID: loginID, Device: device, DeviceID: deviceID})
		if loginErr != nil {
			t.Fatalf("Auth.Login(%s) error = %v", loginID, loginErr)
		}
		return token
	}

	logoutToken := login("auth-logout-token", "web", "logout-token")
	if err = auth.LogoutByToken(ctx, logoutToken); err != nil || auth.IsLogin(ctx, logoutToken) {
		t.Fatalf("Auth.LogoutByToken() error = %v, token active = %v", err, auth.IsLogin(ctx, logoutToken))
	}
	logoutConcreteToken := login("auth-logout-concrete", "web", "logout-concrete")
	if err = auth.LogoutByDeviceAndDeviceID(ctx, "auth-logout-concrete", "web", "logout-concrete"); err != nil || auth.IsLogin(ctx, logoutConcreteToken) {
		t.Fatalf("Auth.LogoutByDeviceAndDeviceID() error = %v, token active = %v", err, auth.IsLogin(ctx, logoutConcreteToken))
	}
	logoutDeviceToken := login("auth-logout-device", "web", "logout-device")
	if err = auth.LogoutByDevice(ctx, "auth-logout-device", "web"); err != nil || auth.IsLogin(ctx, logoutDeviceToken) {
		t.Fatalf("Auth.LogoutByDevice() error = %v, token active = %v", err, auth.IsLogin(ctx, logoutDeviceToken))
	}
	logoutLoginToken := login("auth-logout-login", "mobile", "logout-login")
	if err = auth.LogoutByLoginID(ctx, "auth-logout-login"); err != nil || auth.IsLogin(ctx, logoutLoginToken) {
		t.Fatalf("Auth.LogoutByLoginID() error = %v, token active = %v", err, auth.IsLogin(ctx, logoutLoginToken))
	}

	kickToken := login("auth-kick-token", "web", "kick-token")
	if err = auth.KickoutByToken(ctx, kickToken); err != nil || auth.IsLogin(ctx, kickToken) {
		t.Fatalf("Auth.KickoutByToken() error = %v, token active = %v", err, auth.IsLogin(ctx, kickToken))
	}
	kickConcreteToken := login("auth-kick-concrete", "web", "kick-concrete")
	if err = auth.KickoutByDeviceAndDeviceID(ctx, "auth-kick-concrete", "web", "kick-concrete"); err != nil || auth.IsLogin(ctx, kickConcreteToken) {
		t.Fatalf("Auth.KickoutByDeviceAndDeviceID() error = %v, token active = %v", err, auth.IsLogin(ctx, kickConcreteToken))
	}
	kickDeviceToken := login("auth-kick-device", "web", "kick-device")
	if err = auth.KickoutByDevice(ctx, "auth-kick-device", "web"); err != nil || auth.IsLogin(ctx, kickDeviceToken) {
		t.Fatalf("Auth.KickoutByDevice() error = %v, token active = %v", err, auth.IsLogin(ctx, kickDeviceToken))
	}
	kickLoginToken := login("auth-kick-login", "mobile", "kick-login")
	if err = auth.KickoutByLoginID(ctx, "auth-kick-login"); err != nil || auth.IsLogin(ctx, kickLoginToken) {
		t.Fatalf("Auth.KickoutByLoginID() error = %v, token active = %v", err, auth.IsLogin(ctx, kickLoginToken))
	}
	kickOptionsToken := login("auth-kick-options", "mobile", "kick-options")
	if err = auth.Kickout(ctx, LogoutOptions{Token: kickOptionsToken}); err != nil || auth.IsLogin(ctx, kickOptionsToken) {
		t.Fatalf("Auth.Kickout(options) error = %v, token active = %v", err, auth.IsLogin(ctx, kickOptionsToken))
	}

	replaceToken := login("auth-replace-token", "web", "replace-token")
	if err = auth.ReplaceByToken(ctx, replaceToken); err != nil || auth.IsLogin(ctx, replaceToken) {
		t.Fatalf("Auth.ReplaceByToken() error = %v, token active = %v", err, auth.IsLogin(ctx, replaceToken))
	}
	replaceConcreteToken := login("auth-replace-concrete", "web", "replace-concrete")
	if err = auth.ReplaceByDeviceAndDeviceID(ctx, "auth-replace-concrete", "web", "replace-concrete"); err != nil || auth.IsLogin(ctx, replaceConcreteToken) {
		t.Fatalf("Auth.ReplaceByDeviceAndDeviceID() error = %v, token active = %v", err, auth.IsLogin(ctx, replaceConcreteToken))
	}
	replaceDeviceToken := login("auth-replace-device", "web", "replace-device")
	if err = auth.ReplaceByDevice(ctx, "auth-replace-device", "web"); err != nil || auth.IsLogin(ctx, replaceDeviceToken) {
		t.Fatalf("Auth.ReplaceByDevice() error = %v, token active = %v", err, auth.IsLogin(ctx, replaceDeviceToken))
	}
	replaceLoginToken := login("auth-replace-login", "mobile", "replace-login")
	if err = auth.ReplaceByLoginID(ctx, "auth-replace-login"); err != nil || auth.IsLogin(ctx, replaceLoginToken) {
		t.Fatalf("Auth.ReplaceByLoginID() error = %v, token active = %v", err, auth.IsLogin(ctx, replaceLoginToken))
	}
	replaceOptionsToken := login("auth-replace-options", "mobile", "replace-options")
	if err = auth.Replace(ctx, LogoutOptions{Token: replaceOptionsToken}); err != nil || auth.IsLogin(ctx, replaceOptionsToken) {
		t.Fatalf("Auth.Replace(options) error = %v, token active = %v", err, auth.IsLogin(ctx, replaceOptionsToken))
	}
}

// TestInstanceLoginRefreshAndMetadataFacades verifies instance login variants and refresh lifecycle. TestInstanceLoginRefreshAndMetadataFacades 验证实例登录变体及刷新生命周期。
func TestInstanceLoginRefreshAndMetadataFacades(t *testing.T) {
	ctx := context.Background()
	mgr, err := NewBuilder().IsPrintBanner(false).AutoRenew(false).AuthType("instance-login-complete").Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	auth := New(mgr)
	t.Cleanup(auth.Close)

	token, err := auth.LoginID(ctx, "login-id-user")
	if err != nil {
		t.Fatalf("Auth.LoginID() error = %v", err)
	}
	if err = auth.LoginByToken(ctx, token); err != nil {
		t.Fatalf("Auth.LoginByToken() error = %v", err)
	}
	if loginID, err := auth.GetLoginID(ctx, token); err != nil || loginID != "login-id-user" {
		t.Fatalf("Auth.GetLoginID() = %q, %v", loginID, err)
	}
	if deviceID, err := auth.GetDeviceID(ctx, token); err != nil || deviceID != "" {
		t.Fatalf("Auth.GetDeviceID() = %q, %v", deviceID, err)
	}
	if createTime, err := auth.GetTokenCreateTime(ctx, token); err != nil || createTime <= 0 {
		t.Fatalf("Auth.GetTokenCreateTime() = %d, %v", createTime, err)
	}
	if ttl, err := auth.GetTokenTTL(ctx, token); err != nil || ttl <= 0 {
		t.Fatalf("Auth.GetTokenTTL() = %d, %v", ttl, err)
	}
	if introspection, err := auth.IntrospectToken(ctx, token); err != nil || introspection == nil || !introspection.Active {
		t.Fatalf("Auth.IntrospectToken() = %+v, %v", introspection, err)
	}
	if err = auth.RenewTimeout(ctx, token, time.Minute); err != nil {
		t.Fatalf("Auth.RenewTimeout() error = %v", err)
	}
	if info, err := auth.GetTokenInfo(ctx, token); err != nil || info == nil || info.LoginID != "login-id-user" {
		t.Fatalf("Auth.GetTokenInfo() = %+v, %v", info, err)
	}

	if _, err = auth.LoginWithTimeout(ctx, "timeout-user", time.Minute); err != nil {
		t.Fatalf("Auth.LoginWithTimeout() error = %v", err)
	}
	pair, err := auth.LoginWithRefreshToken(ctx, "refresh-user")
	if err != nil || pair == nil || pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatalf("Auth.LoginWithRefreshToken() = %+v, %v", pair, err)
	}
	if ttl, err := auth.GetRefreshTokenTTL(ctx, pair.RefreshToken); err != nil || ttl <= 0 {
		t.Fatalf("Auth.GetRefreshTokenTTL() = %d, %v", ttl, err)
	}
	rotated, err := auth.RefreshToken(ctx, pair.RefreshToken)
	if err != nil || rotated == nil || rotated.AccessToken == pair.AccessToken || rotated.RefreshToken == pair.RefreshToken {
		t.Fatalf("Auth.RefreshToken() = %+v, %v", rotated, err)
	}
	if err = auth.RevokeRefreshToken(ctx, rotated.RefreshToken); err != nil {
		t.Fatalf("Auth.RevokeRefreshToken() error = %v", err)
	}
	optionPair, err := auth.LoginWithRefreshTokenOptions(ctx, RefreshTokenOptions{LoginOptions: LoginOptions{LoginID: "refresh-options"}, RefreshTimeout: time.Minute})
	if err != nil || optionPair == nil || optionPair.AccessToken == "" || optionPair.RefreshToken == "" {
		t.Fatalf("Auth.LoginWithRefreshTokenOptions() = %+v, %v", optionPair, err)
	}
	if err = auth.RevokeRefreshToken(ctx, optionPair.RefreshToken); err != nil {
		t.Fatalf("Auth.RevokeRefreshToken(options) error = %v", err)
	}
	if err = auth.LogoutByToken(ctx, token); err != nil {
		t.Fatalf("Auth.LogoutByToken() cleanup error = %v", err)
	}
}

// TestInstanceSessionAndAccessFacades verifies instance session, permission, and role forwarding. TestInstanceSessionAndAccessFacades 验证实例会话、权限和角色转发。
func TestInstanceSessionAndAccessFacades(t *testing.T) {
	ctx := context.Background()
	mgr, err := NewBuilder().IsPrintBanner(false).AutoRenew(false).IsShare(false).AuthType("instance-session-access").Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	auth := New(mgr)
	t.Cleanup(auth.Close)

	token1, err := auth.Login(ctx, LoginOptions{LoginID: "session-access-user", Device: "web", DeviceID: "browser"})
	if err != nil {
		t.Fatalf("Auth.Login(first) error = %v", err)
	}
	token2, err := auth.Login(ctx, LoginOptions{LoginID: "session-access-user", Device: "mobile", DeviceID: "phone"})
	if err != nil {
		t.Fatalf("Auth.Login(second) error = %v", err)
	}
	if count, err := auth.GetOnlineTerminalCount(ctx, "session-access-user"); err != nil || count != 2 {
		t.Fatalf("Auth.GetOnlineTerminalCount() = %d, %v", count, err)
	}
	if count, err := auth.GetOnlineTerminalCountByDevice(ctx, "session-access-user", "web"); err != nil || count != 1 {
		t.Fatalf("Auth.GetOnlineTerminalCountByDevice() = %d, %v", count, err)
	}
	if count, err := auth.GetOnlineTerminalCountByDeviceAndDeviceID(ctx, "session-access-user", "mobile", "phone"); err != nil || count != 1 {
		t.Fatalf("Auth.GetOnlineTerminalCountByDeviceAndDeviceID() = %d, %v", count, err)
	}

	session, err := auth.GetSession(ctx, "session-access-user")
	if err != nil || session == nil || session.LoginID != "session-access-user" {
		t.Fatalf("Auth.GetSession() = %+v, %v", session, err)
	}
	byToken, err := auth.GetSessionByToken(ctx, token1)
	if err != nil || byToken == nil || byToken.LoginID != session.LoginID {
		t.Fatalf("Auth.GetSessionByToken() = %+v, %v", byToken, err)
	}
	if values, err := auth.GetTokenValueListByLoginID(ctx, "session-access-user", true); err != nil || len(values) != 2 {
		t.Fatalf("Auth.GetTokenValueListByLoginID() = %v, %v", values, err)
	}
	if values, err := auth.GetTokenValueListByDevice(ctx, "session-access-user", "web", true); err != nil || len(values) != 1 || values[0] != token1 {
		t.Fatalf("Auth.GetTokenValueListByDevice() = %v, %v", values, err)
	}
	if values, err := auth.GetTokenValueListByDeviceAndDeviceID(ctx, "session-access-user", "mobile", "phone", true); err != nil || len(values) != 1 || values[0] != token2 {
		t.Fatalf("Auth.GetTokenValueListByDeviceAndDeviceID() = %v, %v", values, err)
	}
	if terminals, err := auth.GetTerminalListByLoginID(ctx, "session-access-user"); err != nil || len(terminals) != 2 {
		t.Fatalf("Auth.GetTerminalListByLoginID() = %v, %v", terminals, err)
	}
	if terminals, err := auth.GetTerminalListByLoginIDAndDevice(ctx, "session-access-user", "mobile"); err != nil || len(terminals) != 1 || terminals[0].Token != token2 {
		t.Fatalf("Auth.GetTerminalListByLoginIDAndDevice() = %v, %v", terminals, err)
	}
	if terminal, err := auth.GetTerminalInfoByToken(ctx, token1); err != nil || terminal == nil || terminal.DeviceID != "browser" {
		t.Fatalf("Auth.GetTerminalInfoByToken() = %+v, %v", terminal, err)
	}
	visited := 0
	if err = auth.ForEachTerminal(ctx, "session-access-user", func(manager.TerminalInfo) bool {
		visited++
		return true
	}); err != nil || visited != 2 {
		t.Fatalf("Auth.ForEachTerminal() visited = %d, %v", visited, err)
	}
	deviceVisited := 0
	if err = auth.ForEachTerminalByDevice(ctx, "session-access-user", "web", func(manager.TerminalInfo) bool {
		deviceVisited++
		return true
	}); err != nil || deviceVisited != 1 {
		t.Fatalf("Auth.ForEachTerminalByDevice() visited = %d, %v", deviceVisited, err)
	}
	if err = auth.SetSessionValue(ctx, "session-access-user", "trace", "value"); err != nil {
		t.Fatalf("Auth.SetSessionValue() error = %v", err)
	}
	if value, ok, err := auth.GetSessionValue(ctx, "session-access-user", "trace"); err != nil || !ok || value != "value" {
		t.Fatalf("Auth.GetSessionValue() = %#v/%v, %v", value, ok, err)
	}
	if err = auth.DeleteSessionValue(ctx, "session-access-user", "trace"); err != nil {
		t.Fatalf("Auth.DeleteSessionValue() error = %v", err)
	}
	if value, ok, err := auth.GetSessionValue(ctx, "session-access-user", "trace"); err != nil || ok || value != nil {
		t.Fatalf("Auth.GetSessionValue(after delete) = %#v/%v, %v", value, ok, err)
	}
	if latest, err := auth.GetTokenValueByLoginID(ctx, "session-access-user"); err != nil || latest == "" {
		t.Fatalf("Auth.GetTokenValueByLoginID() = %q, %v", latest, err)
	}
	if latest, err := auth.GetTokenValueByLoginIDAndDevice(ctx, "session-access-user", "web"); err != nil || latest != token1 {
		t.Fatalf("Auth.GetTokenValueByLoginIDAndDevice() = %q, %v", latest, err)
	}
	if values, err := auth.SearchTokenValue(ctx, token1, 0, -1); err != nil || !dtokenContains(values, token1) {
		t.Fatalf("Auth.SearchTokenValue() = %v, %v", values, err)
	}
	if values, err := auth.SearchSessionId(ctx, "session-access-user", 0, -1); err != nil || !dtokenContains(values, "session-access-user") {
		t.Fatalf("Auth.SearchSessionId() = %v, %v", values, err)
	}

	requireFacadeNoError(t, "Auth.AddPermissions(login)", auth.AddPermissions(ctx, PermissionOptions{LoginID: "session-access-user", Permissions: []string{"read", "write"}}))
	requireFacadeNoError(t, "Auth.AddPermissionsByToken", auth.AddPermissionsByToken(ctx, token1, []string{"delete"}))
	requireFacadeNoError(t, "Auth.CheckPermission(login)", auth.CheckPermission(ctx, PermissionOptions{LoginID: "session-access-user", Permission: "read"}))
	requireFacadeNoError(t, "Auth.CheckPermission(token)", auth.CheckPermission(ctx, PermissionOptions{Token: token1, Permission: "delete"}))
	requireFacadeNoError(t, "Auth.CheckPermissionsAnd(login)", auth.CheckPermissionsAnd(ctx, PermissionOptions{LoginID: "session-access-user", Permissions: []string{"read", "write"}}))
	requireFacadeNoError(t, "Auth.CheckPermissionsAnd(token)", auth.CheckPermissionsAnd(ctx, PermissionOptions{Token: token1, Permissions: []string{"read", "delete"}}))
	requireFacadeNoError(t, "Auth.CheckPermissionsOr(login)", auth.CheckPermissionsOr(ctx, PermissionOptions{LoginID: "session-access-user", Permissions: []string{"missing", "write"}}))
	requireFacadeNoError(t, "Auth.CheckPermissionsOr(token)", auth.CheckPermissionsOr(ctx, PermissionOptions{Token: token1, Permissions: []string{"missing", "delete"}}))
	if permissions, err := auth.GetPermissions(ctx, "session-access-user"); err != nil || !dtokenContains(permissions, "read") || !dtokenContains(permissions, "delete") {
		t.Fatalf("Auth.GetPermissions() = %v, %v", permissions, err)
	}
	if permissions, err := auth.GetPermissionsByToken(ctx, token1); err != nil || !dtokenContains(permissions, "write") {
		t.Fatalf("Auth.GetPermissionsByToken() = %v, %v", permissions, err)
	}
	if !auth.HasPermission(ctx, "session-access-user", "read") || !auth.HasPermissionByToken(ctx, token1, "delete") || !auth.HasPermissionsAnd(ctx, "session-access-user", []string{"read", "write"}) || !auth.HasPermissionsAndByToken(ctx, token1, []string{"read", "delete"}) || !auth.HasPermissionsOr(ctx, "session-access-user", []string{"missing", "write"}) || !auth.HasPermissionsOrByToken(ctx, token1, []string{"missing", "delete"}) {
		t.Fatal("instance permission predicates returned an unexpected result")
	}
	requireFacadeNoError(t, "Auth.CheckPermissionByToken", auth.CheckPermissionByToken(ctx, token1, "delete"))
	requireFacadeNoError(t, "Auth.CheckPermissionAnd", auth.CheckPermissionAnd(ctx, "session-access-user", []string{"read", "write"}))
	requireFacadeNoError(t, "Auth.CheckPermissionAndByToken", auth.CheckPermissionAndByToken(ctx, token1, []string{"read", "delete"}))
	requireFacadeNoError(t, "Auth.CheckPermissionOr", auth.CheckPermissionOr(ctx, "session-access-user", []string{"missing", "write"}))
	requireFacadeNoError(t, "Auth.CheckPermissionOrByToken", auth.CheckPermissionOrByToken(ctx, token1, []string{"missing", "delete"}))
	requireFacadeNoError(t, "Auth.RemovePermissions", auth.RemovePermissions(ctx, PermissionOptions{LoginID: "session-access-user", Permissions: []string{"write"}}))
	requireFacadeNoError(t, "Auth.RemovePermissionsByToken", auth.RemovePermissionsByToken(ctx, token1, []string{"delete"}))

	requireFacadeNoError(t, "Auth.AddRoles(login)", auth.AddRoles(ctx, RoleOptions{LoginID: "session-access-user", Roles: []string{"reader", "operator"}}))
	requireFacadeNoError(t, "Auth.AddRolesByToken", auth.AddRolesByToken(ctx, token1, []string{"admin"}))
	requireFacadeNoError(t, "Auth.CheckRole(login)", auth.CheckRole(ctx, RoleOptions{LoginID: "session-access-user", Role: "reader"}))
	requireFacadeNoError(t, "Auth.CheckRole(token)", auth.CheckRole(ctx, RoleOptions{Token: token1, Role: "admin"}))
	requireFacadeNoError(t, "Auth.CheckRolesAnd(login)", auth.CheckRolesAnd(ctx, RoleOptions{LoginID: "session-access-user", Roles: []string{"reader", "operator"}}))
	requireFacadeNoError(t, "Auth.CheckRolesAnd(token)", auth.CheckRolesAnd(ctx, RoleOptions{Token: token1, Roles: []string{"reader", "admin"}}))
	requireFacadeNoError(t, "Auth.CheckRolesOr(login)", auth.CheckRolesOr(ctx, RoleOptions{LoginID: "session-access-user", Roles: []string{"missing", "operator"}}))
	requireFacadeNoError(t, "Auth.CheckRolesOr(token)", auth.CheckRolesOr(ctx, RoleOptions{Token: token1, Roles: []string{"missing", "admin"}}))
	if roles, err := auth.GetRoles(ctx, "session-access-user"); err != nil || !dtokenContains(roles, "reader") || !dtokenContains(roles, "admin") {
		t.Fatalf("Auth.GetRoles() = %v, %v", roles, err)
	}
	if roles, err := auth.GetRolesByToken(ctx, token1); err != nil || !dtokenContains(roles, "operator") {
		t.Fatalf("Auth.GetRolesByToken() = %v, %v", roles, err)
	}
	if !auth.HasRole(ctx, "session-access-user", "reader") || !auth.HasRoleByToken(ctx, token1, "admin") || !auth.HasRolesAnd(ctx, "session-access-user", []string{"reader", "operator"}) || !auth.HasRolesAndByToken(ctx, token1, []string{"reader", "admin"}) || !auth.HasRolesOr(ctx, "session-access-user", []string{"missing", "operator"}) || !auth.HasRolesOrByToken(ctx, token1, []string{"missing", "admin"}) {
		t.Fatal("instance role predicates returned an unexpected result")
	}
	requireFacadeNoError(t, "Auth.CheckRoleByToken", auth.CheckRoleByToken(ctx, token1, "admin"))
	requireFacadeNoError(t, "Auth.CheckRoleAnd", auth.CheckRoleAnd(ctx, "session-access-user", []string{"reader", "operator"}))
	requireFacadeNoError(t, "Auth.CheckRoleAndByToken", auth.CheckRoleAndByToken(ctx, token1, []string{"reader", "admin"}))
	requireFacadeNoError(t, "Auth.CheckRoleOr", auth.CheckRoleOr(ctx, "session-access-user", []string{"missing", "operator"}))
	requireFacadeNoError(t, "Auth.CheckRoleOrByToken", auth.CheckRoleOrByToken(ctx, token1, []string{"missing", "admin"}))
	requireFacadeNoError(t, "Auth.RemoveRoles", auth.RemoveRoles(ctx, RoleOptions{LoginID: "session-access-user", Roles: []string{"operator"}}))
	requireFacadeNoError(t, "Auth.RemoveRolesByToken", auth.RemoveRolesByToken(ctx, token1, []string{"admin"}))
}

// TestInstanceDisableAndCredentialFacades verifies instance disable, ticket, and short-key forwarding. TestInstanceDisableAndCredentialFacades 验证实例封禁、Ticket 和短 Key 转发。
func TestInstanceDisableAndCredentialFacades(t *testing.T) {
	ctx := context.Background()
	mgr, err := NewBuilder().IsPrintBanner(false).AutoRenew(false).EnableTicket().EnableShortKey().AuthType("instance-disable-credential").Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	auth := New(mgr)
	t.Cleanup(auth.Close)

	requireFacadeNoError(t, "Auth.Disable", auth.Disable(ctx, DisableOptions{LoginID: "disabled-user", Duration: time.Minute, Reason: "risk"}))
	if !auth.IsDisable(ctx, "disabled-user") {
		t.Fatal("Auth.IsDisable() = false, want true")
	}
	if err := auth.CheckDisable(ctx, "disabled-user"); !errors.Is(err, derror.ErrAccountDisabled) {
		t.Fatalf("Auth.CheckDisable() error = %v, want ErrAccountDisabled", err)
	}
	if info, err := auth.GetDisableInfo(ctx, "disabled-user"); err != nil || info == nil || info.DisableReason != "risk" {
		t.Fatalf("Auth.GetDisableInfo() = %+v, %v", info, err)
	}
	if ttl, err := auth.GetDisableTTL(ctx, "disabled-user"); err != nil || ttl <= 0 {
		t.Fatalf("Auth.GetDisableTTL() = %d, %v", ttl, err)
	}
	requireFacadeNoError(t, "Auth.Untie", auth.Untie(ctx, "disabled-user"))

	requireFacadeNoError(t, "Auth.DisableService(options)", auth.DisableService(ctx, ServiceDisableOptions{LoginID: "service-user", Service: "billing", Duration: time.Minute, Reason: "service-risk"}))
	requireFacadeNoError(t, "Auth.DisableServiceWithReason", auth.DisableServiceWithReason(ctx, "service-user", "reports", time.Minute, "reports-risk"))
	requireFacadeNoError(t, "Auth.DisableServiceLevel", auth.DisableServiceLevel(ctx, "service-user", "payments", 3, time.Minute))
	requireFacadeNoError(t, "Auth.DisableServiceLevelWithReason", auth.DisableServiceLevelWithReason(ctx, "service-user", "admin", 4, time.Minute, "admin-risk"))
	if !auth.IsDisableService(ctx, "service-user", "billing") || !auth.IsDisableServiceLevel(ctx, "service-user", "payments", 2) || auth.IsDisableServiceLevel(ctx, "service-user", "payments", 4) {
		t.Fatal("instance service disable predicates returned an unexpected result")
	}
	if err := auth.CheckDisableService(ctx, "service-user", "missing", "billing"); !errors.Is(err, derror.ErrServiceDisabled) {
		t.Fatalf("Auth.CheckDisableService() error = %v, want ErrServiceDisabled", err)
	}
	if err := auth.CheckDisableServiceLevel(ctx, "service-user", "payments", 2); !errors.Is(err, derror.ErrServiceDisabled) {
		t.Fatalf("Auth.CheckDisableServiceLevel() error = %v, want ErrServiceDisabled", err)
	}
	if info, err := auth.GetDisableServiceInfo(ctx, "service-user", "billing"); err != nil || info == nil || info.DisableReason != "service-risk" {
		t.Fatalf("Auth.GetDisableServiceInfo() = %+v, %v", info, err)
	}
	if ttl, err := auth.GetDisableServiceTTL(ctx, "service-user", "billing"); err != nil || ttl <= 0 {
		t.Fatalf("Auth.GetDisableServiceTTL() = %d, %v", ttl, err)
	}
	requireFacadeNoError(t, "Auth.UntieService", auth.UntieService(ctx, "service-user", "billing"))

	requireFacadeNoError(t, "Auth.DisableDevice(type)", auth.DisableDevice(ctx, DeviceDisableOptions{LoginID: "device-user", Device: "web", Duration: time.Minute, Reason: "device-risk"}))
	requireFacadeNoError(t, "Auth.DisableDevice(type, reason)", auth.DisableDeviceWithReason(ctx, "device-user", "mobile", time.Minute, "mobile-risk"))
	requireFacadeNoError(t, "Auth.DisableDevice(concrete)", auth.DisableDeviceAndDeviceID(ctx, "device-user", "tablet", "tablet-1", time.Minute))
	requireFacadeNoError(t, "Auth.DisableDevice(concrete, reason)", auth.DisableDeviceAndDeviceIDWithReason(ctx, "device-user", "desktop", "desktop-1", time.Minute, "desktop-risk"))
	if !auth.IsDisableDevice(ctx, "device-user", "web") || !auth.IsDisableDeviceAndDeviceID(ctx, "device-user", "tablet", "tablet-1") {
		t.Fatal("instance device disable predicates returned false")
	}
	if err := auth.CheckDisableDevice(ctx, "device-user", "web"); !errors.Is(err, derror.ErrDeviceDisabled) {
		t.Fatalf("Auth.CheckDisableDevice() error = %v, want ErrDeviceDisabled", err)
	}
	if err := auth.CheckDisableDeviceAndDeviceID(ctx, "device-user", "tablet", "tablet-1"); !errors.Is(err, derror.ErrDeviceDisabled) {
		t.Fatalf("Auth.CheckDisableDeviceAndDeviceID() error = %v, want ErrDeviceDisabled", err)
	}
	if info, err := auth.GetDisableDeviceInfo(ctx, "device-user", "mobile"); err != nil || info == nil || info.DisableReason != "mobile-risk" {
		t.Fatalf("Auth.GetDisableDeviceInfo() = %+v, %v", info, err)
	}
	if info, err := auth.GetDisableDeviceAndDeviceIDInfo(ctx, "device-user", "desktop", "desktop-1"); err != nil || info == nil || info.DisableReason != "desktop-risk" {
		t.Fatalf("Auth.GetDisableDeviceAndDeviceIDInfo() = %+v, %v", info, err)
	}
	if ttl, err := auth.GetDisableDeviceTTL(ctx, "device-user", "web"); err != nil || ttl <= 0 {
		t.Fatalf("Auth.GetDisableDeviceTTL() = %d, %v", ttl, err)
	}
	if ttl, err := auth.GetDisableDeviceAndDeviceIDTTL(ctx, "device-user", "desktop", "desktop-1"); err != nil || ttl <= 0 {
		t.Fatalf("Auth.GetDisableDeviceAndDeviceIDTTL() = %d, %v", ttl, err)
	}
	requireFacadeNoError(t, "Auth.UntieDevice", auth.UntieDevice(ctx, "device-user", "web"))
	requireFacadeNoError(t, "Auth.UntieDeviceAndDeviceID", auth.UntieDeviceAndDeviceID(ctx, "device-user", "tablet", "tablet-1"))

	ticketValue, err := auth.CreateTicket(ctx, "ticket-user")
	if err != nil {
		t.Fatalf("Auth.CreateTicket() error = %v", err)
	}
	if validated, err := auth.ValidateTicket(ctx, ticketValue.Ticket); err != nil || validated == nil || validated.LoginID != "ticket-user" {
		t.Fatalf("Auth.ValidateTicket() = %+v, %v", validated, err)
	}
	if ttl, err := auth.GetTicketTTL(ctx, ticketValue.Ticket); err != nil || ttl <= 0 {
		t.Fatalf("Auth.GetTicketTTL() = %d, %v", ttl, err)
	}
	if status, err := auth.GetTicketStatus(ctx, ticketValue.Ticket); err != nil || status != ticket.StatusValid {
		t.Fatalf("Auth.GetTicketStatus() = %q, %v", status, err)
	}
	optionTicket, err := auth.CreateTicketWithOptions(ctx, ticket.CreateOptions{LoginID: "option-ticket", Device: "web", DeviceID: "browser", SourceApp: "issuer", TargetApp: "consumer", Timeout: time.Minute})
	if err != nil {
		t.Fatalf("Auth.CreateTicketWithOptions() error = %v", err)
	}
	if validated, err := auth.ValidateTicketWithOptions(ctx, optionTicket.Ticket, ticket.ValidateOptions{LoginID: "option-ticket", Device: "web"}); err != nil || validated.TargetApp != "consumer" {
		t.Fatalf("Auth.ValidateTicketWithOptions() = %+v, %v", validated, err)
	}
	if result, err := auth.ConsumeTicket(ctx, ticketValue.Ticket); err != nil || result == nil || result.Ticket == nil || result.Ticket.Status != ticket.StatusConsumed {
		t.Fatalf("Auth.ConsumeTicket() = %+v, %v", result, err)
	}
	if result, err := auth.ConsumeTicketWithOptions(ctx, optionTicket.Ticket, ticket.ValidateOptions{LoginID: "option-ticket"}); err != nil || result == nil || result.Ticket.Status != ticket.StatusConsumed {
		t.Fatalf("Auth.ConsumeTicketWithOptions() = %+v, %v", result, err)
	}
	requireFacadeNoError(t, "Auth.RevokeTicket", auth.RevokeTicket(ctx, ticketValue.Ticket))

	shortKeyValue, err := auth.CreateShortKey(ctx)
	if err != nil {
		t.Fatalf("Auth.CreateShortKey() error = %v", err)
	}
	if status, err := auth.GetShortKeyStatus(ctx, shortKeyValue.Key); err != nil || status != shortkey.StatusPending {
		t.Fatalf("Auth.GetShortKeyStatus() = %q, %v", status, err)
	}
	if ttl, err := auth.GetShortKeyTTL(ctx, shortKeyValue.Key); err != nil || ttl <= 0 {
		t.Fatalf("Auth.GetShortKeyTTL() = %d, %v", ttl, err)
	}
	optionKey, err := auth.CreateShortKeyWithOptions(ctx, shortkey.CreateOptions{Scene: "qr", SourceApp: "issuer", TargetApp: "consumer", Timeout: time.Minute})
	if err != nil {
		t.Fatalf("Auth.CreateShortKeyWithOptions() error = %v", err)
	}
	if confirmed, err := auth.ConfirmShortKey(ctx, shortKeyValue.Key, "short-user"); err != nil || confirmed.LoginID != "short-user" {
		t.Fatalf("Auth.ConfirmShortKey() = %+v, %v", confirmed, err)
	}
	if confirmed, err := auth.ConfirmShortKeyWithOptions(ctx, optionKey.Key, shortkey.ConfirmOptions{LoginID: "option-short", Device: "web", DeviceID: "browser"}); err != nil || confirmed.LoginID != "option-short" {
		t.Fatalf("Auth.ConfirmShortKeyWithOptions() = %+v, %v", confirmed, err)
	}
	if validated, err := auth.ValidateShortKey(ctx, shortKeyValue.Key); err != nil || validated.Status != shortkey.StatusConfirmed {
		t.Fatalf("Auth.ValidateShortKey() = %+v, %v", validated, err)
	}
	if validated, err := auth.ValidateShortKeyWithOptions(ctx, optionKey.Key, shortkey.ValidateOptions{LoginID: "option-short", Scene: "qr"}); err != nil || validated.Status != shortkey.StatusConfirmed {
		t.Fatalf("Auth.ValidateShortKeyWithOptions() = %+v, %v", validated, err)
	}
	if result, err := auth.ConsumeShortKey(ctx, shortKeyValue.Key); err != nil || result == nil || result.ShortKey == nil || result.ShortKey.Status != shortkey.StatusConsumed {
		t.Fatalf("Auth.ConsumeShortKey() = %+v, %v", result, err)
	}
	if result, err := auth.ConsumeShortKeyWithOptions(ctx, optionKey.Key, shortkey.ValidateOptions{LoginID: "option-short"}); err != nil || result == nil || result.ShortKey.Status != shortkey.StatusConsumed {
		t.Fatalf("Auth.ConsumeShortKeyWithOptions() = %+v, %v", result, err)
	}
	requireFacadeNoError(t, "Auth.RevokeShortKey", auth.RevokeShortKey(ctx, shortKeyValue.Key))
}

// TestInstanceOAuth2Facade verifies all instance OAuth2 forwarding routes. TestInstanceOAuth2Facade 验证实例 OAuth2 全部转发入口。
func TestInstanceOAuth2Facade(t *testing.T) {
	ctx := context.Background()
	mgr, err := NewBuilder().IsPrintBanner(false).AutoRenew(false).EnableOAuth2().AuthType("instance-oauth2").Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	auth := New(mgr)
	t.Cleanup(auth.Close)

	client := &oauth2.Client{
		ClientID:     "instance-oauth-client",
		ClientSecret: "instance-oauth-secret",
		RedirectURIs: []string{"https://instance.example/callback"},
		GrantTypes: []oauth2.GrantType{
			oauth2.GrantTypeAuthorizationCode,
			oauth2.GrantTypeClientCredentials,
			oauth2.GrantTypePassword,
			oauth2.GrantTypeRefreshToken,
		},
		Scopes: []string{"read", "write"},
	}
	requireFacadeNoError(t, "Auth.RegisterOAuth2Client", auth.RegisterOAuth2Client(client))
	if got, err := auth.GetOAuth2Client(client.ClientID); err != nil || got == nil || got.ClientSecret != client.ClientSecret {
		t.Fatalf("Auth.GetOAuth2Client() = %+v, %v", got, err)
	}

	code, err := auth.GenerateOAuth2AuthorizationCode(ctx, client.ClientID, "code-user", client.RedirectURIs[0], []string{"read"})
	if err != nil {
		t.Fatalf("Auth.GenerateOAuth2AuthorizationCode() error = %v", err)
	}
	issued, err := auth.ExchangeOAuth2CodeForToken(ctx, code.Code, client.ClientID, client.ClientSecret, client.RedirectURIs[0])
	if err != nil || issued == nil || issued.Token == "" {
		t.Fatalf("Auth.ExchangeOAuth2CodeForToken() = %+v, %v", issued, err)
	}

	pkceCode, err := auth.GenerateOAuth2AuthorizationCodeWithPKCE(ctx, client.ClientID, "pkce-user", client.RedirectURIs[0], []string{"write"}, "verifier", oauth2.CodeChallengeMethodPlain)
	if err != nil {
		t.Fatalf("Auth.GenerateOAuth2AuthorizationCodeWithPKCE() error = %v", err)
	}
	if _, err = auth.ExchangeOAuth2CodeForTokenWithPKCE(ctx, pkceCode.Code, client.ClientID, client.ClientSecret, client.RedirectURIs[0], "verifier"); err != nil {
		t.Fatalf("Auth.ExchangeOAuth2CodeForTokenWithPKCE() error = %v", err)
	}

	clientToken, err := auth.OAuth2ClientCredentialsToken(ctx, client.ClientID, client.ClientSecret, []string{"read"})
	if err != nil || clientToken == nil || clientToken.Token == "" {
		t.Fatalf("Auth.OAuth2ClientCredentialsToken() = %+v, %v", clientToken, err)
	}
	passwordToken, err := auth.OAuth2PasswordGrantToken(ctx, client.ClientID, client.ClientSecret, "alice", "password", []string{"write"}, func(username, password string) (string, error) {
		if username == "alice" && password == "password" {
			return "alice-id", nil
		}
		return "", errors.New("invalid credentials")
	})
	if err != nil || passwordToken == nil || passwordToken.UserID != "alice-id" {
		t.Fatalf("Auth.OAuth2PasswordGrantToken() = %+v, %v", passwordToken, err)
	}
	refreshed, err := auth.RefreshOAuth2AccessToken(ctx, client.ClientID, clientToken.RefreshToken, client.ClientSecret)
	if err != nil || refreshed == nil || refreshed.Token == clientToken.Token {
		t.Fatalf("Auth.RefreshOAuth2AccessToken() = %+v, %v", refreshed, err)
	}
	if unified, err := auth.OAuth2Token(ctx, &oauth2.TokenRequest{GrantType: oauth2.GrantTypeRefreshToken, ClientID: client.ClientID, ClientSecret: client.ClientSecret, RefreshToken: refreshed.RefreshToken}, nil); err != nil || unified == nil || unified.Token == refreshed.Token {
		t.Fatalf("Auth.OAuth2Token(refresh) = %+v, %v", unified, err)
	}
	if !auth.ValidateOAuth2AccessToken(ctx, issued.Token) {
		t.Fatal("Auth.ValidateOAuth2AccessToken() = false, want true")
	}
	if info, err := auth.ValidateOAuth2AccessTokenAndGetInfo(ctx, issued.Token); err != nil || info == nil || info.UserID != "code-user" {
		t.Fatalf("Auth.ValidateOAuth2AccessTokenAndGetInfo() = %+v, %v", info, err)
	}
	requireFacadeNoError(t, "Auth.RevokeOAuth2Token", auth.RevokeOAuth2Token(ctx, issued.Token))
	if auth.ValidateOAuth2AccessToken(ctx, issued.Token) {
		t.Fatal("Auth.ValidateOAuth2AccessToken() = true after revoke, want false")
	}
	requireFacadeNoError(t, "Auth.UnregisterOAuth2Client", auth.UnregisterOAuth2Client(client.ClientID))
	if _, err = auth.GetOAuth2Client(client.ClientID); !errors.Is(err, derror.ErrClientNotFound) {
		t.Fatalf("Auth.GetOAuth2Client(after unregister) error = %v, want ErrClientNotFound", err)
	}
}

// TestOptionalFacadeErrorsAfterClose verifies optional status helpers return their documented zero status after close. TestOptionalFacadeErrorsAfterClose 验证关闭后可选状态方法返回约定零值和错误。
func TestOptionalFacadeErrorsAfterClose(t *testing.T) {
	auth := New(nil)
	ctx := context.Background()
	if status, err := auth.GetTicketStatus(ctx, "ticket"); !errors.Is(err, derror.ErrManagerNotFound) || status != ticket.StatusInvalid {
		t.Fatalf("GetTicketStatus(nil manager) = %q, %v", status, err)
	}
	if status, err := auth.GetShortKeyStatus(ctx, "key"); !errors.Is(err, derror.ErrManagerNotFound) || status != shortkey.StatusInvalid {
		t.Fatalf("GetShortKeyStatus(nil manager) = %q, %v", status, err)
	}
	auth.Close()
}

func requireFacadeNoError(t *testing.T, name string, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s error = %v", name, err)
	}
}
