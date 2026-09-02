package listener

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
)

// TestManagerTriggerOrderFiltersAndStats verifies listener ordering, filters, and stats. TestManagerTriggerOrderFiltersAndStats 验证监听器顺序、过滤器和统计。
func TestManagerTriggerOrderFiltersAndStats(t *testing.T) {
	manager := NewManager()
	manager.EnableStats(true)

	var calls []string
	manager.RegisterFuncWithConfig(EventLogin, func(*EventData) {
		calls = append(calls, "low")
	}, ListenerConfig{Async: false, Priority: 1, ID: "low"})
	manager.RegisterFuncWithConfig(EventLogin, func(*EventData) {
		calls = append(calls, "high")
	}, ListenerConfig{Async: false, Priority: 10, ID: "high"})
	manager.RegisterFuncWithConfig(EventAll, func(*EventData) {
		calls = append(calls, "all")
	}, ListenerConfig{Async: false, Priority: 0, ID: "all"})

	manager.TriggerSync(&EventData{Event: EventLogin, LoginID: "u1"})
	if !reflect.DeepEqual(calls, []string{"high", "low", "all"}) {
		t.Fatalf("calls = %v, want [high low all]", calls)
	}
	stats := manager.GetStats()
	if stats.TotalTriggered != 1 || stats.EventCounts[EventLogin] != 1 {
		t.Fatalf("stats = %+v, want one login trigger", stats)
	}

	manager.AddFilter(func(data *EventData) bool {
		return data.LoginID != "blocked"
	})
	manager.TriggerSync(&EventData{Event: EventLogin, LoginID: "blocked"})
	if len(calls) != 3 {
		t.Fatalf("filtered event changed calls to %v, want unchanged", calls)
	}

	eventSwitch := NewManager()
	eventSwitch.RegisterFunc(EventLogin, func(*EventData) {})
	eventSwitch.DisableEvent(EventLogin)
	if eventSwitch.IsEventEnabled(EventLogin) {
		t.Fatal("IsEventEnabled(login) = true after DisableEvent, want false")
	}
	eventSwitch.EnableEvent(EventLogin)
	if !eventSwitch.IsEventEnabled(EventLogin) {
		t.Fatal("IsEventEnabled(login) = false after EnableEvent, want true")
	}
}

// TestManagerPanicHandlerRecovers verifies panic handling does not stop dispatch. TestManagerPanicHandlerRecovers 验证 panic 处理不会中断分发。
func TestManagerPanicHandlerRecovers(t *testing.T) {
	manager := NewManager()
	var recovered any
	manager.SetPanicHandler(func(_ Event, _ *EventData, value any) {
		recovered = value
	})

	manager.RegisterFuncWithConfig(EventLogin, func(*EventData) {
		panic("boom")
	}, ListenerConfig{Async: false})

	manager.TriggerSync(&EventData{Event: EventLogin})
	if recovered != "boom" {
		t.Fatalf("recovered = %v, want boom", recovered)
	}
}

// TestManagerDefensiveNoops verifies nil inputs do not panic or register listeners. TestManagerDefensiveNoops 验证空输入不会 panic 或注册监听器。
func TestManagerDefensiveNoops(t *testing.T) {
	manager := NewManager()
	manager.EnableStats(true)

	manager.Trigger(nil)
	manager.TriggerAsync(nil)
	manager.TriggerSync(nil)
	manager.AddFilter(nil)

	if id := manager.Register(EventLogin, nil); id != "" {
		t.Fatalf("Register(nil) id = %q, want empty", id)
	}
	if id := manager.RegisterFunc(EventLogin, nil); id != "" {
		t.Fatalf("RegisterFunc(nil) id = %q, want empty", id)
	}
	if id := manager.RegisterFuncWithConfig(EventLogin, nil, ListenerConfig{}); id != "" {
		t.Fatalf("RegisterFuncWithConfig(nil) id = %q, want empty", id)
	}
	if manager.Count() != 0 {
		t.Fatalf("Count() = %d, want 0", manager.Count())
	}

	stats := manager.GetStats()
	if stats.TotalTriggered != 0 {
		t.Fatalf("stats.TotalTriggered = %d, want 0", stats.TotalTriggered)
	}
}

