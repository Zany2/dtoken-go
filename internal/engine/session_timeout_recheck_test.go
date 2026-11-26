package engine

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/config"
	derror "github.com/Zany2/dtoken-go/errors"
	listener "github.com/Zany2/dtoken-go/event"
)

// TestManagerTimeoutRecheckPreservesCompletedActivityRefresh exercises maintenance completing after an expired snapshot was read. TestManagerTimeoutRecheckPreservesCompletedActivityRefresh 覆盖读取超时快照后维护任务完成活跃刷新的交错。
func TestManagerTimeoutRecheckPreservesCompletedActivityRefresh(t *testing.T) {
	ctx := context.Background()
	for _, allowRenew := range []bool{false, true} {
		name := "without maintenance"
		if allowRenew {
			name = "with maintenance"
		}
		t.Run(name, func(t *testing.T) {
			mgr := newTestManager(t, func(cfg *config.Config) { cfg.ActiveTimeout = 60 })
			pair, err := mgr.LoginWithRefreshToken(ctx, "timeout-recheck", "web", "browser")
			if err != nil {
				t.Fatal(err)
			}
			pool := &managerQueuedMaintenancePool{}
			mgr.pool = pool
			t.Cleanup(pool.runAll)
			record, err := mgr.getTokenRecord(ctx, pair.AccessToken)
			if err != nil {
				t.Fatal(err)
			}

			// Start with an expired snapshot; another request publishes its accepted activity during the session read. 从超时快照开始，另一请求在读取 Session 期间提交已接受的活跃时间。
			if err := mgr.storage.Set(ctx, mgr.getActiveKey(pair.AccessToken), time.Now().Add(-2*time.Minute).Unix(), time.Hour); err != nil {
				t.Fatal(err)
			}
			inline := &managerAccessInlinePool{mgr: mgr, loginID: pair.LoginID}
			storage := &managerAfterSessionReadStorage{Storage: mgr.storage, sessionKey: mgr.getSessionKey(pair.LoginID)}
			mgr.storage = storage
			var before terminalLifecycleSnapshot
			storage.afterRead = func() {
				mgr.submitLoginMaintenance(ctx, pair.AccessToken, record, 60)
				pool.runAll()
				if err := mgr.SetSessionValue(ctx, pair.LoginID, "revision", "fresh"); err != nil {
					t.Fatal(err)
				}
				before = captureTerminalLifecycle(t, mgr, ctx, terminalLifecycleKeys(mgr, pair.AccessToken, pair.RefreshToken, storage.sessionKey))
				mgr.pool = inline
			}
			events := registerTerminalLifecycleEvents(mgr)
			sess, info, err := mgr.checkLoginAndGetContextWithOptions(ctx, pair.AccessToken, checkLoginOptions{allowRenew: allowRenew})
			if err != nil || sess == nil || info == nil {
				t.Fatalf("refreshed validation = %+v, %+v, %v, want valid context", sess, info, err)
			}
			if storage.afterRead != nil || sess.Data["revision"] != "fresh" || info.LoginID != pair.LoginID {
				t.Fatalf("validation did not return the reloaded context: %+v, %+v", sess, info)
			}
			wantSubmissions := 0
			if allowRenew {
				wantSubmissions = 1
			}
			if inline.locked || inline.executed != wantSubmissions {
				t.Fatalf("maintenance locked/executed = %v/%d, want false/%d", inline.locked, inline.executed, wantSubmissions)
			}

			// A successful check may advance activity; credentials and the session must remain unchanged. 成功校验可推进活跃时间，但凭据和会话必须保持不变。
			if allowRenew {
				delete(before.values, mgr.getActiveKey(pair.AccessToken))
			}
			assertTerminalLifecycleUnchanged(t, mgr, ctx, before)
			if result, err := mgr.IntrospectToken(ctx, pair.AccessToken); err != nil || result == nil || !result.Active {
				t.Fatalf("refreshed introspection = %+v, %v, want active", result, err)
			}
			if len(events()) != 0 {
				t.Fatalf("refreshed login emitted lifecycle events: %+v", events())
			}
		})
	}
}

