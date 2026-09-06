package manager

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/listener"
)

// TestManagerDisableCrossFormatCompatibility verifies field matching, write protection, read precedence, and untie scope across overlapping key formats. TestManagerDisableCrossFormatCompatibility 验证重叠键格式之间的字段匹配、写入保护、读取优先级与解封范围。
func TestManagerDisableCrossFormatCompatibility(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name        string
		loginID     string
		key         func(*Manager, string) string
		legacyKey   func(*Manager, string) string
		foreignKey  func(*Manager) string
		info        any
		foreignInfo any
		isDisabled  func(*Manager, string) bool
		getInfo     func(*Manager, string) (any, error)
		check       func(*Manager, string) error
		ttl         func(*Manager, string) (int64, error)
		disable     func(*Manager, string, int, time.Duration, string) error
		untie       func(*Manager, string) error
		notDisabled error
		disabled    error
		event       listener.Event
	}{
		{
			name:        "service",
			loginID:     "user:one",
			key:         func(m *Manager, id string) string { return m.getDisableServiceKey(id, "billing") },
			legacyKey:   func(m *Manager, id string) string { return m.getLegacyDisableServiceKey(id, "billing") },
			foreignKey:  func(m *Manager) string { return m.getLegacyDisableServiceKey(`user\`, "one:billing") },
			info:        &ServiceDisableInfo{Service: "billing", Level: 2, DisableReason: "target"},
			foreignInfo: &ServiceDisableInfo{Service: "one:billing", Level: 3, DisableReason: "foreign"},
			isDisabled:  func(m *Manager, id string) bool { return m.IsDisableService(ctx, id, "billing") },
			getInfo: func(m *Manager, id string) (any, error) {
				return m.GetDisableServiceInfo(ctx, id, "billing")
			},
			check: func(m *Manager, id string) error { return m.CheckDisableService(ctx, id, "billing") },
			ttl:   func(m *Manager, id string) (int64, error) { return m.GetDisableServiceTTL(ctx, id, "billing") },
			disable: func(m *Manager, id string, level int, duration time.Duration, reason string) error {
				return m.DisableServiceLevel(ctx, id, "billing", level, duration, reason)
			},
			untie:       func(m *Manager, id string) error { return m.UntieService(ctx, id, "billing") },
			notDisabled: derror.ErrServiceNotDisabled,
			disabled:    derror.ErrServiceDisabled,
			event:       listener.EventUntieService,
		},
		{
			name:        "device type",
			loginID:     "user:one",
			key:         func(m *Manager, id string) string { return m.getDisableDeviceKey(id, "web") },
			legacyKey:   func(m *Manager, id string) string { return m.getLegacyDisableDeviceKey(id, "web") },
			foreignKey:  func(m *Manager) string { return m.getLegacyDisableDeviceKey(`user\`, "one:web") },
			info:        &DeviceDisableInfo{Device: "web", DisableReason: "target"},
			foreignInfo: &DeviceDisableInfo{Device: "one:web", DisableReason: "foreign"},
			isDisabled:  func(m *Manager, id string) bool { return m.IsDisableDevice(ctx, id, "web") },
			getInfo: func(m *Manager, id string) (any, error) {
				return m.GetDisableDeviceInfo(ctx, id, "web")
			},
			check: func(m *Manager, id string) error { return m.CheckDisableDevice(ctx, id, "web") },
			ttl:   func(m *Manager, id string) (int64, error) { return m.GetDisableDeviceTTL(ctx, id, "web") },
			disable: func(m *Manager, id string, _ int, duration time.Duration, reason string) error {
				return m.DisableDevice(ctx, id, "web", duration, reason)
			},
			untie:       func(m *Manager, id string) error { return m.UntieDevice(ctx, id, "web") },
			notDisabled: derror.ErrDeviceNotDisabled,
			disabled:    derror.ErrDeviceDisabled,
			event:       listener.EventUntieDevice,
		},
		{
			name:    "concrete device",
			loginID: "user:web",
			key: func(m *Manager, id string) string {
				return m.getDisableDeviceAndDeviceIDKey(id, "web", "phone")
			},
			legacyKey: func(m *Manager, id string) string {
				return m.getLegacyDisableDeviceAndDeviceIDKey(id, "web", "phone")
			},
			foreignKey: func(m *Manager) string {
				return m.getLegacyDisableDeviceAndDeviceIDKey(`user\`, "web", "web:phone")
			},
			info:        &DeviceDisableInfo{Device: "web", DeviceID: "phone", DisableReason: "target"},
			foreignInfo: &DeviceDisableInfo{Device: "web", DeviceID: "web:phone", DisableReason: "foreign"},
			isDisabled: func(m *Manager, id string) bool {
				return m.IsDisableDeviceAndDeviceID(ctx, id, "web", "phone")
			},
			getInfo: func(m *Manager, id string) (any, error) {
				return m.GetDisableDeviceAndDeviceIDInfo(ctx, id, "web", "phone")
			},
			check: func(m *Manager, id string) error {
				return m.CheckDisableDeviceAndDeviceID(ctx, id, "web", "phone")
			},
			ttl: func(m *Manager, id string) (int64, error) {
				return m.GetDisableDeviceAndDeviceIDTTL(ctx, id, "web", "phone")
			},
			disable: func(m *Manager, id string, _ int, duration time.Duration, reason string) error {
				return m.DisableDeviceAndDeviceID(ctx, id, "web", "phone", duration, reason)
			},
			untie: func(m *Manager, id string) error {
				return m.UntieDeviceAndDeviceID(ctx, id, "web", "phone")
			},
			notDisabled: derror.ErrDeviceNotDisabled,
			disabled:    derror.ErrDeviceDisabled,
			event:       listener.EventUntieDevice,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := newTestManager(t, nil)
			key, legacyKey := tt.key(mgr, tt.loginID), tt.legacyKey(mgr, tt.loginID)
			if key == legacyKey || key != tt.foreignKey(mgr) {
				t.Fatal("fixture must collide with another identity's legacy key, not its own legacy key")
			}
			events := 0
			mgr.GetEventManager().RegisterFuncWithConfig(tt.event, func(*listener.EventData) {
				events++
			}, listener.ListenerConfig{Async: false})
			disableEvent := listener.EventDisableDevice
			if tt.event == listener.EventUntieService {
				disableEvent = listener.EventDisableService
			}
			disableEvents := 0
			mgr.GetEventManager().RegisterFuncWithConfig(disableEvent, func(*listener.EventData) {
				disableEvents++
			}, listener.ListenerConfig{Async: false})

			// A foreign legacy marker must not be trusted just because it occupies the current key. 其他身份的旧标记不能仅因占据当前键就被信任。
			if err := mgr.saveToStorage(ctx, key, tt.foreignInfo, time.Minute); err != nil {
				t.Fatalf("save foreign marker error = %v", err)
			}
			foreignData, err := mgr.storage.Get(ctx, key)
			if err != nil {
				t.Fatalf("read foreign marker error = %v", err)
			}

			// Refuse to overwrite a recognizable foreign marker or renew its lifetime. 拒绝覆盖可识别的其他身份标记，也不改动其有效期。
			if err = tt.disable(mgr, tt.loginID, 4, 0, "must-not-write"); !errors.Is(err, derror.ErrInvalidParam) {
				t.Fatalf("disable(foreign marker) error = %v, want ErrInvalidParam", err)
			}
			if got, err := mgr.storage.Get(ctx, key); err != nil || !reflect.DeepEqual(got, foreignData) || disableEvents != 0 {
				t.Fatalf("conflicting disable changed foreign marker or events: value=%v, error=%v, events=%d", got, err, disableEvents)
			}
			if ttl, err := mgr.storage.TTL(ctx, key); err != nil || ttl <= 0 || ttl > time.Minute {
				t.Fatalf("foreign marker TTL = %v, %v, want remaining original lifetime", ttl, err)
			}

			if tt.isDisabled(mgr, tt.loginID) {
				t.Fatal("foreign marker disabled the requested identity")
			}
			if err = tt.check(mgr, tt.loginID); err != nil {
				t.Fatalf("check(foreign marker) error = %v, want nil", err)
			}
			if _, err = tt.getInfo(mgr, tt.loginID); !errors.Is(err, tt.notDisabled) {
				t.Fatalf("getInfo(foreign marker) error = %v, want %v", err, tt.notDisabled)
			}
			if ttl, err := tt.ttl(mgr, tt.loginID); err != nil || ttl != -2 {
				t.Fatalf("ttl(foreign marker) = %d, %v, want -2, nil", ttl, err)
			}
			if err = tt.untie(mgr, tt.loginID); err != nil {
				t.Fatalf("untie(foreign marker) error = %v", err)
			}
			if got, err := mgr.storage.Get(ctx, key); err != nil || !reflect.DeepEqual(got, foreignData) || events != 0 {
				t.Fatalf("no-op untie changed foreign marker or events: value=%v, error=%v, events=%d", got, err, events)
			}

			// A matching legacy marker remains usable behind the foreign current-key record. 当前键为其他身份记录时，匹配的旧标记仍可使用。
			if err = mgr.saveToStorage(ctx, legacyKey, tt.info, 0); err != nil {
				t.Fatalf("save matching legacy marker error = %v", err)
			}
			if !tt.isDisabled(mgr, tt.loginID) {
				t.Fatal("matching legacy marker was ignored")
			}
			if err = tt.check(mgr, tt.loginID); !errors.Is(err, tt.disabled) {
				t.Fatalf("check(matching legacy) error = %v, want %v", err, tt.disabled)
			}
			if info, err := tt.getInfo(mgr, tt.loginID); err != nil || !reflect.DeepEqual(info, tt.info) {
				t.Fatalf("getInfo(matching legacy) = %+v, %v, want %+v", info, err, tt.info)
			}
			if ttl, err := tt.ttl(mgr, tt.loginID); err != nil || ttl != -1 {
				t.Fatalf("ttl(matching legacy) = %d, %v, want -1, nil", ttl, err)
			}
			if err = tt.untie(mgr, tt.loginID); err != nil {
				t.Fatalf("untie(matching legacy) error = %v", err)
			}
			if got, err := mgr.storage.Get(ctx, key); err != nil || !reflect.DeepEqual(got, foreignData) {
				t.Fatalf("untie(matching legacy) changed foreign marker: value=%v, error=%v", got, err)
			}
			if mgr.storage.Exists(ctx, legacyKey) || events != 1 {
				t.Fatalf("legacy untie did not remove only matching state: events=%d, want 1", events)
			}

			// Valid current state takes precedence without reading malformed legacy data. 有效当前状态优先返回，无需读取损坏的旧数据。
			if err = mgr.saveToStorage(ctx, key, tt.info, 0); err != nil {
				t.Fatalf("save matching current marker error = %v", err)
			}
			if err = mgr.storage.Set(ctx, legacyKey, []byte("not-json"), time.Minute); err != nil {
				t.Fatalf("save malformed legacy marker error = %v", err)
			}
			storage := &managerGetCountingStorage{Storage: mgr.storage}
			mgr.storage = storage
			if !tt.isDisabled(mgr, tt.loginID) {
				t.Fatal("matching current marker was ignored")
			}
			if info, err := tt.getInfo(mgr, tt.loginID); err != nil || !reflect.DeepEqual(info, tt.info) {
				t.Fatalf("getInfo(matching current) = %+v, %v, want %+v", info, err, tt.info)
			}
			if err = tt.check(mgr, tt.loginID); !errors.Is(err, tt.disabled) {
				t.Fatalf("check(matching current) error = %v, want %v", err, tt.disabled)
			}
			if ttl, err := tt.ttl(mgr, tt.loginID); err != nil || ttl != -1 {
				t.Fatalf("ttl(matching current) = %d, %v, want -1, nil", ttl, err)
			}
			if count := storage.getCount(legacyKey); count != 0 {
				t.Fatalf("legacy reads after current match = %d, want 0", count)
			}

			// Untie still validates every candidate before deleting any record. 解封仍在删除任何记录前校验全部候选键。
			if err = tt.untie(mgr, tt.loginID); !errors.Is(err, derror.ErrSerializeFailed) {
				t.Fatalf("untie(malformed legacy) error = %v, want ErrSerializeFailed", err)
			}
			if !mgr.storage.Exists(ctx, key) || !mgr.storage.Exists(ctx, legacyKey) || events != 1 {
				t.Fatalf("failed untie changed state or emitted events: events=%d, want 1", events)
			}
			if err = mgr.saveToStorage(ctx, legacyKey, tt.info, time.Minute); err != nil {
				t.Fatalf("save matching legacy marker error = %v", err)
			}
			if err = tt.untie(mgr, tt.loginID); err != nil {
				t.Fatalf("untie(both matches) error = %v", err)
			}
			if mgr.storage.Exists(ctx, key) || mgr.storage.Exists(ctx, legacyKey) || events != 2 {
				t.Fatalf("untie(both matches) left state or emitted duplicate events: events=%d, want 2", events)
			}

			// Valid creation and updates still replace reason, service level, and expiration. 正常创建和更新仍可替换原因、服务等级与有效期。
			if err = tt.disable(mgr, tt.loginID, 1, 0, "initial"); err != nil {
				t.Fatalf("disable(new marker) error = %v", err)
			}
			if ttl, err := tt.ttl(mgr, tt.loginID); err != nil || ttl != -1 {
				t.Fatalf("ttl(new marker) = %d, %v, want -1, nil", ttl, err)
			}
			if err = tt.disable(mgr, tt.loginID, 4, time.Minute, "updated"); err != nil {
				t.Fatalf("disable(existing marker) error = %v", err)
			}
			updated, err := tt.getInfo(mgr, tt.loginID)
			if err != nil {
				t.Fatalf("getInfo(updated marker) error = %v", err)
			}
			switch info := updated.(type) {
			case *ServiceDisableInfo:
				if info.DisableReason != "updated" || info.Level != 4 {
					t.Fatalf("updated service marker = %+v, want new reason and level", info)
				}
			case *DeviceDisableInfo:
				if info.DisableReason != "updated" {
					t.Fatalf("updated device marker = %+v, want new reason", info)
				}
			default:
				t.Fatalf("unexpected disable info type %T", updated)
			}
			if ttl, err := tt.ttl(mgr, tt.loginID); err != nil || ttl < 1 || ttl > 60 {
				t.Fatalf("ttl(updated marker) = %d, %v, want 1..60 seconds", ttl, err)
			}
			if disableEvents != 2 {
				t.Fatalf("successful disable events = %d, want 2", disableEvents)
			}

			// Failed reads must not turn into unconditional overwrites. 读取失败不能退化为无条件覆盖。
			before, err := mgr.storage.Get(ctx, key)
			if err != nil {
				t.Fatalf("read before failed update error = %v", err)
			}
			mgr.storage = &managerFailingStorage{Storage: storage, getErr: errors.New("read failed")}
			if err = tt.disable(mgr, tt.loginID, 5, 0, "must-not-write"); !errors.Is(err, derror.ErrStorageUnavailable) {
				t.Fatalf("disable(read failure) error = %v, want ErrStorageUnavailable", err)
			}
			mgr.storage = storage
			if got, err := mgr.storage.Get(ctx, key); err != nil || !reflect.DeepEqual(got, before) || disableEvents != 2 {
				t.Fatalf("failed update changed marker or events: value=%v, error=%v, events=%d", got, err, disableEvents)
			}

			// Undecodable records stay intact because their scope cannot be checked. 无法解码的记录无法核对范围，因此保留原数据。
			malformed := []byte("not-json")
			if err = mgr.storage.Set(ctx, key, malformed, time.Minute); err != nil {
				t.Fatalf("save malformed current marker error = %v", err)
			}
			before, err = mgr.storage.Get(ctx, key)
			if err != nil {
				t.Fatalf("read malformed current marker error = %v", err)
			}
			if err = tt.disable(mgr, tt.loginID, 5, 0, "must-not-write"); !errors.Is(err, derror.ErrSerializeFailed) {
				t.Fatalf("disable(decode failure) error = %v, want ErrSerializeFailed", err)
			}
			if got, err := mgr.storage.Get(ctx, key); err != nil || !reflect.DeepEqual(got, before) || disableEvents != 2 {
				t.Fatalf("decode failure changed marker or events: value=%v, error=%v, events=%d", got, err, disableEvents)
			}
		})
	}
}