// TestDisableKnownEventBeforeRegister verifies built-in events can be disabled before listeners are registered. TestDisableKnownEventBeforeRegister 验证内置事件可在注册监听器前禁用。
func TestDisableKnownEventBeforeRegister(t *testing.T) {
	manager := NewManager()
	manager.DisableEvent(EventNonceGenerate)
	if manager.IsEventEnabled(EventNonceGenerate) {
		t.Fatal("IsEventEnabled(nonceGenerate) = true, want false")
	}

	called := false
	manager.RegisterFuncWithConfig(EventNonceGenerate, func(*EventData) {
		called = true
	}, ListenerConfig{Async: false})
	manager.TriggerSync(&EventData{Event: EventNonceGenerate})
	if called {
		t.Fatal("disabled event listener was called")
	}
}

// TestFilterCanUseManager verifies filters run outside the manager lock. TestFilterCanUseManager 验证过滤器在管理器锁外执行。
func TestFilterCanUseManager(t *testing.T) {
	manager := NewManager()
	manager.RegisterFuncWithConfig(EventLogin, func(*EventData) {}, ListenerConfig{Async: false})
	manager.AddFilter(func(*EventData) bool {
		manager.RegisterFuncWithConfig(EventLogout, func(*EventData) {}, ListenerConfig{Async: false})
		return true
	})

	manager.TriggerSync(&EventData{Event: EventLogin})
	if manager.CountForEvent(EventLogout) != 1 {
		t.Fatalf("CountForEvent(logout) = %d, want 1", manager.CountForEvent(EventLogout))
	}
}

// TestTriggerSyncWaitsForAsyncListeners verifies synchronous triggering waits for its asynchronous listeners. TestTriggerSyncWaitsForAsyncListeners 验证同步触发会等待本次异步监听器完成。
func TestTriggerSyncWaitsForAsyncListeners(t *testing.T) {
	manager := NewManager()
	listenerStarted := make(chan struct{})
	listenerRelease := make(chan struct{})
	triggerDone := make(chan struct{})
	var releaseOnce sync.Once
	releaseListener := func() {
		releaseOnce.Do(func() {
			close(listenerRelease)
		})
	}
	defer releaseListener()

	manager.RegisterFuncWithConfig(EventLogin, func(*EventData) {
		close(listenerStarted)
		<-listenerRelease
	}, ListenerConfig{Async: true})

	go func() {
		manager.TriggerSync(&EventData{Event: EventLogin})
		close(triggerDone)
	}()

	waitForListenerSignal(t, listenerStarted, "async listener did not start")
	assertListenerSignalPending(t, triggerDone, "TriggerSync returned before its async listener completed")
	releaseListener()
	waitForListenerSignal(t, triggerDone, "TriggerSync did not return after its async listener completed")
}

