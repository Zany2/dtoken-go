package manager

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/listener"
)

// TestManagerLateValidationMaintenanceRejectsTokenReuse resumes a validated request after logout and same-second reuse. TestManagerLateValidationMaintenanceRejectsTokenReuse 在登出并同秒复用 Token 后恢复已校验请求。
func TestManagerLateValidationMaintenanceRejectsTokenReuse(t *testing.T) {
	ctx := context.Background()
	for _, query := range []struct {
		name string
		run  func(*Manager, string) error
	}{
		{"login check", func(m *Manager, token string) error { return m.CheckLogin(ctx, token) }},
		{"terminal query", func(m *Manager, token string) error {
			_, err := m.GetTerminalInfoByToken(ctx, token)
			return err
		}},
	} {
		t.Run(query.name, func(t *testing.T) {
			mgr := newTestManager(t, func(cfg *config.Config) {
				cfg.Timeout = 60
				cfg.ActiveTimeout = 60
				cfg.AutoRenew = true
				cfg.RenewMaxRefresh = 60
			})
			opts := LoginOptions{LoginID: "late-reuse", Token: "late-reuse-token", Device: "web"}
			token, err := mgr.LoginWithOptions(ctx, opts)
			if err != nil {
				t.Fatal(err)
			}
			original, err := mgr.getTokenRecord(ctx, token)
			if err != nil {
				t.Fatal(err)
			}
			pool := &managerQueuedMaintenancePool{}
			mgr.pool = pool
			t.Cleanup(pool.runAll)
			events := registerManagerValidationEventCollector(mgr, listener.EventRenew)
			var before terminalLifecycleSnapshot
			storage := &managerAfterSessionReadStorage{Storage: mgr.storage, sessionKey: mgr.getSessionKey(opts.LoginID)}
			mgr.storage = storage
			storage.afterRead = func() {
				if err := mgr.Logout(ctx, token); err != nil {
					t.Fatal(err)
				}
				if _, err := mgr.LoginWithOptions(ctx, opts); err != nil {
					t.Fatal(err)
				}
				current, err := mgr.getTokenRecord(ctx, token)
				if err != nil {
					t.Fatal(err)
				}
				if current.AccessID == "" || current.AccessID == original.AccessID {
					t.Fatal("replacement login did not receive a new lifecycle identity")
				}

				// Equalize timestamps without relying on execution speed or sleeping. 对齐时间戳，使同秒复用用例不依赖执行速度或等待。
				current.CreateTime = original.CreateTime
				if err := mgr.saveToStorage(ctx, mgr.getTokenKey(token), *current, time.Minute); err != nil {
					t.Fatal(err)
				}
				sess, err := mgr.getSession(ctx, opts.LoginID)
				if err != nil {
					t.Fatal(err)
				}
				sess.TerminalInfos[0].CreateTime = original.CreateTime
				if err := mgr.saveToStorage(ctx, storage.sessionKey, *sess); err != nil {
					t.Fatal(err)
				}
				if err := mgr.storage.Set(ctx, mgr.getActiveKey(token), time.Now().Add(-5*time.Second).Unix(), time.Minute); err != nil {
					t.Fatal(err)
				}
				before = captureTerminalLifecycle(t, mgr, ctx, []string{
					mgr.getTokenKey(token), storage.sessionKey, mgr.getActiveKey(token), mgr.getRenewKey(token),
				})
			}

			if err := query.run(mgr, token); err != nil {
				t.Fatal(err)
			}
			if storage.afterRead != nil || pool.taskCount() != 1 {
				t.Fatal("expected token replacement followed by one late maintenance task")
			}
			pool.runAll()
			assertTerminalLifecycleUnchanged(t, mgr, ctx, before)
			if len(events()) != 0 {
				t.Fatal("old validation renewed the replacement token")
			}

			// The rejected task must release its slot so a fresh request can maintain the current login. 拒绝旧任务后须释放名额，使新请求能够维护当前登录。
			if err := mgr.CheckLogin(ctx, token); err != nil {
				t.Fatal(err)
			}
			pool.runAll()
			if len(events()) != 1 {
				t.Fatal("current lifecycle did not renew after the stale task finished")
			}
		})
	}
}

// managerAfterSessionReadStorage interleaves one lifecycle write after validation reads a session snapshot. managerAfterSessionReadStorage 在校验读取会话快照后插入一次生命周期写入。
type managerAfterSessionReadStorage struct {
	adapter.Storage
	sessionKey string
	afterRead  func()
}

func (s *managerAfterSessionReadStorage) Get(ctx context.Context, key string) (any, error) {
	value, err := s.Storage.Get(ctx, key)
	if err == nil && key == s.sessionKey && s.afterRead != nil {
		afterRead := s.afterRead
		s.afterRead = nil
		afterRead()
	}
	return value, err
}

