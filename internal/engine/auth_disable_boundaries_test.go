package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	base64codec "github.com/Zany2/dtoken-go/codec/base64"
	jsoncodec "github.com/Zany2/dtoken-go/codec/json"
	"github.com/Zany2/dtoken-go/config"
	adapter "github.com/Zany2/dtoken-go/contract"
	derror "github.com/Zany2/dtoken-go/errors"
)

// TestAccountRetirementPreservesReusedExpiredTokenOwner verifies all retirement paths preserve foreign refresh credentials after access expiry. TestAccountRetirementPreservesReusedExpiredTokenOwner 验证各类下线路径在访问 Token 过期后仍保留其他账号的刷新凭据。
func TestAccountRetirementPreservesReusedExpiredTokenOwner(t *testing.T) {
	ctx := context.Background()
	for _, action := range []TerminateAction{"disable", TerminateActionLogout, TerminateActionKickout, TerminateActionReplace} {
		for _, scope := range []string{"account", "device", "concrete device"} {
			if action == "disable" && scope != "account" {
				continue
			}
			for _, storageMode := range []string{"atomic", "basic"} {
				for _, format := range []string{"current", "legacy"} {
					t.Run(string(action)+"/"+scope+"/"+storageMode+"/"+format, func(t *testing.T) {
						mgr := newTestManager(t, func(cfg *config.Config) {
							cfg.Timeout = 3600
							cfg.IsConcurrent = true
							cfg.IsShare = false
						})
						if storageMode == "basic" {
							mgr.storage = nonAtomicManagerStorage{Storage: mgr.storage}
						}

						// Keep the old account session alive beyond its short-lived terminal. 保持旧账号会话的有效期长于其短期终端。
						if _, err := mgr.Login(ctx, "old-owner", "mobile"); err != nil {
							t.Fatal(err)
						}
						opts := LoginOptions{LoginID: "old-owner", Token: "reused-disable-token", Device: "web", DeviceID: "browser", Timeout: time.Minute}
						if _, err := mgr.LoginWithOptions(ctx, opts); err != nil {
							t.Fatal(err)
						}

						// Remove only expired access data, retaining the old session's terminal entry. 仅移除过期访问数据，保留旧会话的终端条目。
						if err := mgr.storage.Delete(ctx, mgr.getTokenKey(opts.Token), mgr.getActiveKey(opts.Token), mgr.getRenewKey(opts.Token)); err != nil {
							t.Fatal(err)
						}
						opts.LoginID = "new-owner"
						pair, err := mgr.LoginWithRefreshTokenOptions(ctx, RefreshTokenOptions{LoginOptions: opts, RefreshTimeout: time.Hour})
						if err != nil {
							t.Fatal(err)
						}
						if format == "legacy" {
							info, err := mgr.getRefreshTokenInfo(ctx, pair.RefreshToken)
							if err != nil {
								t.Fatal(err)
							}
							if err := mgr.saveToStorage(ctx, mgr.getRefreshTokenKey(pair.RefreshToken), info.RefreshTokenInfo, time.Hour); err != nil {
								t.Fatal(err)
							}
						}
						if err := mgr.storage.Delete(ctx, mgr.getTokenKey(opts.Token), mgr.getActiveKey(opts.Token), mgr.getRenewKey(opts.Token)); err != nil {
							t.Fatal(err)
						}
						before := captureTerminalLifecycle(t, mgr, ctx, []string{
							mgr.getSessionKey(pair.LoginID), mgr.getTokenKey(opts.Token),
							mgr.getTokenRefreshKey(opts.Token), mgr.getRefreshTokenKey(pair.RefreshToken),
						})
						events := registerTerminalLifecycleEvents(mgr)
						if action == "disable" {
							err = mgr.Disable(ctx, "old-owner", time.Hour)
						} else {
							termination := TerminateOptions{Action: action, LoginID: "old-owner"}
							if scope != "account" {
								termination.Device = "web"
							}
							if scope == "concrete device" {
								termination.DeviceID = "browser"
							}
							err = mgr.Terminate(ctx, termination)
						}
						if err != nil {
							t.Fatal(err)
						}
						assertTerminalLifecycleUnchanged(t, mgr, ctx, before)
						assertNoTerminalRetirementEvent(t, events(), opts.Token)
						if scope == "account" {
							if mgr.storage.Exists(ctx, mgr.getSessionKey("old-owner")) {
								t.Fatal("retired account session was retained")
							}
						} else {
							sess, err := mgr.getSession(ctx, "old-owner")
							if err != nil || len(sess.TerminalInfos) != 1 || sess.TerminalInfos[0].Device != "mobile" {
								t.Fatalf("scoped retirement did not preserve only the unrelated terminal: %+v, %v", sess, err)
							}
						}
						fresh, err := mgr.RefreshToken(ctx, pair.RefreshToken)
						if err != nil {
							t.Fatalf("foreign refresh was revoked: %v", err)
						}
						if fresh.LoginID != pair.LoginID {
							t.Fatalf("refreshed owner = %q, want %q", fresh.LoginID, pair.LoginID)
						}
					})
				}
			}
		}
	}
}

