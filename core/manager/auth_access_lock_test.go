package manager

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/listener"
)

// TestManagerAccessWritesHandleActiveTimeout verifies all token-based access writes release locks and publish timeout events. TestManagerAccessWritesHandleActiveTimeout 验证所有按 Token 的权限角色写入都会释放锁并发布超时事件。
func TestManagerAccessWritesHandleActiveTimeout(t *testing.T) {
	ctx := context.Background()
	for _, operation := range []struct {
		name        string
		call        func(*Manager, context.Context, string, []string) error
		secondCheck bool
	}{
		{"add permissions", (*Manager).AddPermissionsByToken, false},
		{"remove permissions", (*Manager).RemovePermissionsByToken, false},
		{"add roles", (*Manager).AddRolesByToken, false},
		{"remove roles", (*Manager).RemoveRolesByToken, false},
		{"set session data", func(m *Manager, ctx context.Context, token string, _ []string) error {
			return m.SetSessionValueByToken(ctx, token, "key", "changed")
		}, true},
		{"delete session data", func(m *Manager, ctx context.Context, token string, _ []string) error {
			return m.DeleteSessionValueByToken(ctx, token, "key")
		}, true},
	} {
		for _, sessionMode := range []string{"last terminal", "other terminal remains", "detached terminal"} {
			keepSession := sessionMode != "last terminal"
			name := operation.name + "/" + sessionMode
			t.Run(name, func(t *testing.T) {
				mgr := newTestManager(t, func(cfg *config.Config) {
					cfg.Timeout = 60
					cfg.ActiveTimeout = 1
				})
				pair, err := mgr.LoginWithRefreshToken(ctx, "access-lock", "web")
				if err != nil {
					t.Fatalf("login error = %v", err)
				}
				for _, setup := range []func(context.Context, string, []string) error{mgr.AddPermissions, mgr.AddRoles} {
					if err = setup(ctx, pair.LoginID, []string{"existing"}); err != nil {
						t.Fatalf("prepare access values error = %v", err)
					}
				}
				if sessionMode == "other terminal remains" {
					if _, err = mgr.Login(ctx, pair.LoginID, "app"); err != nil {
						t.Fatalf("other terminal login error = %v", err)
					}
				}
				if sessionMode == "detached terminal" {
					sess, getErr := mgr.getSession(ctx, pair.LoginID)
					if getErr != nil {
						t.Fatalf("get detached session error = %v", getErr)
					}
					sess.removeTerminalByToken(pair.AccessToken)
					if err = mgr.saveToStorage(ctx, mgr.getSessionKey(pair.LoginID), *sess); err != nil {
						t.Fatalf("save detached session error = %v", err)
					}
				}
				if err = mgr.storage.Set(ctx, mgr.getActiveKey(pair.AccessToken), time.Now().Add(-3*time.Second).Unix(), time.Minute); err != nil {
					t.Fatalf("expire active marker error = %v", err)
				}
				if operation.secondCheck {
					mgr.storage = &managerFreshActiveOnceStorage{Storage: mgr.storage, key: mgr.getActiveKey(pair.AccessToken)}
				}
				var events []listener.Event
				mgr.GetEventManager().RegisterFuncWithConfig(listener.EventAll, func(data *listener.EventData) {
					// A synchronous listener may acquire the same account lock. 同步监听器可以获取同一账号锁。
					unlock := mgr.lockLoginWrite(pair.LoginID)
					unlock()
					events = append(events, data.Event)
				}, listener.ListenerConfig{Async: false})
				done := make(chan error, 1)
				go func() { done <- operation.call(mgr, ctx, pair.AccessToken, []string{"existing"}) }()
				select {
				case err = <-done:
					if !errors.Is(err, derror.ErrActiveTimeout) {
						t.Fatalf("access write error = %v, want ErrActiveTimeout", err)
					}
				case <-time.After(5 * time.Second):
					t.Fatal("access write or timeout listener reentered the account lock")
				}
				want := []listener.Event{listener.EventActiveTimeout}
				if !keepSession {
					want = append([]listener.Event{listener.EventDestroySession}, want...)
				}
				if !reflect.DeepEqual(events, want) {
					t.Fatalf("events = %v, want %v", events, want)
				}
				if _, err = mgr.getTokenInfo(ctx, pair.AccessToken); !errors.Is(err, derror.ErrActiveTimeout) {
					t.Fatalf("persisted token state error = %v, want ErrActiveTimeout", err)
				}
				for _, key := range []string{mgr.getActiveKey(pair.AccessToken), mgr.getTokenRefreshKey(pair.AccessToken), mgr.getRefreshTokenKey(pair.RefreshToken)} {
					if mgr.storage.Exists(ctx, key) {
						t.Fatalf("inactive token metadata remains: %s", key)
					}
				}
				if keepSession {
					wantTerminals := 1
					if sessionMode == "detached terminal" {
						wantTerminals = 0
					}
					sess, getErr := mgr.getSession(ctx, pair.LoginID)
					if getErr != nil || len(sess.TerminalInfos) != wantTerminals || !reflect.DeepEqual(sess.Permissions, []string{"existing"}) || !reflect.DeepEqual(sess.Roles, []string{"existing"}) {
						t.Fatalf("remaining session = %+v, %v, want unchanged access values and %d terminals", sess, getErr, wantTerminals)
					}
				}
			})
		}
	}
}

