package dtoken

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/config"

	manager "github.com/Zany2/dtoken-go/internal/engine"
)

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

	mgr, err := buildTestManager(b)
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

// TestRootTerminalOperationFamilies verifies global logout, kickout, and replace routes. TestRootTerminalOperationFamilies 验证全局 logout、kickout 和 replace 路由。
func TestRootTerminalOperationFamilies(t *testing.T) {
	DeleteAllManager()
	t.Cleanup(DeleteAllManager)

	ctx := context.Background()
	mgr, err := buildTestManager(NewBuilder().IsPrintBanner(false).AutoRenew(false).IsShare(false).AuthType("global-terminal"))
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if err := SetManager(mgr); err != nil {
		t.Fatalf("register authentication: %v", err)
	}

	login := func(loginID, device, deviceID string) string {
		token, loginErr := AuthType("global-terminal").Login(ctx, LoginOptions{LoginID: loginID, Device: device, DeviceID: deviceID})
		if loginErr != nil {
			t.Fatalf("Login(%s) error = %v", loginID, loginErr)
		}
		return token
	}

	logoutDeviceToken := login("logout-device", "web", "logout-1")
	login("logout-device", "web", "logout-2")
	if err = AuthType("global-terminal").LogoutByDevice(ctx, "logout-device", "web"); err != nil || AuthType("global-terminal").IsLogin(ctx, logoutDeviceToken) {
		t.Fatalf("LogoutByDevice() error = %v, token active = %v", err, AuthType("global-terminal").IsLogin(ctx, logoutDeviceToken))
	}
	logoutLoginToken := login("logout-login", "mobile", "logout-login-1")
	if err = AuthType("global-terminal").LogoutByLoginID(ctx, "logout-login"); err != nil || AuthType("global-terminal").IsLogin(ctx, logoutLoginToken) {
		t.Fatalf("LogoutByLoginID() error = %v, token active = %v", err, AuthType("global-terminal").IsLogin(ctx, logoutLoginToken))
	}
	directLogoutToken := login("logout-direct", "web", "logout-direct-1")
	if err = AuthType("global-terminal").LogoutByToken(ctx, directLogoutToken); err != nil || AuthType("global-terminal").IsLogin(ctx, directLogoutToken) {
		t.Fatalf("Logout() error = %v, token active = %v", err, AuthType("global-terminal").IsLogin(ctx, directLogoutToken))
	}

	kickToken := login("kick-token", "web", "kick-token-1")
	if err = AuthType("global-terminal").KickoutByToken(ctx, kickToken); err != nil || AuthType("global-terminal").IsLogin(ctx, kickToken) {
		t.Fatalf("Kickout() error = %v, token active = %v", err, AuthType("global-terminal").IsLogin(ctx, kickToken))
	}
	kickConcreteToken := login("kick-concrete", "web", "kick-concrete-1")
	if err = AuthType("global-terminal").KickoutByDeviceAndDeviceID(ctx, "kick-concrete", "web", "kick-concrete-1"); err != nil || AuthType("global-terminal").IsLogin(ctx, kickConcreteToken) {
		t.Fatalf("KickoutByDeviceAndDeviceID() error = %v, token active = %v", err, AuthType("global-terminal").IsLogin(ctx, kickConcreteToken))
	}
	kickDeviceToken := login("kick-device", "web", "kick-device-1")
	if err = AuthType("global-terminal").KickoutByDevice(ctx, "kick-device", "web"); err != nil || AuthType("global-terminal").IsLogin(ctx, kickDeviceToken) {
		t.Fatalf("KickoutByDevice() error = %v, token active = %v", err, AuthType("global-terminal").IsLogin(ctx, kickDeviceToken))
	}
	kickLoginToken := login("kick-login", "mobile", "kick-login-1")
	if err = AuthType("global-terminal").KickoutByLoginID(ctx, "kick-login"); err != nil || AuthType("global-terminal").IsLogin(ctx, kickLoginToken) {
		t.Fatalf("KickoutByLoginID() error = %v, token active = %v", err, AuthType("global-terminal").IsLogin(ctx, kickLoginToken))
	}

	replaceToken := login("replace-token", "web", "replace-token-1")
	if err = AuthType("global-terminal").ReplaceByToken(ctx, replaceToken); err != nil || AuthType("global-terminal").IsLogin(ctx, replaceToken) {
		t.Fatalf("Replace() error = %v, token active = %v", err, AuthType("global-terminal").IsLogin(ctx, replaceToken))
	}
	replaceConcreteToken := login("replace-concrete", "web", "replace-concrete-1")
	if err = AuthType("global-terminal").ReplaceByDeviceAndDeviceID(ctx, "replace-concrete", "web", "replace-concrete-1"); err != nil || AuthType("global-terminal").IsLogin(ctx, replaceConcreteToken) {
		t.Fatalf("ReplaceByDeviceAndDeviceID() error = %v, token active = %v", err, AuthType("global-terminal").IsLogin(ctx, replaceConcreteToken))
	}
	replaceDeviceToken := login("replace-device", "web", "replace-device-1")
	if err = AuthType("global-terminal").ReplaceByDevice(ctx, "replace-device", "web"); err != nil || AuthType("global-terminal").IsLogin(ctx, replaceDeviceToken) {
		t.Fatalf("ReplaceByDevice() error = %v, token active = %v", err, AuthType("global-terminal").IsLogin(ctx, replaceDeviceToken))
	}
	replaceLoginToken := login("replace-login", "mobile", "replace-login-1")
	if err = AuthType("global-terminal").ReplaceByLoginID(ctx, "replace-login"); err != nil || AuthType("global-terminal").IsLogin(ctx, replaceLoginToken) {
		t.Fatalf("ReplaceByLoginID() error = %v, token active = %v", err, AuthType("global-terminal").IsLogin(ctx, replaceLoginToken))
	}
}

// TestInstanceTerminalOperationFamilies verifies instance terminal shortcuts for all lifecycle actions. TestInstanceTerminalOperationFamilies 验证实例门面全部终端生命周期快捷方法。
func TestInstanceTerminalOperationFamilies(t *testing.T) {
	ctx := context.Background()
	mgr, err := buildTestManager(NewBuilder().IsPrintBanner(false).AutoRenew(false).IsShare(false).AuthType("instance-terminal-complete"))
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
	mgr, err := buildTestManager(NewBuilder().IsPrintBanner(false).AutoRenew(false).AuthType("instance-login-complete"))
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
	mgr, err := buildTestManager(NewBuilder().IsPrintBanner(false).AutoRenew(false).IsShare(false).AuthType("instance-session-access"))
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

func requireFacadeNoError(t *testing.T, name string, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s error = %v", name, err)
	}
}
