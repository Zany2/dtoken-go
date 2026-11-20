package engine

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/config"
	adapter "github.com/Zany2/dtoken-go/contract"
	derror "github.com/Zany2/dtoken-go/errors"
	listener "github.com/Zany2/dtoken-go/event"
)

// TestManagerSessionTTLFailurePreservesOtherTerminals checks every renewal path refuses to overwrite a session after a TTL read failure. TestManagerSessionTTLFailurePreservesOtherTerminals 验证各续期路径在 TTL 读取失败后均不覆盖 Session。
func TestManagerSessionTTLFailurePreservesOtherTerminals(t *testing.T) {
	ctx := context.Background()
	for _, operation := range []string{"new login", "shared login", "manual renewal", "automatic renewal", "forced renewal"} {
		t.Run(operation, func(t *testing.T) {
			mgr := newTestManager(t, func(cfg *config.Config) {
				cfg.Timeout = 60
				cfg.AutoRenew = true
				cfg.RenewMaxRefresh = 60
				cfg.IsConcurrent = true
				cfg.IsShare = true
			})
			id := "session-ttl-failure"
			if _, err := mgr.LoginWithOptions(ctx, LoginOptions{LoginID: id, Device: "mobile", Timeout: 24 * time.Hour}); err != nil {
				t.Fatal(err)
			}
			token, err := mgr.Login(ctx, id, "web")
			if err != nil {
				t.Fatal(err)
			}
			key := mgr.getSessionKey(id)
			before := captureTerminalLifecycle(t, mgr, ctx, []string{key})
			storage := &managerSaveTTLStorage{Storage: mgr.storage, key: key, ttlErr: errors.New("TTL unavailable")}
			mgr.storage = storage
			pool := &managerQueuedMaintenancePool{}
			mgr.pool = pool
			t.Cleanup(pool.runAll)
			var renews int
			mgr.GetEventManager().RegisterFuncWithConfig(listener.EventRenew, func(*listener.EventData) { renews++ }, listener.ListenerConfig{Async: false})
			switch operation {
			case "new login":
				_, err = mgr.Login(ctx, id, "desktop")
			case "shared login":
				_, err = mgr.Login(ctx, id, "web")
			case "manual renewal":
				err = mgr.RenewTimeout(ctx, token, time.Minute)
			case "automatic renewal":
				err = mgr.CheckLogin(ctx, token)
			case "forced renewal":
				err = mgr.LoginByToken(ctx, token)
			}
			if operation == "automatic renewal" || operation == "forced renewal" {
				if err != nil || pool.taskCount() != 1 {
					t.Fatalf("maintenance admission = %v, tasks = %d", err, pool.taskCount())
				}
				pool.runAll()
			} else if !errors.Is(err, derror.ErrStorageUnavailable) {
				t.Fatalf("operation error = %v, want ErrStorageUnavailable", err)
			}
			if storage.ttlReads != 1 || storage.writes != 0 || renews != 0 {
				t.Fatalf("TTL reads/writes/renew events = %d/%d/%d, want 1/0/0", storage.ttlReads, storage.writes, renews)
			}
			mgr.storage = storage.Storage
			assertTerminalLifecycleUnchanged(t, mgr, ctx, before)
		})
	}
}

// TestManagerSessionUpdatesRejectExpiredTTL verifies updates cannot extend or recreate an expiring session. TestManagerSessionUpdatesRejectExpiredTTL 验证更新不能延长或重建到期会话。
func TestManagerSessionUpdatesRejectExpiredTTL(t *testing.T) {
	ctx := context.Background()
	for _, timeout := range []int64{60, config.NoLimit} {
		for _, ttl := range []time.Duration{0, adapter.TTLNotFound} {
			for _, operation := range []struct {
				name string
				run  func(*Manager, string) error
			}{
				{"set data", func(m *Manager, id string) error { return m.SetSessionValue(ctx, id, "key", "changed") }},
				{"delete data", func(m *Manager, id string) error { return m.DeleteSessionValue(ctx, id, "key") }},
				{"add permission", func(m *Manager, id string) error { return m.AddPermissions(ctx, id, []string{"read"}) }},
			} {
				t.Run(fmt.Sprintf("timeout=%d/ttl=%d/%s", timeout, ttl, operation.name), func(t *testing.T) {
					mgr := newTestManager(t, func(cfg *config.Config) { cfg.Timeout = timeout })
					id := "session-update-boundary"
					if _, err := mgr.LoginWithTimeout(ctx, id, time.Minute); err != nil {
						t.Fatal(err)
					}
					if err := mgr.SetSessionValue(ctx, id, "key", "original"); err != nil {
						t.Fatal(err)
					}
					key := mgr.getSessionKey(id)
					before, err := mgr.storage.Get(ctx, key)
					if err != nil {
						t.Fatal(err)
					}
					storage := &managerSaveTTLStorage{Storage: mgr.storage, key: key, ttl: ttl}
					mgr.storage = storage

					if err := operation.run(mgr, id); !errors.Is(err, derror.ErrKeyNotFound) {
						t.Fatalf("update error = %v, want ErrKeyNotFound", err)
					}
					if storage.ttlReads != 1 || storage.writes != 0 {
						t.Fatalf("TTL reads/writes = %d/%d, want 1/0", storage.ttlReads, storage.writes)
					}
					after, err := storage.Storage.Get(ctx, key)
					if err != nil {
						t.Fatal(err)
					}
					if ttl == adapter.TTLNotFound {
						if after != nil {
							t.Fatal("expired session was recreated")
						}
					} else if !reflect.DeepEqual(after, before) {
						t.Fatal("session at zero TTL was overwritten")
					}
				})
			}
		}
	}
}

