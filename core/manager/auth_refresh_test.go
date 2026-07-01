// @Author daixk 2025/12/22 15:56:00
package manager

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
)

// TestManagerRefreshTokenFlow verifies login, rotation, and revocation. TestManagerRefreshTokenFlow 验证登录、轮换和撤销流程。
func TestManagerRefreshTokenFlow(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.Timeout = 60
		cfg.RefreshTokenTimeout = 120
	})

	pair, err := mgr.LoginWithRefreshTokenOptions(ctx, RefreshTokenOptions{
		LoginOptions: LoginOptions{
			LoginID: "refresh-user",
			Device:  "web",
			Extra:   map[string]any{"trace": "login"},
		},
		RefreshTimeout: 2 * time.Minute,
	})
	if err != nil {
		t.Fatalf("LoginWithRefreshTokenOptions() error = %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" || pair.AccessToken == pair.RefreshToken {
		t.Fatalf("token pair = %+v, want distinct non-empty tokens", pair)
	}
	if pair.LoginID != "refresh-user" || pair.Device != "web" {
		t.Fatalf("token pair subject = %s/%s, want refresh-user/web", pair.LoginID, pair.Device)
	}
	if pair.ExpiresIn <= 0 || pair.RefreshExpiresIn <= 0 {
		t.Fatalf("token pair ttl = %d/%d, want positive ttl", pair.ExpiresIn, pair.RefreshExpiresIn)
	}

	nextPair, err := mgr.RefreshToken(ctx, pair.RefreshToken)
	if err != nil {
		t.Fatalf("RefreshToken() error = %v", err)
	}
	if nextPair.AccessToken == pair.AccessToken {
		t.Fatal("RefreshToken() returned the old access token")
	}
	if nextPair.RefreshToken == pair.RefreshToken {
		t.Fatal("RefreshToken() returned the old refresh token")
	}
	if _, err = mgr.RefreshToken(ctx, pair.RefreshToken); !errors.Is(err, derror.ErrInvalidRefreshToken) {
		t.Fatalf("RefreshToken(old refresh token) error = %v, want ErrInvalidRefreshToken", err)
	}
	if err = mgr.CheckLogin(ctx, pair.AccessToken); !errors.Is(err, derror.ErrInvalidToken) {
		t.Fatalf("CheckLogin(old access token) error = %v, want ErrInvalidToken", err)
	}
	if info, err := mgr.GetTokenInfo(ctx, nextPair.AccessToken); err != nil || info.Extra["trace"] != "login" {
		t.Fatalf("new access token extra = %+v, %v, want trace=login", info, err)
	}

	if err = mgr.RevokeRefreshToken(ctx, nextPair.RefreshToken); err != nil {
		t.Fatalf("RevokeRefreshToken() error = %v", err)
	}
	if err = mgr.CheckLogin(ctx, nextPair.AccessToken); !errors.Is(err, derror.ErrInvalidToken) {
		t.Fatalf("CheckLogin(revoked access token) error = %v, want ErrInvalidToken", err)
	}
}

// TestManagerRefreshTokenAllowsExpiredAccessToken verifies refresh token is independent from access ttl. TestManagerRefreshTokenAllowsExpiredAccessToken 验证刷新令牌不依赖访问令牌有效期。
func TestManagerRefreshTokenAllowsExpiredAccessToken(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.Timeout = 1
		cfg.RefreshTokenTimeout = 60
	})

	pair, err := mgr.LoginWithRefreshToken(ctx, "expired-access-user", "app")
	if err != nil {
		t.Fatalf("LoginWithRefreshToken() error = %v", err)
	}
	time.Sleep(1100 * time.Millisecond)
	if err = mgr.CheckLogin(ctx, pair.AccessToken); !errors.Is(err, derror.ErrInvalidToken) {
		t.Fatalf("CheckLogin(expired access token) error = %v, want ErrInvalidToken", err)
	}

	nextPair, err := mgr.RefreshToken(ctx, pair.RefreshToken)
	if err != nil {
		t.Fatalf("RefreshToken(expired access token) error = %v", err)
	}
	if nextPair.AccessToken == "" || nextPair.RefreshToken == "" {
		t.Fatalf("RefreshToken() pair = %+v, want non-empty tokens", nextPair)
	}
}

