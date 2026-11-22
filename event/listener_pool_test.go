package event

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	adapter "github.com/Zany2/dtoken-go/contract"
)

type listenerTestPool struct {
	submit func(func()) error
}

var _ adapter.Pool = (*listenerTestPool)(nil)

// Submit delegates to the test scheduler. Submit 委托给测试调度器。
func (p *listenerTestPool) Submit(task func()) error { return p.submit(task) }

// Stop detects unexpected ownership transfer. Stop 检测意外的所有权转移。
func (p *listenerTestPool) Stop() { panic("event manager must not stop caller-owned pool") }

// Stats supplies the pool contract for test schedulers. Stats 为测试调度器提供池契约。
func (p *listenerTestPool) Stats() (int, int, float64) { return 0, 1, 0 }

// TestAsyncPoolTracksQueuedDispatchAndListeners verifies queued work, snapshots, and pool replacement remain tracked. TestAsyncPoolTracksQueuedDispatchAndListeners 验证排队任务、快照及替换池后任务仍受跟踪。
func TestAsyncPoolTracksQueuedDispatchAndListeners(t *testing.T) {
	manager := NewManager()
	queue := make(chan func(), 2)
	manager.SetAsyncPool(&listenerTestPool{submit: func(task func()) error {
		select {
		case queue <- task:
			return nil
		default:
			return fmt.Errorf("pool full")
		}
	}})
	var calls atomic.Int32
	observed := make(chan *EventData, 2)
	manager.RegisterFunc(EventLogin, func(data *EventData) {
		calls.Add(1)
		observed <- data
	})
	input := &EventData{Event: EventLogin, LoginID: "original"}
	before := time.Now().Unix()
	manager.TriggerAsync(input)
	after := time.Now().Unix()
	input.LoginID = "changed"
	if calls.Load() != 0 {
		t.Fatal("queued dispatch executed before its worker ran")
	}

	done := make(chan struct{})
	go func() { manager.Wait(); close(done) }()
	assertListenerSignalPending(t, done, "Wait ignored queued dispatch")
	(<-queue)()
	assertListenerSignalPending(t, done, "Wait ignored queued listener after dispatch completed")
	if calls.Load() != 0 {
		t.Fatal("asynchronous listener bypassed configured pool")
	}

	// Reset affects new submissions, while old queued work remains owned by the original pool. 重置只影响新提交，旧排队任务仍由原池执行。
	manager.SetAsyncPool(nil)
	manager.Trigger(&EventData{Event: EventLogin, LoginID: "new"})
	select {
	case got := <-observed:
		if got.LoginID != "new" {
			t.Fatalf("reset listener received %q, want new", got.LoginID)
		}
	case <-time.After(time.Second):
		t.Fatal("nil pool did not restore goroutine execution")
	}
	assertListenerSignalPending(t, done, "reset discarded previously queued listener")
	(<-queue)()
	waitForListenerSignal(t, done, "Wait did not finish after both pools drained")
	got := <-observed
	if got.LoginID != "original" || got.Timestamp < before || got.Timestamp > after || input.Timestamp != 0 {
		t.Fatalf("queued event lost submission snapshot: %+v, input timestamp=%d", got, input.Timestamp)
	}
	if calls.Load() != 2 {
		t.Fatalf("listener calls = %d, want 2", calls.Load())
	}
}

// TestAsyncPoolRejectionFallsBack verifies rejected dispatch and listener tasks run once and retain panic handling. TestAsyncPoolRejectionFallsBack 验证被拒绝的分发与监听任务回退执行一次且保留 panic 处理。
func TestAsyncPoolRejectionFallsBack(t *testing.T) {
	manager := NewManager()
	var submissions, calls, panics atomic.Int32
	manager.SetAsyncPool(&listenerTestPool{submit: func(func()) error {
		submissions.Add(1)
		return fmt.Errorf("pool full")
	}})
	manager.SetPanicHandler(func(Event, *EventData, any) { panics.Add(1) })
	manager.RegisterFunc(EventLogin, func(*EventData) {
		calls.Add(1)
		panic("listener failure")
	})
	manager.TriggerAsync(&EventData{Event: EventLogin})
	done := make(chan struct{})
	go func() { manager.Wait(); close(done) }()
	waitForListenerSignal(t, done, "fallback or panic leaked task accounting")
	if submissions.Load() != 2 || calls.Load() != 1 || panics.Load() != 1 {
		t.Fatalf("submissions/calls/panics = %d/%d/%d, want 2/1/1", submissions.Load(), calls.Load(), panics.Load())
	}
}