// TestManagerSaveToStorageTTLPolicy verifies retained TTLs, read errors, and explicit creation semantics. TestManagerSaveToStorageTTLPolicy 验证 TTL 保留、读取错误及显式创建语义。
func TestManagerSaveToStorageTTLPolicy(t *testing.T) {
	ctx := context.Background()
	for _, tt := range []struct {
		name       string
		ttl        time.Duration
		ttlErr     error
		expiration []time.Duration
		wantTTL    time.Duration
		wantErr    error
	}{
		{name: "finite update", ttl: 1500 * time.Millisecond, wantTTL: 1500 * time.Millisecond},
		{name: "permanent update", ttl: adapter.TTLNoExpire, wantTTL: 0},
		{name: "TTL read failure", ttlErr: errors.New("TTL unavailable"), wantErr: derror.ErrStorageUnavailable},
		{name: "create finite", ttl: adapter.TTLNotFound, expiration: []time.Duration{time.Minute}, wantTTL: time.Minute},
		{name: "create permanent", ttl: adapter.TTLNotFound, expiration: []time.Duration{0}, wantTTL: 0},
		{name: "explicit renewal", ttl: 0, expiration: []time.Duration{time.Minute}, wantTTL: time.Minute},
	} {
		t.Run(tt.name, func(t *testing.T) {
			mgr := newTestManager(t, nil)
			key := mgr.getSessionKey("ttl-policy")
			if tt.expiration == nil {
				if err := mgr.storage.Set(ctx, key, []byte("original"), time.Hour); err != nil {
					t.Fatal(err)
				}
			}
			storage := &managerSaveTTLStorage{Storage: mgr.storage, key: key, ttl: tt.ttl, ttlErr: tt.ttlErr}
			mgr.storage = storage
			err := mgr.saveToStorage(ctx, key, "updated", tt.expiration...)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("save error = %v, want %v", err, tt.wantErr)
			}
			wantReads := 1
			if tt.expiration != nil {
				wantReads = 0
			}
			if storage.ttlReads != wantReads {
				t.Fatalf("TTL reads = %d, want %d", storage.ttlReads, wantReads)
			}
			if tt.wantErr != nil {
				if storage.writes != 0 {
					t.Fatal("failed TTL read still wrote data")
				}
				return
			}
			if storage.writes != 1 || storage.savedTTL != tt.wantTTL {
				t.Fatalf("writes/TTL = %d/%v, want 1/%v", storage.writes, storage.savedTTL, tt.wantTTL)
			}
			data, err := storage.Storage.Get(ctx, key)
			if err != nil || data == nil {
				t.Fatalf("saved data = %v, %v, want a stored value", data, err)
			}
		})
	}
}

// managerSaveTTLStorage injects expiration boundaries between session reads and writes. managerSaveTTLStorage 在会话读写之间注入过期边界。
type managerSaveTTLStorage struct {
	adapter.Storage
	key      string
	ttl      time.Duration
	ttlErr   error
	ttlReads int
	writes   int
	savedTTL time.Duration
}

// TTL simulates zero remaining time or expiry after the session was read. TTL 模拟会话读取后剩余时长为零或已过期。
func (s *managerSaveTTLStorage) TTL(ctx context.Context, key string) (time.Duration, error) {
	if key != s.key {
		return s.Storage.TTL(ctx, key)
	}
	s.ttlReads++
	if s.ttl == adapter.TTLNotFound && s.ttlErr == nil {
		if err := s.Storage.Delete(ctx, key); err != nil {
			return 0, err
		}
	}
	return s.ttl, s.ttlErr
}

// Set records whether persistence used the intended lifetime. Set 记录持久化是否使用预期有效期。
func (s *managerSaveTTLStorage) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	if key == s.key {
		s.writes++
		s.savedTTL = expiration
	}
	return s.Storage.Set(ctx, key, value, expiration)
}