// TestManagerRefreshTokenBoundaries verifies invalid, ttl, revoke, and expiry behavior. TestManagerRefreshTokenBoundaries 验证刷新令牌非法值、TTL、撤销和过期行为。
func TestManagerRefreshTokenBoundaries(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.Timeout = 60
		cfg.RefreshTokenTimeout = 120
	})

	if _, err := mgr.RefreshToken(ctx, ""); !errors.Is(err, derror.ErrInvalidRefreshToken) {
		t.Fatalf("RefreshToken(empty) error = %v, want ErrInvalidRefreshToken", err)
	}
	if _, err := mgr.RefreshToken(ctx, "missing"); !errors.Is(err, derror.ErrInvalidRefreshToken) {
		t.Fatalf("RefreshToken(missing) error = %v, want ErrInvalidRefreshToken", err)
	}
	if _, err := mgr.GetRefreshTokenTTL(ctx, ""); !errors.Is(err, derror.ErrInvalidRefreshToken) {
		t.Fatalf("GetRefreshTokenTTL(empty) error = %v, want ErrInvalidRefreshToken", err)
	}
	if ttl, err := mgr.GetRefreshTokenTTL(ctx, "missing"); err != nil || ttl != -2 {
		t.Fatalf("GetRefreshTokenTTL(missing) = %d, %v, want -2, nil", ttl, err)
	}
	if err := mgr.RevokeRefreshToken(ctx, "missing"); err != nil {
		t.Fatalf("RevokeRefreshToken(missing) error = %v, want nil", err)
	}

	pair, err := mgr.LoginWithRefreshTokenOptions(ctx, RefreshTokenOptions{
		LoginOptions: LoginOptions{
			LoginID:  "refresh-boundary",
			Device:   "web",
			DeviceID: "browser-1",
			Extra:    map[string]any{"trace": "boundary"},
		},
		RefreshTimeout: time.Minute,
	})
	if err != nil {
		t.Fatalf("LoginWithRefreshTokenOptions() error = %v", err)
	}
	ttl, err := mgr.GetRefreshTokenTTL(ctx, pair.RefreshToken)
	if err != nil {
		t.Fatalf("GetRefreshTokenTTL() error = %v", err)
	}
	if ttl <= 0 || ttl > 60 {
		t.Fatalf("GetRefreshTokenTTL() = %d, want 1..60", ttl)
	}

	nextPair, err := mgr.RefreshToken(ctx, pair.RefreshToken)
	if err != nil {
		t.Fatalf("RefreshToken() error = %v", err)
	}
	if nextPair.LoginID != "refresh-boundary" || nextPair.Device != "web" || nextPair.DeviceID != "browser-1" {
		t.Fatalf("RefreshToken() subject = %+v, want inherited subject", nextPair)
	}
	info, err := mgr.GetTokenInfo(ctx, nextPair.AccessToken)
	if err != nil {
		t.Fatalf("GetTokenInfo(refreshed) error = %v", err)
	}
	if info.Extra["trace"] != "boundary" {
		t.Fatalf("refreshed token extra = %+v, want trace=boundary", info.Extra)
	}
}

// TestManagerRefreshTokenExpires verifies expired refresh tokens cannot rotate. TestManagerRefreshTokenExpires 验证过期刷新令牌不能轮换。
func TestManagerRefreshTokenExpires(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.Timeout = 60
		cfg.RefreshTokenTimeout = 60
	})

	pair, err := mgr.LoginWithRefreshTokenOptions(ctx, RefreshTokenOptions{
		LoginOptions:   LoginOptions{LoginID: "refresh-expiring"},
		RefreshTimeout: 20 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("LoginWithRefreshTokenOptions() error = %v", err)
	}
	time.Sleep(30 * time.Millisecond)
	if _, err = mgr.RefreshToken(ctx, pair.RefreshToken); !errors.Is(err, derror.ErrInvalidRefreshToken) {
		t.Fatalf("RefreshToken(expired) error = %v, want ErrInvalidRefreshToken", err)
	}
	if ttl, err := mgr.GetRefreshTokenTTL(ctx, pair.RefreshToken); err != nil || ttl != -2 {
		t.Fatalf("GetRefreshTokenTTL(expired) = %d, %v, want -2, nil", ttl, err)
	}
}

