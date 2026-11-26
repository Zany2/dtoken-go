package engine

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	adapter "github.com/Zany2/dtoken-go/contract"
	derror "github.com/Zany2/dtoken-go/errors"
	listener "github.com/Zany2/dtoken-go/event"
)

// TestManagerDisableCanonicalCosts checks reads, deletion modes, and event behavior across all restriction scopes. TestManagerDisableCanonicalCosts 检查所有封禁作用域的读取开销、删除模式和事件行为。
func TestManagerDisableCanonicalCosts(t *testing.T) {
	ctx := context.Background()
	for _, atomicMode := range []bool{false, true} {
		for _, kind := range []string{"account", "service", "device", "concrete"} {
			for _, outcome := range []string{"success", "failure", "expired"} {
				t.Run(fmt.Sprintf("%s/%s/atomic=%v", kind, outcome, atomicMode), func(t *testing.T) {
					mgr := newTestManager(t, nil)
					id := `cost:owner\one`
					key := mgr.getDisableKey(id)
					write := func() error { return mgr.Disable(ctx, id, time.Hour) }
					check := func() error { return mgr.CheckDisable(ctx, id) }
					ttl := func() (int64, error) { return mgr.GetDisableTTL(ctx, id) }
					untie := func() error { return mgr.Untie(ctx, id) }
					blocked, event, reads := derror.ErrAccountDisabled, listener.EventUntie, 1
					switch kind {
					case "service":
						key = mgr.getDisableServiceKey(id, "pay")
						write = func() error { return mgr.DisableServiceLevel(ctx, id, "pay", 3, time.Hour) }
						check = func() error { return mgr.CheckDisableServiceLevel(ctx, id, "pay", 3) }
						ttl = func() (int64, error) { return mgr.GetDisableServiceTTL(ctx, id, "pay") }
						untie = func() error { return mgr.UntieService(ctx, id, "pay") }
						blocked, event = derror.ErrServiceDisabled, listener.EventUntieService
					case "device":
						key = mgr.getDisableDeviceKey(id, "web")
						write = func() error { return mgr.DisableDevice(ctx, id, "web", time.Hour) }
						check = func() error { return mgr.CheckDisableDevice(ctx, id, "web") }
						ttl = func() (int64, error) { return mgr.GetDisableDeviceTTL(ctx, id, "web") }
						untie = func() error { return mgr.UntieDevice(ctx, id, "web") }
						blocked, event = derror.ErrDeviceDisabled, listener.EventUntieDevice
					case "concrete":
						key = mgr.getDisableDeviceAndDeviceIDKey(id, "web", "phone")
						write = func() error { return mgr.DisableDeviceAndDeviceID(ctx, id, "web", "phone", time.Hour) }
						check = func() error { return mgr.CheckDisableDeviceAndDeviceID(ctx, id, "web", "phone") }
						ttl = func() (int64, error) { return mgr.GetDisableDeviceAndDeviceIDTTL(ctx, id, "web", "phone") }
						untie = func() error { return mgr.UntieDeviceAndDeviceID(ctx, id, "web", "phone") }
						blocked, event, reads = derror.ErrDeviceDisabled, listener.EventUntieDevice, 2
					}
					base := mgr.storage
					storage := &disableCostStorage{Storage: base}
					mgr.storage = storage
					if atomicMode {
						mgr.storage = &disableAtomicCostStorage{disableCostStorage: storage, atomic: base.(adapter.AtomicStorage)}
					}
					if err := check(); err != nil || storage.gets != reads {
						t.Fatalf("missing restriction = %v, reads=%d, want %d", err, storage.gets, reads)
					}
					storage.gets = 0
					if err := write(); err != nil {
						t.Fatal(err)
					}
					wantWriteReads := 1
					if kind == "account" {
						// Account disable also captures Session for credential retirement. 账号封禁还需读取 Session 以废弃凭据。
						wantWriteReads++
					}
					if storage.gets != wantWriteReads {
						t.Fatalf("write reads = %d, want %d", storage.gets, wantWriteReads)
					}
					storage.gets = 0
					if err := check(); !errors.Is(err, blocked) || storage.gets != reads {
						t.Fatalf("active restriction = %v, reads=%d, want %d", err, storage.gets, reads)
					}
					storage.gets = 0
					if remaining, err := ttl(); err != nil || remaining <= 0 || storage.gets != 1 || storage.ttls != 1 {
						t.Fatalf("TTL = %d, %v, reads/TTLs=%d/%d", remaining, err, storage.gets, storage.ttls)
					}
					events := 0
					mgr.GetEventManager().RegisterFuncWithConfig(event, func(*listener.EventData) { events++ }, listener.ListenerConfig{Async: false})
					storage.gets, storage.deletes = 0, 0
					if outcome == "failure" {
						storage.deleteErr = errors.New("delete unavailable")
					} else if outcome == "expired" {
						if err := base.Delete(ctx, key); err != nil {
							t.Fatal(err)
						}
					}
					err := untie()
					if outcome == "failure" && !errors.Is(err, derror.ErrStorageUnavailable) || outcome != "failure" && err != nil {
						t.Fatalf("untie = %v", err)
					}
					wantEvents, wantDeletes := 0, 1
					if outcome == "success" {
						wantEvents = 1
					} else if outcome == "expired" {
						wantDeletes = 0
					}
					if storage.gets != 1 || storage.deletes != wantDeletes || events != wantEvents {
						t.Fatalf("untie reads/deletes/events=%d/%d/%d", storage.gets, storage.deletes, events)
					}
					if base.Exists(ctx, key) != (outcome == "failure") {
						t.Fatal("unexpected restriction after untie")
					}
				})
			}
		}
	}
}

type disableCostStorage struct {
	adapter.Storage
	gets, ttls, deletes int
	deleteErr           error
}

// Get counts restriction reads. Get 统计封禁读取。
func (s *disableCostStorage) Get(ctx context.Context, key string) (any, error) {
	s.gets++
	return s.Storage.Get(ctx, key)
}

// TTL counts remaining-lifetime lookups. TTL 统计剩余有效期查询。
func (s *disableCostStorage) TTL(ctx context.Context, key string) (time.Duration, error) {
	s.ttls++
	return s.Storage.TTL(ctx, key)
}

// Delete counts ordinary deletion attempts and injects failures. Delete 统计普通删除尝试并注入故障。
func (s *disableCostStorage) Delete(ctx context.Context, keys ...string) error {
	s.deletes++
	if s.deleteErr != nil {
		return s.deleteErr
	}
	return s.Storage.Delete(ctx, keys...)
}

type disableAtomicCostStorage struct {
	*disableCostStorage
	atomic adapter.AtomicStorage
}

// GetAndDelete counts atomic deletion attempts and injects failures. GetAndDelete 统计原子删除尝试并注入故障。
func (s *disableAtomicCostStorage) GetAndDelete(ctx context.Context, key string) (any, error) {
	s.deletes++
	if s.deleteErr != nil {
		return nil, s.deleteErr
	}
	return s.atomic.GetAndDelete(ctx, key)
}

// SetIfAbsent forwards the optional atomic capability. SetIfAbsent 转发可选原子能力。
func (s *disableAtomicCostStorage) SetIfAbsent(ctx context.Context, key string, value any, ttl time.Duration) (bool, error) {
	return s.atomic.SetIfAbsent(ctx, key, value, ttl)
}
