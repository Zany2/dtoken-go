package dtoken

import (
	"context"
	"errors"
	"testing"
	"time"

	djson "github.com/Zany2/dtoken-go/com/codec/json"
	"github.com/Zany2/dtoken-go/com/generator/dgenerator"
	"github.com/Zany2/dtoken-go/com/storage/memory"
	"github.com/Zany2/dtoken-go/core/adapter"
	corebuilder "github.com/Zany2/dtoken-go/core/builder"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/manager"
	"github.com/Zany2/dtoken-go/core/oauth2"
	"github.com/Zany2/dtoken-go/core/shortkey"
	"github.com/Zany2/dtoken-go/core/ticket"
	"github.com/Zany2/dtoken-go/defaults"
)

// TestBuilderCloneIsolationAndMustBuild verifies builder copies and panic boundaries. TestBuilderCloneIsolationAndMustBuild 验证 Builder 克隆隔离和 panic 边界。
func TestBuilderCloneIsolationAndMustBuild(t *testing.T) {
	original := NewBuilder().
		IsPrintBanner(false).
		AutoRenew(false).
		AuthType("builder-original").
		CookiePath("/original").
		LoggerPrefix("original").
		NonceTTL(time.Minute).
		OAuth2TokenExpiration(2 * time.Hour).
		TicketTTL(3 * time.Minute).
		ShortKeyLength(8)
	clone := original.Clone()

	if clone == original || clone.GetConfig() == original.GetConfig() {
		t.Fatal("Clone() should return an independent builder and core config")
	}
	if clone.GetRenewPoolConfig() == original.GetRenewPoolConfig() || clone.GetLoggerConfig() == original.GetLoggerConfig() {
		t.Fatal("Clone() should copy module configs")
	}
	if clone.GetNonceConfig() == original.GetNonceConfig() || clone.GetOAuth2Config() == original.GetOAuth2Config() || clone.GetTicketConfig() == original.GetTicketConfig() || clone.GetShortKeyConfig() == original.GetShortKeyConfig() {
		t.Fatal("Clone() should copy optional module configs")
	}

	clone.AuthType("builder-clone").CookiePath("/clone").LoggerPrefix("clone").NonceTTL(2 * time.Minute).OAuth2TokenExpiration(3 * time.Hour).TicketTTL(4 * time.Minute).ShortKeyLength(10)
	if original.GetConfig().AuthType != "builder-original:" || original.GetConfig().CookieConfig.Path != "/original" {
		t.Fatalf("original config changed through clone: %+v", original.GetConfig())
	}
	if original.GetLoggerConfig().Prefix != "original" || original.GetNonceConfig().TTL != time.Minute || original.GetOAuth2Config().TokenExpiration != 2*time.Hour || original.GetTicketConfig().TTL != 3*time.Minute || original.GetShortKeyConfig().Length != 8 {
		t.Fatal("original module config changed through clone")
	}

	first, err := original.Build()
	if err != nil {
		t.Fatalf("original Build() error = %v", err)
	}
	defer first.CloseManager()
	second, err := clone.Build()
	if err != nil {
		t.Fatalf("clone Build() error = %v", err)
	}
	defer second.CloseManager()
	if first.GetConfig().AuthType != "builder-original:" || second.GetConfig().AuthType != "builder-clone:" {
		t.Fatalf("built auth types = %q/%q", first.GetConfig().AuthType, second.GetConfig().AuthType)
	}

	must := NewBuilder().IsPrintBanner(false).AutoRenew(false).MustBuild()
	must.CloseManager()

	func() {
		defer func() {
			if recovered := recover(); recovered == nil {
				t.Fatal("MustBuild() should panic for invalid module configuration")
			}
		}()
		NewBuilder().IsPrintBanner(false).NonceTTL(0).MustBuild()
	}()
}

// TestBuilderCustomFactoriesReceiveConfig verifies custom factories take precedence and see normalized config. TestBuilderCustomFactoriesReceiveConfig 验证自定义工厂优先级及收到规范化配置。
func TestBuilderCustomFactoriesReceiveConfig(t *testing.T) {
	var generatorCalls, storageCalls, codecCalls, logCalls, poolCalls int
	var seenAuthType string
	customStorage := memory.NewStorage()
	customCodec := djson.NewJSONSerializer()
	customGenerator := dgenerator.NewGenerator(60, "factory-secret", adapter.TokenStyleUUID)
	customLogger := &registryTestLogger{NopLogger: adapter.NewNopLogger()}
	customPool := &registryTestPool{}

	mgr, err := NewBuilder().
		IsPrintBanner(false).
		IsLog(true).
		AutoRenew(true).
		AuthType("factory").
		SetGeneratorFactory(func(cfg *config.Config) (adapter.Generator, error) {
			generatorCalls++
			seenAuthType = cfg.AuthType
			return customGenerator, nil
		}).
		SetStorageFactory(func(cfg *config.Config) (adapter.Storage, error) {
			storageCalls++
			seenAuthType = cfg.AuthType
			return customStorage, nil
		}).
		SetCodecFactory(func(cfg *config.Config) (adapter.Codec, error) {
			codecCalls++
			seenAuthType = cfg.AuthType
			return customCodec, nil
		}).
		SetLogFactory(func(cfg *config.Config) (adapter.Log, error) {
			logCalls++
			seenAuthType = cfg.AuthType
			return customLogger, nil
		}).
		SetPoolFactory(func(cfg *config.Config) (adapter.Pool, error) {
			poolCalls++
			seenAuthType = cfg.AuthType
			return customPool, nil
		}).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if generatorCalls != 1 || storageCalls != 1 || codecCalls != 1 || logCalls != 1 || poolCalls != 1 {
		t.Fatalf("factory calls = generator:%d storage:%d codec:%d log:%d pool:%d, want one each", generatorCalls, storageCalls, codecCalls, logCalls, poolCalls)
	}
	if seenAuthType != "factory:" {
		t.Fatalf("factory auth type = %q, want factory:", seenAuthType)
	}
	if mgr.GetGenerator() != customGenerator || mgr.GetStorage() != customStorage || mgr.GetSerializer() != customCodec || mgr.GetLogger() != customLogger || mgr.GetPool() != customPool {
		t.Fatal("custom factory components were not wired into manager")
	}
	mgr.CloseManager()
	if customPool.stops() != 1 || customLogger.closes() != 1 {
		t.Fatalf("factory component close counts = pool:%d logger:%d, want one each", customPool.stops(), customLogger.closes())
	}
}