// TestTriggerAsyncWaitTracksDispatchAndSnapshotsInput verifies asynchronous dispatch is tracked and owns its caller snapshot. TestTriggerAsyncWaitTracksDispatchAndSnapshotsInput 验证异步分发被 Wait 跟踪并持有调用时快照。
func TestTriggerAsyncWaitTracksDispatchAndSnapshotsInput(t *testing.T) {
	manager := NewManager()
	listenerStarted := make(chan struct{})
	listenerRelease := make(chan struct{})
	observed := make(chan listenerEventObservation, 1)
	waitDone := make(chan struct{})
	var releaseOnce sync.Once
	releaseListener := func() {
		releaseOnce.Do(func() {
			close(listenerRelease)
		})
	}
	defer releaseListener()

	manager.RegisterFuncWithConfig(EventLogin, func(data *EventData) {
		close(listenerStarted)
		<-listenerRelease
		observed <- observeListenerEvent(data, "state")
	}, ListenerConfig{Async: false})

	input := &EventData{
		Event:   EventLogin,
		LoginID: "original-login-id",
		Extra: map[string]any{
			"state": "original-state",
		},
	}
	manager.TriggerAsync(input)
	waitForListenerSignal(t, listenerStarted, "TriggerAsync listener did not start")

	input.LoginID = "caller-mutated-login-id"
	input.Extra["state"] = "caller-mutated-state"
	go func() {
		manager.Wait()
		close(waitDone)
	}()

	assertListenerSignalPending(t, waitDone, "Wait returned while TriggerAsync dispatch was still running")
	releaseListener()
	waitForListenerSignal(t, waitDone, "Wait did not return after TriggerAsync dispatch completed")

	got := <-observed
	if got.loginID != "original-login-id" || got.extraValue != "original-state" {
		t.Fatalf("listener observed loginID/extra = %q/%v, want original-login-id/original-state", got.loginID, got.extraValue)
	}
	if got.timestamp == 0 {
		t.Fatal("listener timestamp = 0, want trigger timestamp")
	}
	if input.Timestamp != 0 {
		t.Fatalf("caller input timestamp = %d, want unchanged zero value", input.Timestamp)
	}
}

// TestAsyncListenerCanReenterTriggerSync verifies an async listener can synchronously trigger another event without waiting on itself. TestAsyncListenerCanReenterTriggerSync 验证异步监听器可同步重入触发其他事件且不会等待自身。
func TestAsyncListenerCanReenterTriggerSync(t *testing.T) {
	manager := NewManager()
	nestedCalled := make(chan struct{})
	reentrantDone := make(chan struct{})

	manager.RegisterFuncWithConfig(EventLogout, func(*EventData) {
		close(nestedCalled)
	}, ListenerConfig{Async: true})
	manager.RegisterFuncWithConfig(EventLogin, func(*EventData) {
		manager.TriggerSync(&EventData{Event: EventLogout})
		close(reentrantDone)
	}, ListenerConfig{Async: true})

	manager.Trigger(&EventData{Event: EventLogin})
	waitForListenerSignal(t, nestedCalled, "reentrant TriggerSync did not run its listener")
	waitForListenerSignal(t, reentrantDone, "reentrant TriggerSync waited on its calling async listener")
	manager.Wait()
}

// TestListenersReceiveIndependentEventData verifies each listener receives isolated event fields and top-level Extra data. TestListenersReceiveIndependentEventData 验证每个监听器收到独立的事件字段和顶层 Extra 数据。
func TestListenersReceiveIndependentEventData(t *testing.T) {
	manager := NewManager()
	firstMutated := make(chan struct{})
	firstObserved := make(chan listenerEventObservation, 1)
	secondObserved := make(chan listenerEventObservation, 1)

	manager.RegisterFuncWithConfig(EventLogin, func(data *EventData) {
		data.LoginID = "first-mutated-login-id"
		data.Extra["state"] = "first-mutated-state"
		firstObserved <- observeListenerEvent(data, "state")
		close(firstMutated)
	}, ListenerConfig{Async: true, Priority: 10})
	manager.RegisterFuncWithConfig(EventLogin, func(data *EventData) {
		<-firstMutated
		secondObserved <- observeListenerEvent(data, "state")
	}, ListenerConfig{Async: true, Priority: 1})

	input := &EventData{
		Event:   EventLogin,
		LoginID: "original-login-id",
		Extra: map[string]any{
			"state": "original-state",
		},
	}
	manager.TriggerSync(input)

	first := <-firstObserved
	second := <-secondObserved
	if first.data == second.data || first.data == input || second.data == input {
		t.Fatal("listeners and caller received shared EventData pointers")
	}
	if first.loginID != "first-mutated-login-id" || first.extraValue != "first-mutated-state" {
		t.Fatalf("first listener observation = %q/%v, want its mutations", first.loginID, first.extraValue)
	}
	if second.loginID != "original-login-id" || second.extraValue != "original-state" {
		t.Fatalf("second listener observation = %q/%v, want original values", second.loginID, second.extraValue)
	}
	if input.LoginID != "original-login-id" || input.Extra["state"] != "original-state" {
		t.Fatalf("caller input was mutated: loginID/extra = %q/%v", input.LoginID, input.Extra["state"])
	}
}

