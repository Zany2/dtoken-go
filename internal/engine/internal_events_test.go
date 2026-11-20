package engine

import (
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/config"
	listener "github.com/Zany2/dtoken-go/event"
)

// TestManagerEmitHonorsShutdown verifies public event admission while preserving already accepted work. TestManagerEmitHonorsShutdown 验证公开事件入口遵守关闭状态，同时保留已接收任务。
func TestManagerEmitHonorsShutdown(t *testing.T) {
	for _, async := range []bool{false, true} {
		name := "synchronous"
		if async {
			name = "asynchronous"
		}
		t.Run(name, func(t *testing.T) {
			mgr := newTestManager(t, func(cfg *config.Config) { cfg.AsyncEvent = async })
			pool := &managerQueuedMaintenancePool{}
			mgr.pool = pool
			t.Cleanup(pool.runAll)
			var tokens []string
			mgr.GetEventManager().RegisterFuncWithConfig(listener.EventLogin, func(data *listener.EventData) {
				tokens = append(tokens, data.Token)
			}, listener.ListenerConfig{Async: false})
			mgr.Emit(listener.EventLogin, "emit-close", "", "", "accepted", nil)

			// Hold shutdown open with accepted work to check both closing and closed states. 用已接收任务保持关闭流程未完成，分别检查关闭中和关闭后状态。
			if !mgr.submitAsync("hold shutdown", func() {}) {
				t.Fatal("manager rejected work before shutdown")
			}
			done := make(chan struct{})
			go func() { mgr.CloseManager(); close(done) }()
			waitForManagerTest(t, time.Second, func() bool {
				mgr.asyncMu.Lock()
				defer mgr.asyncMu.Unlock()
				return mgr.asyncClosed
			})
			mgr.Emit(listener.EventLogin, "emit-close", "", "", "during-close", nil)
			pool.runAll()
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("shutdown did not drain accepted work")
			}
			mgr.Emit(listener.EventLogin, "emit-close", "", "", "after-close", nil)
			mgr.GetEventManager().Wait()
			if len(tokens) != 1 || tokens[0] != "accepted" {
				t.Fatalf("delivered tokens = %v, want only accepted", tokens)
			}
		})
	}

	var missing *Manager
	missing.Emit(listener.EventLogin, "", "", "", "", nil)
}

// TestAsyncEventSnapshotsSlicePayloadBeforePoolExecution verifies queued events own built-in slice payloads at submission time. TestAsyncEventSnapshotsSlicePayloadBeforePoolExecution 验证排队事件在提交时即持有内置切片载荷快照。
func TestAsyncEventSnapshotsSlicePayloadBeforePoolExecution(t *testing.T) {
	pool := &managerDeferredEventPool{}
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.AsyncEvent = true
	})
	mgr.pool = pool

	observed := make(chan string, 1)
	mgr.GetEventManager().RegisterFuncWithConfig(listener.EventPermissionCheck, func(data *listener.EventData) {
		observed <- data.Extra[listener.ExtraKeyPermissions].([]string)[0]
	}, listener.ListenerConfig{Async: false})

	permissions := []string{"read"}
	mgr.triggerEvent(listener.EventPermissionCheck, "user-1", "", "", "", map[string]any{
		listener.ExtraKeyPermissions: permissions,
	})
	permissions[0] = "write"

	pool.run(t)
	if permission := <-observed; permission != "read" {
		t.Fatalf("listener observed permission = %q, want read", permission)
	}
}

type managerDeferredEventPool struct {
	task func()
}

func (p *managerDeferredEventPool) Submit(task func()) error {
	p.task = task
	return nil
}

func (*managerDeferredEventPool) Stop() {}

func (*managerDeferredEventPool) Stats() (int, int, float64) { return 0, 1, 0 }

func (p *managerDeferredEventPool) run(t *testing.T) {
	t.Helper()
	if p.task == nil {
		t.Fatal("event task was not submitted")
	}
	task := p.task
	p.task = nil
	task()
}
