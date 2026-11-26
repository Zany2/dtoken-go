package event

import (
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"

	adapter "github.com/Zany2/dtoken-go/contract"
)

// TestUnobservedEventsSkipAllocationAndScheduling checks all public trigger modes before payload copying. TestUnobservedEventsSkipAllocationAndScheduling 检查所有公开触发方式在载荷复制前跳过无观察需求的事件。
func TestUnobservedEventsSkipAllocationAndScheduling(t *testing.T) {
	for _, state := range []string{"empty", "unrelated", "disabled"} {
		t.Run(state, func(t *testing.T) {
			manager := NewManager()
			if state == "unrelated" {
				manager.RegisterFunc(EventLogout, func(*EventData) { t.Error("unrelated listener called") })
			} else if state == "disabled" {
				manager.RegisterFunc(EventAll, func(*EventData) { t.Error("disabled listener called") })
				manager.AddFilter(func(*EventData) bool { t.Error("disabled filter called"); return true })
				manager.EnableStats(true)
				manager.DisableEvent(EventLogin)
			}
			var submissions int
			manager.SetAsyncPool(&listenerTestPool{submit: func(func()) error { submissions++; return fmt.Errorf("unexpected submission") }})
			data := &EventData{Event: EventLogin, Extra: map[string]any{"permissions": []string{"read", "write"}}}
			for _, trigger := range []func(*EventData){manager.Trigger, manager.TriggerSync, manager.TriggerAsync} {
				if allocations := testing.AllocsPerRun(100, func() { trigger(data) }); allocations != 0 {
					t.Fatalf("unobserved event allocations = %v, want 0", allocations)
				}
			}
			manager.Wait()
			if manager.NeedsDispatch(EventLogin) || submissions != 0 || manager.GetStats().TotalTriggered != 0 || data.Timestamp != 0 {
				t.Fatal("skipped event allocated work, recorded statistics or mutated input")
			}
		})
	}
}

// TestEventObservationIncludesFiltersStatsAndLogs keeps non-listener side effects and wildcard delivery. TestEventObservationIncludesFiltersStatsAndLogs 保留非监听器副作用和通配事件投递。
func TestEventObservationIncludesFiltersStatsAndLogs(t *testing.T) {
	for _, observer := range []string{"filter", "statistics", "logger", "wildcard", "specific"} {
		t.Run(observer, func(t *testing.T) {
			logger := &listenerCaptureLogger{NopLogger: adapter.NewNopLogger()}
			manager := NewManager()
			calls := 0
			switch observer {
			case "filter":
				manager.AddFilter(func(*EventData) bool { calls++; return false })
			case "statistics":
				manager.EnableStats(true)
			case "logger":
				manager = NewManager(logger)
			case "wildcard":
				manager.RegisterFuncWithConfig(EventAll, func(*EventData) { calls++ }, ListenerConfig{Async: false})
			case "specific":
				manager.RegisterFuncWithConfig(EventLogin, func(*EventData) { calls++ }, ListenerConfig{Async: false})
			}
			if !manager.NeedsDispatch(EventLogin) {
				t.Fatal("observer was missed")
			}
			var queued []func()
			manager.SetAsyncPool(&listenerTestPool{submit: func(task func()) error { queued = append(queued, task); return nil }})
			manager.Trigger(&EventData{Event: EventLogin})
			manager.TriggerSync(&EventData{Event: EventLogin})
			manager.TriggerAsync(&EventData{Event: EventLogin})
			if len(queued) != 1 {
				t.Fatalf("queued dispatches = %d, want 1", len(queued))
			}
			queued[0]()
			manager.Wait()
			switch observer {
			case "statistics":
				if manager.GetStats().EventCounts[EventLogin] != 3 {
					t.Fatal("events without listeners lost statistics")
				}
			case "logger":
				if len(logger.infoMessages()) != 3 {
					t.Fatal("events without listeners lost logs")
				}
			default:
				if calls != 3 {
					t.Fatalf("observer calls = %d, want 3", calls)
				}
			}
		})
	}
}

