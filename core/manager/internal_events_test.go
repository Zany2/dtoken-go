package manager

import (
	"testing"

	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/listener"
)

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