// TestManagerRefreshTokenLoginDoesNotShareAccessToken verifies refresh-token login always gets an independent access token. TestManagerRefreshTokenLoginDoesNotShareAccessToken 验证刷新令牌登录不会共享 access token。
func TestManagerRefreshTokenLoginDoesNotShareAccessToken(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.Timeout = 60
		cfg.RefreshTokenTimeout = 60
		cfg.IsShare = true
	})

	first, err := mgr.LoginWithRefreshToken(ctx, "refresh-share", "web", "browser")
	if err != nil {
		t.Fatalf("first LoginWithRefreshToken() error = %v", err)
	}
	second, err := mgr.LoginWithRefreshToken(ctx, "refresh-share", "web", "browser")
	if err != nil {
		t.Fatalf("second LoginWithRefreshToken() error = %v", err)
	}
	if second.AccessToken == first.AccessToken {
		t.Fatalf("second access token = %q, want independent token", second.AccessToken)
	}
	if _, err = mgr.RefreshToken(ctx, first.RefreshToken); err != nil {
		t.Fatalf("RefreshToken(first refresh token) error = %v", err)
	}
	if _, err = mgr.RefreshToken(ctx, second.RefreshToken); err != nil {
		t.Fatalf("RefreshToken(second refresh token) error = %v", err)
	}
}

// TestManagerRefreshTokenKeepsOldAccessTokenOnNewTokenFailure verifies old access token is kept when rotation fails. TestManagerRefreshTokenKeepsOldAccessTokenOnNewTokenFailure 验证轮换失败时旧 access token 仍保留。
func TestManagerRefreshTokenKeepsOldAccessTokenOnNewTokenFailure(t *testing.T) {
	ctx := context.Background()
	cfg := config.DefaultConfig()
	cfg.IsPrintBanner = false
	cfg.IsLog = false
	cfg.AsyncEvent = false
	cfg.AutoRenew = false
	cfg.RenewInterval = config.NoLimit
	cfg.ActiveTimeout = config.NoLimit
	cfg.Timeout = 60
	cfg.RefreshTokenTimeout = 60
	if err := cfg.Validate(); err != nil {
		t.Fatalf("test config invalid: %v", err)
	}

	mgr := NewManager(
		cfg,
		&managerTestGenerator{},
		newManagerTestStorage(),
		managerTestCodec{},
		adapter.NewNopLogger(),
		nil,
		nil,
	)
	t.Cleanup(mgr.CloseManager)

	pair, err := mgr.LoginWithRefreshToken(ctx, "refresh-rollback", "web")
	if err != nil {
		t.Fatalf("LoginWithRefreshToken() error = %v", err)
	}
	baseStorage := requireManagerTestStorage(t, mgr).(*managerTestStorage)
	mgr.storage = &managerTestFailingSetStorage{
		managerTestStorage: baseStorage,
		failKey:            mgr.getTokenKey("token-refresh-rollback-web--2"),
		failSetIfAbsent:    true,
	}
	if _, err = mgr.RefreshToken(ctx, pair.RefreshToken); !errors.Is(err, derror.ErrStorageUnavailable) {
		t.Fatalf("RefreshToken() error = %v, want ErrStorageUnavailable", err)
	}
	if err = mgr.CheckLogin(ctx, pair.AccessToken); err != nil {
		t.Fatalf("old access token CheckLogin() error = %v, want nil", err)
	}
}