// TestManagerTimeoutRecheckPropagatesCurrentState prevents old timeout decisions from masking replacement, disable or storage failures. TestManagerTimeoutRecheckPropagatesCurrentState 防止旧超时判定掩盖生命周期替换、封禁或存储故障。
func TestManagerTimeoutRecheckPropagatesCurrentState(t *testing.T) {
	ctx := context.Background()
	for _, state := range []struct {
		name string
		err  error
	}{
		{"replacement", derror.ErrInvalidToken},
		{"disabled device", derror.ErrDeviceDisabled},
		{"missing activity", derror.ErrInvalidToken},
		{"activity read failure", derror.ErrStorageUnavailable},
		{"missing session", derror.ErrInvalidToken},
	} {
		t.Run(state.name, func(t *testing.T) {
			mgr := newTestManager(t, func(cfg *config.Config) { cfg.ActiveTimeout = 60 })
			pair, err := mgr.LoginWithRefreshToken(ctx, "timeout-current-state", "web", "browser")
			if err != nil {
				t.Fatal(err)
			}
			if err := mgr.storage.Set(ctx, mgr.getActiveKey(pair.AccessToken), time.Now().Add(-2*time.Minute).Unix(), time.Hour); err != nil {
				t.Fatal(err)
			}
			storage := &managerAfterSessionReadStorage{Storage: mgr.storage, sessionKey: mgr.getSessionKey(pair.LoginID)}
			mgr.storage = storage
			keys := terminalLifecycleKeys(mgr, pair.AccessToken, pair.RefreshToken, storage.sessionKey)
			var before terminalLifecycleSnapshot
			var events func() []*listener.EventData
			storage.afterRead = func() {
				switch state.name {
				case "replacement":
					if err := mgr.Logout(ctx, pair.AccessToken); err != nil {
						t.Fatal(err)
					}
					// Simulate expiry of the retained logout marker before a new login reuses the token value. 模拟保留的注销标记过期后，新登录复用相同 Token 值。
					if err := mgr.storage.Delete(ctx, mgr.getTokenKey(pair.AccessToken)); err != nil {
						t.Fatal(err)
					}
					current, err := mgr.LoginWithRefreshTokenOptions(ctx, RefreshTokenOptions{LoginOptions: LoginOptions{
						LoginID: pair.LoginID, Token: pair.AccessToken, Device: "web", DeviceID: "browser",
					}})
					if err != nil {
						t.Fatal(err)
					}
					keys = append(keys, mgr.getRefreshTokenKey(current.RefreshToken))
				case "disabled device":
					if err := mgr.DisableDevice(ctx, pair.LoginID, "web", time.Hour); err != nil {
						t.Fatal(err)
					}
				case "missing activity":
					if err := mgr.storage.Delete(ctx, mgr.getActiveKey(pair.AccessToken)); err != nil {
						t.Fatal(err)
					}
				case "missing session":
					if err := mgr.storage.Delete(ctx, storage.sessionKey); err != nil {
						t.Fatal(err)
					}
				}
				before = captureTerminalLifecycle(t, mgr, ctx, keys)
				events = registerTerminalLifecycleEvents(mgr)
				if state.name == "activity read failure" {
					mgr.storage = &managerSessionReadErrorStorage{Storage: storage, key: mgr.getActiveKey(pair.AccessToken)}
				}
			}
			sess, info, err := mgr.checkLoginAndGetContextWithOptions(ctx, pair.AccessToken, checkLoginOptions{checkDisable: true})
			if !errors.Is(err, state.err) || sess != nil || info != nil {
				t.Fatalf("rechecked context = %+v, %+v, %v, want nil context and %v", sess, info, err, state.err)
			}
			if storage.afterRead != nil {
				t.Fatal("state change hook was not reached")
			}
			mgr.storage = storage.Storage
			assertTerminalLifecycleUnchanged(t, mgr, ctx, before)
			if len(events()) != 0 {
				t.Fatalf("rejected stale check emitted lifecycle events: %+v", events())
			}
		})
	}
}