// TestRegistryDefaultAndBuildAndSetCoreBuilder verifies default lookup and core-builder registration. TestRegistryDefaultAndBuildAndSetCoreBuilder 验证默认查找和 core Builder 注册。
func TestRegistryDefaultAndBuildAndSetCoreBuilder(t *testing.T) {
	DeleteAllManager()
	t.Cleanup(DeleteAllManager)

	core := defaults.NewBuilder().IsPrintBanner(false).AutoRenew(false)
	mgr, err := BuildAndSetManager(core, "core-override")
	if err != nil {
		t.Fatalf("BuildAndSetManager(core builder) error = %v", err)
	}
	if mgr.GetConfig().AuthType != "core-override:" {
		t.Fatalf("core builder auth type = %q, want core-override:", mgr.GetConfig().AuthType)
	}
	if got, err := GetManager(); got != nil || !errors.Is(err, derror.ErrManagerNotFound) {
		t.Fatalf("GetManager() = %v, %v, want default manager not found", got, err)
	}
	if got, err := GetManager("core-override"); err != nil || got != mgr {
		t.Fatalf("GetManager(core-override) = %v, %v", got, err)
	}
	if err := DeleteManager("core-override"); err != nil {
		t.Fatalf("DeleteManager() error = %v", err)
	}
	if err := DeleteManager("core-override"); !errors.Is(err, derror.ErrManagerNotFound) {
		t.Fatalf("second DeleteManager() error = %v, want ErrManagerNotFound", err)
	}
}

// TestDefaultAndMustDefaultSuccess verifies default registry accessors return the registered manager. TestDefaultAndMustDefaultSuccess 验证默认注册表访问器返回已注册的 Manager。
func TestDefaultAndMustDefaultSuccess(t *testing.T) {
	DeleteAllManager()
	t.Cleanup(DeleteAllManager)

	mgr, err := NewBuilder().IsPrintBanner(false).AutoRenew(false).Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	SetManager(mgr)
	auth, err := Default()
	if err != nil || auth.Manager() != mgr {
		t.Fatalf("Default() = %v, %v, want registered manager", auth, err)
	}
	auth.Close()

	// Register a fresh manager for the panic-free MustDefault success path. 为 MustDefault 成功路径重新注册一个 Manager。
	mgr, err = NewBuilder().IsPrintBanner(false).AutoRenew(false).Build()
	if err != nil {
		t.Fatalf("second Build() error = %v", err)
	}
	SetManager(mgr)
	must := MustDefault()
	if must.Manager() != mgr {
		t.Fatal("MustDefault() did not return the registered default manager")
	}
	must.Close()
}

// TestMustDefaultPreservesRegistryError verifies MustDefault does not hide registry errors. TestMustDefaultPreservesRegistryError 验证 MustDefault 不会丢失注册表错误。
func TestMustDefaultPreservesRegistryError(t *testing.T) {
	DeleteAllManager()
	t.Cleanup(DeleteAllManager)
	globalManagerMap.Store(config.DefaultAuthType, struct{}{})
	defer globalManagerMap.Delete(config.DefaultAuthType)

	defer func() {
		recovered := recover()
		err, ok := recovered.(error)
		if !ok || !errors.Is(err, derror.ErrManagerInvalidType) {
			t.Fatalf("MustDefault() panic = %#v, want ErrManagerInvalidType", recovered)
		}
	}()
	MustDefault()
}

