package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/config"
	adapter "github.com/Zany2/dtoken-go/contract"
	derror "github.com/Zany2/dtoken-go/errors"
	"github.com/Zany2/dtoken-go/internal/utils"
)

// renewalCostStorage counts value access separately from expiration operations. renewalCostStorage 分别统计值访问和过期操作。
type renewalCostStorage struct {
	adapter.Storage
	sessionKey, tokenKey        string
	gets, sets, expires, exists int
}

// Get counts Session value reads. Get 统计 Session 值读取。
func (s *renewalCostStorage) Get(ctx context.Context, key string) (any, error) {
	if key == s.sessionKey {
		s.gets++
	}
	return s.Storage.Get(ctx, key)
}

// Set counts Session value writes. Set 统计 Session 值写入。
func (s *renewalCostStorage) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	if key == s.sessionKey {
		s.sets++
	}
	return s.Storage.Set(ctx, key, value, ttl)
}

// Expire counts Session expiration updates. Expire 统计 Session 有效期更新。
func (s *renewalCostStorage) Expire(ctx context.Context, key string, ttl time.Duration) error {
	if key == s.sessionKey {
		s.expires++
	}
	return s.Storage.Expire(ctx, key, ttl)
}

// Exists counts redundant Token existence checks. Exists 统计额外的 Token 存在性检查。
func (s *renewalCostStorage) Exists(ctx context.Context, key string) bool {
	if key == s.tokenKey {
		s.exists++
	}
	return s.Storage.Exists(ctx, key)
}

// TestRenewalUsesSessionTTLOnly checks finite renewal costs and preservation of longer Session lifetimes. TestRenewalUsesSessionTTLOnly 验证有限期续期成本及更长 Session 有效期的保留。
func TestRenewalUsesSessionTTLOnly(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name    string
		ttl     time.Duration
		expires int
	}{
		{"shorter", 30 * time.Second, 1}, {"longer", time.Hour, 0}, {"permanent", 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mgr := newTestManager(t, func(cfg *config.Config) { cfg.Timeout = 60 })
			token, err := mgr.Login(ctx, "renew-cost")
			if err != nil {
				t.Fatal(err)
			}
			key := mgr.getSessionKey("renew-cost")
			// A raw value deliberately rejects any accidental Session decoding. 原始值用于暴露意外的 Session 解码。
			if err = mgr.storage.Set(ctx, key, "opaque-session", tc.ttl); err != nil {
				t.Fatal(err)
			}
			storage := &renewalCostStorage{Storage: mgr.storage, sessionKey: key, tokenKey: mgr.getTokenKey(token)}
			mgr.storage = storage
			mgr.renewFunc(ctx, token, "renew-cost", false)
			if storage.gets != 0 || storage.sets != 0 || storage.expires != tc.expires || storage.exists != 0 {
				t.Fatalf("gets/sets/expires/token exists = %d/%d/%d/%d", storage.gets, storage.sets, storage.expires, storage.exists)
			}
			if value, err := storage.Storage.Get(ctx, key); err != nil || value != "opaque-session" {
				t.Fatalf("Session changed = %v, %v", value, err)
			}
			if ttl, err := storage.TTL(ctx, key); err != nil || (tc.ttl == 0 && ttl != adapter.TTLNoExpire) || (tc.ttl > 0 && ttl < 59*time.Second) {
				t.Fatalf("Session TTL = %v, %v", ttl, err)
			}
		})
	}
}

// TestSessionTTLFallbackNeverCreatesMissingData checks persistence without an optional storage capability. TestSessionTTLFallbackNeverCreatesMissingData 验证无可选能力时的永久化且不重建缺失数据。
func TestSessionTTLFallbackNeverCreatesMissingData(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, nil)
	key := mgr.getSessionKey("persist-fallback")
	if err := mgr.storage.Set(ctx, key, "raw", time.Minute); err != nil {
		t.Fatal(err)
	}
	storage := &renewalCostStorage{Storage: mgr.storage, sessionKey: key}
	mgr.storage = storage
	if err := mgr.renewSessionTTL(ctx, key, 0); err != nil {
		t.Fatal(err)
	}
	if storage.gets != 1 || storage.sets != 1 || storage.expires != 0 {
		t.Fatal("ordinary persistence did not retain raw data")
	}
	if ttl, err := storage.TTL(ctx, key); err != nil || ttl != adapter.TTLNoExpire {
		t.Fatalf("TTL = %v, %v", ttl, err)
	}
	if err := storage.Delete(ctx, key); err != nil {
		t.Fatal(err)
	}
	for _, ttl := range []time.Duration{0, time.Minute} {
		if err := mgr.renewSessionTTL(ctx, key, ttl); !errors.Is(err, derror.ErrKeyNotFound) {
			t.Fatalf("missing Session = %v", err)
		}
	}
	if storage.sets != 1 {
		t.Fatal("missing Session recreated")
	}
}

// TestActivityMaintenanceSkipsSession verifies a valid credential refreshes activity independently of Session data. TestActivityMaintenanceSkipsSession 验证有效凭据独立于 Session 数据刷新活跃时间。
func TestActivityMaintenanceSkipsSession(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) { cfg.ActiveTimeout = 60 })
	token, err := mgr.Login(ctx, "activity-cost")
	if err != nil {
		t.Fatal(err)
	}
	if err = mgr.storage.Delete(ctx, mgr.getSessionKey("activity-cost")); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-10 * time.Second).Unix()
	if err = mgr.storage.Set(ctx, mgr.getActiveKey(token), old, time.Minute); err != nil {
		t.Fatal(err)
	}
	storage := &renewalCostStorage{Storage: mgr.storage, sessionKey: mgr.getSessionKey("activity-cost")}
	mgr.storage = storage
	pool := &managerQueuedMaintenancePool{}
	mgr.pool = pool
	t.Cleanup(pool.runAll)
	if err = mgr.CheckLogin(ctx, token); err != nil {
		t.Fatal(err)
	}
	pool.runAll()
	if storage.gets != 0 || storage.sets != 0 || storage.expires != 0 {
		t.Fatal("activity maintenance accessed Session")
	}
	value, err := storage.Get(ctx, mgr.getActiveKey(token))
	if err != nil {
		t.Fatal(err)
	}
	activeAt, err := utils.ToInt64(value)
	if err != nil || activeAt <= old {
		t.Fatalf("activity was not refreshed: %v, %v", value, err)
	}
}