// TestTriggerLogDoesNotExposeSensitiveEventData verifies event logs omit token and Extra secrets. TestTriggerLogDoesNotExposeSensitiveEventData 验证事件日志不会暴露 Token 和 Extra 密钥。
func TestTriggerLogDoesNotExposeSensitiveEventData(t *testing.T) {
	logger := &listenerCaptureLogger{NopLogger: adapter.NewNopLogger()}
	manager := NewManager(logger)
	manager.TriggerSync(&EventData{
		Event:    EventLogin,
		AuthType: "member",
		LoginID:  "login-id-secret-593ea4",
		Device:   "device-secret-6bc129",
		DeviceID: "device-id-secret-a91e48",
		Token:    "access-token-secret-3f617d",
		Extra: map[string]any{
			ExtraKeyRefreshToken: "refresh-token-secret-934cea",
			"credential":         "extra-secret-0be239",
		},
	})

	messages := strings.Join(logger.infoMessages(), "\n")
	if messages == "" {
		t.Fatal("event trigger did not emit an info log")
	}
	for _, secret := range []string{
		"login-id-secret-593ea4",
		"device-secret-6bc129",
		"device-id-secret-a91e48",
		"access-token-secret-3f617d",
		"refresh-token-secret-934cea",
		"extra-secret-0be239",
	} {
		if strings.Contains(messages, secret) {
			t.Fatalf("event log exposed secret %q: %s", secret, messages)
		}
	}
	if !strings.Contains(messages, "event=login") || !strings.Contains(messages, "authType=member") {
		t.Fatalf("event log lost non-sensitive context: %s", messages)
	}
}

// TestEventAndWildcardListenersSharePriorityOrder verifies specific and wildcard listeners use one priority order. TestEventAndWildcardListenersSharePriorityOrder 验证具体事件与通配监听器使用统一优先级顺序。
func TestEventAndWildcardListenersSharePriorityOrder(t *testing.T) {
	manager := NewManager()
	var calls []string

	manager.RegisterFuncWithConfig(EventLogin, func(*EventData) {
		calls = append(calls, "specific-low")
	}, ListenerConfig{Async: false, Priority: 1})
	manager.RegisterFuncWithConfig(EventAll, func(*EventData) {
		calls = append(calls, "wildcard-high")
	}, ListenerConfig{Async: false, Priority: 10})

	manager.TriggerSync(&EventData{Event: EventLogin})
	if !reflect.DeepEqual(calls, []string{"wildcard-high", "specific-low"}) {
		t.Fatalf("calls = %v, want [wildcard-high specific-low]", calls)
	}
}

// TestTriggerEventAllDoesNotDuplicateWildcardListeners verifies triggering the wildcard event dispatches each listener once. TestTriggerEventAllDoesNotDuplicateWildcardListeners 验证触发通配事件时每个监听器只分发一次。
func TestTriggerEventAllDoesNotDuplicateWildcardListeners(t *testing.T) {
	manager := NewManager()
	called := 0
	manager.RegisterFuncWithConfig(EventAll, func(*EventData) {
		called++
	}, ListenerConfig{Async: false})

	manager.TriggerSync(&EventData{Event: EventAll})
	if called != 1 {
		t.Fatalf("wildcard listener called %d times, want 1", called)
	}
}