// TestGlobalSessionFacades verifies global session, token metadata, search, and terminal iteration helpers. TestGlobalSessionFacades 验证全局会话、Token 元数据、搜索和终端遍历门面。
func TestGlobalSessionFacades(t *testing.T) {
	DeleteAllManager()
	t.Cleanup(DeleteAllManager)

	ctx := context.Background()
	mgr, err := NewBuilder().IsPrintBanner(false).AutoRenew(false).IsShare(false).AuthType("session-facade").Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	SetManager(mgr)

	token1, err := Login(ctx, "session-user", "web", "browser-1", "session-facade")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	token2, err := LoginWithTimeout(ctx, "session-user", 2*time.Minute, "mobile", "phone-1", "session-facade")
	if err != nil {
		t.Fatalf("LoginWithTimeout() error = %v", err)
	}
	if token1 == token2 || !IsLogin(ctx, token1, "session-facade") || !IsLogin(ctx, token2, "session-facade") {
		t.Fatal("expected two distinct active tokens")
	}
	if err = LoginByToken(ctx, token1, "session-facade"); err != nil {
		t.Fatalf("LoginByToken() error = %v", err)
	}
	if err = CheckLogin(ctx, token1, "session-facade"); err != nil {
		t.Fatalf("CheckLogin() error = %v", err)
	}

	info, err := GetTokenInfo(ctx, token1, "session-facade")
	if err != nil || info.LoginID != "session-user" || info.Device != "web" || info.DeviceID != "browser-1" {
		t.Fatalf("GetTokenInfo() = %+v, %v", info, err)
	}
	if loginID, err := GetLoginID(ctx, token1, "session-facade"); err != nil || loginID != "session-user" {
		t.Fatalf("GetLoginID() = %q, %v", loginID, err)
	}
	if deviceID, err := GetDeviceID(ctx, token1, "session-facade"); err != nil || deviceID != "browser-1" {
		t.Fatalf("GetDeviceID() = %q, %v", deviceID, err)
	}
	if createTime, err := GetTokenCreateTime(ctx, token1, "session-facade"); err != nil || createTime <= 0 {
		t.Fatalf("GetTokenCreateTime() = %d, %v", createTime, err)
	}
	if ttl, err := GetTokenTTL(ctx, token1, "session-facade"); err != nil || ttl <= 0 {
		t.Fatalf("GetTokenTTL() = %d, %v", ttl, err)
	}
	if err = RenewTimeout(ctx, token1, time.Minute, "session-facade"); err != nil {
		t.Fatalf("RenewTimeout() error = %v", err)
	}
	if introspection, err := IntrospectToken(ctx, token1, "session-facade"); err != nil || !introspection.Active {
		t.Fatalf("IntrospectToken() = %+v, %v", introspection, err)
	}

	if count, err := GetOnlineTerminalCount(ctx, "session-user", "session-facade"); err != nil || count != 2 {
		t.Fatalf("GetOnlineTerminalCount() = %d, %v", count, err)
	}
	if count, err := GetOnlineTerminalCountByDevice(ctx, "session-user", "web", "session-facade"); err != nil || count != 1 {
		t.Fatalf("GetOnlineTerminalCountByDevice() = %d, %v", count, err)
	}
	if count, err := GetOnlineTerminalCountByDeviceAndDeviceID(ctx, "session-user", "web", "browser-1", "session-facade"); err != nil || count != 1 {
		t.Fatalf("GetOnlineTerminalCountByDeviceAndDeviceID() = %d, %v", count, err)
	}
	if values, err := GetTokenValueListByLoginID(ctx, "session-user", true, "session-facade"); err != nil || len(values) != 2 {
		t.Fatalf("GetTokenValueListByLoginID() = %v, %v", values, err)
	}
	if values, err := GetTokenValueListByDevice(ctx, "session-user", "web", true, "session-facade"); err != nil || len(values) != 1 || values[0] != token1 {
		t.Fatalf("GetTokenValueListByDevice() = %v, %v", values, err)
	}
	if values, err := GetTokenValueListByDeviceAndDeviceID(ctx, "session-user", "mobile", "phone-1", true, "session-facade"); err != nil || len(values) != 1 || values[0] != token2 {
		t.Fatalf("GetTokenValueListByDeviceAndDeviceID() = %v, %v", values, err)
	}

	sess, err := GetSession(ctx, "session-user", "session-facade")
	if err != nil || sess.LoginID != "session-user" || len(sess.TerminalInfos) != 2 {
		t.Fatalf("GetSession() = %+v, %v", sess, err)
	}
	byToken, err := GetSessionByToken(ctx, token1, "session-facade")
	if err != nil || byToken.LoginID != sess.LoginID {
		t.Fatalf("GetSessionByToken() = %+v, %v", byToken, err)
	}
	terminals, err := GetTerminalListByLoginID(ctx, "session-user", "session-facade")
	if err != nil || len(terminals) != 2 {
		t.Fatalf("GetTerminalListByLoginID() = %v, %v", terminals, err)
	}
	filtered, err := GetTerminalListByLoginIDAndDevice(ctx, "session-user", "mobile", "session-facade")
	if err != nil || len(filtered) != 1 || filtered[0].Token != token2 {
		t.Fatalf("GetTerminalListByLoginIDAndDevice() = %v, %v", filtered, err)
	}
	terminal, err := GetTerminalInfoByToken(ctx, token1, "session-facade")
	if err != nil || terminal.DeviceID != "browser-1" {
		t.Fatalf("GetTerminalInfoByToken() = %+v, %v", terminal, err)
	}

	visited := 0
	if err = ForEachTerminal(ctx, "session-user", func(_ manager.TerminalInfo) bool {
		visited++
		return visited < 2
	}, "session-facade"); err != nil || visited != 2 {
		t.Fatalf("ForEachTerminal() visited = %d, %v", visited, err)
	}
	deviceVisited := 0
	if err = ForEachTerminalByDevice(ctx, "session-user", "web", func(_ manager.TerminalInfo) bool {
		deviceVisited++
		return true
	}, "session-facade"); err != nil || deviceVisited != 1 {
		t.Fatalf("ForEachTerminalByDevice() visited = %d, %v", deviceVisited, err)
	}

	if err = SetSessionValue(ctx, "session-user", " trace ", "value", "session-facade"); err != nil {
		t.Fatalf("SetSessionValue() error = %v", err)
	}
	if value, ok, err := GetSessionValue(ctx, "session-user", "trace", "session-facade"); err != nil || !ok || value != "value" {
		t.Fatalf("GetSessionValue() = %#v/%v, %v", value, ok, err)
	}
	if err = DeleteSessionValue(ctx, "session-user", "trace", "session-facade"); err != nil {
		t.Fatalf("DeleteSessionValue() error = %v", err)
	}
	if _, ok, err := GetSessionValue(ctx, "session-user", "trace", "session-facade"); err != nil || ok {
		t.Fatalf("GetSessionValue(after delete) = ok:%v, err:%v", ok, err)
	}
	if latest, err := GetTokenValueByLoginID(ctx, "session-user", "session-facade"); err != nil || latest != token2 {
		t.Fatalf("GetTokenValueByLoginID() = %q, %v", latest, err)
	}
	if latest, err := GetTokenValueByLoginIDAndDevice(ctx, "session-user", "web", "session-facade"); err != nil || latest != token1 {
		t.Fatalf("GetTokenValueByLoginIDAndDevice() = %q, %v", latest, err)
	}
	if values, err := SearchTokenValue(ctx, token1, 0, -1, "session-facade"); err != nil || !dtokenContains(values, token1) {
		t.Fatalf("SearchTokenValue() = %v, %v", values, err)
	}
	if values, err := SearchSessionId(ctx, "session-user", 0, -1, "session-facade"); err != nil || !dtokenContains(values, "session-user") {
		t.Fatalf("SearchSessionId() = %v, %v", values, err)
	}

	pair, err := LoginWithRefreshTokenOptions(ctx, RefreshTokenOptions{LoginOptions: LoginOptions{AuthType: "session-facade", LoginID: "refresh-options"}, RefreshTimeout: time.Minute}, "session-facade")
	if err != nil || pair == nil || pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatalf("LoginWithRefreshTokenOptions() = %+v, %v", pair, err)
	}
	if err = RevokeRefreshToken(ctx, pair.RefreshToken, "session-facade"); err != nil {
		t.Fatalf("RevokeRefreshToken() error = %v", err)
	}
}