// TestAsyncPoolNestedTriggerSyncBypassesQueuedWorker verifies nested synchronous dispatch cannot queue behind its own worker. TestAsyncPoolNestedTriggerSyncBypassesQueuedWorker 验证嵌套同步分发不会排队等待自身工作协程。
func TestAsyncPoolNestedTriggerSyncBypassesQueuedWorker(t *testing.T) {
	manager := NewManager()
	queue := make(chan func(), 2)
	var submissions atomic.Int32
	manager.SetAsyncPool(&listenerTestPool{submit: func(task func()) error {
		submissions.Add(1)
		select {
		case queue <- task:
			return nil
		default:
			return fmt.Errorf("pool full")
		}
	}})
	nested := make(chan struct{})
	done := make(chan struct{})
	manager.RegisterFunc(EventLogout, func(*EventData) { close(nested) })
	manager.RegisterFunc(EventLogin, func(*EventData) {
		manager.TriggerSync(&EventData{Event: EventLogout})
		close(done)
	})
	manager.Trigger(&EventData{Event: EventLogin})
	go (<-queue)()
	waitForListenerSignal(t, done, "nested TriggerSync waited for occupied pool worker")
	waitForListenerSignal(t, nested, "nested listener did not complete")
	manager.Wait()
	if submissions.Load() != 1 {
		t.Fatalf("pool submissions = %d, want only the outer listener", submissions.Load())
	}
}

// TestAsyncPoolSaturationDoesNotBlockNestedDispatch verifies a busy nonblocking pool falls back for nested listeners. TestAsyncPoolSaturationDoesNotBlockNestedDispatch 验证非阻塞池忙碌时嵌套监听器可回退执行。
func TestAsyncPoolSaturationDoesNotBlockNestedDispatch(t *testing.T) {
	manager := NewManager()
	var busy atomic.Bool
	var rejected atomic.Int32
	workerDone := make(chan struct{})
	manager.SetAsyncPool(&listenerTestPool{submit: func(task func()) error {
		if !busy.CompareAndSwap(false, true) {
			rejected.Add(1)
			return fmt.Errorf("pool full")
		}
		go func() {
			defer close(workerDone)
			defer busy.Store(false)
			task()
		}()
		return nil
	}})
	called := make(chan struct{})
	manager.RegisterFunc(EventLogin, func(*EventData) { close(called) })
	manager.TriggerAsync(&EventData{Event: EventLogin})
	waitForListenerSignal(t, called, "nested listener was lost when dispatch occupied the pool")
	waitForListenerSignal(t, workerDone, "dispatch worker did not finish")
	manager.Wait()
	if rejected.Load() != 1 {
		t.Fatalf("rejected submissions = %d, want 1", rejected.Load())
	}
}

// TestAsyncPoolLeavesSynchronousListenersInline verifies synchronous callbacks do not submit pool work. TestAsyncPoolLeavesSynchronousListenersInline 验证同步回调不会提交池任务。
func TestAsyncPoolLeavesSynchronousListenersInline(t *testing.T) {
	manager := NewManager()
	manager.SetAsyncPool(&listenerTestPool{submit: func(func()) error {
		t.Fatal("synchronous listener submitted to pool")
		return nil
	}})
	called := false
	manager.RegisterFuncWithConfig(EventLogin, func(*EventData) { called = true }, ListenerConfig{Async: false})
	manager.Trigger(&EventData{Event: EventLogin})
	if !called {
		t.Fatal("synchronous listener did not complete inline")
	}
}
