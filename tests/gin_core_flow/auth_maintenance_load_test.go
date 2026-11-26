package gin_core_flow_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go"
	"github.com/Zany2/dtoken-go/config"
	adapter "github.com/Zany2/dtoken-go/contract"
	listener "github.com/Zany2/dtoken-go/event"
	"github.com/Zany2/dtoken-go/internal/engine"
	"github.com/Zany2/dtoken-go/internal/utils"
	"github.com/Zany2/dtoken-go/pool/ants"
	"github.com/Zany2/dtoken-go/storage/memory"
)

// maintenanceGatePool holds accepted work inside a real worker until the test releases it. maintenanceGatePool 在真实工作线程内暂缓已接收任务，直到测试放行。
type maintenanceGatePool struct {
	adapter.Pool
	gate    <-chan struct{}
	started chan struct{}
	once    sync.Once
}

// Submit preserves real pool admission and controls only the start of accepted work. Submit 保留真实池接收行为，仅控制已接收任务的开始时间。
func (p *maintenanceGatePool) Submit(task func()) error {
	return p.Pool.Submit(func() {
		p.once.Do(func() { close(p.started) })
		<-p.gate
		task()
	})
}

// newMaintenanceLoadAuth uses Memory and a single real worker, observing admission without changing it. newMaintenanceLoadAuth 使用 Memory 及单个真实工作线程，只观察而不改变任务接收行为。
func newMaintenanceLoadAuth(t *testing.T, active, renew bool, gate <-chan struct{}) (*dtoken.Auth, *benchmarkTaskPool, *benchmarkErrorLog, *maintenanceGatePool) {
	t.Helper()
	worker, err := ants.NewRenewPoolManagerWithConfig(ants.DefaultRenewPoolConfig().SetMinSize(1).SetMaxSize(1))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(worker.Stop)
	controlled := &maintenanceGatePool{Pool: worker, gate: gate, started: make(chan struct{})}
	pool := &benchmarkTaskPool{Pool: controlled}
	logger := &benchmarkErrorLog{Log: adapter.NewNopLogger()}
	cfg := config.DefaultConfig()
	cfg.KeyPrefix = flowKeyPrefix(t)
	cfg.Timeout, cfg.RenewMaxRefresh, cfg.RenewInterval = 600, 300, config.NoLimit
	cfg.ActiveTimeout = config.NoLimit
	if active {
		cfg.ActiveTimeout = 60
	}
	cfg.AutoRenew = renew
	cfg.IsShare, cfg.IsConcurrent, cfg.MaxLoginCount = false, true, config.NoLimit
	cfg.IsPrintBanner, cfg.IsLog, cfg.AsyncEvent = false, true, true
	auth, err := dtoken.NewBuilder().Config(cfg).Storage(memory.NewStorage()).SetPool(pool).SetLog(logger).BuildInstance()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(auth.Close)
	// Isolate maintenance from login-event logging while retaining submission-error observation. 隔离登录事件日志任务，同时保留提交错误观察。
	auth.EventManager().EnableEvent(listener.EventRenew)
	return auth, pool, logger, controlled
}

// seedMaintenanceToken prepares an eligible renewal and an older but valid activity marker. seedMaintenanceToken 准备满足续期条件且活跃标记较旧但有效的凭据。
func seedMaintenanceToken(t *testing.T, auth *dtoken.Auth, loginID, deviceID string, active bool) (string, string, string) {
	t.Helper()
	ctx := context.Background()
	token, err := auth.Login(ctx, dtoken.LoginOptions{LoginID: loginID, Device: "web", DeviceID: deviceID})
	if err != nil {
		t.Fatal(err)
	}
	cfg := auth.Config()
	namespace := utils.StorageNamespace(cfg.KeyPrefix, cfg.AuthType)
	tokenKey, activeKey := namespace+config.TokenKeyPrefix+token, namespace+engine.ActivePrefix+token
	if err = auth.Storage().Expire(ctx, tokenKey, time.Minute); err != nil {
		t.Fatal(err)
	}
	if active {
		if err = auth.Storage().Set(ctx, activeKey, time.Now().Add(-10*time.Second).Unix(), time.Minute); err != nil {
			t.Fatal(err)
		}
	}
	return token, tokenKey, activeKey
}

