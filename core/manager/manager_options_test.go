package manager

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
)

func TestManagerLoginWithOptionsStoresOverrides(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.Timeout = 600
		cfg.IsConcurrent = true
		cfg.IsShare = true
	})

	token, err := mgr.LoginWithOptions(ctx, LoginOptions{
		LoginID:       "opt-user",
		Device:        "web",
		DeviceID:      "browser-1",
		Timeout:       90 * time.Second,
		ActiveTimeout: 30 * time.Second,
		Token:         "custom-token",
		Extra:         map[string]any{"trace": "token-extra"},
		TerminalExtra: map[string]any{"terminal": "extra"},
	})
	if err != nil {
		t.Fatalf("LoginWithOptions() error = %v", err)
	}
	if token != "custom-token" {
		t.Fatalf("LoginWithOptions() token = %q, want custom-token", token)
	}

	info, err := mgr.GetTokenInfo(ctx, token)
	if err != nil {
		t.Fatalf("GetTokenInfo() error = %v", err)
	}
	if info.Timeout != 90 || info.ActiveTimeout != 30 {
		t.Fatalf("TokenInfo timeout = %d active = %d, want 90 and 30", info.Timeout, info.ActiveTimeout)
	}
	if info.Extra["trace"] != "token-extra" {
		t.Fatalf("TokenInfo.Extra = %+v, want trace", info.Extra)
	}

	terminal, err := mgr.GetTerminalInfoByToken(ctx, token)
	if err != nil {
		t.Fatalf("GetTerminalInfoByToken() error = %v", err)
	}
	if terminal.Extra["terminal"] != "extra" {
		t.Fatalf("TerminalInfo.Extra = %+v, want terminal extra", terminal.Extra)
	}
}

func TestManagerLoginWithOptionsOverridesConcurrencyPolicy(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.IsConcurrent = true
		cfg.IsShare = true
		cfg.MaxLoginCount = config.NoLimit
	})

	isShare := false
	maxLoginCount := int64(1)
	overflowMode := config.LogoutModeKickout

	first, err := mgr.LoginWithOptions(ctx, LoginOptions{
		LoginID:            "policy-user",
		Device:             "web",
		DeviceID:           "a",
		IsShare:            &isShare,
		MaxLoginCount:      &maxLoginCount,
		OverflowLogoutMode: &overflowMode,
	})
	if err != nil {
		t.Fatalf("first LoginWithOptions() error = %v", err)
	}
	second, err := mgr.LoginWithOptions(ctx, LoginOptions{
		LoginID:            "policy-user",
		Device:             "mobile",
		DeviceID:           "b",
		IsShare:            &isShare,
		MaxLoginCount:      &maxLoginCount,
		OverflowLogoutMode: &overflowMode,
	})
	if err != nil {
		t.Fatalf("second LoginWithOptions() error = %v", err)
	}
	if first == second {
		t.Fatalf("LoginWithOptions() reused token %q, want new token", second)
	}
	if err = mgr.CheckLogin(ctx, first); !errors.Is(err, derror.ErrTokenKickout) {
		t.Fatalf("first CheckLogin() error = %v, want ErrTokenKickout", err)
	}
	if err = mgr.CheckLogin(ctx, second); err != nil {
		t.Fatalf("second CheckLogin() error = %v", err)
	}
}

func TestManagerSessionData(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, nil)

	if _, err := mgr.Login(ctx, "session-user"); err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if err := mgr.SetSessionValue(ctx, "session-user", "theme", "dark"); err != nil {
		t.Fatalf("SetSessionValue() error = %v", err)
	}
	value, ok, err := mgr.GetSessionValue(ctx, "session-user", "theme")
	if err != nil {
		t.Fatalf("GetSessionValue() error = %v", err)
	}
	if !ok || value != "dark" {
		t.Fatalf("GetSessionValue() = %v, %v, want dark, true", value, ok)
	}
	if err = mgr.DeleteSessionValue(ctx, "session-user", "theme"); err != nil {
		t.Fatalf("DeleteSessionValue() error = %v", err)
	}
	value, ok, err = mgr.GetSessionValue(ctx, "session-user", "theme")
	if err != nil {
		t.Fatalf("GetSessionValue(after delete) error = %v", err)
	}
	if ok || value != nil {
		t.Fatalf("GetSessionValue(after delete) = %v, %v, want nil, false", value, ok)
	}

	if err = mgr.SetSessionValue(ctx, "", "theme", "dark"); !errors.Is(err, derror.ErrIDIsEmpty) {
		t.Fatalf("SetSessionValue(empty id) error = %v, want ErrIDIsEmpty", err)
	}
	if _, _, err = mgr.GetSessionValue(ctx, "session-user", ""); !errors.Is(err, derror.ErrInvalidParam) {
		t.Fatalf("GetSessionValue(empty key) error = %v, want ErrInvalidParam", err)
	}
}

