package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/config"
	derror "github.com/Zany2/dtoken-go/errors"
)

// TestManagerLoginSkipsUnrelatedTokenReads verifies ordinary and shared logins do not sweep other credentials. TestManagerLoginSkipsUnrelatedTokenReads 验证普通及共享登录不清扫其他凭据。
func TestManagerLoginSkipsUnrelatedTokenReads(t *testing.T) {
	ctx := context.Background()
	for _, mode := range []string{"unlimited", "below limit", "shared"} {
		t.Run(mode, func(t *testing.T) {
			mgr := newTestManager(t, func(cfg *config.Config) {
				cfg.IsShare = false
				cfg.MaxLoginCount = config.NoLimit
			})
			unrelated, err := mgr.Login(ctx, "login-cost", "mobile", "other")
			if err != nil {
				t.Fatal(err)
			}
			candidate, err := mgr.Login(ctx, "login-cost", "web", "browser")
			if err != nil {
				t.Fatal(err)
			}
			// Corrupt unrelated data must not affect an operation that does not need it. 无需读取的无关数据损坏不应影响本次操作。
			if err = mgr.storage.Set(ctx, mgr.getTokenKey(unrelated), "{", time.Minute); err != nil {
				t.Fatal(err)
			}
			if mode != "unlimited" {
				mgr.config.MaxLoginCount = 12
			}
			mgr.config.IsShare = mode == "shared"
			storage := &managerGetCountingStorage{Storage: mgr.storage}
			mgr.storage = storage
			token, err := mgr.Login(ctx, "login-cost", "web", "browser")
			if err != nil {
				t.Fatal(err)
			}
			want := 0
			if mode == "shared" {
				want = 1
				if token != candidate {
					t.Fatal("shared login changed token")
				}
			}
			if storage.getCount(mgr.getTokenKey(unrelated)) != 0 || storage.getCount(mgr.getTokenKey(candidate)) != want {
				t.Fatalf("unrelated/candidate reads = %d/%d, want 0/%d", storage.getCount(mgr.getTokenKey(unrelated)), storage.getCount(mgr.getTokenKey(candidate)), want)
			}
		})
	}
}

// TestManagerLoginLimitPrunesOnlyItsDevice verifies expired slots do not evict a valid token or inspect unrelated devices. TestManagerLoginLimitPrunesOnlyItsDevice 验证过期名额不会淘汰有效 Token 或检查无关设备。
func TestManagerLoginLimitPrunesOnlyItsDevice(t *testing.T) {
	ctx := context.Background()
	for _, share := range []bool{false, true} {
		name := "new token"
		if share {
			name = "shared token"
		}
		t.Run(name, func(t *testing.T) {
			mgr := newTestManager(t, func(cfg *config.Config) {
				cfg.IsShare = false
				cfg.MaxLoginCount = config.NoLimit
				cfg.ConcurrencyScope = config.ConcurrencyScopeDevice
			})
			valid, err := mgr.Login(ctx, "scope-limit", "web", "old")
			if err != nil {
				t.Fatal(err)
			}
			stale, err := mgr.LoginWithRefreshToken(ctx, "scope-limit", "web", "expired")
			if err != nil {
				t.Fatal(err)
			}
			other, err := mgr.Login(ctx, "scope-limit", "mobile", "other")
			if err != nil {
				t.Fatal(err)
			}
			var candidate string
			if share {
				candidate, err = mgr.Login(ctx, "scope-limit", "web", "shared")
				if err != nil {
					t.Fatal(err)
				}
			}
			if err = mgr.storage.Delete(ctx, mgr.getTokenKey(stale.AccessToken)); err != nil {
				t.Fatal(err)
			}
			if err = mgr.storage.Set(ctx, mgr.getTokenKey(other), "{", time.Minute); err != nil {
				t.Fatal(err)
			}
			mgr.config.MaxLoginCount, mgr.config.IsShare = 2, share
			storage := &managerGetCountingStorage{Storage: mgr.storage}
			mgr.storage = storage
			token, err := mgr.Login(ctx, "scope-limit", "web", "shared")
			if err != nil {
				t.Fatal(err)
			}
			if share && token != candidate {
				t.Fatal("shared token was not retained")
			}
			if storage.getCount(mgr.getTokenKey(other)) != 0 {
				t.Fatal("device policy read another device token")
			}
			if err = mgr.CheckLogin(ctx, valid); err != nil {
				t.Fatalf("expired entry evicted valid token: %v", err)
			}
			if !mgr.storage.Exists(ctx, mgr.getRefreshTokenKey(stale.RefreshToken)) {
				t.Fatal("natural expiration revoked independent refresh credential")
			}
			sess, err := mgr.getSession(ctx, "scope-limit")
			if err != nil {
				t.Fatal(err)
			}
			if len(sess.TerminalInfos) != 3 || sessionHasTerminalToken(sess, stale.AccessToken) || !sessionHasTerminalToken(sess, other) {
				t.Fatalf("unexpected scoped cleanup: %+v", sess.TerminalInfos)
			}
		})
	}
}