// checkMaintenanceBurst waits for concurrent public checks before any shutdown or assertions. checkMaintenanceBurst 在关闭或断言前等待并发公开校验返回。
func checkMaintenanceBurst(t *testing.T, auth *dtoken.Auth, tokens []string) {
	t.Helper()
	start := make(chan struct{})
	errors := make(chan error, len(tokens))
	for _, token := range tokens {
		go func() {
			<-start
			errors <- auth.CheckLogin(context.Background(), token)
		}()
	}
	close(start)
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	for range tokens {
		select {
		case err := <-errors:
			if err != nil {
				t.Errorf("CheckLogin: %v", err)
			}
		case <-deadline.C:
			t.Fatal("concurrent checks did not return while the worker was occupied")
		}
	}
}

// TestAuthRealPoolCoalescingAndClose verifies merged checks and shutdown across worker and listener barriers. TestAuthRealPoolCoalescingAndClose 验证校验合并，以及跨工作线程和监听器屏障的关闭等待。
func TestAuthRealPoolCoalescingAndClose(t *testing.T) {
	gate, eventGate := make(chan struct{}), make(chan struct{})
	release, releaseEvent := sync.OnceFunc(func() { close(gate) }), sync.OnceFunc(func() { close(eventGate) })
	auth, pool, logger, controlled := newMaintenanceLoadAuth(t, true, true, gate)
	// Release both barriers before cleanup tries to drain the Auth. 清理等待 Auth 排空前先释放两个屏障。
	t.Cleanup(release)
	t.Cleanup(releaseEvent)
	token, tokenKey, activeKey := seedMaintenanceToken(t, auth, "merged", "one", true)
	storage := auth.Storage()
	var renewals atomic.Int64
	eventStarted := make(chan struct{})
	// Share the occupied worker to verify listener rejection falls back without deadlocking shutdown. 共享已占用的工作线程，验证监听器被拒绝后回退且不阻塞关闭流程。
	auth.EventManager().SetAsyncPool(pool)
	auth.EventManager().RegisterFuncWithConfig(listener.EventRenew, func(*listener.EventData) {
		renewals.Add(1)
		close(eventStarted)
		<-eventGate
	}, listener.ListenerConfig{Async: true})
	startedAt := time.Now().Unix()
	if err := auth.CheckLogin(context.Background(), token); err != nil {
		t.Fatal(err)
	}
	select {
	case <-controlled.started:
	case <-time.After(5 * time.Second):
		t.Fatal("accepted worker did not start")
	}
	tokens := make([]string, 64)
	for i := range tokens {
		tokens[i] = token
	}
	checkMaintenanceBurst(t, auth, tokens)
	if pool.accepted.Load() != 1 || pool.rejected.Load() != 0 || renewals.Load() != 0 {
		t.Fatalf("accepted/rejected/renewals = %d/%d/%d, want 1/0/0", pool.accepted.Load(), pool.rejected.Load(), renewals.Load())
	}

	closed := make(chan struct{})
	go func() { auth.Close(); close(closed) }()
	waitForFlow(t, 5*time.Second, "Auth shutdown admission boundary", auth.IsClosed)
	select {
	case <-closed:
		t.Fatal("Close returned before accepted maintenance ran")
	default:
	}
	if err := auth.CheckLogin(context.Background(), token); err == nil {
		t.Fatal("closed Auth admitted a new check")
	}
	release()
	select {
	case <-eventStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("renewal did not reach its listener during shutdown")
	}
	select {
	case <-closed:
		t.Fatal("Close returned before the accepted listener completed")
	default:
	}
	if ttl, err := storage.TTL(context.Background(), tokenKey); err != nil || ttl <= time.Minute {
		t.Fatalf("accepted renewal TTL = %v, %v", ttl, err)
	}
	value, err := storage.Get(context.Background(), activeKey)
	if err != nil {
		t.Fatal(err)
	}
	if at, err := utils.ToInt64(value); err != nil || at < startedAt {
		t.Fatalf("merged activity = %v, %v", value, err)
	}
	releaseEvent()
	select {
	case <-closed:
	case <-time.After(5 * time.Second):
		t.Fatal("Close did not finish after both barriers opened")
	}
	if pool.accepted.Load() != 1 || pool.rejected.Load() != 1 || logger.errors.Load() != 0 || renewals.Load() != 1 {
		t.Fatalf("accepted/rejected/errors/renewals = %d/%d/%d/%d", pool.accepted.Load(), pool.rejected.Load(), logger.errors.Load(), renewals.Load())
	}
}