func TestManagerTerminate(t *testing.T) {
	ctx := context.Background()

	t.Run("default logout by token", func(t *testing.T) {
		mgr := newTestManager(t, nil)
		token, err := mgr.Login(ctx, "term-user", "web")
		if err != nil {
			t.Fatalf("Login() error = %v", err)
		}
		if err = mgr.Terminate(ctx, TerminateOptions{Token: token}); err != nil {
			t.Fatalf("Terminate() error = %v", err)
		}
		if err = mgr.CheckLogin(ctx, token); !errors.Is(err, derror.ErrInvalidToken) {
			t.Fatalf("CheckLogin() error = %v, want ErrInvalidToken", err)
		}
	})

	t.Run("kickout by device id", func(t *testing.T) {
		mgr := newTestManager(t, nil)
		first, err := mgr.Login(ctx, "term-device-user", "web", "a")
		if err != nil {
			t.Fatalf("first Login() error = %v", err)
		}
		second, err := mgr.Login(ctx, "term-device-user", "web", "b")
		if err != nil {
			t.Fatalf("second Login() error = %v", err)
		}
		if err = mgr.Terminate(ctx, TerminateOptions{
			Action:   TerminateActionKickout,
			LoginID:  "term-device-user",
			Device:   "web",
			DeviceID: "a",
		}); err != nil {
			t.Fatalf("Terminate(kickout device id) error = %v", err)
		}
		if err = mgr.CheckLogin(ctx, first); !errors.Is(err, derror.ErrTokenKickout) {
			t.Fatalf("first CheckLogin() error = %v, want ErrTokenKickout", err)
		}
		if err = mgr.CheckLogin(ctx, second); err != nil {
			t.Fatalf("second CheckLogin() error = %v", err)
		}
	})

	t.Run("invalid options", func(t *testing.T) {
		mgr := newTestManager(t, nil)
		if err := mgr.Terminate(ctx, TerminateOptions{}); !errors.Is(err, derror.ErrIDIsEmpty) {
			t.Fatalf("Terminate(empty) error = %v, want ErrIDIsEmpty", err)
		}
		if err := mgr.Terminate(ctx, TerminateOptions{Token: "x", Action: "unknown"}); !errors.Is(err, derror.ErrInvalidParam) {
			t.Fatalf("Terminate(invalid action) error = %v, want ErrInvalidParam", err)
		}
	})
}

func TestManagerStrategy(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, nil)

	strategy := mgr.GetStrategy()
	if strategy == nil {
		t.Fatal("GetStrategy() = nil, want default strategy")
	}
	if !strategy.PermissionMatcher("order/*", "order/read") {
		t.Fatal("default PermissionMatcher(order/*, order/read) = false, want true")
	}

	customRoleMatcher := func(pattern, role string) bool {
		return strings.EqualFold(pattern, role)
	}
	customMgr := newTestManagerWithStrategy(t, &Strategy{RoleMatcher: customRoleMatcher})
	if _, err := customMgr.Login(ctx, "strategy-user"); err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if err := customMgr.AddRoles(ctx, "strategy-user", []string{"Admin"}); err != nil {
		t.Fatalf("AddRoles() error = %v", err)
	}
	if !customMgr.HasRole(ctx, "strategy-user", "admin") {
		t.Fatal("HasRole(admin) = false, want custom role matcher to ignore case")
	}
}

func newTestManagerWithStrategy(t *testing.T, strategy *Strategy) *Manager {
	t.Helper()

	cfg := config.DefaultConfig()
	cfg.IsPrintBanner = false
	cfg.IsLog = false
	cfg.AsyncEvent = false
	cfg.AutoRenew = false
	cfg.RenewInterval = config.NoLimit
	cfg.ActiveTimeout = config.NoLimit
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
		WithStrategy(strategy),
	)
	t.Cleanup(mgr.CloseManager)
	return mgr
}
