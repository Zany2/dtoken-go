package engine

import (
	"context"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/config"
	adapter "github.com/Zany2/dtoken-go/contract"
	listener "github.com/Zany2/dtoken-go/event"
	"github.com/Zany2/dtoken-go/internal/utils"
)

// TestMaintenanceRejectionPreservesActivityAndAllowsRetry verifies optional renewal is skipped without dropping activity or leaking reservations. TestMaintenanceRejectionPreservesActivityAndAllowsRetry 验证可选续期被跳过但不丢失活跃更新或泄漏任务占位。
func TestMaintenanceRejectionPreservesActivityAndAllowsRetry(t *testing.T) {
	for name, pool := range map[string]adapter.Pool{"absent": nil, "full": managerSubmitErrorPool{}} {
		for _, active := range []bool{false, true} {
			label := name + "/renewal"
			if active {
				label += " and activity"
			}
			t.Run(label, func(t *testing.T) {
				ctx := context.Background()
				mgr := newTestManager(t, func(cfg *config.Config) {
					cfg.Timeout = 600
					cfg.AutoRenew = true
					cfg.RenewMaxRefresh = 600
					if active {
						cfg.ActiveTimeout = 60
					}
				})
				token, err := mgr.Login(ctx, "rejected-maintenance")
				if err != nil {
					t.Fatal(err)
				}
				mgr.pool = pool
				if err = mgr.storage.Expire(ctx, mgr.getTokenKey(token), time.Minute); err != nil {
					t.Fatal(err)
				}
				old := time.Now().Add(-10 * time.Second).Unix()
				if active {
					if err = mgr.storage.Set(ctx, mgr.getActiveKey(token), old, time.Minute); err != nil {
						t.Fatal(err)
					}
				}
				if err = mgr.CheckLogin(ctx, token); err != nil {
					t.Fatal(err)
				}
				if ttl, err := mgr.storage.TTL(ctx, mgr.getTokenKey(token)); err != nil || ttl <= 0 || ttl > time.Minute {
					t.Fatalf("optional renewal changed TTL: %v, %v", ttl, err)
				}
				if active {
					value, err := mgr.storage.Get(ctx, mgr.getActiveKey(token))
					if err != nil {
						t.Fatal(err)
					}
					at, err := utils.ToInt64(value)
					if err != nil || at <= old {
						t.Fatalf("activity dropped: %v, %v", value, err)
					}
					if ttl, err := mgr.storage.TTL(ctx, mgr.getActiveKey(token)); err != nil || ttl <= 0 || ttl > time.Minute {
						t.Fatalf("activity extended TTL: %v, %v", ttl, err)
					}
				}
				queued := &managerQueuedMaintenancePool{}
				mgr.pool = queued
				t.Cleanup(queued.runAll)
				if err = mgr.CheckLogin(ctx, token); err != nil {
					t.Fatal(err)
				}
				if queued.taskCount() != 1 {
					t.Fatal("rejected work retained its reservation")
				}
			})
		}
	}
}

// TestForcedMaintenanceFallbackKeepsRenewalEvent verifies explicit renewal completes after rejection and publishes outside the account lock. TestForcedMaintenanceFallbackKeepsRenewalEvent 验证显式续期被拒绝后仍完成，并在账号锁外发布事件。
func TestForcedMaintenanceFallbackKeepsRenewalEvent(t *testing.T) {
	for name, pool := range map[string]adapter.Pool{"absent": nil, "full": managerSubmitErrorPool{}} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			mgr := newTestManager(t, func(cfg *config.Config) {
				cfg.Timeout = 600
				cfg.AsyncEvent = true
			})
			token, err := mgr.Login(ctx, "forced-fallback")
			if err != nil {
				t.Fatal(err)
			}
			mgr.pool = pool
			if err = mgr.storage.Expire(ctx, mgr.getTokenKey(token), time.Minute); err != nil {
				t.Fatal(err)
			}
			var events int
			mgr.GetEventManager().RegisterFunc(listener.EventRenew, func(data *listener.EventData) {
				unlock := mgr.lockLoginWrite(data.LoginID)
				unlock()
				events++
			})
			if err = mgr.LoginByToken(ctx, token); err != nil {
				t.Fatal(err)
			}
			if events != 1 {
				t.Fatalf("renewal events = %d, want 1 before return", events)
			}
			if ttl, err := mgr.storage.TTL(ctx, mgr.getTokenKey(token)); err != nil || ttl <= time.Minute {
				t.Fatalf("forced renewal skipped: %v, %v", ttl, err)
			}
		})
	}
}
