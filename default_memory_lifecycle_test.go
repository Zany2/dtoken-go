package dtoken

import (
	"context"
	"testing"
	"time"

	manager "github.com/Zany2/dtoken-go/internal/engine"
	"github.com/Zany2/dtoken-go/storage/memory"
)

// TestDefaultMemoryLifecycle checks registry routing and shared session data through public default APIs. TestDefaultMemoryLifecycle 通过公开默认 API 验证注册表路由和共享会话数据。
func TestDefaultMemoryLifecycle(t *testing.T) {
	DeleteAllManager()
	t.Cleanup(CloseAll)
	ctx := context.Background()
	storage := memory.NewStorage()
	b := NewBuilder().Storage(storage).IsPrintBanner(false).IsLog(false).AutoRenew(false).IsShare(false)
	auth, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}
	named, err := b.Clone().AuthType("isolated").Build()
	if err != nil {
		t.Fatal(err)
	}
	other, err := named.LoginID(ctx, "user")
	if err != nil {
		t.Fatal(err)
	}
	token, err := Login(ctx, LoginOptions{LoginID: "user", Device: "web", DeviceID: "browser", Extra: map[string]any{"source": "test"}})
	if err != nil {
		t.Fatal(err)
	}
	second, err := LoginWithTimeout(ctx, "user", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if !auth.IsLogin(ctx, token) || !IsLogin(ctx, second) || IsLogin(ctx, other) || named.IsLogin(ctx, token) {
		t.Fatal("namespace or instance routing failed")
	}
	if err = LoginByToken(ctx, token); err != nil {
		t.Fatal(err)
	}
	if id, err := GetLoginID(ctx, token); err != nil || id != "user" {
		t.Fatalf("identity = %q, %v", id, err)
	}
	if info, err := GetTokenInfo(ctx, token); err != nil || info == nil || info.Extra["source"] != "test" {
		t.Fatalf("token info = %+v, %v", info, err)
	}
	if device, err := GetDevice(ctx, token); err != nil || device != "web" {
		t.Fatalf("device = %q, %v", device, err)
	}
	if id, err := GetDeviceID(ctx, token); err != nil || id != "browser" {
		t.Fatalf("device id = %q, %v", id, err)
	}
	if device, id, err := GetDeviceAndDeviceID(ctx, token); err != nil || device != "web" || id != "browser" {
		t.Fatalf("device pair = %q/%q, %v", device, id, err)
	}
	if created, err := GetTokenCreateTime(ctx, token); err != nil || created <= 0 {
		t.Fatalf("created = %d, %v", created, err)
	}
	if err = RenewTimeout(ctx, token, time.Minute); err != nil {
		t.Fatal(err)
	}
	if ttl, err := GetTokenTTL(ctx, token); err != nil || ttl < 1 || ttl > 60 {
		t.Fatalf("ttl = %d, %v", ttl, err)
	}
	if state, err := IntrospectToken(ctx, token); err != nil || state == nil || !state.Active {
		t.Fatalf("introspection = %+v, %v", state, err)
	}
	if err = SetSessionValueByToken(ctx, token, "theme", "dark"); err != nil {
		t.Fatal(err)
	}
	if value, ok, err := GetSessionValueByToken(ctx, second, "theme"); err != nil || !ok || value != "dark" {
		t.Fatalf("shared value = %v, %v, %v", value, ok, err)
	}
	if _, ok, err := named.GetSessionValueByToken(ctx, other, "theme"); err != nil || ok {
		t.Fatalf("namespace value leaked: %v, %v", ok, err)
	}
	if err = DeleteSessionValueByToken(ctx, second, "theme"); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := GetSessionValue(ctx, "user", "theme"); err != nil || ok {
		t.Fatalf("deleted value = %v, %v", ok, err)
	}
	if err = SetSessionValue(ctx, "user", "theme", "light"); err != nil {
		t.Fatal(err)
	}
	if err = LogoutByToken(ctx, token); err != nil {
		t.Fatal(err)
	}
	if err = SetSessionValueByToken(ctx, token, "theme", "invalid"); err == nil {
		t.Fatal("revoked token wrote session")
	}
	if err = DeleteSessionValueByToken(ctx, token, "theme"); err == nil {
		t.Fatal("revoked token deleted session")
	}
	if _, _, err = GetSessionValueByToken(ctx, token, "theme"); err == nil {
		t.Fatal("revoked token read session")
	}
	if value, ok, err := GetSessionValueByToken(ctx, second, "theme"); err != nil || !ok || value != "light" {
		t.Fatalf("surviving session = %v, %v, %v", value, ok, err)
	}
	if err = DeleteSessionValue(ctx, "user", "theme"); err != nil {
		t.Fatal(err)
	}
	if err = LogoutByLoginID(ctx, "user"); err != nil {
		t.Fatal(err)
	}
	if IsLogin(ctx, second) || !named.IsLogin(ctx, other) {
		t.Fatal("logout crossed namespace boundary")
	}
	CloseAll()
	if _, err = LoginID(ctx, "user"); err == nil {
		t.Fatal("login succeeded after registry close")
	}
	if err = SetSessionValueByToken(ctx, second, "theme", "closed"); err == nil {
		t.Fatal("closed registry accepted write")
	}
}

// TestDefaultMemoryRefresh verifies rotation, replay rejection and linked access revocation. TestDefaultMemoryRefresh 验证刷新轮换、防重放与关联访问令牌撤销。
func TestDefaultMemoryRefresh(t *testing.T) {
	DeleteAllManager()
	t.Cleanup(CloseAll)
	_, err := NewBuilder().IsPrintBanner(false).IsLog(false).AutoRenew(false).IsShare(false).RefreshTokenTimeout(300).Build()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, login := range []func() (*manager.RefreshTokenPair, error){
		func() (*manager.RefreshTokenPair, error) { return LoginWithRefreshToken(ctx, "basic") },
		func() (*manager.RefreshTokenPair, error) {
			return LoginWithRefreshTokenOptions(ctx, RefreshTokenOptions{LoginOptions: LoginOptions{LoginID: "options", Device: "web"}, RefreshTimeout: time.Minute})
		},
	} {
		pair, err := login()
		if err != nil {
			t.Fatal(err)
		}
		if ttl, err := GetRefreshTokenTTL(ctx, pair.RefreshToken); err != nil || ttl <= 0 || ttl > 300 {
			t.Fatalf("refresh ttl = %d, %v", ttl, err)
		}
		rotated, err := RefreshToken(ctx, pair.RefreshToken)
		if err != nil {
			t.Fatal(err)
		}
		if IsLogin(ctx, pair.AccessToken) || !IsLogin(ctx, rotated.AccessToken) {
			t.Fatal("rotation did not replace access token")
		}
		if _, err = RefreshToken(ctx, pair.RefreshToken); err == nil {
			t.Fatal("refresh replay succeeded")
		}
		if err = RevokeRefreshToken(ctx, rotated.RefreshToken); err != nil {
			t.Fatal(err)
		}
		if _, err = RefreshToken(ctx, rotated.RefreshToken); err == nil {
			t.Fatal("revoked refresh accepted")
		}
		if IsLogin(ctx, rotated.AccessToken) {
			t.Fatal("refresh revoke left linked access valid")
		}
	}
}
