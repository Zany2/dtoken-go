// @Author daixk 2025/12/22 15:56:00
package manager

import (
	"context"
	"errors"
	"testing"
	"time"

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
