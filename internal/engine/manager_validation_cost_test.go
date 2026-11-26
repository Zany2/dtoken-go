package engine

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/config"
	adapter "github.com/Zany2/dtoken-go/contract"
	listener "github.com/Zany2/dtoken-go/event"
)

// TestManagerLoginChecksDeferSessionAndRenewalReads checks the foreground cost and background renewal decisions independently. TestManagerLoginChecksDeferSessionAndRenewalReads 分别验证前台校验开销和后台续期决策。
func TestManagerLoginChecksDeferSessionAndRenewalReads(t *testing.T) {
	ctx := context.Background()
	for _, input := range []struct {
		name       string
		ttl        time.Duration
		marker     bool
		ttlErr     error
		wantRenew  int
		wantMarker int64
	}{
		{name: "above threshold", ttl: 30 * time.Second},
		{name: "due", ttl: 2 * time.Second, wantRenew: 1, wantMarker: 1},
		{name: "interval blocked", ttl: 2 * time.Second, marker: true, wantMarker: 1},
		{name: "TTL unavailable", ttlErr: errors.New("TTL unavailable")},
	} {
		t.Run(input.name, func(t *testing.T) {
			mgr := newTestManager(t, func(cfg *config.Config) {
				cfg.Timeout = 60
				cfg.AutoRenew = true
				cfg.RenewMaxRefresh = 5
				cfg.RenewInterval = 10
			})
			id := "validation-cost"
			token, err := mgr.Login(ctx, id, "web")
			if err != nil {
				t.Fatal(err)
			}
			if err := mgr.SetSessionValue(ctx, id, "large payload", strings.Repeat("x", 64*1024)); err != nil {
				t.Fatal(err)
			}
			if !input.marker {
				if err := mgr.storage.Delete(ctx, mgr.getRenewKey(token)); err != nil {
					t.Fatal(err)
				}
			}
			storage := &managerValidationCountingStorage{
				managerGetCountingStorage: &managerGetCountingStorage{Storage: &managerFixedRenewTTLStorage{
					Storage: mgr.storage, key: mgr.getTokenKey(token), ttl: input.ttl,
				}},
				tokenKey: mgr.getTokenKey(token), renewKey: mgr.getRenewKey(token), ttlErr: input.ttlErr,
			}
			codec := &managerValidationCountingCodec{Codec: mgr.serializer}
			mgr.storage, mgr.serializer = storage, codec
			pool := &managerQueuedMaintenancePool{}
			mgr.pool = pool
			t.Cleanup(pool.runAll)
			renews := 0
			mgr.GetEventManager().RegisterFuncWithConfig(listener.EventRenew, func(*listener.EventData) {
				renews++
			}, listener.ListenerConfig{Async: false})

			if !mgr.IsLogin(ctx, token) {
				t.Fatal("IsLogin rejected valid token")
			}
			if err := mgr.CheckLogin(ctx, token); err != nil {
				t.Fatal(err)
			}
			if loginID, err := mgr.GetLoginID(ctx, token); err != nil || loginID != id {
				t.Fatalf("GetLoginID = %q, %v", loginID, err)
			}
			if storage.getCount(storage.tokenKey) != 3 || codec.tokens.Load() != 0 {
				t.Fatalf("token reads/codec decodes = %d/%d, want 3/0", storage.getCount(storage.tokenKey), codec.tokens.Load())
			}
			storage.mu.Lock()
			totalGets := 0
			for _, count := range storage.gets {
				totalGets += count
			}
			storage.mu.Unlock()
			if totalGets != 3 {
				t.Fatalf("foreground total GETs = %d, want one per check", totalGets)
			}
			if storage.getCount(mgr.getSessionKey(id)) != 0 || codec.sessions.Load() != 0 || storage.ttls.Load() != 0 || storage.markers.Load() != 0 {
				t.Fatal("foreground validation loaded session data or checked renewal eligibility")
			}
			if pool.taskCount() != 1 {
				t.Fatalf("coalesced tasks = %d, want 1", pool.taskCount())
			}

			pool.runAll()
			if renews != input.wantRenew || storage.ttls.Load() != 1 || storage.markers.Load() != input.wantMarker {
				t.Fatalf("renewals/TTL checks/marker checks = %d/%d/%d, want %d/1/%d", renews, storage.ttls.Load(), storage.markers.Load(), input.wantRenew, input.wantMarker)
			}
			if storage.getCount(mgr.getSessionKey(id)) != 0 || codec.sessions.Load() != 0 || codec.tokens.Load() != 0 || storage.getCount(storage.tokenKey) != 3+input.wantRenew {
				t.Fatalf("background record reads were not limited to due renewal: session reads/decodes=%d/%d, token decodes=%d", storage.getCount(mgr.getSessionKey(id)), codec.sessions.Load(), codec.tokens.Load())
			}
			mgr.maintenanceMu.Lock()
			pending := len(mgr.maintenance)
			mgr.maintenanceMu.Unlock()
			if pending != 0 {
				t.Fatal("completed eligibility check retained its maintenance slot")
			}
		})
	}
}

// managerValidationCountingStorage separates renewal metadata reads from credential reads. managerValidationCountingStorage 分别统计续期元数据和凭据读取。
type managerValidationCountingStorage struct {
	*managerGetCountingStorage
	tokenKey string
	renewKey string
	ttlErr   error
	ttls     atomic.Int64
	markers  atomic.Int64
}

// TTL counts token eligibility reads and can simulate their failure. TTL 统计 Token 续期资格读取，并可模拟故障。
func (s *managerValidationCountingStorage) TTL(ctx context.Context, key string) (time.Duration, error) {
	if key == s.tokenKey {
		s.ttls.Add(1)
		if s.ttlErr != nil {
			return 0, s.ttlErr
		}
	}
	return s.Storage.TTL(ctx, key)
}

// Exists counts renewal throttle lookups. Exists 统计续期间隔查询。
func (s *managerValidationCountingStorage) Exists(ctx context.Context, key string) bool {
	if key == s.renewKey {
		s.markers.Add(1)
	}
	return s.Storage.Exists(ctx, key)
}

// managerValidationCountingCodec tracks decoded records without changing their representation. managerValidationCountingCodec 统计解码记录，保留原有数据表示。
type managerValidationCountingCodec struct {
	adapter.Codec
	tokens   atomic.Int64
	sessions atomic.Int64
	extras   atomic.Int64
}

// Decode counts credential and account data decoding. Decode 统计凭据与账号数据解码。
func (c *managerValidationCountingCodec) Decode(data []byte, target any) error {
	switch target.(type) {
	case *tokenRecord:
		c.tokens.Add(1)
	case *Session, **Session:
		c.sessions.Add(1)
	case *map[string]any:
		c.extras.Add(1)
	}
	return c.Codec.Decode(data, target)
}
