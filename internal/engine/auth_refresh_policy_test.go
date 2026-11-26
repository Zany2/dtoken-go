package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/config"
	derror "github.com/Zany2/dtoken-go/errors"
	listener "github.com/Zany2/dtoken-go/event"
)

// TestManagerRefreshHonorsOccupiedLoginSlots verifies rotation only bypasses policy when replacing its own attached terminal. TestManagerRefreshHonorsOccupiedLoginSlots 验证刷新仅在替换自身已挂载终端时跳过登录策略。
func TestManagerRefreshHonorsOccupiedLoginSlots(t *testing.T) {
	ctx := context.Background()
	for _, storageMode := range []string{"atomic", "basic"} {
		for _, scope := range []config.ConcurrencyScope{config.ConcurrencyScopeAccount, config.ConcurrencyScopeDevice} {
			for _, policy := range []string{"max count", "replace old", "reject new"} {
				for _, lifecycle := range []string{"attached", "expired"} {
					t.Run(storageMode+"/"+string(scope)+"/"+policy+"/"+lifecycle, func(t *testing.T) {
						mgr := newTerminalLifecycleTestManager(t, func(cfg *config.Config) {
							cfg.ConcurrencyScope = scope
							cfg.MaxLoginCount = 1
							cfg.OverflowLogoutMode = config.LogoutModeKickout
							cfg.IsConcurrent = policy == "max count"
							cfg.ReplacedLoginExitMode = config.ReplacedLoginExitModeOldDevice
							if policy == "reject new" {
								cfg.ReplacedLoginExitMode = config.ReplacedLoginExitModeNewDevice
							}
						})
						if storageMode == "basic" {
							mgr.storage = nonAtomicManagerStorage{Storage: mgr.storage}
						}
						old, err := mgr.LoginWithRefreshToken(ctx, "refresh-slot", "web", "original")
						if err != nil {
							t.Fatal(err)
						}
						var occupying *RefreshTokenPair
						if lifecycle == "expired" {
							// Preserve refresh metadata while an ordinary login takes the released access slot. 保留刷新元数据，由普通登录占用访问令牌过期后释放的名额。
							if err = mgr.storage.Delete(ctx, mgr.getTokenKey(old.AccessToken)); err != nil {
								t.Fatal(err)
							}
							occupying, err = mgr.LoginWithRefreshToken(ctx, old.LoginID, "web", "occupying")
							if err != nil {
								t.Fatal(err)
							}
						}
						var otherDevice string
						if scope == config.ConcurrencyScopeDevice {
							otherDevice, err = mgr.Login(ctx, old.LoginID, "mobile", "unrelated")
							if err != nil {
								t.Fatal(err)
							}
						}
						if err = mgr.SetSessionValue(ctx, old.LoginID, "theme", "dark"); err != nil {
							t.Fatal(err)
						}
						events := registerTerminalLifecycleEvents(mgr)
						next, err := mgr.RefreshToken(ctx, old.RefreshToken)
						rejected := lifecycle == "expired" && policy == "reject new"
						if rejected {
							if next != nil || !errors.Is(err, derror.ErrLoginLimitExceeded) {
								t.Fatalf("occupied slot refresh = %+v, %v, want ErrLoginLimitExceeded", next, err)
							}
							if len(events()) != 0 {
								t.Fatalf("rejected refresh emitted events: %+v", events())
							}
							if _, err = mgr.getRefreshTokenInfo(ctx, occupying.RefreshToken); err != nil {
								t.Fatalf("rejection revoked the occupying terminal: %v", err)
							}
						} else {
							if err != nil || next == nil {
								t.Fatalf("refresh = %+v, %v", next, err)
							}
							if occupying != nil {
								wantErr := derror.ErrTokenKickout
								if policy == "replace old" {
									wantErr = derror.ErrTokenReplaced
								}
								if _, err = mgr.getTokenRecord(ctx, occupying.AccessToken); !errors.Is(err, wantErr) {
									t.Fatalf("occupying token error = %v, want %v", err, wantErr)
								}
								if mgr.storage.Exists(ctx, mgr.getRefreshTokenKey(occupying.RefreshToken)) {
									t.Fatal("evicted terminal retained its refresh credential")
								}
							}
						}
						if mgr.storage.Exists(ctx, mgr.getRefreshTokenKey(old.RefreshToken)) {
							t.Fatal("old refresh token was not consumed")
						}
						sess, err := mgr.getSession(ctx, old.LoginID)
						if err != nil {
							t.Fatal(err)
						}
						web := sessionGetTerminalsByDevice(sess, "web")
						wantToken := ""
						if rejected {
							wantToken = occupying.AccessToken
						} else {
							wantToken = next.AccessToken
						}
						// Rejected admission preserves stale indexes; only live terminals occupy slots. 拒绝接收保留残留索引，仅存活终端占用名额。
						wantRecorded := 1
						if rejected {
							wantRecorded = 2
						}
						live, liveErr := mgr.GetTokenValueListByDevice(ctx, old.LoginID, "web", true)
						if liveErr != nil || len(live) != 1 || live[0] != wantToken || len(web) != wantRecorded || sess.Data["theme"] != "dark" {
							t.Fatalf("rotation exceeded the slot limit or lost session data: %+v", sess)
						}
						if otherDevice != "" && !sessionHasTerminalToken(sess, otherDevice) {
							t.Fatal("device-scoped rotation removed another device")
						}
					})
				}
			}
		}
	}
}

