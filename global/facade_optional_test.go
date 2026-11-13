package global

import (
	"context"

	"testing"
)

// TestOptionalFacadesReportDisabledModules verifies optional facades preserve disabled-module errors and zero values. TestOptionalFacadesReportDisabledModules 验证可选模块门面保留未启用错误和约定零值。

// TestGlobalRefreshAndIntrospectionFacades verifies refresh-token and introspection global helpers. TestGlobalRefreshAndIntrospectionFacades 验证刷新令牌和令牌检查全局门面。
func TestGlobalRefreshAndIntrospectionFacades(t *testing.T) {
	DeleteAllManager()
	t.Cleanup(DeleteAllManager)

	ctx := context.Background()
	mgr, err := buildTestManager(NewBuilder().
		IsPrintBanner(false).
		AutoRenew(false).
		AuthType("refresh-facade"),
	)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	SetManager(mgr)

	pair, err := LoginWithRefreshToken(ctx, "refresh-user", "web", "browser", "refresh-facade")
	if err != nil {
		t.Fatalf("LoginWithRefreshToken() error = %v", err)
	}
	if pair.LoginID != "refresh-user" || pair.Device != "web" || pair.DeviceID != "browser" {
		t.Fatalf("refresh pair = %+v, want login and device metadata", pair)
	}

	info, err := IntrospectToken(ctx, pair.AccessToken, "refresh-facade")
	if err != nil {
		t.Fatalf("IntrospectToken() error = %v", err)
	}
	if !info.Active || info.LoginID != "refresh-user" {
		t.Fatalf("introspection = %+v, want active refresh-user", info)
	}

	ttl, err := GetRefreshTokenTTL(ctx, pair.RefreshToken, "refresh-facade")
	if err != nil {
		t.Fatalf("GetRefreshTokenTTL() error = %v", err)
	}
	if ttl <= 0 {
		t.Fatalf("refresh ttl = %d, want positive", ttl)
	}

	rotated, err := RefreshToken(ctx, pair.RefreshToken, "refresh-facade")
	if err != nil {
		t.Fatalf("RefreshToken() error = %v", err)
	}
	if rotated.RefreshToken == pair.RefreshToken || rotated.AccessToken == pair.AccessToken {
		t.Fatal("RefreshToken() should rotate both refresh and access tokens")
	}
	if err = RevokeRefreshToken(ctx, rotated.RefreshToken, "refresh-facade"); err != nil {
		t.Fatalf("RevokeRefreshToken() error = %v", err)
	}
	if IsLogin(ctx, rotated.AccessToken, "refresh-facade") {
		t.Fatal("access token should be logged out after refresh token revoke")
	}
}

// TestGlobalTicketAndShortKeyFacades verifies lifecycle helpers for optional credential modules. TestGlobalTicketAndShortKeyFacades 验证可选凭证模块的生命周期门面。

// TestInstanceTerminalShortcutFacades verifies instance terminal action shortcuts. TestInstanceTerminalShortcutFacades 验证实例终端操作快捷门面。
func TestInstanceTerminalShortcutFacades(t *testing.T) {
	ctx := context.Background()
	mgr, err := buildTestManager(NewBuilder().
		IsPrintBanner(false).
		AutoRenew(false).
		AuthType("terminal-facade"),
	)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	auth := New(mgr)
	t.Cleanup(auth.Close)

	token, err := auth.Login(ctx, LoginOptions{LoginID: "terminal-user", Device: "web", DeviceID: "browser"})
	if err != nil {
		t.Fatalf("Auth.Login() error = %v", err)
	}
	if err = auth.KickoutByDeviceAndDeviceID(ctx, "terminal-user", "web", "browser"); err != nil {
		t.Fatalf("KickoutByDeviceAndDeviceID() error = %v", err)
	}
	if auth.IsLogin(ctx, token) {
		t.Fatal("token should not be logged in after kickout")
	}

	token, err = auth.Login(ctx, LoginOptions{LoginID: "terminal-user", Device: "web", DeviceID: "browser"})
	if err != nil {
		t.Fatalf("Auth.Login(second) error = %v", err)
	}
	if err = auth.ReplaceByLoginID(ctx, "terminal-user"); err != nil {
		t.Fatalf("ReplaceByLoginID() error = %v", err)
	}
	if auth.IsLogin(ctx, token) {
		t.Fatal("token should not be logged in after replace")
	}

	token, err = auth.Login(ctx, LoginOptions{LoginID: "terminal-user", Device: "web", DeviceID: "browser"})
	if err != nil {
		t.Fatalf("Auth.Login(third) error = %v", err)
	}
	if err = auth.Logout(ctx, LogoutOptions{Token: token}); err != nil {
		t.Fatalf("Logout(options) error = %v", err)
	}
	if auth.IsLogin(ctx, token) {
		t.Fatal("token should not be logged in after option logout")
	}
}