// TestListenerIDsRejectDuplicatesAndSkipAutomaticCollisions verifies IDs are globally unique across explicit and automatic registration. TestListenerIDsRejectDuplicatesAndSkipAutomaticCollisions 验证显式与自动注册的 ID 在全局保持唯一。
func TestListenerIDsRejectDuplicatesAndSkipAutomaticCollisions(t *testing.T) {
	manager := NewManager()
	if id := manager.RegisterFuncWithConfig(EventLogin, func(*EventData) {}, ListenerConfig{ID: "listener_1"}); id != "listener_1" {
		t.Fatalf("first explicit ID = %q, want listener_1", id)
	}
	if id := manager.RegisterFuncWithConfig(EventLogout, func(*EventData) {}, ListenerConfig{ID: "listener_1"}); id != "" {
		t.Fatalf("duplicate explicit ID = %q, want empty", id)
	}
	if id := manager.RegisterFunc(EventLogout, func(*EventData) {}); id != "listener_2" {
		t.Fatalf("automatic ID = %q, want listener_2 after listener_1 collision", id)
	}
	if manager.Count() != 2 {
		t.Fatalf("Count() = %d, want 2 unique listeners", manager.Count())
	}
}

// TestEventStateOverridesSupportWildcardAndCustomEvents verifies disabling one event still works with wildcard listeners and leaves unrelated custom events enabled. TestEventStateOverridesSupportWildcardAndCustomEvents 验证存在通配监听器时仍可禁用单个事件，且无关自定义事件保持启用。
func TestEventStateOverridesSupportWildcardAndCustomEvents(t *testing.T) {
	manager := NewManager()
	customEvent := Event("customEvent")
	var calls []Event

	manager.RegisterFuncWithConfig(EventAll, func(data *EventData) {
		calls = append(calls, data.Event)
	}, ListenerConfig{Async: false, ID: "wildcard"})

	manager.DisableEvent(EventLogin)
	if manager.IsEventEnabled(EventLogin) {
		t.Fatal("IsEventEnabled(login) = true after DisableEvent(login)")
	}
	if !manager.IsEventEnabled(EventLogout) || !manager.IsEventEnabled(customEvent) {
		t.Fatal("DisableEvent(login) changed an unrelated built-in or custom event")
	}

	manager.TriggerSync(&EventData{Event: EventLogin})
	manager.TriggerSync(&EventData{Event: customEvent})
	if !reflect.DeepEqual(calls, []Event{customEvent}) {
		t.Fatalf("wildcard calls = %v, want [%s]", calls, customEvent)
	}

	manager.EnableEvent(EventAll)
	manager.DisableEvent(EventLogout)
	if manager.IsEventEnabled(EventLogout) || !manager.IsEventEnabled(customEvent) {
		t.Fatal("specific disable did not override the EventAll allow entry")
	}

	manager.EnableEvent(EventLogin)
	if !manager.IsEventEnabled(EventLogin) || manager.IsEventEnabled(customEvent) {
		t.Fatal("EnableEvent(login) did not replace the event allow-list")
	}

	manager.EnableEvent()
	if !manager.IsEventEnabled(customEvent) {
		t.Fatal("EnableEvent() did not restore all events")
	}

	manager.DisableEvent(EventAll)
	if manager.IsEventEnabled(EventLogin) || manager.IsEventEnabled(customEvent) {
		t.Fatal("DisableEvent(EventAll) did not disable all events")
	}
}

// TestFilterPanicIsRecoveredAndStopsDispatch verifies filter panics are routed to the panic handler without dispatching listeners. TestFilterPanicIsRecoveredAndStopsDispatch 验证过滤器 panic 会交给 panic 处理器且不会继续分发监听器。
func TestFilterPanicIsRecoveredAndStopsDispatch(t *testing.T) {
	manager := NewManager()
	manager.EnableStats(true)
	recovered := make(chan any, 2)
	listenerCalled := make(chan struct{}, 1)
	manager.SetPanicHandler(func(_ Event, _ *EventData, value any) {
		recovered <- value
	})
	manager.AddFilter(func(*EventData) bool {
		panic("filter-boom")
	})
	manager.RegisterFuncWithConfig(EventLogin, func(*EventData) {
		listenerCalled <- struct{}{}
	}, ListenerConfig{Async: false})

	manager.TriggerSync(&EventData{Event: EventLogin})
	manager.TriggerAsync(&EventData{Event: EventLogin})
	manager.Wait()

	for i := 0; i < 2; i++ {
		if value := <-recovered; value != "filter-boom" {
			t.Fatalf("recovered panic = %v, want filter-boom", value)
		}
	}
	select {
	case <-listenerCalled:
		t.Fatal("listener ran after filter panic")
	default:
	}
	if stats := manager.GetStats(); stats.TotalTriggered != 0 {
		t.Fatalf("stats.TotalTriggered = %d after rejected events, want 0", stats.TotalTriggered)
	}
}