// TestGlobalPermissionAndRoleFacades verifies every global permission and role operation family. TestGlobalPermissionAndRoleFacades 验证全局权限和角色操作族。
func TestGlobalPermissionAndRoleFacades(t *testing.T) {
	DeleteAllManager()
	t.Cleanup(DeleteAllManager)
	ctx := context.Background()
	mgr, err := NewBuilder().IsPrintBanner(false).AutoRenew(false).AuthType("access-facade").Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	SetManager(mgr)
	token, err := Login(ctx, "access-user", "web", "browser", "access-facade")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	if err = AddPermissions(ctx, "access-user", []string{"read", "write"}, "access-facade"); err != nil {
		t.Fatalf("AddPermissions() error = %v", err)
	}
	if err = AddPermissionsByToken(ctx, token, []string{"delete"}, "access-facade"); err != nil {
		t.Fatalf("AddPermissionsByToken() error = %v", err)
	}
	if permissions, err := GetPermissions(ctx, "access-user", "access-facade"); err != nil || !dtokenContains(permissions, "read") || !dtokenContains(permissions, "delete") {
		t.Fatalf("GetPermissions() = %v, %v", permissions, err)
	}
	if permissions, err := GetPermissionsByToken(ctx, token, "access-facade"); err != nil || !dtokenContains(permissions, "write") {
		t.Fatalf("GetPermissionsByToken() = %v, %v", permissions, err)
	}
	if !HasPermission(ctx, "access-user", "read", "access-facade") || !HasPermissionByToken(ctx, token, "delete", "access-facade") || !HasPermissionsAnd(ctx, "access-user", []string{"read", "write"}, "access-facade") || !HasPermissionsAndByToken(ctx, token, []string{"read", "delete"}, "access-facade") || !HasPermissionsOr(ctx, "access-user", []string{"missing", "write"}, "access-facade") || !HasPermissionsOrByToken(ctx, token, []string{"missing", "delete"}, "access-facade") {
		t.Fatal("permission predicates returned an unexpected result")
	}
	for _, check := range []func() error{
		func() error { return CheckPermission(ctx, "access-user", "read", "access-facade") },
		func() error { return CheckPermissionByToken(ctx, token, "delete", "access-facade") },
		func() error {
			return CheckPermissionAnd(ctx, "access-user", []string{"read", "write"}, "access-facade")
		},
		func() error {
			return CheckPermissionAndByToken(ctx, token, []string{"read", "delete"}, "access-facade")
		},
		func() error {
			return CheckPermissionOr(ctx, "access-user", []string{"missing", "write"}, "access-facade")
		},
		func() error {
			return CheckPermissionOrByToken(ctx, token, []string{"missing", "delete"}, "access-facade")
		},
	} {
		if err := check(); err != nil {
			t.Fatalf("permission check error = %v", err)
		}
	}
	if err = RemovePermissionsByToken(ctx, token, []string{"delete"}, "access-facade"); err != nil {
		t.Fatalf("RemovePermissionsByToken() error = %v", err)
	}
	if err = RemovePermissions(ctx, "access-user", []string{"write"}, "access-facade"); err != nil {
		t.Fatalf("RemovePermissions() error = %v", err)
	}
	if HasPermission(ctx, "access-user", "write", "access-facade") || HasPermissionByToken(ctx, token, "delete", "access-facade") {
		t.Fatal("removed permissions are still visible")
	}

	if err = AddRoles(ctx, "access-user", []string{"reader", "operator"}, "access-facade"); err != nil {
		t.Fatalf("AddRoles() error = %v", err)
	}
	if err = AddRolesByToken(ctx, token, []string{"admin"}, "access-facade"); err != nil {
		t.Fatalf("AddRolesByToken() error = %v", err)
	}
	if roles, err := GetRoles(ctx, "access-user", "access-facade"); err != nil || !dtokenContains(roles, "reader") || !dtokenContains(roles, "admin") {
		t.Fatalf("GetRoles() = %v, %v", roles, err)
	}
	if roles, err := GetRolesByToken(ctx, token, "access-facade"); err != nil || !dtokenContains(roles, "operator") {
		t.Fatalf("GetRolesByToken() = %v, %v", roles, err)
	}
	if !HasRole(ctx, "access-user", "reader", "access-facade") || !HasRoleByToken(ctx, token, "admin", "access-facade") || !HasRolesAnd(ctx, "access-user", []string{"reader", "operator"}, "access-facade") || !HasRolesAndByToken(ctx, token, []string{"reader", "admin"}, "access-facade") || !HasRolesOr(ctx, "access-user", []string{"missing", "operator"}, "access-facade") || !HasRolesOrByToken(ctx, token, []string{"missing", "admin"}, "access-facade") {
		t.Fatal("role predicates returned an unexpected result")
	}
	for _, check := range []func() error{
		func() error { return CheckRole(ctx, "access-user", "reader", "access-facade") },
		func() error { return CheckRoleByToken(ctx, token, "admin", "access-facade") },
		func() error { return CheckRoleAnd(ctx, "access-user", []string{"reader", "operator"}, "access-facade") },
		func() error { return CheckRoleAndByToken(ctx, token, []string{"reader", "admin"}, "access-facade") },
		func() error { return CheckRoleOr(ctx, "access-user", []string{"missing", "operator"}, "access-facade") },
		func() error { return CheckRoleOrByToken(ctx, token, []string{"missing", "admin"}, "access-facade") },
	} {
		if err := check(); err != nil {
			t.Fatalf("role check error = %v", err)
		}
	}
	if err = RemoveRolesByToken(ctx, token, []string{"admin"}, "access-facade"); err != nil {
		t.Fatalf("RemoveRolesByToken() error = %v", err)
	}
	if err = RemoveRoles(ctx, "access-user", []string{"operator"}, "access-facade"); err != nil {
		t.Fatalf("RemoveRoles() error = %v", err)
	}
	if HasRole(ctx, "access-user", "operator", "access-facade") || HasRoleByToken(ctx, token, "admin", "access-facade") {
		t.Fatal("removed roles are still visible")
	}
}