// TestAuthRealPoolSaturationAndRecovery checks bounded fallback and renewed admission after actual saturation. TestAuthRealPoolSaturationAndRecovery 验证真实池饱和后的有界回退及恢复接收。
func TestAuthRealPoolSaturationAndRecovery(t *testing.T) {
	for _, mode := range []struct {
		name          string
		active, renew bool
	}{{"activity", true, false}, {"renewal", false, true}, {"both", true, true}} {
		for _, scope := range []string{"same-account", "many-accounts"} {
			t.Run(mode.name+"/"+scope, func(t *testing.T) {
				openGate := make(chan struct{})
				close(openGate)
				auth, pool, logger, controlled := newMaintenanceLoadAuth(t, mode.active, mode.renew, openGate)
				blocked := make(chan struct{})
				release := sync.OnceFunc(func() { close(blocked) })
				t.Cleanup(release)
				// Occupy the real worker outside the observer so application admissions remain distinguishable. 在观察器外占用真实工作线程，单独统计应用任务接收。
				if err := controlled.Pool.Submit(func() { <-blocked }); err != nil {
					t.Fatal(err)
				}
				var renewals atomic.Int64
				auth.EventManager().RegisterFuncWithConfig(listener.EventRenew, func(*listener.EventData) {
					renewals.Add(1)
				}, listener.ListenerConfig{Async: true})
				tokens, tokenKeys, activeKeys := make([]string, 32), make([]string, 32), make([]string, 32)
				for i := range tokens {
					id := "overload"
					if scope == "many-accounts" {
						id = fmt.Sprintf("overload-%d", i)
					}
					tokens[i], tokenKeys[i], activeKeys[i] = seedMaintenanceToken(t, auth, id, fmt.Sprint(i), mode.active)
				}
				startedAt := time.Now().Unix()
				checkMaintenanceBurst(t, auth, tokens)
				if pool.accepted.Load() != 0 || pool.rejected.Load() != 32 || renewals.Load() != 0 {
					t.Fatalf("saturated accepted/rejected/renewals = %d/%d/%d", pool.accepted.Load(), pool.rejected.Load(), renewals.Load())
				}
				for i, tokenKey := range tokenKeys {
					if ttl, err := auth.Storage().TTL(context.Background(), tokenKey); err != nil || ttl <= 0 || ttl > time.Minute {
						t.Fatalf("optional renewal changed TTL = %v, %v", ttl, err)
					}
					if mode.active {
						value, err := auth.Storage().Get(context.Background(), activeKeys[i])
						if err != nil {
							t.Fatal(err)
						}
						if at, err := utils.ToInt64(value); err != nil || at < startedAt {
							t.Fatalf("fallback activity = %v, %v", value, err)
						}
						if ttl, err := auth.Storage().TTL(context.Background(), activeKeys[i]); err != nil || ttl <= 0 || ttl > time.Minute {
							t.Fatalf("fallback activity extended TTL = %v, %v", ttl, err)
						}
					}
				}

				// Worker completion can precede its return to the idle queue; retry until admission recovers. 工作线程完成与回到空闲队列存在间隔，重试直到恢复接收。
				release()
				waitForFlow(t, 5*time.Second, "maintenance admission recovery", func() bool {
					if err := auth.CheckLogin(context.Background(), tokens[0]); err != nil {
						t.Fatal(err)
					}
					return pool.accepted.Load() == 1
				})
				// Drain outside requests so assertions include both work and event completion. 请求结束后排空，断言同时覆盖任务及事件完成。
				auth.Close()
				wantRenewals := int64(0)
				if mode.renew {
					wantRenewals = 1
				}
				if renewals.Load() != wantRenewals || logger.errors.Load() != pool.rejected.Load() {
					t.Fatalf("renewals/errors/rejections = %d/%d/%d", renewals.Load(), logger.errors.Load(), pool.rejected.Load())
				}
			})
		}
	}
}
