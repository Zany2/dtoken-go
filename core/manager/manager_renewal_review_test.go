package manager

import (
	"context"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/listener"
)

// TestManagerAutoRenewUsesExactTTL checks fractional-second threshold boundaries without sleeping. TestManagerAutoRenewUsesExactTTL 无需等待即可验证不足一秒及续期阈值边界。
func TestManagerAutoRenewUsesExactTTL(t *testing.T) {
	ctx := context.Background()
	for _, input := range []struct {
		name string
		ttl  time.Duration
		want bool
	}{
		{"subsecond", 500 * time.Millisecond, true},
		{"threshold", 5 * time.Second, true},
		{"above threshold", 5*time.Second + time.Nanosecond, false},
		{"missing", adapter.TTLNotFound, false},
		{"unlimited", adapter.TTLNoExpire, false},
		{"zero", 0, false},
	} {
		t.Run(input.name, func(t *testing.T) {
			mgr := newTestManager(t, func(cfg *config.Config) {
				cfg.Timeout = 60
				cfg.AutoRenew = true
				cfg.RenewMaxRefresh = 5
			})
			token, err := mgr.Login(ctx, "precise-renewal", "web")
			if err != nil {
				t.Fatal(err)
			}
			mgr.storage = &managerFixedRenewTTLStorage{Storage: mgr.storage, key: mgr.getTokenKey(token), ttl: input.ttl}
			pool := &managerQueuedMaintenancePool{}
			mgr.pool = pool
			t.Cleanup(pool.runAll)
			if err := mgr.CheckLogin(ctx, token); err != nil {
				t.Fatal(err)
			}
			wantTasks := 0
			if input.want {
				wantTasks = 1
			}
			if pool.taskCount() != wantTasks {
				t.Fatalf("queued renewals = %d, want %d", pool.taskCount(), wantTasks)
			}
			var renews int
			mgr.GetEventManager().RegisterFuncWithConfig(listener.EventRenew, func(*listener.EventData) { renews++ }, listener.ListenerConfig{Async: false})
			pool.runAll()
			if renews != wantTasks {
				t.Fatalf("renewal events = %d, want %d", renews, wantTasks)
			}
		})
	}
}

// managerFixedRenewTTLStorage exposes deterministic renewal boundaries. managerFixedRenewTTLStorage 提供确定的续期时长边界。
type managerFixedRenewTTLStorage struct {
	adapter.Storage
	key string
	ttl time.Duration
}

func (s *managerFixedRenewTTLStorage) TTL(ctx context.Context, key string) (time.Duration, error) {
	if key == s.key {
		return s.ttl, nil
	}
	return s.Storage.TTL(ctx, key)
}

// TestManagerSharedLoginInvalidatesOlderActivityWrite prevents a queued validation from moving activity backwards. TestManagerSharedLoginInvalidatesOlderActivityWrite 防止排队校验任务使共享登录后的活跃时间倒退。
func TestManagerSharedLoginInvalidatesOlderActivityWrite(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.ActiveTimeout = 30
		cfg.IsShare = true
	})
	token, err := mgr.Login(ctx, "shared-activity", "web", "browser")
	if err != nil {
		t.Fatal(err)
	}
	pool := &managerQueuedMaintenancePool{}
	mgr.pool = pool
	t.Cleanup(pool.runAll)
	if err := mgr.CheckLogin(ctx, token); err != nil {
		t.Fatal(err)
	}
	if pool.taskCount() != 1 {
		t.Fatal("expected a queued activity update")
	}

	// Simulate an earlier checked request without a wall-clock delay. 不等待真实时间，模拟较早请求捕获的活跃时间。
	mgr.maintenanceMu.Lock()
	state := mgr.maintenance[token]
	state.activeAt = time.Now().Add(-10 * time.Second).Unix()
	mgr.maintenance[token] = state
	mgr.maintenanceMu.Unlock()
	shared, err := mgr.Login(ctx, "shared-activity", "web", "browser")
	if err != nil || shared != token {
		t.Fatalf("shared login = %q, %v", shared, err)
	}
	before := captureTerminalLifecycle(t, mgr, ctx, []string{mgr.getTokenKey(token), mgr.getActiveKey(token), mgr.getSessionKey("shared-activity")})
	pool.runAll()
	assertTerminalLifecycleUnchanged(t, mgr, ctx, before)
}

// TestManagerRenewalDoesNotReportExpiredToken checks expiry between token validation and renewal. TestManagerRenewalDoesNotReportExpiredToken 验证校验后续期前过期不会上报成功或延长 Session。
func TestManagerRenewalDoesNotReportExpiredToken(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, nil)
	token, err := mgr.Login(ctx, "renewal-expiry", "web")
	if err != nil {
		t.Fatal(err)
	}
	base := mgr.storage
	sessionKey := mgr.getSessionKey("renewal-expiry")
	before := captureTerminalLifecycle(t, mgr, ctx, []string{sessionKey})
	mgr.storage = &managerExpireBeforeRenewStorage{Storage: base, tokenKey: mgr.getTokenKey(token), sessionKey: sessionKey}
	events := registerTerminalLifecycleEvents(mgr)
	mgr.renewFunc(ctx, token, "renewal-expiry", false)
	assertTerminalLifecycleUnchanged(t, mgr, ctx, before)
	if len(events()) != 0 {
		t.Fatalf("expired token emitted renewal success: %+v", events())
	}
	value, err := base.Get(ctx, mgr.getTokenKey(token))
	if err != nil || value != nil {
		t.Fatalf("expired token restored: %v, %v", value, err)
	}
}

// managerExpireBeforeRenewStorage expires the token after its session is read. managerExpireBeforeRenewStorage 在读取 Session 后使 Token 过期。
type managerExpireBeforeRenewStorage struct {
	adapter.Storage
	tokenKey   string
	sessionKey string
}

func (s *managerExpireBeforeRenewStorage) Get(ctx context.Context, key string) (any, error) {
	value, err := s.Storage.Get(ctx, key)
	if err == nil && key == s.sessionKey {
		err = s.Storage.Delete(ctx, s.tokenKey)
	}
	return value, err
}