// TestGlobalTypedTerminationAndAccessOptions verifies typed global helpers dispatch all option branches. TestGlobalTypedTerminationAndAccessOptions 验证全局类型化选项入口覆盖所有分支。
func TestGlobalTypedTerminationAndAccessOptions(t *testing.T) {
	DeleteAllManager()
	t.Cleanup(DeleteAllManager)
	ctx := context.Background()
	mgr, err := NewBuilder().IsPrintBanner(false).AutoRenew(false).AuthType("typed-complete").Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	SetManager(mgr)

	token, err := LoginWithOptions(ctx, LoginOptions{AuthType: "typed-complete", LoginID: "typed-termination", Device: "web", DeviceID: "browser"})
	if err != nil {
		t.Fatalf("LoginWithOptions() error = %v", err)
	}
	if err = KickoutWithOptions(ctx, LogoutOptions{AuthType: "typed-complete", Token: token}); err != nil || IsLogin(ctx, token, "typed-complete") {
		t.Fatalf("KickoutWithOptions() error = %v, logged in = %v", err, IsLogin(ctx, token, "typed-complete"))
	}
	token, err = LoginWithOptions(ctx, LoginOptions{AuthType: "typed-complete", LoginID: "typed-termination", Device: "web", DeviceID: "browser"})
	if err != nil {
		t.Fatalf("second LoginWithOptions() error = %v", err)
	}
	if err = ReplaceWithOptions(ctx, LogoutOptions{AuthType: "typed-complete", Token: token}); err != nil || IsLogin(ctx, token, "typed-complete") {
		t.Fatalf("ReplaceWithOptions() error = %v, logged in = %v", err, IsLogin(ctx, token, "typed-complete"))
	}

	if err = DisableWithOptions(ctx, DisableOptions{AuthType: "typed-complete", LoginID: "typed-disabled", Duration: time.Minute, Reason: "risk"}); err != nil {
		t.Fatalf("DisableWithOptions() error = %v", err)
	}
	if !IsDisable(ctx, "typed-disabled", "typed-complete") {
		t.Fatal("DisableWithOptions() did not disable the account")
	}
	if err = Untie(ctx, "typed-disabled", "typed-complete"); err != nil {
		t.Fatalf("Untie() error = %v", err)
	}

	accessToken, err := LoginWithOptions(ctx, LoginOptions{AuthType: "typed-complete", LoginID: "typed-access"})
	if err != nil {
		t.Fatalf("access LoginWithOptions() error = %v", err)
	}
	if err = AddPermissionsWithOptions(ctx, PermissionOptions{AuthType: "typed-complete", Token: accessToken, Permissions: []string{"read", "write"}}); err != nil {
		t.Fatalf("AddPermissionsWithOptions() error = %v", err)
	}
	if err = CheckPermissionsAndWithOptions(ctx, PermissionOptions{AuthType: "typed-complete", Token: accessToken, Permissions: []string{"read", "write"}}); err != nil {
		t.Fatalf("CheckPermissionsAndWithOptions() error = %v", err)
	}
	if err = CheckPermissionsOrWithOptions(ctx, PermissionOptions{AuthType: "typed-complete", LoginID: "typed-access", Permissions: []string{"missing", "read"}}); err != nil {
		t.Fatalf("CheckPermissionsOrWithOptions() error = %v", err)
	}
	if err = RemovePermissionsWithOptions(ctx, PermissionOptions{AuthType: "typed-complete", LoginID: "typed-access", Permissions: []string{"write"}}); err != nil {
		t.Fatalf("RemovePermissionsWithOptions() error = %v", err)
	}
	if err = AddRolesWithOptions(ctx, RoleOptions{AuthType: "typed-complete", LoginID: "typed-access", Roles: []string{"reader", "operator"}}); err != nil {
		t.Fatalf("AddRolesWithOptions() error = %v", err)
	}
	if err = CheckRoleWithOptions(ctx, RoleOptions{AuthType: "typed-complete", Token: accessToken, Role: "reader"}); err != nil {
		t.Fatalf("CheckRoleWithOptions() error = %v", err)
	}
	if err = CheckRolesAndWithOptions(ctx, RoleOptions{AuthType: "typed-complete", Token: accessToken, Roles: []string{"reader", "operator"}}); err != nil {
		t.Fatalf("CheckRolesAndWithOptions() error = %v", err)
	}
	if err = CheckRolesOrWithOptions(ctx, RoleOptions{AuthType: "typed-complete", LoginID: "typed-access", Roles: []string{"missing", "reader"}}); err != nil {
		t.Fatalf("CheckRolesOrWithOptions() error = %v", err)
	}
	if err = RemoveRolesWithOptions(ctx, RoleOptions{AuthType: "typed-complete", Token: accessToken, Roles: []string{"operator"}}); err != nil {
		t.Fatalf("RemoveRolesWithOptions() error = %v", err)
	}

	if err = DisableServiceWithOptions(ctx, ServiceDisableOptions{AuthType: "typed-complete", LoginID: "typed-access", Service: "billing", Level: 2, Duration: time.Minute}); err != nil {
		t.Fatalf("DisableServiceWithOptions() error = %v", err)
	}
	if err = DisableDeviceWithOptions(ctx, DeviceDisableOptions{AuthType: "typed-complete", LoginID: "typed-access", Device: "web", DeviceID: "browser", Duration: time.Minute}); err != nil {
		t.Fatalf("DisableDeviceWithOptions() error = %v", err)
	}
}

