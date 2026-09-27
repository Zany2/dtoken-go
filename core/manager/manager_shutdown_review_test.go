package manager

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/listener"
)

// managerNonReentrantPool detects nested submission without hanging a regression test. managerNonReentrantPool 检测嵌套提交，避免回归测试自身挂起。
type managerNonReentrantPool struct {
	mu     sync.Mutex
	busy   bool
	nested int
}

func (p *managerNonReentrantPool) Submit(task func()) error {
	p.mu.Lock()
	if p.busy {
		p.nested++
		p.mu.Unlock()
		return errors.New("nested submission would block a single-worker pool")
	}
	p.busy = true
	p.mu.Unlock()
	defer func() {
		p.mu.Lock()
		p.busy = false
		p.mu.Unlock()
	}()
	task()
	return nil
}

func (*managerNonReentrantPool) Stop() {}

func (*managerNonReentrantPool) Stats() (int, int, float64) { return 0, 1, 0 }

// managerObservedStopPool exposes resource release to the shutdown regression. managerObservedStopPool 为关闭回归用例提供资源释放信号。
type managerObservedStopPool struct {
	managerLifecycleTestPool
	stopped chan struct{}
}

func (p *managerObservedStopPool) Stop() {
	p.managerLifecycleTestPool.Stop()
	close(p.stopped)
}

// TestManagerRenewEventDoesNotResubmitToPool verifies async renewal events do not reenter their worker pool. TestManagerRenewEventDoesNotResubmitToPool 验证异步续期事件不会再次进入当前工作协程池。
func TestManagerRenewEventDoesNotResubmitToPool(t *testing.T) {
	ctx := context.Background()
	for _, forced := range []bool{false, true} {
		name := "automatic"
		if forced {
			name = "forced"
		}
		t.Run(name, func(t *testing.T) {
			mgr := newTestManager(t, func(cfg *config.Config) {
				cfg.AsyncEvent = true
				cfg.AutoRenew = true
				cfg.Timeout = 60
				cfg.RenewMaxRefresh = 60
			})
			pool := &managerNonReentrantPool{}
			mgr.pool = pool
			token, err := mgr.Login(ctx, "nested-renewal", "web")
			if err != nil {
				t.Fatal(err)
			}
			events := make(chan *listener.EventData, 2)
			mgr.GetEventManager().RegisterFuncWithConfig(listener.EventRenew, func(data *listener.EventData) {
				// The event must still run after the account lock is released. 事件仍须在账号锁释放后执行。
				unlock := mgr.lockLoginWrite(data.LoginID)
				unlock()
				events <- data
			}, listener.ListenerConfig{Async: false})
			if forced {
				err = mgr.LoginByToken(ctx, token)
			} else {
				err = mgr.CheckLogin(ctx, token)
			}
			if err != nil {
				t.Fatal(err)
			}
			mgr.asyncWG.Wait()
			pool.mu.Lock()
			nested := pool.nested
			pool.mu.Unlock()
			if nested != 0 || len(events) != 1 {
				t.Fatalf("nested submissions=%d renewal events=%d, want 0/1", nested, len(events))
			}
			if event := <-events; event.Token != token || event.LoginID != "nested-renewal" {
				t.Fatalf("renewal event = %+v", event)
			}
		})
	}
}

// TestManagerCloseDrainsRenewalEvents verifies accepted maintenance still reports completion during shutdown. TestManagerCloseDrainsRenewalEvents 验证已接收的维护任务在关闭期间仍报告完成事件。
func TestManagerCloseDrainsRenewalEvents(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.AsyncEvent = true
		cfg.Timeout = 60
	})
	pool := &managerQueuedMaintenancePool{}
	mgr.pool = pool
	t.Cleanup(pool.runAll)
	token, err := mgr.Login(ctx, "closing-renewal", "web")
	if err != nil {
		t.Fatal(err)
	}
	pool.runAll()
	events := make(chan *listener.EventData, 1)
	mgr.GetEventManager().RegisterFuncWithConfig(listener.EventRenew, func(data *listener.EventData) {
		events <- data
	}, listener.ListenerConfig{Async: true})
	if err := mgr.LoginByToken(ctx, token); err != nil {
		t.Fatal(err)
	}
	if pool.taskCount() != 1 {
		t.Fatal("expected one accepted renewal task")
	}
	closed := make(chan struct{})
	go func() { mgr.CloseManager(); close(closed) }()
	waitForManagerTest(t, time.Second, func() bool {
		mgr.asyncMu.Lock()
		defer mgr.asyncMu.Unlock()
		return mgr.asyncClosed
	})
	pool.runAll()
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("close did not drain renewal and listeners")
	}
	if len(events) != 1 || pool.taskCount() != 0 {
		t.Fatalf("drained events=%d queued tasks=%d, want 1/0", len(events), pool.taskCount())
	}
}

