package manager

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/listener"
)

// TestManagerAccountDisableRetiresCredentials verifies untie and a new session cannot revive old credentials. TestManagerAccountDisableRetiresCredentials 验证解封及新建会话不会使旧凭证复活。
func TestManagerAccountDisableRetiresCredentials(t *testing.T) {
	ctx := context.Background()
	for _, ordinary := range []bool{false, true} {
		for _, activeTimeout := range []int64{config.NoLimit, 60} {
			for _, release := range []string{"untie", "expired marker"} {
				t.Run(fmt.Sprintf("ordinary=%v/active=%d/%s", ordinary, activeTimeout, release), func(t *testing.T) {
					mgr := newTestManager(t, func(cfg *config.Config) {
						cfg.ActiveTimeout = activeTimeout
						cfg.Timeout = 300
					})
					if ordinary {
						mgr.storage = &managerStorageOnly{inner: newManagerTestStorage()}
					}
					id := "disabled-lifecycle"
					pair, err := mgr.LoginWithRefreshToken(ctx, id, "web", "old")
					if err != nil {
						t.Fatal(err)
					}
					beforeTTL, err := mgr.storage.TTL(ctx, mgr.getTokenKey(pair.AccessToken))
					if err != nil {
						t.Fatal(err)
					}
					pool := &managerQueuedMaintenancePool{}
					mgr.pool = pool
					t.Cleanup(pool.runAll)
					if err := mgr.Disable(ctx, id, time.Hour); err != nil {
						t.Fatal(err)
					}
					if pool.taskCount() != 0 {
						t.Fatal("account retirement still depends on an asynchronous worker")
					}
					if err := mgr.CheckLogin(ctx, pair.AccessToken); !errors.Is(err, derror.ErrAccountDisabled) {
						t.Fatalf("disabled access error = %v", err)
					}
					for _, key := range []string{mgr.getSessionKey(id), mgr.getActiveKey(pair.AccessToken), mgr.getRenewKey(pair.AccessToken), mgr.getRefreshTokenKey(pair.RefreshToken), mgr.getTokenRefreshKey(pair.AccessToken)} {
						if mgr.storage.Exists(ctx, key) {
							t.Fatalf("Disable retained %q", key)
						}
					}
					if ttl, err := mgr.storage.TTL(ctx, mgr.getTokenKey(pair.AccessToken)); err != nil || ttl <= 0 || ttl > beforeTTL+time.Second {
						t.Fatalf("retired token TTL = %v, %v, want no extension from %v", ttl, err, beforeTTL)
					}
					if release == "untie" {
						err = mgr.Untie(ctx, id)
					} else {
						// Removing only the marker reproduces storage expiry without sleeps. 仅移除封禁标记以模拟存储到期，无需等待。
						err = mgr.storage.Delete(ctx, mgr.getDisableKey(id))
					}
					if err != nil {
						t.Fatal(err)
					}
					fresh, err := mgr.Login(ctx, id, "web", "new")
					if err != nil {
						t.Fatal(err)
					}
					if err := mgr.CheckLogin(ctx, pair.AccessToken); !errors.Is(err, derror.ErrInvalidToken) {
						t.Fatalf("retired token revived after new login: %v", err)
					}
					if _, err := mgr.RefreshToken(ctx, pair.RefreshToken); !errors.Is(err, derror.ErrInvalidRefreshToken) {
						t.Fatalf("retired refresh revived after new login: %v", err)
					}
					if err := mgr.CheckLogin(ctx, fresh); err != nil {
						t.Fatalf("new login error = %v", err)
					}
				})
			}
		}
	}
}

