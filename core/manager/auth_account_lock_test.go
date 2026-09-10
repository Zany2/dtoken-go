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

// managerAccountSwapStorage interleaves a lifecycle replacement after reading the original token snapshot. managerAccountSwapStorage 在读取原 Token 快照后插入生命周期替换操作。
type managerAccountSwapStorage struct {
	adapter.Storage
	key       string
	afterRead func() error
}

// Get returns the captured value after running the one-time interleaving hook. Get 执行一次性交错钩子后返回已捕获的值。
func (s *managerAccountSwapStorage) Get(ctx context.Context, key string) (any, error) {
	value, err := s.Storage.Get(ctx, key)
	if err != nil || key != s.key || s.afterRead == nil {
		return value, err
	}
	hook := s.afterRead
	s.afterRead = nil
	if err = hook(); err != nil {
		return nil, err
	}
	return value, nil
}

// TestManagerTokenWritesRejectLifecycleSwap verifies stale requests cannot mutate a replacement lifecycle or emit its events. TestManagerTokenWritesRejectLifecycleSwap 验证旧请求不能变更替代生命周期或触发其事件。
func TestManagerTokenWritesRejectLifecycleSwap(t *testing.T) {
	ctx := context.Background()
	for _, operation := range []struct {
		name string
		run  func(*Manager, context.Context, string) error
	}{
		{"login by token", (*Manager).LoginByToken},
		{"renew timeout", func(m *Manager, ctx context.Context, token string) error {
			return m.RenewTimeout(ctx, token, time.Hour)
		}},
		{"add permissions", func(m *Manager, ctx context.Context, token string) error {
			return m.AddPermissionsByToken(ctx, token, []string{"new"})
		}},
		{"remove permissions", func(m *Manager, ctx context.Context, token string) error {
			return m.RemovePermissionsByToken(ctx, token, []string{"existing"})
		}},
		{"add roles", func(m *Manager, ctx context.Context, token string) error {
			return m.AddRolesByToken(ctx, token, []string{"new"})
		}},
		{"remove roles", func(m *Manager, ctx context.Context, token string) error {
			return m.RemoveRolesByToken(ctx, token, []string{"existing"})
		}},
		{"set session data", func(m *Manager, ctx context.Context, token string) error {
			return m.SetSessionValueByToken(ctx, token, "key", "changed")
		}},
		{"delete session data", func(m *Manager, ctx context.Context, token string) error {
			return m.DeleteSessionValueByToken(ctx, token, "key")
		}},
	} {
		for _, replacement := range []struct {
			name    string
			loginID string
		}{
			{"different account", "lock-replacement"},
			{"same account", "lock-original"},
		} {
			for _, state := range []string{"active", "active timeout", "corrupt active marker"} {
				t.Run(operation.name+"/"+replacement.name+"/"+state, func(t *testing.T) {
					mgr := newTestManager(t, func(cfg *config.Config) {
						cfg.Timeout, cfg.RefreshTokenTimeout = 600, 1200
						cfg.ActiveTimeout, cfg.RenewInterval = 60, 30
					})
					const oldLoginID, token = "lock-original", "lock-reused-token"
					if _, err := mgr.LoginWithOptions(ctx, LoginOptions{LoginID: oldLoginID, Token: token, Device: "web"}); err != nil {
						t.Fatalf("original login error = %v", err)
					}
					originalRecord, err := mgr.getTokenRecord(ctx, token)
					if err != nil {
						t.Fatalf("load original token record error = %v", err)
					}

					// Keep all maintenance synchronous so unexpected submissions remain observable. 保持维护同步执行，使意外提交可被观察。
					pool := &managerAccessInlinePool{mgr: mgr, loginID: replacement.loginID}
					mgr.pool = pool
					var events []listener.Event
					before := make(map[string]any)
					beforeTTL := make(map[string]time.Duration)
					swapped := false
					storage := &managerAccountSwapStorage{Storage: mgr.storage, key: mgr.getTokenKey(token)}
					storage.afterRead = func() error {
						if err := mgr.Logout(ctx, token); err != nil {
							return err
						}
						pair, err := mgr.LoginWithRefreshTokenOptions(ctx, RefreshTokenOptions{LoginOptions: LoginOptions{
							LoginID: replacement.loginID, Token: token, Device: "web",
						}})
						if err != nil {
							return err
						}

						// Make public metadata identical so only the private AccessID can distinguish same-account reuse. 使公开元数据完全相同，确保同账号复用只能由私有 AccessID 区分。
						replacementRecord, err := mgr.getTokenRecord(ctx, token)
						if err != nil {
							return err
						}
						if replacementRecord.AccessID == "" || replacementRecord.AccessID == originalRecord.AccessID {
							return errors.New("replacement token did not receive a fresh access ID")
						}
						if replacementRecord.TerminalIndex != originalRecord.TerminalIndex {
							return errors.New("replacement token did not reuse the original terminal index")
						}
						replacementRecord.CreateTime = originalRecord.CreateTime
						if err = mgr.saveToStorage(ctx, mgr.getTokenKey(token), *replacementRecord); err != nil {
							return err
						}
						sess, err := mgr.getSession(ctx, replacement.loginID)
						if err != nil {
							return err
						}
						terminalFound := false
						for i := range sess.TerminalInfos {
							if sess.TerminalInfos[i].Token == token {
								sess.TerminalInfos[i].CreateTime = originalRecord.CreateTime
								terminalFound = true
							}
						}
						if !terminalFound {
							return errors.New("replacement session did not contain the reused token")
						}
						if err = mgr.saveToStorage(ctx, mgr.getSessionKey(replacement.loginID), *sess); err != nil {
							return err
						}
						for _, setup := range []func(context.Context, string, []string) error{mgr.AddPermissions, mgr.AddRoles} {
							if err = setup(ctx, replacement.loginID, []string{"existing"}); err != nil {
								return err
							}
						}
						if err = mgr.SetSessionValue(ctx, replacement.loginID, "key", "original"); err != nil {
							return err
						}

						// Exercise both timeout cleanup and corrupt-marker deletion after the lifecycle changes. 在生命周期变化后覆盖超时清理及损坏标记删除路径。
						var activeValue any = time.Now().Unix()
						switch state {
						case "active timeout":
							activeValue = time.Now().Add(-2 * time.Minute).Unix()
						case "corrupt active marker":
							activeValue = "invalid-timestamp"
						}
						if err = mgr.storage.Set(ctx, mgr.getActiveKey(token), activeValue, 10*time.Minute); err != nil {
							return err
						}
						keys := []string{
							mgr.getTokenKey(token), mgr.getSessionKey(replacement.loginID), mgr.getSessionKey(oldLoginID),
							mgr.getActiveKey(token), mgr.getRenewKey(token), mgr.getTokenRefreshKey(token), mgr.getRefreshTokenKey(pair.RefreshToken),
						}
						for _, key := range keys {
							before[key], err = mgr.storage.Get(ctx, key)
							if err != nil {
								return err
							}
							beforeTTL[key], err = mgr.storage.TTL(ctx, key)
							if err != nil {
								return err
							}
						}
						mgr.GetEventManager().RegisterFuncWithConfig(listener.EventAll, func(data *listener.EventData) {
							events = append(events, data.Event)
						}, listener.ListenerConfig{Async: false})
						swapped = true
						return nil
					}
					mgr.storage = storage

					// The request still holds the old read snapshot, but all persisted data now belongs to the replacement lifecycle. 请求仍持有旧读取快照，但持久化数据已属于替代生命周期。
					err := operation.run(mgr, ctx, token)
					if !swapped {
						t.Fatalf("lifecycle replacement did not complete: %v", err)
					}
					if !errors.Is(err, derror.ErrInvalidToken) {
						t.Errorf("stale request error = %v, want ErrInvalidToken", err)
					}
					for key, want := range before {
						got, getErr := mgr.storage.Get(ctx, key)
						if getErr != nil || !reflect.DeepEqual(got, want) {
							t.Errorf("replacement key %q changed: got=%v want=%v error=%v", key, got, want, getErr)
						}
						ttl, ttlErr := mgr.storage.TTL(ctx, key)
						if ttlErr != nil || (beforeTTL[key] <= 0 && ttl != beforeTTL[key]) || (beforeTTL[key] > 0 && (ttl <= 0 || ttl > beforeTTL[key])) {
							t.Errorf("replacement key %q TTL changed unexpectedly: got=%v before=%v error=%v", key, ttl, beforeTTL[key], ttlErr)
						}
					}
					if len(events) != 0 || pool.executed != 0 || pool.locked {
						t.Errorf("stale request caused side effects: events=%v tasks=%d locked=%v", events, pool.executed, pool.locked)
					}
				})
			}
		}
	}
}
