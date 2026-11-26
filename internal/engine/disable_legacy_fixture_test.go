package engine

import (
	"context"
	"strings"
	"testing"
	"time"

	adapter "github.com/Zany2/dtoken-go/contract"
)

// upgradeTestDisableMarker converts an isolated old fixture before exercising runtime APIs. upgradeTestDisableMarker 在调用运行时 API 前转换隔离的旧封禁用例。
func upgradeTestDisableMarker(t *testing.T, mgr *Manager, key string) {
	t.Helper()
	ctx := context.Background()
	value, err := mgr.storage.Get(ctx, key)
	if err != nil || value == nil {
		t.Fatalf("read legacy fixture: %v", err)
	}
	now := time.Now()
	ttl, err := mgr.storage.TTL(ctx, key)
	if err != nil || ttl != adapter.TTLNoExpire && ttl <= 0 {
		t.Fatalf("legacy fixture TTL: %v, %v", ttl, err)
	}
	suffix := strings.TrimPrefix(key, mgr.storageNamespace())
	changes, err := UpgradeDisableSnapshot(ctx, mgr.serializer, map[string]any{suffix: value})
	if err != nil {
		t.Fatal(err)
	}
	change, ok := changes[suffix]
	if !ok {
		t.Fatal("fixture was not converted")
	}
	if ttl == adapter.TTLNoExpire {
		ttl = 0
	} else {
		ttl = time.Until(now.Add(ttl))
		if ttl <= 0 {
			t.Fatal("legacy fixture expired")
		}
	}
	target := mgr.storageNamespace() + change.Key
	if err := mgr.storage.Set(ctx, target, change.Value, ttl); err != nil {
		t.Fatal(err)
	}
	if key != target {
		if err := mgr.storage.Delete(ctx, key); err != nil {
			t.Fatal(err)
		}
	}
}

// getLegacyDisableKey constructs a legacy key for migration and isolation fixtures only. 仅为迁移及隔离用例构造旧键。
func (m *Manager) getLegacyDisableKey(loginID string) string {
	return m.storageNamespace() + DisableKeyPrefix + loginID
}

// getLegacyDisableServiceKey constructs a legacy key for migration and isolation fixtures only. 仅为迁移及隔离用例构造旧键。
func (m *Manager) getLegacyDisableServiceKey(loginID, service string) string {
	return m.storageNamespace() + DisableServiceKeyPrefix + loginID + ":" + service
}

// getLegacyDisableDeviceKey constructs a legacy key for migration and isolation fixtures only. 仅为迁移及隔离用例构造旧键。
func (m *Manager) getLegacyDisableDeviceKey(loginID, device string) string {
	return m.storageNamespace() + DisableDeviceKeyPrefix + loginID + ":" + device
}

// getLegacyDisableDeviceAndDeviceIDKey constructs a legacy key for migration and isolation fixtures only. 仅为迁移及隔离用例构造旧键。
func (m *Manager) getLegacyDisableDeviceAndDeviceIDKey(loginID, device, deviceID string) string {
	return m.storageNamespace() + DisableDeviceIDKeyPrefix + loginID + ":" + device + ":" + deviceID
}
