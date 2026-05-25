package listener

import (
	"reflect"
	"testing"
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
