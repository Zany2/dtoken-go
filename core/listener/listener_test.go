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