// TestManagerDisableSupportsInlinePoolAndEmptySessions covers lock reentry and session destroy events. TestManagerDisableSupportsInlinePoolAndEmptySessions 覆盖锁重入与空 Session 销毁事件。
func TestManagerDisableSupportsInlinePoolAndEmptySessions(t *testing.T) {
	ctx := context.Background()
	for _, mode := range []string{"terminal", "empty session", "no session"} {
		t.Run(mode, func(t *testing.T) {
			mgr := newTestManager(t, nil)
			id := "inline-disable"
			if mode == "terminal" {
				if _, err := mgr.LoginWithRefreshToken(ctx, id, "web"); err != nil {
					t.Fatal(err)
				}
			} else if mode == "empty session" {
				if err := mgr.saveToStorage(ctx, mgr.getSessionKey(id), *defaultCreateSession(mgr.config.AuthType, id, time.Now().Unix())); err != nil {
					t.Fatal(err)
				}
			}
			pool := &managerAccessInlinePool{mgr: mgr, loginID: id}
			mgr.pool = pool
			var events []listener.Event
			mgr.GetEventManager().RegisterFuncWithConfig(listener.EventAll, func(data *listener.EventData) {
				unlock := mgr.lockLoginWrite(id)
				unlock()
				events = append(events, data.Event)
			}, listener.ListenerConfig{Async: false})
			if err := mgr.Disable(ctx, id, 0); err != nil {
				t.Fatal(err)
			}
			if pool.locked {
				t.Fatal("Disable submitted work under the account lock")
			}
			want := []listener.Event{listener.EventDisable}
			if mode != "no session" {
				want = append([]listener.Event{listener.EventDestroySession}, want...)
			}
			if !reflect.DeepEqual(events, want) {
				t.Fatalf("events = %v, want %v", events, want)
			}
			if err := mgr.Disable(ctx, id, 0); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(events, append(want, listener.EventDisable)) {
				t.Fatalf("repeated Disable destroyed an absent session: %v", events)
			}
		})
	}
}

// TestManagerDisablePreservesForeignTerminal verifies stale entries do not revoke another account's credentials. TestManagerDisablePreservesForeignTerminal 验证陈旧终端条目不会撤销其他账号的凭证。
func TestManagerDisablePreservesForeignTerminal(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, nil)
	pair, err := mgr.LoginWithRefreshToken(ctx, "kept-owner", "web", "browser")
	if err != nil {
		t.Fatal(err)
	}
	sess, err := mgr.getSession(ctx, pair.LoginID)
	if err != nil {
		t.Fatal(err)
	}
	sess.LoginID = "stale-owner"
	if err := mgr.saveToStorage(ctx, mgr.getSessionKey(sess.LoginID), *sess); err != nil {
		t.Fatal(err)
	}
	before := captureTerminalLifecycle(t, mgr, ctx, []string{mgr.getSessionKey(pair.LoginID), mgr.getTokenKey(pair.AccessToken), mgr.getActiveKey(pair.AccessToken), mgr.getTokenRefreshKey(pair.AccessToken), mgr.getRefreshTokenKey(pair.RefreshToken)})
	if err := mgr.Disable(ctx, sess.LoginID, time.Hour); err != nil {
		t.Fatal(err)
	}
	assertTerminalLifecycleUnchanged(t, mgr, ctx, before)
}

// TestManagerDisableReportsCleanupFailure verifies failed retirement retains the session index and emits no success events. TestManagerDisableReportsCleanupFailure 验证清理失败保留 Session 索引且不发送成功事件。
func TestManagerDisableReportsCleanupFailure(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, nil)
	id := "disable-cleanup-error"
	pair, err := mgr.LoginWithRefreshToken(ctx, id, "web")
	if err != nil {
		t.Fatal(err)
	}
	base := mgr.storage
	mgr.storage = &managerFailingStorage{Storage: base, deleteErr: errors.New("cleanup unavailable")}
	events := registerTerminalLifecycleEvents(mgr)
	if err := mgr.Disable(ctx, id, time.Minute); !errors.Is(err, derror.ErrStorageUnavailable) {
		t.Fatalf("Disable cleanup error = %v", err)
	}
	if !base.Exists(ctx, mgr.getSessionKey(id)) || len(events()) != 0 {
		t.Fatal("failed cleanup lost the retry index or emitted success events")
	}
	mgr.storage = base
	if err := mgr.Disable(ctx, id, time.Minute); err != nil {
		t.Fatalf("retry Disable error = %v", err)
	}
	if base.Exists(ctx, mgr.getRefreshTokenKey(pair.RefreshToken)) {
		t.Fatal("retry retained refresh token")
	}
}