// TestDispatchSnapshotsSurviveReconfiguration checks stable equal-priority order and callback reentrancy. TestDispatchSnapshotsSurviveReconfiguration 检查同优先级稳定顺序及回调重入。
func TestDispatchSnapshotsSurviveReconfiguration(t *testing.T) {
	manager := NewManager()
	var order []string
	add := func(event Event, id string, priority int) {
		manager.RegisterFuncWithConfig(event, func(*EventData) { order = append(order, id) }, ListenerConfig{ID: id, Priority: priority})
	}
	add(EventAll, "wildcard", 5)
	add(EventLogin, "first", 5)
	add(EventLogin, "second", 5)
	changed := false
	manager.RegisterFuncWithConfig(EventLogin, func(*EventData) {
		order = append(order, "control")
		if !changed {
			changed = true
			manager.Unregister("second")
			manager.ClearEvent(EventAll)
			add(EventLogin, "new", 20)
		}
	}, ListenerConfig{Priority: 10})
	add(EventAll, "late-wildcard", 15)
	manager.TriggerSync(&EventData{Event: EventLogin})
	if !reflect.DeepEqual(order, []string{"late-wildcard", "control", "first", "second", "wildcard"}) {
		t.Fatalf("in-flight dispatch changed: %v", order)
	}
	order = nil
	manager.TriggerSync(&EventData{Event: EventLogin})
	if !reflect.DeepEqual(order, []string{"new", "control", "first"}) {
		t.Fatalf("next dispatch ignored changes: %v", order)
	}
	manager.ClearEvent(EventLogin)
	add(EventAll, "remaining", 1)
	order = nil
	manager.TriggerSync(&EventData{Event: EventLogin})
	manager.TriggerSync(&EventData{Event: EventAll})
	if !reflect.DeepEqual(order, []string{"remaining", "remaining"}) {
		t.Fatalf("wildcard fallback or EventAll duplicated: %v", order)
	}
	manager.Unregister("remaining")
	if manager.NeedsDispatch(EventLogin) {
		t.Fatal("removed wildcard retained a dispatch observer")
	}
	manager.Clear()
	if len(manager.dispatch) != 0 {
		t.Fatal("Clear retained dispatch snapshots")
	}
}

// TestFilterSnapshotsSurviveReconfiguration keeps the current filter chain stable. TestFilterSnapshotsSurviveReconfiguration 保持本次过滤器链稳定。
func TestFilterSnapshotsSurviveReconfiguration(t *testing.T) {
	manager := NewManager()
	var filters []string
	manager.AddFilter(func(*EventData) bool {
		filters = append(filters, "first")
		manager.ClearFilters()
		manager.AddFilter(func(*EventData) bool { filters = append(filters, "new"); return false })
		return true
	})
	manager.AddFilter(func(*EventData) bool { filters = append(filters, "second"); return true })
	manager.EnableStats(true)
	manager.TriggerSync(&EventData{Event: EventLogin})
	manager.TriggerSync(&EventData{Event: EventLogin})
	if !reflect.DeepEqual(filters, []string{"first", "second", "new"}) || manager.GetStats().TotalTriggered != 1 {
		t.Fatalf("filter snapshot = %v, stats=%+v", filters, manager.GetStats())
	}
}

// TestDispatchSnapshotCosts keeps repeated snapshots allocation-free without caching arbitrary event names. TestDispatchSnapshotCosts 验证重复快照不分配内存，也不缓存任意事件名称。
func TestDispatchSnapshotCosts(t *testing.T) {
	manager := NewManager()
	manager.RegisterFunc(EventAll, func(*EventData) {})
	manager.RegisterFunc(EventLogin, func(*EventData) {})
	manager.AddFilter(func(*EventData) bool { return true })
	if allocations := testing.AllocsPerRun(100, func() { manager.snapshot(EventLogin) }); allocations != 0 {
		t.Fatalf("dispatch snapshot allocations = %v, want 0", allocations)
	}
	for i := 0; i < 100; i++ {
		manager.TriggerSync(&EventData{Event: Event(fmt.Sprintf("custom-%d", i))})
	}
	if len(manager.dispatch) != 1 {
		t.Fatalf("custom events grew dispatch cache to %d", len(manager.dispatch))
	}
}

// TestDispatchConcurrentRegistration exercises immutable snapshots during concurrent registration changes. TestDispatchConcurrentRegistration 覆盖注册并发变化时的不可变分发快照。
func TestDispatchConcurrentRegistration(t *testing.T) {
	manager := NewManager()
	var calls atomic.Int64
	manager.RegisterFunc(EventLogin, func(*EventData) { calls.Add(1) })
	var workers sync.WaitGroup
	workers.Add(2)
	go func() {
		defer workers.Done()
		for i := 0; i < 100; i++ {
			manager.TriggerSync(&EventData{Event: EventLogin})
		}
	}()
	go func() {
		defer workers.Done()
		for i := 0; i < 100; i++ {
			id := manager.RegisterFunc(EventAll, func(*EventData) {})
			manager.Unregister(id)
			manager.AddFilter(func(*EventData) bool { return true })
			manager.ClearFilters()
		}
	}()
	workers.Wait()
	if calls.Load() != 100 {
		t.Fatalf("stable listener calls = %d, want 100", calls.Load())
	}
}

// BenchmarkEventDispatch records allocation and time costs for empty, concrete and wildcard dispatch. BenchmarkEventDispatch 记录空分发、具体事件及通配分发的分配与耗时。
func BenchmarkEventDispatch(b *testing.B) {
	for _, count := range []int{0, 1, 16} {
		b.Run(fmt.Sprintf("listeners-%d", count), func(b *testing.B) {
			manager := NewManager()
			for i := 0; i < count; i++ {
				event := EventLogin
				if i%2 == 1 {
					event = EventAll
				}
				manager.RegisterFuncWithConfig(event, func(*EventData) {}, ListenerConfig{Priority: i % 3})
			}
			data := &EventData{Event: EventLogin, Extra: map[string]any{"permissions": []string{"read"}}}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				manager.TriggerSync(data)
			}
		})
	}
}