// TestPanicHandlerPanicIsContained verifies a failing custom panic handler cannot escape an asynchronous listener. TestPanicHandlerPanicIsContained 验证自定义 panic 处理器自身故障不会逃逸异步监听器。
func TestPanicHandlerPanicIsContained(t *testing.T) {
	logger := &listenerCaptureLogger{NopLogger: adapter.NewNopLogger()}
	manager := NewManager(logger)
	manager.SetPanicHandler(func(Event, *EventData, any) {
		panic("panic-handler-boom")
	})
	manager.RegisterFuncWithConfig(EventLogin, func(*EventData) {
		panic("listener-boom")
	}, ListenerConfig{Async: true})

	manager.TriggerSync(&EventData{Event: EventLogin})

	messages := strings.Join(logger.errorMessages(), "\n")
	if !strings.Contains(messages, "panic handler panic recovered") || !strings.Contains(messages, "panic-handler-boom") {
		t.Fatalf("panic handler failure was not logged: %s", messages)
	}
}

// TestTriggerSyncDoesNotWaitForUnrelatedAsyncWork verifies scoped synchronous dispatch does not wait for an earlier asynchronous listener. TestTriggerSyncDoesNotWaitForUnrelatedAsyncWork 验证同步分发不会等待更早启动的无关异步监听器。
func TestTriggerSyncDoesNotWaitForUnrelatedAsyncWork(t *testing.T) {
	manager := NewManager()
	unrelatedStarted := make(chan struct{})
	unrelatedRelease := make(chan struct{})
	triggerDone := make(chan struct{})
	var releaseOnce sync.Once
	releaseUnrelated := func() {
		releaseOnce.Do(func() {
			close(unrelatedRelease)
		})
	}
	defer releaseUnrelated()

	manager.RegisterFuncWithConfig(EventLogout, func(*EventData) {
		close(unrelatedStarted)
		<-unrelatedRelease
	}, ListenerConfig{Async: true})
	manager.RegisterFuncWithConfig(EventLogin, func(*EventData) {}, ListenerConfig{Async: false})

	manager.Trigger(&EventData{Event: EventLogout})
	waitForListenerSignal(t, unrelatedStarted, "unrelated async listener did not start")
	go func() {
		manager.TriggerSync(&EventData{Event: EventLogin})
		close(triggerDone)
	}()
	waitForListenerSignal(t, triggerDone, "TriggerSync waited for unrelated async work")

	releaseUnrelated()
	manager.Wait()
}

// TestWaitReleasesAllConcurrentWaiters verifies every lifecycle waiter is notified after asynchronous work drains. TestWaitReleasesAllConcurrentWaiters 验证异步任务排空后所有生命周期等待方都会被唤醒。
func TestWaitReleasesAllConcurrentWaiters(t *testing.T) {
	manager := NewManager()
	dispatchStarted := make(chan struct{})
	dispatchRelease := make(chan struct{})
	waitDone := make(chan struct{}, 3)
	var releaseOnce sync.Once
	releaseDispatch := func() {
		releaseOnce.Do(func() {
			close(dispatchRelease)
		})
	}
	defer releaseDispatch()

	manager.RegisterFuncWithConfig(EventLogin, func(*EventData) {
		close(dispatchStarted)
		<-dispatchRelease
	}, ListenerConfig{Async: false})
	manager.TriggerAsync(&EventData{Event: EventLogin})
	waitForListenerSignal(t, dispatchStarted, "asynchronous dispatch did not start")

	for i := 0; i < cap(waitDone); i++ {
		go func() {
			manager.Wait()
			waitDone <- struct{}{}
		}()
	}
	assertListenerSignalPending(t, waitDone, "Wait returned before asynchronous dispatch completed")

	releaseDispatch()
	for i := 0; i < cap(waitDone); i++ {
		waitForListenerSignal(t, waitDone, "a concurrent Wait caller was not released")
	}
}

