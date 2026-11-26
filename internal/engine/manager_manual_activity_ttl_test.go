package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/config"
	adapter "github.com/Zany2/dtoken-go/contract"
	derror "github.com/Zany2/dtoken-go/errors"
	listener "github.com/Zany2/dtoken-go/event"
	"github.com/Zany2/dtoken-go/internal/utils"
	"github.com/Zany2/dtoken-go/storage/memory"
)

// manualActivityStorage observes activity operations and intervenes after renewal validation. manualActivityStorage 记录活跃标记操作，并在续期校验后注入状态变化。
type manualActivityStorage struct {
	adapter.Storage
	activeKey, tokenKey           string
	gets, sets, expires, persists int
	afterValidation               func() error
	failOperation                 string
}

// Get counts reads and can fail the unlimited-lifetime fallback after validation. Get 统计读取，并可使校验后的永久化回退读取失败。
func (s *manualActivityStorage) Get(ctx context.Context, key string) (any, error) {
	if key == s.activeKey {
		s.gets++
		if s.failOperation == "get" && s.gets > 1 {
			return nil, errors.New("activity read failed")
		}
	}
	return s.Storage.Get(ctx, key)
}

// Set counts activity writes and runs the hook when renewal writes its primary record. Set 统计活跃标记写入，并在续期写主记录时执行钩子。
func (s *manualActivityStorage) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	if key == s.tokenKey && s.afterValidation != nil {
		hook := s.afterValidation
		s.afterValidation = nil
		if err := hook(); err != nil {
			return err
		}
	}
	if key == s.activeKey {
		s.sets++
		if s.failOperation == "set" {
			return errors.New("activity write failed")
		}
	}
	return s.Storage.Set(ctx, key, value, ttl)
}

// Expire counts TTL-only activity updates. Expire 统计仅修改活跃标记有效期的操作。
func (s *manualActivityStorage) Expire(ctx context.Context, key string, ttl time.Duration) error {
	if key == s.activeKey {
		s.expires++
		if s.failOperation == "expire" {
			return errors.New("activity expiration failed")
		}
	}
	return s.Storage.Expire(ctx, key, ttl)
}

// manualPersistentActivityStorage exposes the native persistence capability separately. manualPersistentActivityStorage 单独暴露原生永久化能力。
type manualPersistentActivityStorage struct {
	*manualActivityStorage
}

// Persist counts native activity persistence without reading its value. Persist 统计无需读取活跃标记值的原生永久化操作。
func (s *manualPersistentActivityStorage) Persist(ctx context.Context, key string) error {
	if key == s.activeKey {
		s.persists++
		if s.failOperation == "persist" {
			return errors.New("activity persistence failed")
		}
	}
	return s.Storage.(adapter.PersistentStorage).Persist(ctx, key)
}

