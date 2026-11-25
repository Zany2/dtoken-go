package dtoken

import (
	"context"

	"testing"
)

// TestRootRefreshAndIntrospectionFacades verifies refresh-token and introspection global helpers. TestRootRefreshAndIntrospectionFacades 验证刷新令牌和令牌检查全局门面。
func TestRootRefreshAndIntrospectionFacades(t *testing.T) {
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

	pair, err := AuthType("refresh-facade").LoginWithRefreshTokenOptions(ctx, RefreshTokenOptions{LoginOptions: LoginOptions{LoginID: "refresh-user", Device: "web", DeviceID: "browser"}})
	if err != nil {
		t.Fatalf("LoginWithRefreshToken() error = %v", err)
	}
	if pair.LoginID != "refresh-user" || pair.Device != "web" || pair.DeviceID != "browser" {
		t.Fatalf("refresh pair = %+v, want login and device metadata", pair)
	}

	info, err := AuthType("refresh-facade").IntrospectToken(ctx, pair.AccessToken)
	if err != nil {
		t.Fatalf("IntrospectToken() error = %v", err)
	}
	if !info.Active || info.LoginID != "refresh-user" {
		t.Fatalf("introspection = %+v, want active refresh-user", info)
	}

	ttl, err := AuthType("refresh-facade").GetRefreshTokenTTL(ctx, pair.RefreshToken)
	if err != nil {
		t.Fatalf("GetRefreshTokenTTL() error = %v", err)
	}
	if ttl <= 0 {
		t.Fatalf("refresh ttl = %d, want positive", ttl)
	}

	rotated, err := AuthType("refresh-facade").RefreshToken(ctx, pair.RefreshToken)
	if err != nil {
		t.Fatalf("RefreshToken() error = %v", err)
	}
	if rotated.RefreshToken == pair.RefreshToken || rotated.AccessToken == pair.AccessToken {
		t.Fatal("RefreshToken() should rotate both refresh and access tokens")
	}
	if err = AuthType("refresh-facade").RevokeRefreshToken(ctx, rotated.RefreshToken); err != nil {
		t.Fatalf("RevokeRefreshToken() error = %v", err)
	}
	if AuthType("refresh-facade").IsLogin(ctx, rotated.AccessToken) {
		t.Fatal("access token should be logged out after refresh token revoke")
	}
}

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