// TestManagerRefreshRetiresInactiveLifecycle verifies inactivity cannot be bypassed by refreshing before authentication. TestManagerRefreshRetiresInactiveLifecycle 验证先刷新不能绕过不活跃超时。
func TestManagerRefreshRetiresInactiveLifecycle(t *testing.T) {
	ctx := context.Background()
	for _, storageMode := range []string{"atomic", "basic"} {
		for _, scenario := range []string{"last terminal", "other terminal", "missing session", "missing reverse", "authentication first"} {
			t.Run(storageMode+"/"+scenario, func(t *testing.T) {
				mgr := newTerminalLifecycleTestManager(t, nil)
				if storageMode == "basic" {
					mgr.storage = nonAtomicManagerStorage{Storage: mgr.storage}
				}
				pair, err := mgr.LoginWithRefreshToken(ctx, "inactive-refresh", "web", "expired")
				if err != nil {
					t.Fatal(err)
				}
				var other string
				switch scenario {
				case "other terminal":
					other, err = mgr.Login(ctx, pair.LoginID, "mobile", "kept")
				case "missing session":
					err = mgr.storage.Delete(ctx, mgr.getSessionKey(pair.LoginID))
				case "missing reverse":
					err = mgr.storage.Delete(ctx, mgr.getTokenRefreshKey(pair.AccessToken))
				}
				if err != nil {
					t.Fatal(err)
				}
				if err = mgr.storage.Set(ctx, mgr.getActiveKey(pair.AccessToken), time.Now().Add(-2*time.Minute).Unix(), 10*time.Minute); err != nil {
					t.Fatal(err)
				}
				callbackLocked := false
				mgr.GetEventManager().RegisterFuncWithConfig(listener.EventAll, func(*listener.EventData) {
					mgr.loginLocksMu.Lock()
					if entry := mgr.loginLocks[pair.LoginID]; entry != nil {
						if entry.mu.TryLock() {
							entry.mu.Unlock()
						} else {
							callbackLocked = true
						}
					}
					mgr.loginLocksMu.Unlock()
				}, listener.ListenerConfig{Async: false})
				events := registerTerminalLifecycleEvents(mgr)
				wantErr := derror.ErrActiveTimeout
				if scenario == "authentication first" {
					if err = mgr.CheckLogin(ctx, pair.AccessToken); !errors.Is(err, derror.ErrActiveTimeout) {
						t.Fatalf("authentication error = %v", err)
					}
					wantErr = derror.ErrInvalidRefreshToken
				}
				next, err := mgr.RefreshToken(ctx, pair.RefreshToken)
				if next != nil || !errors.Is(err, wantErr) {
					t.Fatalf("inactive refresh = %+v, %v, want %v", next, err, wantErr)
				}
				for _, key := range []string{mgr.getRefreshTokenKey(pair.RefreshToken), mgr.getTokenRefreshKey(pair.AccessToken), mgr.getActiveKey(pair.AccessToken), mgr.getRenewKey(pair.AccessToken)} {
					if mgr.storage.Exists(ctx, key) {
						t.Fatalf("inactive lifecycle retained key %q", key)
					}
				}
				if _, err = mgr.getTokenRecord(ctx, pair.AccessToken); !errors.Is(err, derror.ErrActiveTimeout) {
					t.Fatalf("retired access error = %v", err)
				}
				if _, err = mgr.RefreshToken(ctx, pair.RefreshToken); !errors.Is(err, derror.ErrInvalidRefreshToken) {
					t.Fatalf("replayed refresh error = %v", err)
				}
				timeouts, destroys := 0, 0
				for _, event := range events() {
					switch event.Event {
					case listener.EventActiveTimeout:
						timeouts++
					case listener.EventDestroySession:
						destroys++
					default:
						t.Fatalf("unexpected event: %+v", event)
					}
				}
				wantDestroys := 1
				if scenario == "missing session" || other != "" {
					wantDestroys = 0
				}
				if timeouts != 1 || destroys != wantDestroys {
					t.Fatalf("events = %d timeouts, %d destroys; want 1, %d", timeouts, destroys, wantDestroys)
				}
				if callbackLocked {
					t.Fatal("timeout event ran while the account lock was held")
				}
				if other != "" {
					sess, err := mgr.getSession(ctx, pair.LoginID)
					if err != nil || len(sess.TerminalInfos) != 1 || sess.TerminalInfos[0].Token != other {
						t.Fatalf("unrelated terminal changed: %+v, %v", sess, err)
					}
				} else if mgr.storage.Exists(ctx, mgr.getSessionKey(pair.LoginID)) {
					t.Fatal("empty or missing session was retained")
				}
			})
		}
	}
}