// TestManagerExclusiveLoginStopsAtLiveSlot verifies early rejection and exact lifecycle matching. TestManagerExclusiveLoginStopsAtLiveSlot 验证遇到有效名额即拒绝，并精确匹配生命周期。
func TestManagerExclusiveLoginStopsAtLiveSlot(t *testing.T) {
	ctx := context.Background()
	for _, mismatched := range []bool{false, true} {
		name := "live slot"
		if mismatched {
			name = "stale lifecycle"
		}
		t.Run(name, func(t *testing.T) {
			mgr := newTestManager(t, func(cfg *config.Config) {
				cfg.IsShare = false
				cfg.MaxLoginCount = config.NoLimit
			})
			old, err := mgr.Login(ctx, "exclusive-cost", "web", "old")
			if err != nil {
				t.Fatal(err)
			}
			latest, err := mgr.Login(ctx, "exclusive-cost", "web", "latest")
			if err != nil {
				t.Fatal(err)
			}
			if err = mgr.storage.Delete(ctx, mgr.getTokenKey(old)); err != nil {
				t.Fatal(err)
			}
			if mismatched {
				sess, err := mgr.getSession(ctx, "exclusive-cost")
				if err != nil {
					t.Fatal(err)
				}
				sess.TerminalInfos[1].Index++
				if err = mgr.saveToStorage(ctx, mgr.getSessionKey(sess.LoginID), *sess); err != nil {
					t.Fatal(err)
				}
			}
			mgr.config.IsConcurrent = false
			mgr.config.ReplacedLoginExitMode = config.ReplacedLoginExitModeNewDevice
			storage := &managerGetCountingStorage{Storage: mgr.storage}
			mgr.storage = storage
			_, err = mgr.Login(ctx, "exclusive-cost", "web", "new")
			if mismatched {
				if err != nil {
					t.Fatalf("stale lifecycle blocked new login: %v", err)
				}
			} else {
				if !errors.Is(err, derror.ErrLoginLimitExceeded) {
					t.Fatalf("exclusive login error = %v", err)
				}
				if storage.getCount(mgr.getTokenKey(old)) != 0 {
					t.Fatal("rejection continued scanning older slots")
				}
			}
			if storage.getCount(mgr.getTokenKey(latest)) != 1 {
				t.Fatal("latest slot was reread")
			}
		})
	}
}

