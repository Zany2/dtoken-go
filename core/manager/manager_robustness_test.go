package manager

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/listener"
)

// TestManagerSharedLoginDoesNotRequireGenerator verifies sharing succeeds even when a generator is unavailable. TestManagerSharedLoginDoesNotRequireGenerator 验证生成器不可用时仍可成功共享 Token。
func TestManagerSharedLoginDoesNotRequireGenerator(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.IsConcurrent = true
		cfg.IsShare = true
	})

	first, err := mgr.Login(ctx, "shared-generator", "web", "browser")
	if err != nil {
		t.Fatalf("first Login() error = %v", err)
	}
	mgr.generator = managerErrorGenerator{}

	second, err := mgr.Login(ctx, "shared-generator", "web", "browser")
	if err != nil {
		t.Fatalf("shared Login() error = %v, want generator-independent success", err)
	}
	if second != first {
		t.Fatalf("shared token = %q, want original token %q", second, first)
	}
}

// TestManagerExplicitTokenBypassesSharing verifies explicit tokens always create their own terminal. TestManagerExplicitTokenBypassesSharing 验证显式 Token 始终创建独立终端。
func TestManagerExplicitTokenBypassesSharing(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.IsConcurrent = true
		cfg.IsShare = true
	})

	first, err := mgr.Login(ctx, "explicit-share", "web", "browser")
	if err != nil {
		t.Fatalf("first Login() error = %v", err)
	}
	second, err := mgr.LoginWithOptions(ctx, LoginOptions{
		LoginID:  "explicit-share",
		Device:   "web",
		DeviceID: "browser",
		Token:    "explicit-token",
	})
	if err != nil {
		t.Fatalf("explicit LoginWithOptions() error = %v", err)
	}
	if second != "explicit-token" || second == first {
		t.Fatalf("explicit token = %q, want explicit-token distinct from %q", second, first)
	}
	terminals, err := mgr.GetTerminalListByLoginID(ctx, "explicit-share")
	if err != nil {
		t.Fatalf("GetTerminalListByLoginID() error = %v", err)
	}
	if len(terminals) != 2 {
		t.Fatalf("terminal count = %d, want 2", len(terminals))
	}
}

// TestManagerGeneratorFailureLeavesExistingSessionUntouched verifies generator errors happen before destructive concurrency work. TestManagerGeneratorFailureLeavesExistingSessionUntouched 验证生成器失败发生在并发淘汰之前，旧会话保持不变。
func TestManagerGeneratorFailureLeavesExistingSessionUntouched(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.IsConcurrent = false
		cfg.IsShare = false
		cfg.ReplacedLoginExitMode = config.ReplacedLoginExitModeOldDevice
	})

	oldToken, err := mgr.Login(ctx, "generator-before-replace", "web")
	if err != nil {
		t.Fatalf("first Login() error = %v", err)
	}
	mgr.generator = managerErrorGenerator{}
	if _, err = mgr.Login(ctx, "generator-before-replace", "mobile"); !errors.Is(err, errManagerGenerator) {
		t.Fatalf("failed Login() error = %v, want generator error", err)
	}
	if err = mgr.CheckLogin(ctx, oldToken); err != nil {
		t.Fatalf("old token CheckLogin() error = %v, want old session preserved", err)
	}
}