// TestEventDataStringAndListenerFunc verifies payload formatting and function listener behavior. TestEventDataStringAndListenerFunc 验证事件载荷格式化和函数监听器行为。
func TestEventDataStringAndListenerFunc(t *testing.T) {
	var nilData *EventData
	if got := nilData.String(); got != "Event<nil>" {
		t.Fatalf("nil EventData.String() = %q, want Event<nil>", got)
	}

	data := &EventData{
		Event:     EventLogin,
		AuthType:  "member",
		LoginID:   "user-1",
		Device:    "web",
		DeviceID:  "browser-1",
		Timestamp: 123,
	}
	want := "Event{type=login,AuthType=member, loginID=user-1, device=web, deviceId=browser-1, timestamp=123}"
	if got := data.String(); got != want {
		t.Fatalf("EventData.String() = %q, want %q", got, want)
	}

	ListenerFunc(nil).OnEvent(nil)
	called := false
	ListenerFunc(func(got *EventData) {
		called = got == data
	}).OnEvent(data)
	if !called {
		t.Fatal("ListenerFunc did not invoke its callback with the event data")
	}
}

// TestManagerRegistrationAndCleanupAPIs verifies registration metadata and cleanup operations. TestManagerRegistrationAndCleanupAPIs 验证注册元数据及清理操作。
func TestManagerRegistrationAndCleanupAPIs(t *testing.T) {
	manager := NewManager()
	if manager.HasListeners(EventLogin) {
		t.Fatal("empty manager reports login listeners")
	}

	firstID := manager.RegisterFuncWithConfig(EventLogin, func(*EventData) {}, ListenerConfig{Async: false, ID: "first"})
	secondID := manager.RegisterFuncWithConfig(EventLogin, func(*EventData) {}, ListenerConfig{Async: false, ID: "second"})
	otherID := manager.Register(EventLogout, ListenerFunc(func(*EventData) {}))
	if firstID != "first" || secondID != "second" || otherID == "" {
		t.Fatalf("registered IDs = %q, %q, %q", firstID, secondID, otherID)
	}
	if !manager.HasListeners(EventLogin) || manager.Count() != 3 || manager.CountForEvent(EventLogin) != 2 {
		t.Fatalf("listener counts = total %d, login %d", manager.Count(), manager.CountForEvent(EventLogin))
	}
	if got := manager.GetListenerIDs(EventLogin); !reflect.DeepEqual(got, []string{"first", "second"}) {
		t.Fatalf("GetListenerIDs(login) = %v, want [first second]", got)
	}

	events := manager.GetAllEvents()
	seen := make(map[Event]bool, len(events))
	for _, event := range events {
		seen[event] = true
	}
	if !seen[EventLogin] || !seen[EventLogout] || len(seen) != 2 {
		t.Fatalf("GetAllEvents() = %v, want login and logout", events)
	}

	if manager.Unregister("missing") {
		t.Fatal("Unregister(missing) = true, want false")
	}
	if !manager.Unregister(firstID) || manager.Unregister(firstID) {
		t.Fatal("Unregister did not remove exactly one listener")
	}
	if got := manager.GetListenerIDs(EventLogin); !reflect.DeepEqual(got, []string{"second"}) {
		t.Fatalf("GetListenerIDs(login) after unregister = %v, want [second]", got)
	}

	manager.ClearEvent(EventLogin)
	if manager.HasListeners(EventLogin) || manager.Count() != 1 {
		t.Fatalf("ClearEvent(login) left listeners: total=%d, hasLogin=%v", manager.Count(), manager.HasListeners(EventLogin))
	}
	manager.Clear()
	if manager.Count() != 0 || len(manager.GetAllEvents()) != 0 || manager.HasListeners(EventLogout) {
		t.Fatalf("Clear() did not remove all listeners")
	}
}