// TestManagerTokenTextReuseReplacesStaleIndex verifies lazy cleanup cannot leave duplicate entries for an explicit token. TestManagerTokenTextReuseReplacesStaleIndex 验证懒清理不会使显式复用的 Token 留下重复条目。
func TestManagerTokenTextReuseReplacesStaleIndex(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.IsShare = false
		cfg.MaxLoginCount = config.NoLimit
	})
	opts := LoginOptions{LoginID: "reuse-index", Token: "reused-text", Device: "web"}
	token, err := mgr.LoginWithOptions(ctx, opts)
	if err != nil {
		t.Fatal(err)
	}
	if err = mgr.storage.Delete(ctx, mgr.getTokenKey(token)); err != nil {
		t.Fatal(err)
	}
	if _, err = mgr.LoginWithOptions(ctx, opts); err != nil {
		t.Fatal(err)
	}
	sess, err := mgr.getSession(ctx, opts.LoginID)
	if err != nil {
		t.Fatal(err)
	}
	record, err := mgr.getTokenRecord(ctx, token)
	if err != nil {
		t.Fatal(err)
	}
	if len(sess.TerminalInfos) != 1 || sess.HistoryTerminalCount != 2 || !terminalMatchesTokenRecord(sess.LoginID, sess.TerminalInfos[0], record) {
		t.Fatalf("reused token index = %+v", sess)
	}
}

// TestManagerRefreshDoesNotSweepOtherTokens checks rotation reads only the affected credential lifecycle. TestManagerRefreshDoesNotSweepOtherTokens 检查刷新轮换只读取涉及的凭据生命周期。
func TestManagerRefreshDoesNotSweepOtherTokens(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.IsShare = false
		cfg.MaxLoginCount = config.NoLimit
	})
	pair, err := mgr.LoginWithRefreshToken(ctx, "rotation-cost", "web")
	if err != nil {
		t.Fatal(err)
	}
	other, err := mgr.Login(ctx, "rotation-cost", "mobile")
	if err != nil {
		t.Fatal(err)
	}
	if err = mgr.storage.Set(ctx, mgr.getTokenKey(other), "{", time.Minute); err != nil {
		t.Fatal(err)
	}
	storage := &managerGetCountingStorage{Storage: mgr.storage}
	mgr.storage = storage
	if _, err = mgr.RefreshToken(ctx, pair.RefreshToken); err != nil {
		t.Fatal(err)
	}
	if storage.getCount(mgr.getTokenKey(other)) != 0 {
		t.Fatal("rotation swept unrelated token")
	}
}

// TestManagerScopedCleanupRetiresInactiveLifecycle keeps timeout cleanup and refresh revocation within scope. TestManagerScopedCleanupRetiresInactiveLifecycle 将超时清理及刷新撤销限制在作用域内。
func TestManagerScopedCleanupRetiresInactiveLifecycle(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.IsShare = false
		cfg.MaxLoginCount = config.NoLimit
		cfg.ActiveTimeout = 30
		cfg.ConcurrencyScope = config.ConcurrencyScopeDevice
	})
	old, err := mgr.LoginWithRefreshTokenOptions(ctx, RefreshTokenOptions{LoginOptions: LoginOptions{LoginID: "scope-timeout", Device: "web", Extra: map[string]any{"old": true}}})
	if err != nil {
		t.Fatal(err)
	}
	record, err := mgr.getTokenRecord(ctx, old.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	other, err := mgr.Login(ctx, old.LoginID, "mobile")
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{old.AccessToken, other} {
		if err = mgr.storage.Set(ctx, mgr.getActiveKey(token), time.Now().Add(-time.Hour).Unix(), time.Hour); err != nil {
			t.Fatal(err)
		}
	}
	mgr.config.MaxLoginCount = 1
	if _, err = mgr.Login(ctx, old.LoginID, "web"); err != nil {
		t.Fatal(err)
	}
	if _, err = mgr.getTokenRecord(ctx, old.AccessToken); !errors.Is(err, derror.ErrActiveTimeout) {
		t.Fatalf("expired slot state = %v", err)
	}
	if mgr.storage.Exists(ctx, mgr.getRefreshTokenKey(old.RefreshToken)) || mgr.storage.Exists(ctx, mgr.getTokenExtraKey(record.AccessID)) {
		t.Fatal("timeout left refresh or extra data")
	}
	if _, err = mgr.getTokenRecord(ctx, other); err != nil {
		t.Fatalf("unrelated device was retired: %v", err)
	}
}