// TestManagerTemporaryDeviceDisablePreservesToken verifies temporary device disable does not destroy recoverable tokens. TestManagerTemporaryDeviceDisablePreservesToken 验证临时设备封禁不会销毁可恢复 Token。
func TestManagerTemporaryDeviceDisablePreservesToken(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.IsConcurrent = true
		cfg.IsShare = false
	})

	token, err := mgr.Login(ctx, "temporary-device-disable", "web", "browser")
	if err != nil {
		t.Fatalf("first Login() error = %v", err)
	}
	if err = mgr.DisableDevice(ctx, "temporary-device-disable", "web", time.Minute); err != nil {
		t.Fatalf("DisableDevice() error = %v", err)
	}
	if err = mgr.CheckLogin(ctx, token); !errors.Is(err, derror.ErrDeviceDisabled) {
		t.Fatalf("CheckLogin(disabled device) error = %v, want ErrDeviceDisabled", err)
	}

	// A login on another device exercises session cleanup while the original device is disabled. 在另一设备登录以覆盖设备封禁期间的会话清理路径。
	if _, err = mgr.Login(ctx, "temporary-device-disable", "mobile", "phone"); err != nil {
		t.Fatalf("mobile Login() during device disable error = %v", err)
	}
	if err = mgr.UntieDevice(ctx, "temporary-device-disable", "web"); err != nil {
		t.Fatalf("UntieDevice() error = %v", err)
	}
	if err = mgr.CheckLogin(ctx, token); err != nil {
		t.Fatalf("original token CheckLogin(after untie) error = %v, want nil", err)
	}
}

// TestManagerCompositeDisableKeysAreInjective verifies separators in key components cannot collide. TestManagerCompositeDisableKeysAreInjective 验证复合键组件中的分隔符不会造成碰撞。
func TestManagerCompositeDisableKeysAreInjective(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, nil)
	keys := []string{
		mgr.getDisableServiceKey("user:one", "billing"),
		mgr.getDisableServiceKey("user", "one:billing"),
		mgr.getDisableServiceKey(`user\\one`, "billing"),
		mgr.getDisableServiceKey("user", `one\\billing`),
		mgr.getDisableDeviceKey("user:one", "web"),
		mgr.getDisableDeviceKey("user", "one:web"),
		mgr.getDisableDeviceAndDeviceIDKey("user:one", "web", "phone"),
		mgr.getDisableDeviceAndDeviceIDKey("user", "one:web", "phone"),
		mgr.getDisableDeviceAndDeviceIDKey("user", "web", "phone:one"),
	}
	seen := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		if _, ok := seen[key]; ok {
			t.Fatalf("composite key collision: %q", key)
		}
		seen[key] = struct{}{}
	}
	if got := mgr.getDisableServiceKey("plain-user", "billing"); got != mgr.config.KeyPrefix+mgr.config.AuthType+DisableServiceKeyPrefix+"plain-user:billing" {
		t.Fatalf("ordinary composite key = %q, want stable unescaped form", got)
	}
	legacyInfo := ServiceDisableInfo{Service: "billing", Level: 1}
	if err := mgr.saveToStorage(ctx, mgr.getLegacyDisableServiceKey("legacy:user", "billing"), legacyInfo, time.Minute); err != nil {
		t.Fatalf("save legacy service key error = %v", err)
	}
	if info, err := mgr.GetDisableServiceInfo(ctx, "legacy:user", "billing"); err != nil || info.Level != 1 {
		t.Fatalf("GetDisableServiceInfo(legacy key) = %+v, %v, want legacy record", info, err)
	}
	if err := mgr.UntieService(ctx, "legacy:user", "billing"); err != nil {
		t.Fatalf("UntieService(legacy key) error = %v", err)
	}
	if mgr.storage.Exists(ctx, mgr.getLegacyDisableServiceKey("legacy:user", "billing")) {
		t.Fatal("legacy service key still exists after untie")
	}
}

// TestManagerSearchPaginationSortsUnorderedScanners verifies pagination is stable for unordered storage scanners. TestManagerSearchPaginationSortsUnorderedScanners 验证无序扫描存储的分页结果稳定。
func TestManagerSearchPaginationSortsUnorderedScanners(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, nil)
	base := newManagerTestStorage()
	mgr.storage = &managerUnorderedScannerStorage{managerTestStorage: base}
	for _, key := range []string{"items/c", "items/a", "items/b"} {
		if err := mgr.storage.Set(ctx, key, key, 0); err != nil {
			t.Fatalf("Set(%s) error = %v", key, err)
		}
	}

	keys, err := mgr.searchKeys(ctx, "items/*", 0, 2)
	if err != nil {
		t.Fatalf("searchKeys() error = %v", err)
	}
	if !reflect.DeepEqual(keys, []string{"items/a", "items/b"}) {
		t.Fatalf("searchKeys() = %v, want sorted first page", keys)
	}
}