// TestScopedRetirementRevokesOwnExpiredRefresh verifies ownership protection still allows revoking the selected account's expired access lifecycle. TestScopedRetirementRevokesOwnExpiredRefresh 验证归属保护不影响撤销目标账号已过期访问生命周期的刷新凭据。
func TestScopedRetirementRevokesOwnExpiredRefresh(t *testing.T) {
	ctx := context.Background()
	for _, action := range []TerminateAction{TerminateActionLogout, TerminateActionKickout, TerminateActionReplace} {
		for _, scope := range []string{"account", "device", "concrete device"} {
			for _, storageMode := range []string{"atomic", "basic"} {
				t.Run(string(action)+"/"+scope+"/"+storageMode, func(t *testing.T) {
					mgr := newTestManager(t, nil)
					if storageMode == "basic" {
						mgr.storage = nonAtomicManagerStorage{Storage: mgr.storage}
					}
					pair, err := mgr.LoginWithRefreshToken(ctx, "expired-owner", "web", "browser")
					if err != nil {
						t.Fatal(err)
					}
					if err := mgr.storage.Delete(ctx, mgr.getTokenKey(pair.AccessToken)); err != nil {
						t.Fatal(err)
					}
					opts := TerminateOptions{Action: action, LoginID: pair.LoginID}
					if scope != "account" {
						opts.Device = "web"
					}
					if scope == "concrete device" {
						opts.DeviceID = "browser"
					}
					if err := mgr.Terminate(ctx, opts); err != nil {
						t.Fatal(err)
					}
					for _, key := range []string{mgr.getSessionKey(pair.LoginID), mgr.getTokenKey(pair.AccessToken), mgr.getTokenRefreshKey(pair.AccessToken), mgr.getRefreshTokenKey(pair.RefreshToken)} {
						if mgr.storage.Exists(ctx, key) {
							t.Fatalf("retirement retained %q", key)
						}
					}
					if _, err := mgr.RefreshToken(ctx, pair.RefreshToken); !errors.Is(err, derror.ErrInvalidRefreshToken) {
						t.Fatalf("retired refresh = %v, want ErrInvalidRefreshToken", err)
					}
				})
			}
		}
	}
}