// TestManualRenewalActivityTTL covers exact lifetime changes without treating renewal as user activity. TestManualRenewalActivityTTL 验证精确有效期变更，不将手动续期视为用户活跃。
func TestManualRenewalActivityTTL(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name                          string
		timeout                       time.Duration
		native, disabled, advance     bool
		gets, sets, expires, persists int
	}{
		{name: "shorten", timeout: time.Minute, gets: 1, expires: 1},
		{name: "extend", timeout: 2 * time.Hour, gets: 1, expires: 1},
		{name: "native permanent", native: true, gets: 1, persists: 1},
		{name: "ordinary permanent", gets: 2, sets: 1},
		{name: "newer activity finite", timeout: time.Minute, advance: true, gets: 1, expires: 1},
		{name: "newer activity permanent", native: true, advance: true, gets: 1, persists: 1},
		{name: "disabled", timeout: time.Minute, disabled: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mgr := newTestManager(t, func(cfg *config.Config) {
				cfg.Timeout = config.NoLimit
				if !tc.disabled {
					cfg.ActiveTimeout = 300
				}
			})
			// Use the bundled native implementation, hiding optional capabilities for fallback cases. 使用内置原生实现，回退场景隐藏可选能力。
			base := memory.NewStorage()
			mgr.storage = base
			token, err := mgr.LoginWithOptions(ctx, LoginOptions{LoginID: "manual-ttl", Timeout: time.Hour, Extra: map[string]any{"source": "test"}})
			if err != nil {
				t.Fatal(err)
			}
			record, err := mgr.getTokenRecord(ctx, token)
			if err != nil {
				t.Fatal(err)
			}
			activeKey := mgr.getActiveKey(token)
			activity := time.Now().Add(-time.Minute).Unix()
			if !tc.disabled {
				if err = base.Set(ctx, activeKey, activity, time.Hour); err != nil {
					t.Fatal(err)
				}
			}
			storage := &manualActivityStorage{Storage: base, activeKey: activeKey, tokenKey: mgr.getTokenKey(token)}
			if tc.advance {
				activity += 30
				storage.afterValidation = func() error { return base.Set(ctx, activeKey, activity, time.Hour) }
			}
			mgr.storage = storage
			if tc.native {
				mgr.storage = &manualPersistentActivityStorage{storage}
			}
			var renews int
			mgr.GetEventManager().RegisterFuncWithConfig(listener.EventRenew, func(*listener.EventData) { renews++ }, listener.ListenerConfig{Async: false})
			if err = mgr.RenewTimeout(ctx, token, tc.timeout); err != nil {
				t.Fatal(err)
			}
			if storage.gets != tc.gets || storage.sets != tc.sets || storage.expires != tc.expires || storage.persists != tc.persists || renews != 1 {
				t.Fatalf("activity get/set/expire/persist/events = %d/%d/%d/%d/%d", storage.gets, storage.sets, storage.expires, storage.persists, renews)
			}
			if !tc.disabled {
				value, err := base.Get(ctx, activeKey)
				if err != nil {
					t.Fatal(err)
				}
				actual, err := utils.ToInt64(value)
				if err != nil || actual != activity {
					t.Fatalf("activity = %v, %v; want %d", value, err, activity)
				}
			}
			for _, key := range []string{mgr.getTokenKey(token), mgr.getTokenExtraKey(record.AccessID), activeKey, mgr.getSessionKey(record.LoginID)} {
				if tc.disabled && key == activeKey {
					continue
				}
				want := tc.timeout
				if key == mgr.getSessionKey(record.LoginID) && want > 0 && want < time.Hour {
					want = time.Hour
				}
				ttl, err := base.TTL(ctx, key)
				if err != nil || want == 0 && ttl != adapter.TTLNoExpire || want > 0 && (ttl > want || ttl < want-5*time.Second) {
					t.Fatalf("TTL for %s = %v, %v; want %v", key, ttl, err, want)
				}
			}
		})
	}
}

// TestManualRenewalActivityFailure rejects missing markers and storage errors without publishing success. TestManualRenewalActivityFailure 验证缺失标记及存储错误不会发布续期成功事件。
func TestManualRenewalActivityFailure(t *testing.T) {
	ctx := context.Background()
	for _, mode := range []string{"finite", "native permanent", "ordinary permanent"} {
		for _, failure := range []string{"missing", "storage", "read"} {
			if failure == "read" && mode != "ordinary permanent" {
				continue
			}
			t.Run(mode+"/"+failure, func(t *testing.T) {
				mgr := newTestManager(t, func(cfg *config.Config) {
					cfg.Timeout = config.NoLimit
					cfg.ActiveTimeout = 300
				})
				base := memory.NewStorage()
				mgr.storage = base
				token, err := mgr.LoginWithTimeout(ctx, "manual-failure", time.Hour)
				if err != nil {
					t.Fatal(err)
				}
				key := mgr.getActiveKey(token)
				storage := &manualActivityStorage{Storage: base, activeKey: key, tokenKey: mgr.getTokenKey(token)}
				mgr.storage = storage
				timeout := time.Duration(0)
				switch mode {
				case "finite":
					timeout, storage.failOperation = time.Minute, "expire"
				case "native permanent":
					mgr.storage = &manualPersistentActivityStorage{storage}
					storage.failOperation = "persist"
				default:
					storage.failOperation = "set"
				}
				if failure == "read" {
					storage.failOperation = "get"
				}
				if failure == "missing" {
					storage.failOperation = ""
					storage.afterValidation = func() error { return base.Delete(ctx, key) }
				}
				var renews int
				mgr.GetEventManager().RegisterFuncWithConfig(listener.EventRenew, func(*listener.EventData) { renews++ }, listener.ListenerConfig{Async: false})
				err = mgr.RenewTimeout(ctx, token, timeout)
				if err == nil || renews != 0 {
					t.Fatalf("renewal error/events = %v/%d", err, renews)
				}
				if failure != "missing" && !errors.Is(err, derror.ErrStorageUnavailable) {
					t.Fatalf("storage failure = %v", err)
				}
				if failure == "missing" && base.Exists(ctx, key) {
					t.Fatal("renewal recreated missing activity")
				}
			})
		}
	}
}