// TestManagerSearchPaginationRejectsIntegerOverflow verifies extreme pagination values do not panic or wrap. TestManagerSearchPaginationRejectsIntegerOverflow 验证极大分页参数不会溢出或触发 panic。
func TestManagerSearchPaginationRejectsIntegerOverflow(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, nil)
	if err := mgr.storage.Set(ctx, "overflow/key", "value", 0); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	maxInt := int(^uint(0) >> 1)
	keys, err := mgr.searchKeys(ctx, "overflow/*", maxInt, maxInt)
	if err != nil {
		t.Fatalf("searchKeys(extreme start) error = %v", err)
	}
	if len(keys) != 0 {
		t.Fatalf("searchKeys(extreme start) = %v, want empty", keys)
	}
	keys, err = mgr.searchKeys(ctx, "overflow/*", 0, maxInt)
	if err != nil || !reflect.DeepEqual(keys, []string{"overflow/key"}) {
		t.Fatalf("searchKeys(extreme size) = %v, %v, want one key", keys, err)
	}
}

// TestManagerConcurrencyLifecycleEvents verifies automatic terminal events match the applied concurrency action. TestManagerConcurrencyLifecycleEvents 验证自动并发终端事件与实际处理动作一致。
func TestManagerConcurrencyLifecycleEvents(t *testing.T) {
	ctx := context.Background()
	for _, tt := range []struct {
		name      string
		mutate    func(*config.Config)
		wantEvent listener.Event
	}{
		{
			name: "overflow kickout",
			mutate: func(cfg *config.Config) {
				cfg.IsConcurrent = true
				cfg.IsShare = false
				cfg.MaxLoginCount = 1
				cfg.OverflowLogoutMode = config.LogoutModeKickout
			},
			wantEvent: listener.EventKickout,
		},
		{
			name: "non concurrent replace",
			mutate: func(cfg *config.Config) {
				cfg.IsConcurrent = false
				cfg.IsShare = false
				cfg.ReplacedLoginExitMode = config.ReplacedLoginExitModeOldDevice
			},
			wantEvent: listener.EventReplace,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			mgr := newTestManager(t, tt.mutate)
			var events []*listener.EventData
			mgr.GetEventManager().RegisterFuncWithConfig(listener.EventAll, func(data *listener.EventData) {
				copyData := *data
				events = append(events, &copyData)
			}, listener.ListenerConfig{Async: false})

			oldToken, err := mgr.Login(ctx, "lifecycle-events-"+tt.name, "web", "old")
			if err != nil {
				t.Fatalf("first Login() error = %v", err)
			}
			newToken, err := mgr.Login(ctx, "lifecycle-events-"+tt.name, "mobile", "new")
			if err != nil {
				t.Fatalf("second Login() error = %v", err)
			}
			if newToken == "" {
				t.Fatal("second Login() returned empty token")
			}
			assertManagerEvent(t, events, tt.wantEvent, "lifecycle-events-"+tt.name, "web", "old", oldToken, nil)
		})
	}
}

// TestManagerRenewFailureDoesNotEmitSuccessEvent verifies failed renewal never emits EventRenew. TestManagerRenewFailureDoesNotEmitSuccessEvent 验证续期失败时不会发送成功续期事件。
func TestManagerRenewFailureDoesNotEmitSuccessEvent(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.Timeout = 30
		cfg.RenewInterval = 5
		cfg.AutoRenew = false
	})
	token, err := mgr.Login(ctx, "renew-event-failure", "web")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	var renewEvents int
	mgr.GetEventManager().RegisterFuncWithConfig(listener.EventRenew, func(*listener.EventData) {
		renewEvents++
	}, listener.ListenerConfig{Async: false})
	baseStorage := mgr.storage
	mgr.storage = &managerRenewFailingStorage{Storage: baseStorage, failKey: mgr.getRenewKey(token)}

	mgr.renewFunc(ctx, token, "renew-event-failure")
	if renewEvents != 0 {
		t.Fatalf("renew events = %d, want 0 after storage failure", renewEvents)
	}
}