// TestManagerRefreshTokenSkipsConcurrencyBeforeNewPairSucceeds verifies rotation failure does not replace old access. TestManagerRefreshTokenSkipsConcurrencyBeforeNewPairSucceeds 验证轮换失败不会提前顶替旧 access。
func TestManagerRefreshTokenSkipsConcurrencyBeforeNewPairSucceeds(t *testing.T) {
	ctx := context.Background()
	cfg := config.DefaultConfig()
	cfg.IsPrintBanner = false
	cfg.IsLog = false
	cfg.AsyncEvent = false
	cfg.AutoRenew = false
	cfg.RenewInterval = config.NoLimit
	cfg.ActiveTimeout = config.NoLimit
	cfg.Timeout = 60
	cfg.RefreshTokenTimeout = 60
	cfg.IsConcurrent = false
	cfg.ConcurrencyScope = config.ConcurrencyScopeAccount
	if err := cfg.Validate(); err != nil {
		t.Fatalf("test config invalid: %v", err)
	}

	mgr := NewManager(
		cfg,
		&managerTestGenerator{},
		newManagerTestStorage(),
		managerTestCodec{},
		adapter.NewNopLogger(),
		nil,
		nil,
	)
	t.Cleanup(mgr.CloseManager)

	pair, err := mgr.LoginWithRefreshToken(ctx, "refresh-no-replace-before-success", "web")
	if err != nil {
		t.Fatalf("LoginWithRefreshToken() error = %v", err)
	}
	baseStorage := requireManagerTestStorage(t, mgr).(*managerTestStorage)
	mgr.storage = &managerTestFailingSetStorage{
		managerTestStorage: baseStorage,
		failKeyPrefix:      mgr.getRefreshTokenKey(""),
		failSetIfAbsent:    true,
	}
	if _, err = mgr.RefreshToken(ctx, pair.RefreshToken); !errors.Is(err, derror.ErrStorageUnavailable) {
		t.Fatalf("RefreshToken() error = %v, want ErrStorageUnavailable", err)
	}
	if err = mgr.CheckLogin(ctx, pair.AccessToken); err != nil {
		t.Fatalf("old access token CheckLogin() error = %v, want nil", err)
	}
}

// TestManagerIntrospectToken verifies active and inactive token responses. TestManagerIntrospectToken 验证活跃和非活跃令牌响应。
func TestManagerIntrospectToken(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManagerWithAccessProvider(t, nil, AccessProviderFunc{
		PermissionFunc: func(_ context.Context, subject AccessSubject) ([]string, error) {
			if subject.LoginID != "inspect-user" {
				t.Fatalf("Permissions subject loginID = %q, want inspect-user", subject.LoginID)
			}
			return []string{"article:read"}, nil
		},
		RoleFunc: func(_ context.Context, subject AccessSubject) ([]string, error) {
			if subject.Device != "web" {
				t.Fatalf("Roles subject device = %q, want web", subject.Device)
			}
			return []string{"admin"}, nil
		},
	})

	token, err := mgr.LoginWithOptions(ctx, LoginOptions{
		LoginID: "inspect-user",
		Device:  "web",
		Extra:   map[string]any{"scene": "introspection"},
	})
	if err != nil {
		t.Fatalf("LoginWithOptions() error = %v", err)
	}

	info, err := mgr.IntrospectToken(ctx, token)
	if err != nil {
		t.Fatalf("IntrospectToken() error = %v", err)
	}
	if !info.Active || info.LoginID != "inspect-user" || info.Device != "web" {
		t.Fatalf("IntrospectToken() = %+v, want active inspect-user/web", info)
	}
	if len(info.Permissions) != 1 || info.Permissions[0] != "article:read" {
		t.Fatalf("Permissions = %+v, want article:read", info.Permissions)
	}
	if len(info.Roles) != 1 || info.Roles[0] != "admin" {
		t.Fatalf("Roles = %+v, want admin", info.Roles)
	}
	if info.Extra["scene"] != "introspection" {
		t.Fatalf("Extra = %+v, want scene=introspection", info.Extra)
	}

	if err = mgr.Logout(ctx, token); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	info, err = mgr.IntrospectToken(ctx, token)
	if err != nil {
		t.Fatalf("IntrospectToken(logged out token) error = %v", err)
	}
	if info.Active || info.Error == "" {
		t.Fatalf("IntrospectToken(logged out token) = %+v, want inactive with error", info)
	}
}