// TestManagerDisableReplacesLegacyRestriction verifies new expiry cannot reveal the previous ban. TestManagerDisableReplacesLegacyRestriction 验证新封禁到期不会恢复旧封禁。
func TestManagerDisableReplacesLegacyRestriction(t *testing.T) {
	ctx := context.Background()
	id := "legacy:account"
	for _, kind := range []string{"account", "service", "device", "concrete device"} {
		for _, legacyMode := range []string{"matching", "foreign", "corrupt"} {
			t.Run(kind+"/"+legacyMode, func(t *testing.T) {
				mgr := newTestManager(t, nil)
				current, legacy := mgr.getDisableKey(id), mgr.getLegacyDisableKey(id)
				marker := disableMarker{LoginID: id, Kind: "account"}
				write := func() error { return mgr.Disable(ctx, id, time.Minute) }
				ttl := func() (int64, error) { return mgr.GetDisableTTL(ctx, id) }
				switch kind {
				case "service":
					current, legacy = mgr.getDisableServiceKey(id, "pay"), mgr.getLegacyDisableServiceKey(id, "pay")
					marker.Kind, marker.Service, marker.Level = "service", "pay", 9
					write = func() error { return mgr.DisableServiceLevel(ctx, id, "pay", 1, time.Minute) }
					ttl = func() (int64, error) { return mgr.GetDisableServiceTTL(ctx, id, "pay") }
				case "device":
					current, legacy = mgr.getDisableDeviceKey(id, "web"), mgr.getLegacyDisableDeviceKey(id, "web")
					marker.Kind, marker.Device = "device", "web"
					write = func() error { return mgr.DisableDevice(ctx, id, "web", time.Minute) }
					ttl = func() (int64, error) { return mgr.GetDisableDeviceTTL(ctx, id, "web") }
				case "concrete device":
					current, legacy = mgr.getDisableDeviceAndDeviceIDKey(id, "web", "phone"), mgr.getLegacyDisableDeviceAndDeviceIDKey(id, "web", "phone")
					marker.Kind, marker.Device, marker.DeviceID = "device", "web", "phone"
					write = func() error { return mgr.DisableDeviceAndDeviceID(ctx, id, "web", "phone", time.Minute) }
					ttl = func() (int64, error) { return mgr.GetDisableDeviceAndDeviceIDTTL(ctx, id, "web", "phone") }
				}
				if current == legacy {
					t.Fatal("fixture must use separate current and legacy keys")
				}
				if legacyMode == "foreign" {
					marker.LoginID = "other-owner"
				} else {
					// Old records had business fields but no explicit owner or kind. 旧记录包含业务字段，但没有显式账号和类型。
					marker.LoginID, marker.Kind = "", ""
				}
				if err := mgr.saveToStorage(ctx, legacy, marker, 0); err != nil {
					t.Fatal(err)
				}
				if legacyMode == "corrupt" {
					if err := mgr.storage.Set(ctx, legacy, []byte("invalid-json"), 0); err != nil {
						t.Fatal(err)
					}
				}
				before := captureTerminalLifecycle(t, mgr, ctx, []string{legacy})
				err := write()
				if legacyMode == "corrupt" {
					if !errors.Is(err, derror.ErrSerializeFailed) || mgr.storage.Exists(ctx, current) {
						t.Fatalf("unverifiable legacy write = %v, want rejection before writing", err)
					}
					assertTerminalLifecycleUnchanged(t, mgr, ctx, before)
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				if got, err := ttl(); err != nil || got <= 0 || got > 60 {
					t.Fatalf("updated TTL = %d, %v", got, err)
				}
				if legacyMode == "matching" && mgr.storage.Exists(ctx, legacy) {
					t.Fatal("obsolete legacy restriction retained")
				}
				if legacyMode == "foreign" {
					assertTerminalLifecycleUnchanged(t, mgr, ctx, before)
				}
				// Simulate expiry of the new marker; the previous restriction must not reappear. 模拟新标记到期，旧限制不能再次生效。
				if err := mgr.storage.Delete(ctx, current); err != nil {
					t.Fatal(err)
				}
				if got, err := ttl(); err != nil || got != -2 {
					t.Fatalf("expired new restriction TTL = %d, %v, want -2", got, err)
				}
			})
		}
	}
}

// TestManagerDisableRevokesRefreshAfterAccessExpiry covers refresh credentials whose access mapping already expired. TestManagerDisableRevokesRefreshAfterAccessExpiry 覆盖访问映射已到期但刷新凭证仍有效的终端。
func TestManagerDisableRevokesRefreshAfterAccessExpiry(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, nil)
	pair, err := mgr.LoginWithRefreshToken(ctx, "expired-access-disable", "web")
	if err != nil {
		t.Fatal(err)
	}
	if err := mgr.storage.Delete(ctx, mgr.getTokenKey(pair.AccessToken)); err != nil {
		t.Fatal(err)
	}
	if err := mgr.Disable(ctx, pair.LoginID, time.Hour); err != nil {
		t.Fatal(err)
	}
	if mgr.storage.Exists(ctx, mgr.getTokenKey(pair.AccessToken)) || mgr.storage.Exists(ctx, mgr.getRefreshTokenKey(pair.RefreshToken)) {
		t.Fatal("Disable recreated expired access metadata or retained its refresh token")
	}
}
