package manager

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/listener"
	"github.com/Zany2/dtoken-go/core/utils"
)

// TestManagerLoginSharingRespectsIndependentDeviceID covers each basic login entry point. TestManagerLoginSharingRespectsIndependentDeviceID 覆盖各基础登录入口的独立设备 ID 共享筛选。
func TestManagerLoginSharingRespectsIndependentDeviceID(t *testing.T) {
	ctx := context.Background()
	for _, entry := range []struct {
		name  string
		login func(*Manager, string, string) (string, error)
	}{
		{"login", func(m *Manager, id, deviceID string) (string, error) {
			return m.Login(ctx, id, " \t", deviceID)
		}},
		{"timeout", func(m *Manager, id, deviceID string) (string, error) {
			return m.LoginWithTimeout(ctx, id, 0, " \t", deviceID)
		}},
		{"options", func(m *Manager, id, deviceID string) (string, error) {
			return m.LoginWithOptions(ctx, LoginOptions{LoginID: id, Device: " \t", DeviceID: deviceID})
		}},
	} {
		t.Run(entry.name, func(t *testing.T) {
			mgr := newTestManager(t, func(cfg *config.Config) {
				cfg.IsConcurrent = true
				cfg.IsShare = true
				cfg.MaxLoginCount = config.NoLimit
			})
			id := "independent-device-id"
			for _, opts := range []LoginOptions{
				{LoginID: id, Device: "mobile", DeviceID: "target", Token: "target-token"},
				{LoginID: id, Device: "web", DeviceID: "other", Token: "other-token"},
			} {
				if _, err := mgr.LoginWithOptions(ctx, opts); err != nil {
					t.Fatal(err)
				}
			}

			// An omitted type must not hide the ID or select the latest unrelated terminal. 缺省设备类型不能忽略 ID，也不能选择最新的不相关终端。
			if token, err := entry.login(mgr, id, " target "); err != nil || token != "target-token" {
				t.Fatalf("shared login = %q, %v, want target-token", token, err)
			}
			token, err := entry.login(mgr, id, " unseen ")
			if err != nil || token == "target-token" || token == "other-token" || token == "" {
				t.Fatalf("new device login = %q, %v, want a fresh token", token, err)
			}
			sess, info, err := mgr.checkLoginAndGetContextNoRenew(ctx, token)
			if err != nil {
				t.Fatal(err)
			}
			if info.Device != "" || info.DeviceID != "unseen" || len(sess.TerminalInfos) != 3 {
				t.Fatalf("new device identity = %+v, terminals = %+v", info, sess.TerminalInfos)
			}
		})
	}
}