// TestManagerConfirmedTimeoutRetiresOnlyItsTerminal verifies confirmed timeouts still clean credentials and emit events after unlocking. TestManagerConfirmedTimeoutRetiresOnlyItsTerminal 验证确认超时后仍清理凭据并在解锁后发布事件。
func TestManagerConfirmedTimeoutRetiresOnlyItsTerminal(t *testing.T) {
	ctx := context.Background()
	for _, layout := range []string{"last terminal", "other terminal", "detached terminal"} {
		t.Run(layout, func(t *testing.T) {
			mgr := newTestManager(t, func(cfg *config.Config) { cfg.ActiveTimeout = 60 })
			pair, err := mgr.LoginWithRefreshToken(ctx, "confirmed-timeout", "web", "browser")
			if err != nil {
				t.Fatal(err)
			}
			if err := mgr.SetSessionValue(ctx, pair.LoginID, "theme", "dark"); err != nil {
				t.Fatal(err)
			}
			if layout == "other terminal" {
				if _, err := mgr.Login(ctx, pair.LoginID, "mobile"); err != nil {
					t.Fatal(err)
				}
			} else if layout == "detached terminal" {
				sess, err := mgr.getSession(ctx, pair.LoginID)
				if err != nil {
					t.Fatal(err)
				}
				sessionRemoveTerminalByToken(sess, pair.AccessToken)
				if err := mgr.saveToStorage(ctx, mgr.getSessionKey(pair.LoginID), *sess); err != nil {
					t.Fatal(err)
				}
			}
			if err := mgr.storage.Set(ctx, mgr.getActiveKey(pair.AccessToken), time.Now().Add(-2*time.Minute).Unix(), time.Hour); err != nil {
				t.Fatal(err)
			}
			var events []listener.Event
			mgr.GetEventManager().RegisterFuncWithConfig(listener.EventAll, func(data *listener.EventData) {
				mgr.loginLocksMu.Lock()
				entry := mgr.loginLocks[pair.LoginID]
				if entry != nil {
					if entry.mu.TryLock() {
						entry.mu.Unlock()
					} else {
						t.Error("timeout event emitted while account lock is held")
					}
				}
				mgr.loginLocksMu.Unlock()
				events = append(events, data.Event)
			}, listener.ListenerConfig{Async: false})
			for i := 0; i < 2; i++ {
				if _, _, err := mgr.checkLoginAndGetContextNoRenew(ctx, pair.AccessToken); !errors.Is(err, derror.ErrActiveTimeout) {
					t.Fatalf("timeout check %d = %v, want ErrActiveTimeout", i, err)
				}
			}
			wantEvents := []listener.Event{listener.EventActiveTimeout}
			if layout == "last terminal" {
				wantEvents = append([]listener.Event{listener.EventDestroySession}, wantEvents...)
			}
			if !reflect.DeepEqual(events, wantEvents) {
				t.Fatalf("events = %v, want %v", events, wantEvents)
			}
			for _, key := range []string{mgr.getActiveKey(pair.AccessToken), mgr.getRenewKey(pair.AccessToken), mgr.getTokenRefreshKey(pair.AccessToken), mgr.getRefreshTokenKey(pair.RefreshToken)} {
				if mgr.storage.Exists(ctx, key) {
					t.Fatalf("confirmed timeout retained metadata %q", key)
				}
			}
			sess, err := mgr.getSession(ctx, pair.LoginID)
			if layout == "last terminal" {
				if !errors.Is(err, derror.ErrSessionNotFound) {
					t.Fatalf("last-terminal session = %+v, %v, want missing", sess, err)
				}
			} else {
				wantCount := 0
				if layout == "other terminal" {
					wantCount = 1
				}
				if err != nil || sess == nil || sess.Data["theme"] != "dark" || sessionHasTerminalToken(sess, pair.AccessToken) || len(sess.TerminalInfos) != wantCount {
					t.Fatalf("remaining session = %+v, %v, want preserved data and %d terminals", sess, err, wantCount)
				}
			}
		})
	}
}