// TestManagerCloseKeepsPoolAliveForListeners verifies all listener dependencies outlive accepted callbacks. TestManagerCloseKeepsPoolAliveForListeners 验证已接收监听器完成前其依赖仍保持可用。
func TestManagerCloseKeepsPoolAliveForListeners(t *testing.T) {
	pool := &managerObservedStopPool{stopped: make(chan struct{})}
	logger := &managerLifecycleTestLogger{}
	mgr := newTestManagerWithRuntime(t, pool, logger)
	t.Cleanup(mgr.CloseManager)
	started, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var once sync.Once
	releaseListener := func() { once.Do(func() { close(release) }) }
	t.Cleanup(releaseListener)
	mgr.GetEventManager().RegisterFuncWithConfig(listener.EventLogin, func(*listener.EventData) {
		close(started)
		<-release
	}, listener.ListenerConfig{Async: true})
	mgr.triggerEvent(listener.EventLogin, "listener-close", "", "", "", nil)
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("listener did not start")
	}
	go func() { mgr.CloseManager(); close(done) }()
	waitForManagerTest(t, time.Second, func() bool {
		mgr.asyncMu.Lock()
		defer mgr.asyncMu.Unlock()
		return mgr.asyncClosed
	})
	select {
	case <-pool.stopped:
		t.Fatal("pool stopped while its listener was still running")
	case <-time.After(20 * time.Millisecond):
	}
	if pool.stopCount() != 0 || logger.closeCount() != 0 {
		t.Fatal("listener dependencies were released before listener completion")
	}
	releaseListener()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("close did not finish after listener completion")
	}
	if pool.stopCount() != 1 || logger.closeCount() != 1 {
		t.Fatal("owned dependencies were not released exactly once")
	}
}

// TestManagerAsyncTaskPanicIsContained verifies fallback and inline execution retain task accounting after panic. TestManagerAsyncTaskPanicIsContained 验证回退与内联执行在 panic 后仍正确完成任务计数。
func TestManagerAsyncTaskPanicIsContained(t *testing.T) {
	for name, pool := range map[string]adapter.Pool{
		"no pool": nil, "rejected submission": managerSubmitErrorPool{}, "inline": &managerNonReentrantPool{},
	} {
		t.Run(name, func(t *testing.T) {
			mgr := newTestManager(t, nil)
			mgr.pool = pool
			if !mgr.submitAsync("panic regression", func() { panic("component panic") }) {
				t.Fatal("task was not accepted")
			}
			completed := make(chan struct{})
			go func() { mgr.asyncWG.Wait(); close(completed) }()
			select {
			case <-completed:
			case <-time.After(time.Second):
				t.Fatal("panicking task leaked its completion count")
			}
			next := make(chan struct{})
			mgr.submitAsync("after panic", func() { close(next) })
			select {
			case <-next:
			case <-time.After(time.Second):
				t.Fatal("subsequent task did not run")
			}
		})
	}
}

// TestManagerLateMaintenanceCannotTouchRevokedToken covers validation finishing after account disable. TestManagerLateMaintenanceCannotTouchRevokedToken 覆盖账号封禁后才完成维护提交的旧校验请求。
func TestManagerLateMaintenanceCannotTouchRevokedToken(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.Timeout = 60
		cfg.ActiveTimeout = 60
		cfg.AutoRenew = true
		cfg.RenewMaxRefresh = 60
	})
	id := "late-disabled-maintenance"
	token, err := mgr.Login(ctx, id, "web")
	if err != nil {
		t.Fatal(err)
	}
	checkedRecord, _, _, err := mgr.inspectLoginToken(ctx, token, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := mgr.Disable(ctx, id, time.Hour); err != nil {
		t.Fatal(err)
	}
	if err := mgr.Untie(ctx, id); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.Login(ctx, id, "mobile"); err != nil {
		t.Fatal(err)
	}
	pool := &managerQueuedMaintenancePool{}
	mgr.pool = pool
	t.Cleanup(pool.runAll)
	before := captureTerminalLifecycle(t, mgr, ctx, []string{mgr.getTokenKey(token), mgr.getSessionKey(id), mgr.getActiveKey(token), mgr.getRenewKey(token)})
	events := registerManagerValidationEventCollector(mgr, listener.EventRenew)
	// Resume the old checked request only after disable and new login have completed. 封禁及新登录完成后才继续旧请求的维护提交。
	mgr.submitLoginMaintenance(ctx, token, checkedRecord, 60)
	if pool.taskCount() != 1 {
		t.Fatal("expected late maintenance to be queued")
	}
	pool.runAll()
	assertTerminalLifecycleUnchanged(t, mgr, ctx, before)
	if len(events()) != 0 {
		t.Fatal("retired token emitted a renewal event")
	}
}