// managerFreshActiveOnceStorage expires activity between the initial and locked checks. managerFreshActiveOnceStorage 使活跃状态在初次检查与锁内复核之间过期。
type managerFreshActiveOnceStorage struct {
	adapter.Storage
	key  string
	read bool
}

// Get keeps only the initial activity check fresh. Get 仅使初次活跃检查通过。
func (s *managerFreshActiveOnceStorage) Get(ctx context.Context, key string) (any, error) {
	if key == s.key && !s.read {
		s.read = true
		return time.Now().Unix(), nil
	}
	return s.Storage.Get(ctx, key)
}

// managerAccessInlinePool detects locked submission without hanging the regression test. managerAccessInlinePool 检测持锁提交，避免回归测试自身挂起。
type managerAccessInlinePool struct {
	mgr      *Manager
	loginID  string
	locked   bool
	executed int
}

// Submit executes inline after checking the caller has released the account lock. Submit 确认调用方已释放账号锁后内联执行。
func (p *managerAccessInlinePool) Submit(task func()) error {
	p.mgr.loginLocksMu.Lock()
	entry := p.mgr.loginLocks[p.loginID]
	locked := entry != nil && !entry.mu.TryLock()
	if entry != nil && !locked {
		entry.mu.Unlock()
	}
	p.mgr.loginLocksMu.Unlock()
	if locked {
		p.locked = true
		return errors.New("maintenance submitted under account lock")
	}
	p.executed++
	task()
	return nil
}

// Stop has no resources to release in this inline fixture. Stop 在此内联用例中无需释放资源。
func (*managerAccessInlinePool) Stop() {}

// Stats reports the fixed capacity of the inline fixture. Stats 返回内联用例的固定容量。
func (*managerAccessInlinePool) Stats() (int, int, float64) { return 0, 1, 0 }

// TestManagerAccessWritesSubmitMaintenanceAfterUnlock verifies effective and no-op mutations support inline pools. TestManagerAccessWritesSubmitMaintenanceAfterUnlock 验证有效和无效变更均支持内联协程池。
func TestManagerAccessWritesSubmitMaintenanceAfterUnlock(t *testing.T) {
	ctx := context.Background()
	for _, operation := range []struct {
		name  string
		call  func(*Manager, context.Context, string, []string) error
		setup func(*Manager, context.Context, string, []string) error
	}{
		{"add permissions", (*Manager).AddPermissionsByToken, nil},
		{"remove permissions", (*Manager).RemovePermissionsByToken, (*Manager).AddPermissions},
		{"add roles", (*Manager).AddRolesByToken, nil},
		{"remove roles", (*Manager).RemoveRolesByToken, (*Manager).AddRoles},
	} {
		t.Run(operation.name, func(t *testing.T) {
			mgr := newTestManager(t, func(cfg *config.Config) { cfg.ActiveTimeout = 60 })
			token, err := mgr.Login(ctx, "access-inline", "web")
			if err != nil {
				t.Fatalf("login error = %v", err)
			}

			// Seed removals so each operation first changes state and then becomes a no-op. 为删除操作准备初值，使每个操作均先有效变更、再成为无效变更。
			if operation.setup != nil {
				if err = operation.setup(mgr, ctx, "access-inline", []string{"value"}); err != nil {
					t.Fatalf("prepare access values error = %v", err)
				}
			}
			pool := &managerAccessInlinePool{mgr: mgr, loginID: "access-inline"}
			mgr.pool = pool
			for i := 0; i < 2; i++ {
				if err = operation.call(mgr, ctx, token, []string{"value"}); err != nil {
					t.Fatalf("access write error = %v", err)
				}
			}
			if pool.locked || pool.executed != 2 {
				t.Fatalf("inline maintenance: locked=%v executed=%d, want false and 2", pool.locked, pool.executed)
			}
		})
	}
}
