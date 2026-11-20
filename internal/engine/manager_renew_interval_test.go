package engine

import (
	"context"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/config"
	adapter "github.com/Zany2/dtoken-go/contract"
	listener "github.com/Zany2/dtoken-go/event"
)

// TestManagerRenewIntervalFitsTokenLifetime covers each writer of the renewal throttle with per-token timeouts. TestManagerRenewIntervalFitsTokenLifetime 覆盖各续期节流写入路径对单次 Token 有效期的处理。
func TestManagerRenewIntervalFitsTokenLifetime(t *testing.T) {
	ctx := context.Background()
	for _, input := range []struct {
		name       string
		expiration time.Duration
		interval   time.Duration
	}{
		{"short login", time.Minute, 30 * time.Second},
		{"equal interval", 5 * time.Minute, 150 * time.Second},
		{"ordinary login", time.Hour, 5 * time.Minute},
		{"subsecond login", 500 * time.Millisecond, 250 * time.Millisecond},
		{"manual shortening", time.Minute, 30 * time.Second},
		{"shared login", time.Minute, 30 * time.Second},
		{"forced renewal", time.Minute, 30 * time.Second},
		{"automatic renewal", time.Minute, 30 * time.Second},
	} {
		t.Run(input.name, func(t *testing.T) {
			mgr := newTestManager(t, func(cfg *config.Config) {
				cfg.Timeout = 3600
				cfg.AutoRenew = true
				cfg.RenewInterval = 300
				cfg.RenewMaxRefresh = 3600
				cfg.IsShare = input.name == "shared login"
			})
			pool := &managerQueuedMaintenancePool{}
			mgr.pool = pool
			t.Cleanup(pool.runAll)
			storage := &managerRenewIntervalStorage{Storage: mgr.storage, expirations: make(map[string]time.Duration)}
			mgr.storage = storage
			expiration := input.expiration
			if input.name == "manual shortening" {
				expiration = time.Hour
			}
			token, err := mgr.LoginWithOptions(ctx, LoginOptions{LoginID: "renew-interval", Device: "web", Timeout: expiration})
			if err != nil {
				t.Fatal(err)
			}
			renewKey := mgr.getRenewKey(token)
			switch input.name {
			case "manual shortening":
				if storage.expirations[renewKey] != 5*time.Minute {
					t.Fatal("initial long-lived token has an unexpected interval")
				}
				err = mgr.RenewTimeout(ctx, token, input.expiration)
			case "shared login":
				delete(storage.expirations, renewKey)
				var shared string
				shared, err = mgr.Login(ctx, "renew-interval", "web")
				if shared != token {
					t.Fatalf("shared token = %q, want %q", shared, token)
				}
			case "forced renewal":
				delete(storage.expirations, renewKey)
				err = mgr.LoginByToken(ctx, token)
				pool.runAll()
			case "automatic renewal":
				if mgr.isAutoRenewDue(ctx, token) {
					t.Fatal("fresh throttle did not suppress renewal")
				}
				// Simulate a later request after the throttle expires without sleeping. 模拟节流标记过期后的请求，不依赖真实等待。
				if err = storage.Delete(ctx, renewKey); err != nil {
					t.Fatal(err)
				}
				if err = storage.Expire(ctx, mgr.getTokenKey(token), 20*time.Second); err != nil {
					t.Fatal(err)
				}
				delete(storage.expirations, renewKey)
				var renews int
				mgr.GetEventManager().RegisterFuncWithConfig(listener.EventRenew, func(*listener.EventData) { renews++ }, listener.ListenerConfig{Async: false})
				err = mgr.CheckLogin(ctx, token)
				pool.runAll()
				if renews != 1 {
					t.Fatalf("automatic renewal events = %d, want 1", renews)
				}
				ttl, ttlErr := storage.TTL(ctx, mgr.getTokenKey(token))
				if ttlErr != nil || ttl <= 50*time.Second || ttl > time.Minute {
					t.Fatalf("renewed token TTL = %v, %v", ttl, ttlErr)
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			if interval := storage.expirations[renewKey]; interval != input.interval {
				t.Fatalf("renewal interval = %v, want %v", interval, input.interval)
			}
		})
	}
}

// TestManagerRenewMarkerBoundarySemantics preserves disabled and unlimited behavior without permanent zero-duration throttles. TestManagerRenewMarkerBoundarySemantics 保留禁用与无限期语义，并避免零时长永久节流标记。
func TestManagerRenewMarkerBoundarySemantics(t *testing.T) {
	ctx := context.Background()
	for _, input := range []struct {
		name       string
		configured int64
		expiration time.Duration
		want       time.Duration
	}{
		{"disabled", config.NoLimit, time.Minute, 0},
		{"unlimited token", 300, adapter.TTLNoExpire, 5 * time.Minute},
		{"minimum duration", 300, time.Nanosecond, 0},
	} {
		t.Run(input.name, func(t *testing.T) {
			mgr := newTestManager(t, func(cfg *config.Config) { cfg.RenewInterval = input.configured })
			storage := &managerRenewIntervalStorage{Storage: mgr.storage, expirations: make(map[string]time.Duration)}
			mgr.storage = storage
			if err := mgr.setRenewMarker(ctx, "boundary", input.expiration); err != nil {
				t.Fatal(err)
			}
			key := mgr.getRenewKey("boundary")
			if got := storage.expirations[key]; got != input.want {
				t.Fatalf("marker duration = %v, want %v", got, input.want)
			}
			if input.want == 0 && storage.Exists(ctx, key) {
				t.Fatal("disabled throttle created a permanent marker")
			}
		})
	}
}

// managerRenewIntervalStorage records exact expiration durations without wall-clock assertions. managerRenewIntervalStorage 记录精确过期时长，避免依赖墙上时钟进行断言。
type managerRenewIntervalStorage struct {
	adapter.Storage
	expirations map[string]time.Duration
}

// Set records successful expiration writes. Set 记录成功写入的过期时长。
func (s *managerRenewIntervalStorage) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	if err := s.Storage.Set(ctx, key, value, expiration); err != nil {
		return err
	}
	s.expirations[key] = expiration
	return nil
}
