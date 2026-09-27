package manager

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/listener"
)

// TestManagerDisabledTerminalRetainsExclusiveSlot prevents temporary bans from bypassing non-concurrent login. TestManagerDisabledTerminalRetainsExclusiveSlot 防止临时封禁绕过非并发登录限制。
func TestManagerDisabledTerminalRetainsExclusiveSlot(t *testing.T) {
	ctx := context.Background()
	for _, scope := range []config.ConcurrencyScope{config.ConcurrencyScopeAccount, config.ConcurrencyScopeDevice} {
		for _, ordinary := range []bool{false, true} {
			name := string(scope) + "/atomic"
			if ordinary {
				name = string(scope) + "/ordinary"
			}
			t.Run(name, func(t *testing.T) {
				mgr := newTestManager(t, func(cfg *config.Config) {
					cfg.IsConcurrent = false
					cfg.ConcurrencyScope = scope
					cfg.ReplacedLoginExitMode = config.ReplacedLoginExitModeNewDevice
				})
				if ordinary {
					mgr.storage = &managerStorageOnly{inner: newManagerTestStorage()}
				}
				id := "disabled-exclusive-slot"
				old, err := mgr.LoginWithRefreshToken(ctx, id, "web", "old")
				if err != nil {
					t.Fatal(err)
				}
				if err := mgr.DisableDeviceAndDeviceID(ctx, id, "web", "old", time.Hour); err != nil {
					t.Fatal(err)
				}

				// The same device scope blocks siblings; the account scope also blocks other device types. 设备级限制同类型终端，账号级还限制其他设备类型。
				device := "web"
				if scope == config.ConcurrencyScopeAccount {
					device = "mobile"
				}
				before := make(map[string]any)
				for _, key := range []string{mgr.getSessionKey(id), mgr.getTokenKey(old.AccessToken), mgr.getRefreshTokenKey(old.RefreshToken), mgr.getTokenRefreshKey(old.AccessToken)} {
					value, err := mgr.storage.Get(ctx, key)
					if err != nil || value == nil {
						t.Fatalf("snapshot %q = %v, %v", key, value, err)
					}
					before[key] = value
				}
				events := 0
				mgr.GetEventManager().RegisterFuncWithConfig(listener.EventAll, func(*listener.EventData) { events++ }, listener.ListenerConfig{Async: false})
				if token, err := mgr.Login(ctx, id, device, "new"); token != "" || !errors.Is(err, derror.ErrLoginLimitExceeded) {
					t.Fatalf("login during temporary ban = %q, %v, want ErrLoginLimitExceeded", token, err)
				}
				for key, value := range before {
					if got, err := mgr.storage.Get(ctx, key); err != nil || !reflect.DeepEqual(got, value) {
						t.Fatalf("rejected login changed %q: %v, %v", key, got, err)
					}
				}
				if events != 0 {
					t.Fatalf("rejected login emitted %d events", events)
				}
				if scope == config.ConcurrencyScopeDevice {
					if _, err := mgr.Login(ctx, id, "mobile", "new"); err != nil {
						t.Fatalf("unrelated device scope blocked: %v", err)
					}
				}
				if err := mgr.UntieDeviceAndDeviceID(ctx, id, "web", "old"); err != nil {
					t.Fatal(err)
				}
				if err := mgr.CheckLogin(ctx, old.AccessToken); err != nil {
					t.Fatalf("old login did not recover after unban: %v", err)
				}
				if err := mgr.Logout(ctx, old.AccessToken); err != nil {
					t.Fatal(err)
				}
				if _, err := mgr.Login(ctx, id, device, "new"); err != nil {
					t.Fatalf("logout did not release slot: %v", err)
				}
			})
		}
	}
}

// TestManagerSharedLoginEventUsesTerminalIdentity covers optional filters and disabled candidates. TestManagerSharedLoginEventUsesTerminalIdentity 覆盖共享事件的可选筛选字段与封禁候选终端。
func TestManagerSharedLoginEventUsesTerminalIdentity(t *testing.T) {
	ctx := context.Background()
	for _, args := range [][]string{nil, {"web"}, {"", "old"}, {"web", "old"}} {
		t.Run("filters="+strings.Join(args, "/"), func(t *testing.T) {
			mgr := newTestManager(t, func(cfg *config.Config) {
				cfg.IsConcurrent = true
				cfg.IsShare = true
			})
			id := "shared-event-identity"
			old, err := mgr.Login(ctx, id, "web", "old")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := mgr.Login(ctx, id, "web", "disabled-newest"); err != nil {
				t.Fatal(err)
			}
			if err := mgr.DisableDeviceAndDeviceID(ctx, id, "web", "disabled-newest", time.Hour); err != nil {
				t.Fatal(err)
			}
			var events []*listener.EventData
			mgr.GetEventManager().RegisterFuncWithConfig(listener.EventAll, func(event *listener.EventData) {
				events = append(events, event)
			}, listener.ListenerConfig{Async: false})
			token, err := mgr.Login(ctx, id, args...)
			if err != nil || token != old {
				t.Fatalf("shared login = %q, %v, want %q", token, err, old)
			}
			if len(events) != 1 {
				t.Fatalf("events = %+v, want one login event", events)
			}
			event := events[0]
			if event.Event != listener.EventLogin || event.LoginID != id || event.Token != old || event.Device != "web" || event.DeviceID != "old" || event.Extra[listener.ExtraKeyShared] != true {
				t.Fatalf("shared event identity = %+v", event)
			}
			sess, err := mgr.getSession(ctx, id)
			if err != nil || len(sess.TerminalInfos) != 2 || sess.HistoryTerminalCount != 2 {
				t.Fatalf("sharing changed terminal count/history: %+v, %v", sess, err)
			}
		})
	}
}