// TestManagerStatsCopyAndReset verifies statistics toggling, defensive copies, and reset behavior. TestManagerStatsCopyAndReset 验证统计开关、防御性副本和重置行为。
func TestManagerStatsCopyAndReset(t *testing.T) {
	manager := NewManager()
	manager.EnableStats(true)
	manager.TriggerSync(&EventData{Event: EventLogin})

	stats := manager.GetStats()
	if stats.TotalTriggered != 1 || stats.EventCounts[EventLogin] != 1 || stats.LastTriggered[EventLogin].IsZero() {
		t.Fatalf("stats = %+v, want one login trigger with timestamp", stats)
	}
	stats.EventCounts[EventLogin] = 99
	delete(stats.LastTriggered, EventLogin)
	if got := manager.GetStats(); got.EventCounts[EventLogin] != 1 || got.LastTriggered[EventLogin].IsZero() {
		t.Fatalf("GetStats returned internal map references: %+v", got)
	}

	manager.ResetStats()
	stats = manager.GetStats()
	if stats.TotalTriggered != 0 || len(stats.EventCounts) != 0 || len(stats.LastTriggered) != 0 {
		t.Fatalf("stats after ResetStats() = %+v, want empty stats", stats)
	}

	manager.EnableStats(false)
	manager.TriggerSync(&EventData{Event: EventLogin})
	if got := manager.GetStats(); got.TotalTriggered != 0 || len(got.EventCounts) != 0 {
		t.Fatalf("stats collected while disabled: %+v", got)
	}
}

// TestManagerClearFiltersAndDirectTrigger verifies filters can be cleared and Trigger tracks default asynchronous listeners. TestManagerClearFiltersAndDirectTrigger 验证过滤器可清除且 Trigger 会跟踪默认异步监听器。
func TestManagerClearFiltersAndDirectTrigger(t *testing.T) {
	manager := NewManager()
	blocked := true
	manager.AddFilter(func(*EventData) bool { return !blocked })
	called := make(chan struct{}, 1)
	manager.Register(EventLogin, ListenerFunc(func(*EventData) {
		called <- struct{}{}
	}))

	manager.Trigger(&EventData{Event: EventLogin})
	manager.Wait()
	select {
	case <-called:
		t.Fatal("listener ran while filter rejected the event")
	default:
	}

	manager.ClearFilters()
	blocked = false
	manager.Trigger(&EventData{Event: EventLogin})
	manager.Wait()
	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatal("listener did not run after filters were cleared")
	}
}

type listenerEventObservation struct {
	data       *EventData
	loginID    string
	extraValue any
	timestamp  int64
}

func observeListenerEvent(data *EventData, extraKey string) listenerEventObservation {
	return listenerEventObservation{
		data:       data,
		loginID:    data.LoginID,
		extraValue: data.Extra[extraKey],
		timestamp:  data.Timestamp,
	}
}

func waitForListenerSignal(t *testing.T, signal <-chan struct{}, failure string) {
	t.Helper()

	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatal(failure)
	}
}

func assertListenerSignalPending(t *testing.T, signal <-chan struct{}, failure string) {
	t.Helper()

	select {
	case <-signal:
		t.Fatal(failure)
	case <-time.After(50 * time.Millisecond):
	}
}

type listenerCaptureLogger struct {
	*adapter.NopLogger
	mu       sync.Mutex
	messages []string
	errors   []string
}

func (l *listenerCaptureLogger) Infof(format string, values ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.messages = append(l.messages, fmt.Sprintf(format, values...))
}

func (l *listenerCaptureLogger) infoMessages() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.messages...)
}

func (l *listenerCaptureLogger) Errorf(format string, values ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.errors = append(l.errors, fmt.Sprintf(format, values...))
}

func (l *listenerCaptureLogger) errorMessages() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.errors...)
}