// TestDisableRejectsInvalidMarkers verifies malformed or unidentified markers fail without mutations or events. TestDisableRejectsInvalidMarkers 验证损坏或缺失身份的封禁记录返回错误，且不修改存储或发送事件。
func TestDisableRejectsInvalidMarkers(t *testing.T) {
	ctx := context.Background()
	for _, codec := range []adapter.Codec{jsoncodec.NewJSONSerializer(), base64codec.NewBase64Serializer()} {
		for _, kind := range []string{"account", "service", "device", "concrete device"} {
			for _, format := range []string{"null", "ownerless", "missing kind"} {
				t.Run(codec.Name()+"/"+kind+"/"+format, func(t *testing.T) {
					mgr := newTestManager(t, nil)
					mgr.serializer = codec
					id := "null:owner"
					current, legacy := mgr.getDisableKey(id), mgr.getLegacyDisableKey(id)
					marker := disableMarker{LoginID: id, Kind: "account"}
					get := func() error { _, err := mgr.GetDisableInfo(ctx, id); return err }
					check := func() error { return mgr.CheckDisable(ctx, id) }
					ttl := func() error { _, err := mgr.GetDisableTTL(ctx, id); return err }
					disable := func() error { return mgr.Disable(ctx, id, time.Minute) }
					untie := func() error { return mgr.Untie(ctx, id) }
					switch kind {
					case "service":
						current, legacy = mgr.getDisableServiceKey(id, "pay"), mgr.getLegacyDisableServiceKey(id, "pay")
						marker.Kind, marker.Service, marker.Level = "service", "pay", 2
						get = func() error { _, err := mgr.GetDisableServiceInfo(ctx, id, "pay"); return err }
						check = func() error { return mgr.CheckDisableService(ctx, id, "pay") }
						ttl = func() error { _, err := mgr.GetDisableServiceTTL(ctx, id, "pay"); return err }
						disable = func() error { return mgr.DisableServiceLevel(ctx, id, "pay", 2, time.Minute) }
						untie = func() error { return mgr.UntieService(ctx, id, "pay") }
					case "device":
						current, legacy = mgr.getDisableDeviceKey(id, "web"), mgr.getLegacyDisableDeviceKey(id, "web")
						marker.Kind, marker.Device = "device", "web"
						get = func() error { _, err := mgr.GetDisableDeviceInfo(ctx, id, "web"); return err }
						check = func() error { return mgr.CheckDisableDevice(ctx, id, "web") }
						ttl = func() error { _, err := mgr.GetDisableDeviceTTL(ctx, id, "web"); return err }
						disable = func() error { return mgr.DisableDevice(ctx, id, "web", time.Minute) }
						untie = func() error { return mgr.UntieDevice(ctx, id, "web") }
					case "concrete device":
						current, legacy = mgr.getDisableDeviceAndDeviceIDKey(id, "web", "phone"), mgr.getLegacyDisableDeviceAndDeviceIDKey(id, "web", "phone")
						marker.Kind, marker.Device, marker.DeviceID = "device", "web", "phone"
						get = func() error { _, err := mgr.GetDisableDeviceAndDeviceIDInfo(ctx, id, "web", "phone"); return err }
						check = func() error { return mgr.CheckDisableDeviceAndDeviceID(ctx, id, "web", "phone") }
						ttl = func() error { _, err := mgr.GetDisableDeviceAndDeviceIDTTL(ctx, id, "web", "phone"); return err }
						disable = func() error { return mgr.DisableDeviceAndDeviceID(ctx, id, "web", "phone", time.Minute) }
						untie = func() error { return mgr.UntieDeviceAndDeviceID(ctx, id, "web", "phone") }
					}
					key := current

					// Canonical objects must decode before invalidating their payload. 规范记录须可解码，再改写为无效载荷。
					if err := mgr.saveToStorage(ctx, key, marker, time.Hour); err != nil {
						t.Fatal(err)
					}
					if err := get(); err != nil {
						t.Fatalf("valid marker rejected: %v", err)
					}
					wantErr := derror.ErrSerializeFailed
					var invalid any = (*disableMarker)(nil)
					if format != "null" {
						wantErr = derror.ErrInvalidParam
						if format == "ownerless" {
							marker.LoginID = ""
						} else {
							marker.Kind = ""
						}
						invalid = marker
					}
					if err := mgr.saveToStorage(ctx, key, invalid, time.Hour); err != nil {
						t.Fatal(err)
					}
					before := captureTerminalLifecycle(t, mgr, ctx, []string{current, legacy})
					events := registerTerminalLifecycleEvents(mgr)
					operations := []struct {
						name string
						call func() error
					}{{"get", get}, {"check", check}, {"ttl", ttl}, {"disable", disable}, {"untie", untie}}
					if kind == "service" {
						if err := mgr.CheckDisableServiceLevel(ctx, id, "pay", 2); !errors.Is(err, wantErr) {
							t.Fatalf("null service level check = %v", err)
						}
					}
					for _, operation := range operations {
						if err := operation.call(); !errors.Is(err, wantErr) {
							t.Errorf("%s(%s) = %v, want %v", operation.name, format, err, wantErr)
						}
						assertTerminalLifecycleUnchanged(t, mgr, ctx, before)
					}
					if len(events()) != 0 {
						t.Fatal("null marker emitted success events")
					}
				})
			}
		}
	}
}