// TestGlobalDisableFacades verifies account, service, and device disable wrappers. TestGlobalDisableFacades 验证账号、服务和设备封禁门面。
func TestGlobalDisableFacades(t *testing.T) {
	DeleteAllManager()
	t.Cleanup(DeleteAllManager)
	ctx := context.Background()
	mgr, err := NewBuilder().IsPrintBanner(false).AutoRenew(false).AuthType("disable-facade").Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	SetManager(mgr)

	if err = Disable(ctx, "account-user", time.Minute, "account-risk", "disable-facade"); err != nil {
		t.Fatalf("Disable() error = %v", err)
	}
	if !IsDisable(ctx, "account-user", "disable-facade") {
		t.Fatal("IsDisable() = false, want true")
	}
	if err = CheckDisable(ctx, "account-user", "disable-facade"); !errors.Is(err, derror.ErrAccountDisabled) {
		t.Fatalf("CheckDisable() error = %v, want ErrAccountDisabled", err)
	}
	if info, err := GetDisableInfo(ctx, "account-user", "disable-facade"); err != nil || info.DisableReason != "account-risk" {
		t.Fatalf("GetDisableInfo() = %+v, %v", info, err)
	}
	if ttl, err := GetDisableTTL(ctx, "account-user", "disable-facade"); err != nil || ttl <= 0 {
		t.Fatalf("GetDisableTTL() = %d, %v", ttl, err)
	}
	if err = Untie(ctx, "account-user", "disable-facade"); err != nil || IsDisable(ctx, "account-user", "disable-facade") {
		t.Fatalf("Untie() error = %v, disabled = %v", err, IsDisable(ctx, "account-user", "disable-facade"))
	}

	if err = DisableService(ctx, "service-user", "billing", time.Minute, "disable-facade"); err != nil {
		t.Fatalf("DisableService() error = %v", err)
	}
	if err = UntieService(ctx, "service-user", "billing", "disable-facade"); err != nil {
		t.Fatalf("UntieService() error = %v", err)
	}
	if err = DisableServiceWithReason(ctx, "service-user", "billing", time.Minute, "service-risk", "disable-facade"); err != nil {
		t.Fatalf("DisableServiceWithReason() error = %v", err)
	}
	if err = DisableServiceLevel(ctx, "service-user", "reports", 3, time.Minute, "disable-facade"); err != nil {
		t.Fatalf("DisableServiceLevel() error = %v", err)
	}
	if err = DisableServiceLevelWithReason(ctx, "service-user", "payments", 4, time.Minute, "level-risk", "disable-facade"); err != nil {
		t.Fatalf("DisableServiceLevelWithReason() error = %v", err)
	}
	if !IsDisableService(ctx, "service-user", "billing", "disable-facade") || !IsDisableServiceLevel(ctx, "service-user", "reports", 2, "disable-facade") || IsDisableServiceLevel(ctx, "service-user", "reports", 4, "disable-facade") {
		t.Fatal("service disable predicates returned unexpected results")
	}
	if err = CheckDisableService(ctx, "service-user", []string{"missing", "billing"}, "disable-facade"); !errors.Is(err, derror.ErrServiceDisabled) {
		t.Fatalf("CheckDisableService() error = %v, want ErrServiceDisabled", err)
	}
	if err = CheckDisableServiceLevel(ctx, "service-user", "reports", 2, "disable-facade"); !errors.Is(err, derror.ErrServiceDisabled) {
		t.Fatalf("CheckDisableServiceLevel() error = %v, want ErrServiceDisabled", err)
	}
	if info, err := GetDisableServiceInfo(ctx, "service-user", "billing", "disable-facade"); err != nil || info.DisableReason != "service-risk" {
		t.Fatalf("GetDisableServiceInfo() = %+v, %v", info, err)
	}
	if ttl, err := GetDisableServiceTTL(ctx, "service-user", "billing", "disable-facade"); err != nil || ttl <= 0 {
		t.Fatalf("GetDisableServiceTTL() = %d, %v", ttl, err)
	}
	if err = UntieService(ctx, "service-user", "billing", "disable-facade"); err != nil || IsDisableService(ctx, "service-user", "billing", "disable-facade") {
		t.Fatalf("UntieService(reason) error = %v, disabled = %v", err, IsDisableService(ctx, "service-user", "billing", "disable-facade"))
	}

	if err = DisableDevice(ctx, "device-user", "web", time.Minute, "disable-facade"); err != nil {
		t.Fatalf("DisableDevice() error = %v", err)
	}
	if err = DisableDeviceWithReason(ctx, "device-user", "mobile", time.Minute, "device-risk", "disable-facade"); err != nil {
		t.Fatalf("DisableDeviceWithReason() error = %v", err)
	}
	if err = DisableDeviceAndDeviceID(ctx, "device-user", "tablet", "tablet-1", time.Minute, "disable-facade"); err != nil {
		t.Fatalf("DisableDeviceAndDeviceID() error = %v", err)
	}
	if err = DisableDeviceAndDeviceIDWithReason(ctx, "device-user", "desktop", "desktop-1", time.Minute, "concrete-risk", "disable-facade"); err != nil {
		t.Fatalf("DisableDeviceAndDeviceIDWithReason() error = %v", err)
	}
	if !IsDisableDevice(ctx, "device-user", "web", "disable-facade") || !IsDisableDeviceAndDeviceID(ctx, "device-user", "tablet", "tablet-1", "disable-facade") {
		t.Fatal("device disable predicates returned false")
	}
	if err = CheckDisableDevice(ctx, "device-user", "web", "disable-facade"); !errors.Is(err, derror.ErrDeviceDisabled) {
		t.Fatalf("CheckDisableDevice() error = %v, want ErrDeviceDisabled", err)
	}
	if err = CheckDisableDeviceAndDeviceID(ctx, "device-user", "tablet", "tablet-1", "disable-facade"); !errors.Is(err, derror.ErrDeviceDisabled) {
		t.Fatalf("CheckDisableDeviceAndDeviceID() error = %v, want ErrDeviceDisabled", err)
	}
	if info, err := GetDisableDeviceInfo(ctx, "device-user", "mobile", "disable-facade"); err != nil || info.DisableReason != "device-risk" {
		t.Fatalf("GetDisableDeviceInfo() = %+v, %v", info, err)
	}
	if info, err := GetDisableDeviceAndDeviceIDInfo(ctx, "device-user", "desktop", "desktop-1", "disable-facade"); err != nil || info.DisableReason != "concrete-risk" {
		t.Fatalf("GetDisableDeviceAndDeviceIDInfo() = %+v, %v", info, err)
	}
	if ttl, err := GetDisableDeviceTTL(ctx, "device-user", "web", "disable-facade"); err != nil || ttl <= 0 {
		t.Fatalf("GetDisableDeviceTTL() = %d, %v", ttl, err)
	}
	if ttl, err := GetDisableDeviceAndDeviceIDTTL(ctx, "device-user", "desktop", "desktop-1", "disable-facade"); err != nil || ttl <= 0 {
		t.Fatalf("GetDisableDeviceAndDeviceIDTTL() = %d, %v", ttl, err)
	}
	if err = UntieDevice(ctx, "device-user", "web", "disable-facade"); err != nil || IsDisableDevice(ctx, "device-user", "web", "disable-facade") {
		t.Fatalf("UntieDevice() error = %v, disabled = %v", err, IsDisableDevice(ctx, "device-user", "web", "disable-facade"))
	}
	if err = UntieDeviceAndDeviceID(ctx, "device-user", "tablet", "tablet-1", "disable-facade"); err != nil || IsDisableDeviceAndDeviceID(ctx, "device-user", "tablet", "tablet-1", "disable-facade") {
		t.Fatalf("UntieDeviceAndDeviceID() error = %v, disabled = %v", err, IsDisableDeviceAndDeviceID(ctx, "device-user", "tablet", "tablet-1", "disable-facade"))
	}
}