// TestManagerLoginRejectsInvalidMetadataBeforeRetiringTokens protects sequential replacement and overflow flows. TestManagerLoginRejectsInvalidMetadataBeforeRetiringTokens 保护顺序执行的顶替和超限淘汰流程。
func TestManagerLoginRejectsInvalidMetadataBeforeRetiringTokens(t *testing.T) {
	ctx := context.Background()
	for _, mode := range []string{"fresh", "replacement", "overflow"} {
		for _, field := range []string{"extra", "terminal extra"} {
			t.Run(mode+"/"+field, func(t *testing.T) {
				mgr := newTestManager(t, func(cfg *config.Config) {
					cfg.IsConcurrent = mode != "replacement"
					cfg.IsShare = false
					cfg.MaxLoginCount = 1
					cfg.ConcurrencyScope = config.ConcurrencyScopeAccount
					cfg.ReplacedLoginExitMode = config.ReplacedLoginExitModeOldDevice
					cfg.ActiveTimeout = 30
					cfg.RenewInterval = 30
				})
				id := "invalid-login-metadata"
				before := make(map[string]any)
				if mode != "fresh" {
					pair, err := mgr.LoginWithRefreshToken(ctx, id, "web", "existing")
					if err != nil {
						t.Fatal(err)
					}
					for _, key := range []string{
						mgr.getSessionKey(id), mgr.getTokenKey(pair.AccessToken),
						mgr.getActiveKey(pair.AccessToken), mgr.getRenewKey(pair.AccessToken),
						mgr.getRefreshTokenKey(pair.RefreshToken), mgr.getTokenRefreshKey(pair.AccessToken),
					} {
						value, err := mgr.storage.Get(ctx, key)
						if err != nil || value == nil {
							t.Fatalf("snapshot(%q) = %v, %v", key, value, err)
						}
						before[key] = value
					}
				}
				events := 0
				mgr.GetEventManager().RegisterFuncWithConfig(listener.EventAll, func(*listener.EventData) {
					events++
				}, listener.ListenerConfig{Async: false})
				opts := LoginOptions{LoginID: id, Token: "invalid-extra-token", Device: "mobile"}
				invalid := map[string]any{"unsupported": make(chan int)}
				if field == "extra" {
					opts.Extra = invalid
				} else {
					opts.TerminalExtra = invalid
				}
				if token, err := mgr.LoginWithOptions(ctx, opts); token != "" || !errors.Is(err, derror.ErrSerializeFailed) {
					t.Fatalf("invalid metadata login = %q, %v, want ErrSerializeFailed", token, err)
				}
				for key, value := range before {
					if got, err := mgr.storage.Get(ctx, key); err != nil || !reflect.DeepEqual(got, value) {
						t.Fatalf("rejected login changed %q: %v, %v", key, got, err)
					}
				}
				if events != 0 || mgr.storage.Exists(ctx, mgr.getTokenKey(opts.Token)) {
					t.Fatalf("rejected login produced token or events: %d", events)
				}
				if mode == "fresh" && mgr.storage.Exists(ctx, mgr.getSessionKey(id)) {
					t.Fatal("rejected login left a session")
				}
			})
		}
	}
}

// TestManagerLoginValidatesCreatedSessionIdentity checks custom sessions before persistence. TestManagerLoginValidatesCreatedSessionIdentity 验证自定义 Session 在持久化前校验并补齐身份。
func TestManagerLoginValidatesCreatedSessionIdentity(t *testing.T) {
	ctx := context.Background()
	for _, identity := range []string{"empty", "matching", "foreign auth type", "foreign login ID"} {
		t.Run(identity, func(t *testing.T) {
			mgr := newTestManagerWithStrategy(t, &Strategy{
				CreateSession: func(authType, loginID string, createTime int64) *Session {
					sess := defaultCreateSession(authType, loginID, createTime)
					sess.Data["custom"] = "preserved"
					switch identity {
					case "empty":
						sess.AuthType, sess.LoginID = "", ""
					case "foreign auth type":
						sess.AuthType = authType + "foreign"
					case "foreign login ID":
						sess.LoginID = loginID + "foreign"
					}
					return sess
				},
			})
			opts := LoginOptions{LoginID: "custom-session-identity", Token: "custom-session-token"}
			token, err := mgr.LoginWithOptions(ctx, opts)
			if identity == "foreign auth type" || identity == "foreign login ID" {
				if token != "" || !errors.Is(err, derror.ErrInvalidParam) {
					t.Fatalf("foreign identity login = %q, %v, want ErrInvalidParam", token, err)
				}
				if mgr.storage.Exists(ctx, mgr.getSessionKey(opts.LoginID)) || mgr.storage.Exists(ctx, mgr.getTokenKey(opts.Token)) {
					t.Fatal("foreign session identity was persisted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}

			// Inspect persisted fields directly so read-time normalization cannot mask a broken write. 直接检查持久化字段，避免读取时规范化掩盖写入错误。
			value, err := mgr.storage.Get(ctx, mgr.getSessionKey(opts.LoginID))
			if err != nil {
				t.Fatal(err)
			}
			data, err := utils.ToBytes(value)
			if err != nil {
				t.Fatal(err)
			}
			var sess Session
			if err := mgr.serializer.Decode(data, &sess); err != nil {
				t.Fatal(err)
			}
			if sess.AuthType != mgr.config.AuthType || sess.LoginID != opts.LoginID || sess.Data["custom"] != "preserved" {
				t.Fatalf("persisted session = %+v, want canonical identity and custom data", sess)
			}
			if _, _, err := mgr.checkLoginAndGetContextNoRenew(ctx, token); err != nil {
				t.Fatalf("new token failed session validation: %v", err)
			}
		})
	}
}
