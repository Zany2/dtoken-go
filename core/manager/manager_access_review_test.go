package manager

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/listener"
)

// TestManagerEmptyAccessMutationsValidateLogin verifies empty changes cannot skip token validation. TestManagerEmptyAccessMutationsValidateLogin 验证空变更不能跳过 Token 校验。
func TestManagerEmptyAccessMutationsValidateLogin(t *testing.T) {
	ctx := context.Background()
	for _, operation := range []struct {
		name string
		call func(*Manager, context.Context, string, []string) error
	}{
		{"add permissions", (*Manager).AddPermissionsByToken},
		{"remove permissions", (*Manager).RemovePermissionsByToken},
		{"add roles", (*Manager).AddRolesByToken},
		{"remove roles", (*Manager).RemoveRolesByToken},
	} {
		for _, state := range []struct {
			name string
			want error
		}{
			{"empty token", derror.ErrInvalidToken},
			{"missing token", derror.ErrInvalidToken},
			{"kickout", derror.ErrTokenKickout},
			{"replaced", derror.ErrTokenReplaced},
			{"disabled account", derror.ErrAccountDisabled},
			{"disabled device", derror.ErrDeviceDisabled},
			{"missing session", derror.ErrInvalidToken},
			{"active timeout", derror.ErrActiveTimeout},
		} {
			for _, input := range []struct {
				name   string
				values []string
			}{{"nil", nil}, {"empty", []string{}}, {"empty strings", []string{"", ""}}} {
				t.Run(operation.name+"/"+state.name+"/"+input.name, func(t *testing.T) {
					mgr := newTestManager(t, func(cfg *config.Config) {
						cfg.Timeout = 300
						cfg.ActiveTimeout = 60
					})
					id := "empty-access"
					token, err := mgr.Login(ctx, id, "web", "browser")
					if err != nil {
						t.Fatal(err)
					}
					pool := &managerQueuedMaintenancePool{}
					mgr.pool = pool
					t.Cleanup(pool.runAll)
					switch state.name {
					case "empty token":
						token = ""
					case "missing token":
						err = mgr.storage.Delete(ctx, mgr.getTokenKey(token))
					case "kickout":
						err = mgr.Kickout(ctx, token)
					case "replaced":
						err = mgr.Replace(ctx, token)
					case "disabled account":
						err = mgr.Disable(ctx, id, time.Minute)
					case "disabled device":
						err = mgr.DisableDevice(ctx, id, "web", time.Minute)
					case "missing session":
						err = mgr.storage.Delete(ctx, mgr.getSessionKey(id))
					case "active timeout":
						err = mgr.storage.Set(ctx, mgr.getActiveKey(token), time.Now().Add(-2*time.Minute).Unix(), 5*time.Minute)
					}
					if err != nil {
						t.Fatal(err)
					}
					events := registerTerminalLifecycleEvents(mgr)
					if err := operation.call(mgr, ctx, token, input.values); !errors.Is(err, state.want) {
						t.Fatalf("empty change error = %v, want %v", err, state.want)
					}
					timeouts := 0
					for _, event := range events() {
						if event.Event == listener.EventPermissionChange || event.Event == listener.EventRoleChange {
							t.Fatalf("empty change emitted %s", event.Event)
						}
						if event.Event == listener.EventActiveTimeout {
							timeouts++
						}
					}
					if state.name == "active timeout" && timeouts != 1 {
						t.Fatalf("timeout events = %d, want 1", timeouts)
					}
				})
			}
		}
	}
}

// TestManagerValidEmptyAccessMutationsRemainNoop checks valid empty changes preserve storage and maintenance state. TestManagerValidEmptyAccessMutationsRemainNoop 验证有效空变更不影响存储和维护状态。
func TestManagerValidEmptyAccessMutationsRemainNoop(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.Timeout = 300
		cfg.ActiveTimeout = 60
		cfg.AutoRenew = true
	})
	id := "valid-empty-access"
	token, err := mgr.Login(ctx, id, "web", "browser")
	if err != nil {
		t.Fatal(err)
	}
	if err := mgr.AddPermissions(ctx, id, []string{"profile:read"}); err != nil {
		t.Fatal(err)
	}
	if err := mgr.AddRoles(ctx, id, []string{"member"}); err != nil {
		t.Fatal(err)
	}
	pool := &managerQueuedMaintenancePool{}
	mgr.pool = pool
	t.Cleanup(pool.runAll)
	storage := &managerSetCountingStorage{Storage: mgr.storage}
	mgr.storage = storage
	keys := []string{mgr.getSessionKey(id), mgr.getTokenKey(token), mgr.getActiveKey(token)}
	before := captureTerminalLifecycle(t, mgr, ctx, keys)
	events := registerTerminalLifecycleEvents(mgr)
	for _, call := range []func(context.Context, string, []string) error{
		mgr.AddPermissionsByToken, mgr.RemovePermissionsByToken, mgr.AddRolesByToken, mgr.RemoveRolesByToken,
	} {
		for _, values := range [][]string{nil, {}, {"", ""}} {
			if err := call(ctx, token, values); err != nil {
				t.Fatalf("valid empty change error = %v", err)
			}
		}
	}
	if pool.taskCount() != 0 || len(events()) != 0 {
		t.Fatalf("empty changes scheduled %d tasks and emitted %d events", pool.taskCount(), len(events()))
	}
	for _, key := range keys {
		if count := storage.setCount(key); count != 0 {
			t.Fatalf("empty changes wrote key %q %d times", key, count)
		}
	}
	assertTerminalLifecycleUnchanged(t, mgr, ctx, before)
}