// TestGlobalNonceTicketShortKeyFacades verifies optional global credential lifecycles. TestGlobalNonceTicketShortKeyFacades 验证可选全局凭证生命周期。
func TestGlobalNonceTicketShortKeyFacades(t *testing.T) {
	DeleteAllManager()
	t.Cleanup(DeleteAllManager)
	ctx := context.Background()
	mgr, err := NewBuilder().IsPrintBanner(false).AutoRenew(false).AuthType("credential-global").EnableNonce().EnableTicket().EnableShortKey().Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	SetManager(mgr)

	defaultNonce, err := GenerateNonce(ctx, "credential-global")
	if err != nil || defaultNonce == "" || !IsNonceValid(ctx, defaultNonce, "credential-global") {
		t.Fatalf("GenerateNonce() = %q, %v", defaultNonce, err)
	}
	if err = VerifyAndConsumeNonce(ctx, defaultNonce, "credential-global"); err != nil {
		t.Fatalf("VerifyAndConsumeNonce(default) error = %v", err)
	}

	nonceValue, err := GenerateNonceWithTimeout(ctx, time.Minute, "credential-global")
	if err != nil || nonceValue == "" || !IsNonceValid(ctx, nonceValue, "credential-global") {
		t.Fatalf("GenerateNonceWithTimeout() = %q, %v", nonceValue, err)
	}
	if ttl, err := GetNonceTTL(ctx, nonceValue, "credential-global"); err != nil || ttl <= 0 {
		t.Fatalf("GetNonceTTL() = %d, %v", ttl, err)
	}
	if !VerifyNonce(ctx, nonceValue, "credential-global") || IsNonceValid(ctx, nonceValue, "credential-global") {
		t.Fatal("VerifyNonce() should consume a valid nonce")
	}
	if err = VerifyAndConsumeNonce(ctx, nonceValue, "credential-global"); err == nil {
		t.Fatal("VerifyAndConsumeNonce() should reject a consumed nonce")
	}

	createdTicket, err := CreateTicket(ctx, "ticket-user", "credential-global")
	if err != nil {
		t.Fatalf("CreateTicket() error = %v", err)
	}
	if validated, err := ValidateTicket(ctx, createdTicket.Ticket, "credential-global"); err != nil || validated.LoginID != "ticket-user" {
		t.Fatalf("ValidateTicket() = %+v, %v", validated, err)
	}
	if err = RevokeTicket(ctx, createdTicket.Ticket, "credential-global"); err != nil {
		t.Fatalf("RevokeTicket() error = %v", err)
	}
	if status, err := GetTicketStatus(ctx, createdTicket.Ticket, "credential-global"); err != nil || status != ticket.StatusRevoked {
		t.Fatalf("GetTicketStatus() = %q, %v", status, err)
	}
	consumableTicket, err := CreateTicket(ctx, "consume-ticket-user", "credential-global")
	if err != nil {
		t.Fatalf("CreateTicket(consumable) error = %v", err)
	}
	consumedTicket, err := ConsumeTicket(ctx, consumableTicket.Ticket, "credential-global")
	if err != nil || consumedTicket == nil || consumedTicket.Ticket == nil || consumedTicket.Ticket.Status != ticket.StatusConsumed {
		t.Fatalf("ConsumeTicket() = %+v, %v", consumedTicket, err)
	}

	createdKey, err := CreateShortKey(ctx, "credential-global")
	if err != nil {
		t.Fatalf("CreateShortKey() error = %v", err)
	}
	if confirmed, err := ConfirmShortKey(ctx, createdKey.Key, "short-user", "credential-global"); err != nil || confirmed.LoginID != "short-user" {
		t.Fatalf("ConfirmShortKey() = %+v, %v", confirmed, err)
	}
	if validated, err := ValidateShortKey(ctx, createdKey.Key, "credential-global"); err != nil || validated.Status != shortkey.StatusConfirmed {
		t.Fatalf("ValidateShortKey() = %+v, %v", validated, err)
	}
	if err = RevokeShortKey(ctx, createdKey.Key, "credential-global"); err != nil {
		t.Fatalf("RevokeShortKey() error = %v", err)
	}
	if status, err := GetShortKeyStatus(ctx, createdKey.Key, "credential-global"); err != nil || status != shortkey.StatusRevoked {
		t.Fatalf("GetShortKeyStatus() = %q, %v", status, err)
	}
	consumableKey, err := CreateShortKey(ctx, "credential-global")
	if err != nil {
		t.Fatalf("CreateShortKey(consumable) error = %v", err)
	}
	if _, err = ConfirmShortKey(ctx, consumableKey.Key, "consume-short-user", "credential-global"); err != nil {
		t.Fatalf("ConfirmShortKey(consumable) error = %v", err)
	}
	consumedKey, err := ConsumeShortKey(ctx, consumableKey.Key, "credential-global")
	if err != nil || consumedKey == nil || consumedKey.ShortKey == nil || consumedKey.ShortKey.Status != shortkey.StatusConsumed {
		t.Fatalf("ConsumeShortKey() = %+v, %v", consumedKey, err)
	}
}