// TestManagerLockedActiveTimeoutEmitsDedicatedEvent verifies lock-held timeout paths publish the exact lifecycle event after unlocking. TestManagerLockedActiveTimeoutEmitsDedicatedEvent 验证持锁活跃超时路径会在解锁后发布准确的生命周期事件。
func TestManagerLockedActiveTimeoutEmitsDedicatedEvent(t *testing.T) {
	for _, operation := range []struct {
		name string
		call func(*Manager, context.Context, string) error
	}{
		{
			name: "login by token",
			call: func(mgr *Manager, ctx context.Context, token string) error {
				return mgr.LoginByToken(ctx, token)
			},
		},
		{
			name: "renew timeout",
			call: func(mgr *Manager, ctx context.Context, token string) error {
				return mgr.RenewTimeout(ctx, token, time.Minute)
			},
		},
	} {
		t.Run(operation.name, func(t *testing.T) {
			ctx := context.Background()
			mgr := newTestManager(t, func(cfg *config.Config) {
				cfg.Timeout = 60
				cfg.ActiveTimeout = 1
				cfg.AutoRenew = false
			})
			token, err := mgr.Login(ctx, "locked-active-timeout-"+operation.name, "web", "browser")
			if err != nil {
				t.Fatalf("Login() error = %v", err)
			}
			if err = mgr.storage.Set(ctx, mgr.getActiveKey(token), time.Now().Add(-2*time.Second).Unix(), time.Minute); err != nil {
				t.Fatalf("Set(active marker) error = %v", err)
			}
			var events []*listener.EventData
			mgr.GetEventManager().RegisterFuncWithConfig(listener.EventAll, func(data *listener.EventData) {
				copyData := *data
				events = append(events, &copyData)
			}, listener.ListenerConfig{Async: false})

			if err = operation.call(mgr, ctx, token); !errors.Is(err, derror.ErrActiveTimeout) {
				t.Fatalf("%s(stale active marker) error = %v, want ErrActiveTimeout", operation.name, err)
			}
			assertManagerEvent(t, events, listener.EventActiveTimeout, "locked-active-timeout-"+operation.name, "web", "browser", token, nil)
			for _, data := range events {
				if data.Event == listener.EventKickout && data.Token == token {
					t.Fatalf("locked active timeout emitted EventKickout: %+v", data)
				}
			}
		})
	}
}

type managerUnorderedScannerStorage struct {
	*managerTestStorage
}

func (s *managerUnorderedScannerStorage) Keys(ctx context.Context, pattern string) ([]string, error) {
	keys, err := s.managerTestStorage.Keys(ctx, pattern)
	for left, right := 0, len(keys)-1; left < right; left, right = left+1, right-1 {
		keys[left], keys[right] = keys[right], keys[left]
	}
	return keys, err
}

type managerRenewFailingStorage struct {
	adapter.Storage
	failKey string
}

func (s *managerRenewFailingStorage) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	if key == s.failKey {
		return errors.New("renew marker write failed")
	}
	return s.Storage.Set(ctx, key, value, expiration)
}

func (s *managerRenewFailingStorage) Close() error {
	if closer, ok := s.Storage.(interface{ Close() error }); ok {
		return closer.Close()
	}
	return nil
}

var _ adapter.ScannerStorage = (*managerUnorderedScannerStorage)(nil)
var _ adapter.Storage = (*managerRenewFailingStorage)(nil)