// TestManagerMaintenanceDoesNotMergeOtherLifecycleActivity protects queued current activity from a late old request. TestManagerMaintenanceDoesNotMergeOtherLifecycleActivity 防止迟到旧请求修改当前排队任务的活跃时间。
func TestManagerMaintenanceDoesNotMergeOtherLifecycleActivity(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) { cfg.ActiveTimeout = 60 })
	token, err := mgr.Login(ctx, "maintenance-merge", "web")
	if err != nil {
		t.Fatal(err)
	}
	current, err := mgr.getTokenRecord(ctx, token)
	if err != nil {
		t.Fatal(err)
	}
	pool := &managerQueuedMaintenancePool{}
	mgr.pool = pool
	t.Cleanup(pool.runAll)
	if err := mgr.CheckLogin(ctx, token); err != nil {
		t.Fatal(err)
	}
	mgr.maintenanceMu.Lock()
	state := mgr.maintenance[token]
	state.activeAt = time.Now().Add(-10 * time.Second).Unix()
	mgr.maintenance[token] = state
	mgr.maintenanceMu.Unlock()
	stale := *current
	stale.AccessID = "previous-lifecycle"
	mgr.submitLoginMaintenance(ctx, token, &stale, 60)
	if activeAt, ok := mgr.getLoginMaintenanceActiveAt(token, state.generation); !ok || activeAt != state.activeAt {
		t.Fatalf("old lifecycle changed queued activity: %d, %v", activeAt, ok)
	}
	if pool.taskCount() != 1 {
		t.Fatal("old lifecycle replaced or added to the current task")
	}
	pool.runAll()
}

// TestManagerRevokedCredentialsRejectStaleSessionSnapshots keeps all live-entry paths aligned after account disable. TestManagerRevokedCredentialsRejectStaleSessionSnapshots 验证账号封禁后所有有效凭证入口均拒绝陈旧会话快照。
func TestManagerRevokedCredentialsRejectStaleSessionSnapshots(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, nil)
	pair, err := mgr.LoginWithRefreshToken(ctx, "revoked-snapshot", "web")
	if err != nil {
		t.Fatal(err)
	}
	sess, err := mgr.getSession(ctx, pair.LoginID)
	if err != nil {
		t.Fatal(err)
	}
	record, err := mgr.getTokenRecord(ctx, pair.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	if err := mgr.Disable(ctx, pair.LoginID, time.Hour); err != nil {
		t.Fatal(err)
	}
	if err := mgr.Untie(ctx, pair.LoginID); err != nil {
		t.Fatal(err)
	}

	// Simulate a stale session snapshot being written back by another storage client. 模拟其他存储客户端回写旧 Session 快照。
	if err := mgr.saveToStorage(ctx, mgr.getSessionKey(pair.LoginID), *sess, time.Hour); err != nil {
		t.Fatal(err)
	}
	before := captureTerminalLifecycle(t, mgr, ctx, terminalLifecycleKeys(mgr, pair.AccessToken, pair.RefreshToken, mgr.getSessionKey(pair.LoginID)))
	if err := mgr.CheckLogin(ctx, pair.AccessToken); !errors.Is(err, derror.ErrInvalidToken) {
		t.Fatalf("CheckLogin() = %v, want invalid token", err)
	}
	if tokens, err := mgr.GetTokenValueListByLoginID(ctx, pair.LoginID, true); err != nil || len(tokens) != 0 {
		t.Fatalf("alive token list = %v, %v", tokens, err)
	}
	if count, err := mgr.GetOnlineTerminalCount(ctx, pair.LoginID); err != nil || count != 0 {
		t.Fatalf("online terminal count = %d, %v", count, err)
	}
	if alive, err := mgr.checkTerminalTokenAlive(ctx, pair.AccessToken); err != nil || alive {
		t.Fatalf("token alive = %v, %v", alive, err)
	}
	if alive, err := mgr.hasActiveTerminal(ctx, sess.TerminalInfos, sess); err != nil || alive {
		t.Fatalf("revoked login occupies a slot: %v, %v", alive, err)
	}
	if shared, err := mgr.getTokenAndShare(ctx, sess, "web", ""); err != nil || shared != "" {
		t.Fatalf("revoked shared token = %q, %v", shared, err)
	}
	expected := &refreshTokenRecord{
		RefreshTokenInfo: RefreshTokenInfo{LoginID: pair.LoginID, AccessToken: pair.AccessToken},
		AccessID:         record.AccessID,
	}
	if issued, err := mgr.issueRefreshToken(ctx, expected, time.Hour, nil); !errors.Is(err, derror.ErrInvalidToken) || issued != nil {
		t.Fatalf("refresh issuance for revoked access = %+v, %v", issued, err)
	}
	assertTerminalLifecycleUnchanged(t, mgr, ctx, before)

	// Raw metadata remains available for diagnostics. 原始元数据仍可用于诊断。
	if info, err := mgr.GetTokenInfo(ctx, pair.AccessToken); err != nil || info.LoginID != pair.LoginID {
		t.Fatalf("raw token metadata = %+v, %v", info, err)
	}
}
