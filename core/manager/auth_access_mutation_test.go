package manager

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/listener"
)

// managerSetCountingStorage counts writes while forwarding storage operations. managerSetCountingStorage 统计写入次数并转发存储操作。
type managerSetCountingStorage struct {
	adapter.Storage
	mu   sync.Mutex     // mu protects write counters. mu 保护写入计数。
	sets map[string]int // sets stores write counts by key. sets 按键存储写入次数。
}

// Set records one write and delegates to the wrapped storage. Set 记录一次写入并转发到被包装存储。
func (s *managerSetCountingStorage) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	s.mu.Lock()
	if s.sets == nil {
		s.sets = make(map[string]int)
	}
	s.sets[key]++
	s.mu.Unlock()
	return s.Storage.Set(ctx, key, value, expiration)
}

// setCount returns the write count for one key. setCount 返回指定键的写入次数。
func (s *managerSetCountingStorage) setCount(key string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sets[key]
}

// TestManagerAccessMutationsSkipUnchangedSession verifies no-op access updates do not persist or emit change events. TestManagerAccessMutationsSkipUnchangedSession 验证无效权限与角色更新不会持久化或触发变更事件。
func TestManagerAccessMutationsSkipUnchangedSession(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.AutoRenew = false
	})

	token, err := mgr.Login(ctx, "access-noop", "web", "browser")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if err = mgr.AddPermissions(ctx, "access-noop", []string{"profile:read"}); err != nil {
		t.Fatalf("AddPermissions() error = %v", err)
	}
	if err = mgr.AddRoles(ctx, "access-noop", []string{"member"}); err != nil {
		t.Fatalf("AddRoles() error = %v", err)
	}

	changeEvents := 0
	mgr.GetEventManager().RegisterFuncWithConfig(listener.EventAll, func(data *listener.EventData) {
		if data.Event == listener.EventPermissionChange || data.Event == listener.EventRoleChange {
			changeEvents++
		}
	}, listener.ListenerConfig{Async: false})

	storage := &managerSetCountingStorage{Storage: mgr.storage}
	mgr.storage = storage

	operations := []struct {
		name string
		run  func() error
	}{
		{name: "add existing permission", run: func() error {
			return mgr.AddPermissions(ctx, "access-noop", []string{"profile:read"})
		}},
		{name: "remove missing permission", run: func() error {
			return mgr.RemovePermissions(ctx, "access-noop", []string{"profile:write"})
		}},
		{name: "add existing permission by token", run: func() error {
			return mgr.AddPermissionsByToken(ctx, token, []string{"profile:read"})
		}},
		{name: "remove missing permission by token", run: func() error {
			return mgr.RemovePermissionsByToken(ctx, token, []string{"profile:write"})
		}},
		{name: "add existing role", run: func() error {
			return mgr.AddRoles(ctx, "access-noop", []string{"member"})
		}},
		{name: "remove missing role", run: func() error {
			return mgr.RemoveRoles(ctx, "access-noop", []string{"admin"})
		}},
		{name: "add existing role by token", run: func() error {
			return mgr.AddRolesByToken(ctx, token, []string{"member"})
		}},
		{name: "remove missing role by token", run: func() error {
			return mgr.RemoveRolesByToken(ctx, token, []string{"admin"})
		}},
	}
	for _, operation := range operations {
		if err = operation.run(); err != nil {
			t.Fatalf("%s error = %v", operation.name, err)
		}
	}

	if got := storage.setCount(mgr.getSessionKey("access-noop")); got != 0 {
		t.Fatalf("session write count = %d, want 0", got)
	}
	if changeEvents != 0 {
		t.Fatalf("access change event count = %d, want 0", changeEvents)
	}
}

// TestManagerAccessMutationEventsReportActualChanges verifies mixed updates report only effective changes. TestManagerAccessMutationEventsReportActualChanges 验证混合更新事件仅上报实际变更。
func TestManagerAccessMutationEventsReportActualChanges(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.AutoRenew = false
	})

	token, err := mgr.Login(ctx, "access-event", "web", "browser")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if err = mgr.AddPermissions(ctx, "access-event", []string{"profile:read"}); err != nil {
		t.Fatalf("AddPermissions() setup error = %v", err)
	}
	if err = mgr.AddRoles(ctx, "access-event", []string{"member"}); err != nil {
		t.Fatalf("AddRoles() setup error = %v", err)
	}

	var changes []*listener.EventData
	mgr.GetEventManager().RegisterFuncWithConfig(listener.EventAll, func(data *listener.EventData) {
		if data.Event == listener.EventPermissionChange || data.Event == listener.EventRoleChange {
			changes = append(changes, data)
		}
	}, listener.ListenerConfig{Async: false})

	if err = mgr.AddPermissions(ctx, "access-event", []string{"profile:read", "profile:write"}); err != nil {
		t.Fatalf("AddPermissions() error = %v", err)
	}
	if err = mgr.RemovePermissionsByToken(ctx, token, []string{"profile:missing", "profile:read"}); err != nil {
		t.Fatalf("RemovePermissionsByToken() error = %v", err)
	}
	if err = mgr.AddRolesByToken(ctx, token, []string{"member", "admin"}); err != nil {
		t.Fatalf("AddRolesByToken() error = %v", err)
	}
	if err = mgr.RemoveRoles(ctx, "access-event", []string{"missing", "member"}); err != nil {
		t.Fatalf("RemoveRoles() error = %v", err)
	}

	want := []struct {
		event  listener.Event
		action string
		key    string
		values []string
	}{
		{listener.EventPermissionChange, listener.ActionAdd, listener.ExtraKeyPermissions, []string{"profile:write"}},
		{listener.EventPermissionChange, listener.ActionRemove, listener.ExtraKeyPermissions, []string{"profile:read"}},
		{listener.EventRoleChange, listener.ActionAdd, listener.ExtraKeyRoles, []string{"admin"}},
		{listener.EventRoleChange, listener.ActionRemove, listener.ExtraKeyRoles, []string{"member"}},
	}
	if len(changes) != len(want) {
		t.Fatalf("access change event count = %d, want %d", len(changes), len(want))
	}
	for i, expectation := range want {
		if changes[i].Event != expectation.event || changes[i].Extra[listener.ExtraKeyAction] != expectation.action {
			t.Fatalf("access change event %d = %+v, want event=%s action=%s", i, changes[i], expectation.event, expectation.action)
		}
		if got := changes[i].Extra[expectation.key]; !reflect.DeepEqual(got, expectation.values) {
			t.Fatalf("access change event %d values = %v, want %v", i, got, expectation.values)
		}
	}
}