// TestManagerLowerLoginLimitRetiresOldestInScope checks repeated eviction, metadata cleanup and session preservation. TestManagerLowerLoginLimitRetiresOldestInScope 验证连续淘汰、元数据清理与 Session 保留。
func TestManagerLowerLoginLimitRetiresOldestInScope(t *testing.T) {
	ctx := context.Background()
	for _, scope := range []config.ConcurrencyScope{config.ConcurrencyScopeAccount, config.ConcurrencyScopeDevice} {
		for _, mode := range []struct {
			mode  config.LogoutMode
			err   error
			event listener.Event
		}{
			{config.LogoutModeLogout, derror.ErrInvalidToken, listener.EventLogout},
			{config.LogoutModeKickout, derror.ErrTokenKickout, listener.EventKickout},
			{config.LogoutModeReplaced, derror.ErrTokenReplaced, listener.EventReplace},
		} {
			t.Run(string(scope)+"/"+string(mode.mode), func(t *testing.T) {
				mgr := newTestManager(t, func(cfg *config.Config) {
					cfg.IsConcurrent = true
					cfg.IsShare = true
					cfg.ConcurrencyScope = scope
					cfg.MaxLoginCount = config.NoLimit
				})
				id := "lower-login-limit"
				var pairs []*RefreshTokenPair
				for _, device := range []string{"mobile", "web", "web", "web"} {
					pair, err := mgr.LoginWithRefreshToken(ctx, id, device, "same-id")
					if err != nil {
						t.Fatal(err)
					}
					pairs = append(pairs, pair)
				}
				if err := mgr.SetSessionValue(ctx, id, "theme", "dark"); err != nil {
					t.Fatal(err)
				}
				before, err := mgr.getSession(ctx, id)
				if err != nil {
					t.Fatal(err)
				}
				if err := mgr.DisableDevice(ctx, id, "mobile", time.Hour); err != nil {
					t.Fatal(err)
				}
				var events []*listener.EventData
				mgr.GetEventManager().RegisterFuncWithConfig(listener.EventAll, func(event *listener.EventData) {
					events = append(events, event)
				}, listener.ListenerConfig{Async: false})
				limit := int64(2)
				token, err := mgr.LoginWithOptions(ctx, LoginOptions{
					LoginID: id, Device: "web", DeviceID: "new", Token: "lower-limit-new",
					MaxLoginCount: &limit, OverflowLogoutMode: &mode.mode,
				})
				if err != nil {
					t.Fatal(err)
				}
				retired := pairs[:3]
				wantTokens := []string{pairs[3].AccessToken, token}
				if scope == config.ConcurrencyScopeDevice {
					retired = pairs[1:3]
					wantTokens = append([]string{pairs[0].AccessToken}, wantTokens...)
				}
				after, err := mgr.getSession(ctx, id)
				if err != nil {
					t.Fatal(err)
				}
				var gotTokens []string
				for _, terminal := range after.TerminalInfos {
					gotTokens = append(gotTokens, terminal.Token)
				}
				if !reflect.DeepEqual(gotTokens, wantTokens) || after.Data["theme"] != "dark" || after.CreateTime != before.CreateTime || after.HistoryTerminalCount != before.HistoryTerminalCount+1 {
					t.Fatalf("session after lowering limit = %+v, want tokens %v and preserved account data", after, wantTokens)
				}
				if len(events) != len(retired)+1 {
					t.Fatalf("events = %+v, want eviction events followed by login", events)
				}
				for i, pair := range retired {
					if _, err := mgr.getTokenRecord(ctx, pair.AccessToken); !errors.Is(err, mode.err) {
						t.Fatalf("retired token state = %v, want %v", err, mode.err)
					}
					for _, key := range []string{mgr.getRefreshTokenKey(pair.RefreshToken), mgr.getTokenRefreshKey(pair.AccessToken)} {
						if mgr.storage.Exists(ctx, key) {
							t.Fatalf("evicted terminal retained refresh binding %q", key)
						}
					}
					if events[i].Event != mode.event || events[i].Token != pair.AccessToken || events[i].Device != pair.Device {
						t.Fatalf("eviction event %d = %+v", i, events[i])
					}
				}
				if events[len(retired)].Event != listener.EventLogin || events[len(retired)].Token != token {
					t.Fatalf("last event = %+v, want new login", events[len(retired)])
				}
				if err := mgr.UntieDevice(ctx, id, "mobile"); err != nil {
					t.Fatal(err)
				}
				for _, kept := range wantTokens {
					if err := mgr.CheckLogin(ctx, kept); err != nil {
						t.Fatalf("retained token failed validation: %v", err)
					}
				}
				if scope == config.ConcurrencyScopeAccount {
					if err := mgr.CheckLogin(ctx, pairs[0].AccessToken); !errors.Is(err, mode.err) {
						t.Fatalf("unban restored an evicted token: %v", err)
					}
				}
			})
		}
	}
}