// TestGlobalOAuth2FacadeFlows verifies all OAuth2 global facade routes. TestGlobalOAuth2FacadeFlows 验证全局 OAuth2 门面路由。
func TestGlobalOAuth2FacadeFlows(t *testing.T) {
	DeleteAllManager()
	t.Cleanup(DeleteAllManager)
	ctx := context.Background()
	mgr, err := NewBuilder().IsPrintBanner(false).AutoRenew(false).AuthType("oauth2-global").EnableOAuth2().Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	SetManager(mgr)
	client := &oauth2.Client{
		ClientID:     "oauth-client",
		ClientSecret: "oauth-secret",
		RedirectURIs: []string{"https://app.example/callback"},
		GrantTypes: []oauth2.GrantType{
			oauth2.GrantTypeAuthorizationCode,
			oauth2.GrantTypeClientCredentials,
			oauth2.GrantTypePassword,
			oauth2.GrantTypeRefreshToken,
		},
		Scopes: []string{"read", "write"},
	}
	if err = RegisterOAuth2Client(client, "oauth2-global"); err != nil {
		t.Fatalf("RegisterOAuth2Client() error = %v", err)
	}
	if got, err := GetOAuth2Client(client.ClientID, "oauth2-global"); err != nil || got.ClientSecret != client.ClientSecret {
		t.Fatalf("GetOAuth2Client() = %+v, %v", got, err)
	}

	code, err := GenerateOAuth2AuthorizationCode(ctx, client.ClientID, "code-user", client.RedirectURIs[0], []string{"read"}, "oauth2-global")
	if err != nil {
		t.Fatalf("GenerateOAuth2AuthorizationCode() error = %v", err)
	}
	issued, err := ExchangeOAuth2CodeForToken(ctx, code.Code, client.ClientID, client.ClientSecret, client.RedirectURIs[0], "oauth2-global")
	if err != nil || issued.Token == "" {
		t.Fatalf("ExchangeOAuth2CodeForToken() = %+v, %v", issued, err)
	}
	if !ValidateOAuth2AccessToken(ctx, issued.Token, "oauth2-global") {
		t.Fatal("ValidateOAuth2AccessToken() = false, want true")
	}
	if info, err := ValidateOAuth2AccessTokenAndGetInfo(ctx, issued.Token, "oauth2-global"); err != nil || info.UserID != "code-user" {
		t.Fatalf("ValidateOAuth2AccessTokenAndGetInfo() = %+v, %v", info, err)
	}
	if err = RevokeOAuth2Token(ctx, issued.Token, "oauth2-global"); err != nil || ValidateOAuth2AccessToken(ctx, issued.Token, "oauth2-global") {
		t.Fatalf("RevokeOAuth2Token() error = %v, valid = %v", err, ValidateOAuth2AccessToken(ctx, issued.Token, "oauth2-global"))
	}

	pkceCode, err := GenerateOAuth2AuthorizationCodeWithPKCE(ctx, client.ClientID, "pkce-user", client.RedirectURIs[0], []string{"write"}, "verifier", oauth2.CodeChallengeMethodPlain, "oauth2-global")
	if err != nil {
		t.Fatalf("GenerateOAuth2AuthorizationCodeWithPKCE() error = %v", err)
	}
	if _, err = ExchangeOAuth2CodeForTokenWithPKCE(ctx, pkceCode.Code, client.ClientID, client.ClientSecret, client.RedirectURIs[0], "verifier", "oauth2-global"); err != nil {
		t.Fatalf("ExchangeOAuth2CodeForTokenWithPKCE() error = %v", err)
	}

	clientToken, err := OAuth2ClientCredentialsToken(ctx, client.ClientID, client.ClientSecret, []string{"read"}, "oauth2-global")
	if err != nil {
		t.Fatalf("OAuth2ClientCredentialsToken() error = %v", err)
	}
	passwordToken, err := OAuth2PasswordGrantToken(ctx, client.ClientID, client.ClientSecret, "alice", "password", []string{"write"}, func(username, password string) (string, error) {
		if username == "alice" && password == "password" {
			return "alice-id", nil
		}
		return "", errors.New("invalid credentials")
	}, "oauth2-global")
	if err != nil || passwordToken.UserID != "alice-id" {
		t.Fatalf("OAuth2PasswordGrantToken() = %+v, %v", passwordToken, err)
	}
	refreshed, err := RefreshOAuth2AccessToken(ctx, client.ClientID, clientToken.RefreshToken, client.ClientSecret, "oauth2-global")
	if err != nil || refreshed.Token == clientToken.Token {
		t.Fatalf("RefreshOAuth2AccessToken() = %+v, %v", refreshed, err)
	}
	unified, err := OAuth2Token(ctx, &oauth2.TokenRequest{GrantType: oauth2.GrantTypeRefreshToken, ClientID: client.ClientID, ClientSecret: client.ClientSecret, RefreshToken: refreshed.RefreshToken}, nil, "oauth2-global")
	if err != nil || unified.Token == refreshed.Token {
		t.Fatalf("OAuth2Token(refresh) = %+v, %v", unified, err)
	}
	if err = RevokeOAuth2Token(ctx, passwordToken.Token, "oauth2-global"); err != nil {
		t.Fatalf("RevokeOAuth2Token(password) error = %v", err)
	}
	if err = UnregisterOAuth2Client(client.ClientID, "oauth2-global"); err != nil {
		t.Fatalf("UnregisterOAuth2Client() error = %v", err)
	}
	if _, err = GetOAuth2Client(client.ClientID, "oauth2-global"); !errors.Is(err, derror.ErrClientNotFound) {
		t.Fatalf("GetOAuth2Client(after unregister) error = %v, want ErrClientNotFound", err)
	}
}

// TestInstanceDisableBranchSelection verifies typed instance options select level and concrete-device operations. TestInstanceDisableBranchSelection 验证实例选项会选择等级和具体设备分支。
func TestInstanceDisableBranchSelection(t *testing.T) {
	ctx := context.Background()
	mgr, err := NewBuilder().IsPrintBanner(false).AutoRenew(false).AuthType("instance-disable").Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	auth := New(mgr)
	t.Cleanup(auth.Close)

	if err = auth.DisableService(ctx, ServiceDisableOptions{LoginID: "instance-user", Service: "billing", Duration: time.Minute, Reason: "risk"}); err != nil {
		t.Fatalf("Auth.DisableService() error = %v", err)
	}
	if err = auth.DisableServiceLevelWithReason(ctx, "instance-user", "reports", 2, time.Minute, "level-risk"); err != nil || !auth.IsDisableServiceLevel(ctx, "instance-user", "reports", 1) {
		t.Fatalf("Auth.DisableServiceLevelWithReason() error = %v, disabled = %v", err, auth.IsDisableServiceLevel(ctx, "instance-user", "reports", 1))
	}
	if err = auth.DisableDevice(ctx, DeviceDisableOptions{LoginID: "instance-user", Device: "web", Duration: time.Minute, Reason: "device-risk"}); err != nil {
		t.Fatalf("Auth.DisableDevice(type) error = %v", err)
	}
	if err = auth.DisableDeviceAndDeviceIDWithReason(ctx, "instance-user", "mobile", "phone", time.Minute, "concrete-risk"); err != nil || !auth.IsDisableDeviceAndDeviceID(ctx, "instance-user", "mobile", "phone") {
		t.Fatalf("Auth.DisableDeviceAndDeviceIDWithReason() error = %v, disabled = %v", err, auth.IsDisableDeviceAndDeviceID(ctx, "instance-user", "mobile", "phone"))
	}
}

func dtokenContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

var _ managerBuilder = (*corebuilder.Builder)(nil)
