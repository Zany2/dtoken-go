package manager

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/listener"
)

// TestManagerBasicLoginValidationSkipsTerminalLookup verifies basic checks use the token mapping without loading terminal context. TestManagerBasicLoginValidationSkipsTerminalLookup 验证基础登录态检查使用 Token 映射且不加载终端上下文。
func TestManagerBasicLoginValidationSkipsTerminalLookup(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.AutoRenew = false
		cfg.ActiveTimeout = 0
		cfg.IsConcurrent = true
		cfg.IsShare = false
	})

	token, err := mgr.Login(ctx, "mapping-source", "web", "first")
	if err != nil {
		t.Fatalf("first Login() error = %v", err)
	}
	if _, err = mgr.Login(ctx, "mapping-source", "web", "second"); err != nil {
		t.Fatalf("second Login() error = %v", err)
	}

	sess, err := mgr.GetSession(ctx, "mapping-source")
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if _, ok := sess.removeTerminalByToken(token); !ok {
		t.Fatal("removeTerminalByToken() did not find target token")
	}
	if err = mgr.saveToStorage(ctx, mgr.getSessionKey("mapping-source"), *sess, mgr.getExpiration()); err != nil {
		t.Fatalf("saveToStorage(session) error = %v", err)
	}

	if err = mgr.CheckLogin(ctx, token); err != nil {
		t.Fatalf("CheckLogin() error = %v, want token mapping to remain valid", err)
	}
	if loginID, loginErr := mgr.GetLoginID(ctx, token); loginErr != nil || loginID != "mapping-source" {
		t.Fatalf("GetLoginID() = %q, %v, want mapping-source, nil", loginID, loginErr)
	}
	tokenSession, err := mgr.GetSessionByToken(ctx, token)
	if err != nil || tokenSession == nil || tokenSession.LoginID != "mapping-source" {
		t.Fatalf("GetSessionByToken() = %#v, %v, want mapping-source session", tokenSession, err)
	}
	if _, err = mgr.GetTerminalInfoByToken(ctx, token); !errors.Is(err, derror.ErrInvalidToken) {
		t.Fatalf("GetTerminalInfoByToken() error = %v, want ErrInvalidToken", err)
	}
}

// TestManagerBasicLoginValidationRejectsDetachedMapping verifies a deleted Account Session keeps residual mappings invalid. TestManagerBasicLoginValidationRejectsDetachedMapping 验证账号 Session 删除后残留映射保持无效。
func TestManagerBasicLoginValidationRejectsDetachedMapping(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.AutoRenew = false
		cfg.ActiveTimeout = 0
	})

	token, err := mgr.Login(ctx, "detached-mapping", "web", "browser")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if err = mgr.GetStorage().Delete(ctx, mgr.getSessionKey("detached-mapping")); err != nil {
		t.Fatalf("Delete(session) error = %v", err)
	}
	if err = mgr.CheckLogin(ctx, token); !errors.Is(err, derror.ErrInvalidToken) {
		t.Fatalf("CheckLogin() error = %v, want ErrInvalidToken", err)
	}
}

// TestManagerDetachedTokenTermination verifies token operations remain effective when terminal metadata is missing. TestManagerDetachedTokenTermination 验证终端元数据缺失时按 Token 下线仍然生效。
func TestManagerDetachedTokenTermination(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name      string
		action    func(*Manager, string) error
		wantErr   error
		wantEvent listener.Event
	}{
		{
			name:      "logout",
			action:    func(mgr *Manager, token string) error { return mgr.Logout(ctx, token) },
			wantErr:   derror.ErrInvalidToken,
			wantEvent: listener.EventLogout,
		},
		{
			name:      "kickout",
			action:    func(mgr *Manager, token string) error { return mgr.Kickout(ctx, token) },
			wantErr:   derror.ErrTokenKickout,
			wantEvent: listener.EventKickout,
		},
		{
			name:      "replace",
			action:    func(mgr *Manager, token string) error { return mgr.Replace(ctx, token) },
			wantErr:   derror.ErrTokenReplaced,
			wantEvent: listener.EventReplace,
		},
		{
			name: "terminate-replace",
			action: func(mgr *Manager, token string) error {
				return mgr.Terminate(ctx, TerminateOptions{Token: token, Action: TerminateActionReplace})
			},
			wantErr:   derror.ErrTokenReplaced,
			wantEvent: listener.EventReplace,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loginID := "detached-termination-" + tt.name
			mgr := newTestManager(t, func(cfg *config.Config) {
				cfg.AutoRenew = false
				cfg.ActiveTimeout = 0
				cfg.IsConcurrent = true
				cfg.IsShare = false
			})
			storage := requireManagerTestStorage(t, mgr)

			pair, err := mgr.LoginWithRefreshToken(ctx, loginID, "web", "detached")
			if err != nil {
				t.Fatalf("LoginWithRefreshToken() error = %v", err)
			}
			keptToken, err := mgr.Login(ctx, loginID, "mobile", "kept")
			if err != nil {
				t.Fatalf("Login(kept) error = %v", err)
			}

			sess, err := mgr.GetSession(ctx, loginID)
			if err != nil {
				t.Fatalf("GetSession() error = %v", err)
			}
			if _, ok := sess.removeTerminalByToken(pair.AccessToken); !ok {
				t.Fatal("removeTerminalByToken() did not find target token")
			}
			if err = mgr.saveToStorage(ctx, mgr.getSessionKey(loginID), *sess, mgr.getExpiration()); err != nil {
				t.Fatalf("saveToStorage(session) error = %v", err)
			}
			if err = storage.Set(ctx, mgr.getRenewKey(pair.AccessToken), time.Now().Unix(), time.Minute); err != nil {
				t.Fatalf("Set(renew marker) error = %v", err)
			}
			if err = storage.Set(ctx, mgr.getActiveKey(pair.AccessToken), time.Now().Unix(), time.Minute); err != nil {
				t.Fatalf("Set(active marker) error = %v", err)
			}

			eventCount := 0
			mgr.GetEventManager().RegisterFuncWithConfig(tt.wantEvent, func(data *listener.EventData) {
				if data.Token == pair.AccessToken {
					eventCount++
				}
			}, listener.ListenerConfig{Async: false})

			if err = tt.action(mgr, pair.AccessToken); err != nil {
				t.Fatalf("first action() error = %v", err)
			}
			if err = tt.action(mgr, pair.AccessToken); err != nil {
				t.Fatalf("repeated action() error = %v", err)
			}
			if err = mgr.CheckLogin(ctx, pair.AccessToken); !errors.Is(err, tt.wantErr) {
				t.Fatalf("CheckLogin(target) error = %v, want %v", err, tt.wantErr)
			}
			if err = mgr.CheckLogin(ctx, keptToken); err != nil {
				t.Fatalf("CheckLogin(kept) error = %v", err)
			}
			if storage.Exists(ctx, mgr.getRenewKey(pair.AccessToken)) || storage.Exists(ctx, mgr.getActiveKey(pair.AccessToken)) {
				t.Fatal("token maintenance metadata remains after termination")
			}
			if ttl, ttlErr := mgr.GetRefreshTokenTTL(ctx, pair.RefreshToken); ttlErr != nil || ttl != -2 {
				t.Fatalf("GetRefreshTokenTTL() = %d, %v, want -2, nil", ttl, ttlErr)
			}
			if eventCount != 1 {
				t.Fatalf("%s events = %d, want 1", tt.wantEvent, eventCount)
			}
		})
	}
}

// TestManagerDetachedTokenActiveTimeoutPersistsState verifies inactive state does not depend on terminal metadata. TestManagerDetachedTokenActiveTimeoutPersistsState 验证不活跃状态落盘不依赖终端元数据。
func TestManagerDetachedTokenActiveTimeoutPersistsState(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.Timeout = 60
		cfg.AutoRenew = false
		cfg.ActiveTimeout = 1
		cfg.IsConcurrent = true
		cfg.IsShare = false
	})
	storage := requireManagerTestStorage(t, mgr)

	token, err := mgr.Login(ctx, "detached-active-timeout", "web", "detached")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	keptToken, err := mgr.Login(ctx, "detached-active-timeout", "mobile", "kept")
	if err != nil {
		t.Fatalf("Login(kept) error = %v", err)
	}
	sess, err := mgr.GetSession(ctx, "detached-active-timeout")
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if _, ok := sess.removeTerminalByToken(token); !ok {
		t.Fatal("removeTerminalByToken() did not find target token")
	}
	if err = mgr.saveToStorage(ctx, mgr.getSessionKey("detached-active-timeout"), *sess, mgr.getExpiration()); err != nil {
		t.Fatalf("saveToStorage(session) error = %v", err)
	}
	if err = storage.Set(ctx, mgr.getActiveKey(token), time.Now().Add(-2*time.Second).Unix(), time.Minute); err != nil {
		t.Fatalf("Set(expired active marker) error = %v", err)
	}

	activeTimeoutEvents := 0
	mgr.GetEventManager().RegisterFuncWithConfig(listener.EventActiveTimeout, func(data *listener.EventData) {
		if data.Token == token {
			activeTimeoutEvents++
		}
	}, listener.ListenerConfig{Async: false})

	if err = mgr.CheckLogin(ctx, token); !errors.Is(err, derror.ErrActiveTimeout) {
		t.Fatalf("first CheckLogin() error = %v, want ErrActiveTimeout", err)
	}
	if err = mgr.CheckLogin(ctx, token); !errors.Is(err, derror.ErrActiveTimeout) {
		t.Fatalf("second CheckLogin() error = %v, want persisted ErrActiveTimeout", err)
	}
	if err = mgr.CheckLogin(ctx, keptToken); err != nil {
		t.Fatalf("CheckLogin(kept) error = %v", err)
	}
	if activeTimeoutEvents != 1 {
		t.Fatalf("active-timeout events = %d, want 1", activeTimeoutEvents)
	}
}

// TestManagerConcurrentChecksCollapseAutoRenew verifies queued validation tasks renew a due token once. TestManagerConcurrentChecksCollapseAutoRenew 验证并发校验排队后只续期一次到期 Token。
func TestManagerConcurrentChecksCollapseAutoRenew(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.Timeout = 20
		cfg.AutoRenew = true
		cfg.RenewMaxRefresh = 5
		cfg.RenewInterval = 4
		cfg.ActiveTimeout = 0
	})
	storage := requireManagerTestStorage(t, mgr)

	token, err := mgr.Login(ctx, "concurrent-auto-renew", "web", "browser")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if err = storage.Delete(ctx, mgr.getRenewKey(token)); err != nil {
		t.Fatalf("Delete(renew marker) error = %v", err)
	}
	if err = storage.Expire(ctx, mgr.getTokenKey(token), 2*time.Second); err != nil {
		t.Fatalf("Expire(token) error = %v", err)
	}

	var eventMu sync.Mutex
	renewEvents := 0
	mgr.GetEventManager().RegisterFuncWithConfig(listener.EventRenew, func(*listener.EventData) {
		eventMu.Lock()
		renewEvents++
		eventMu.Unlock()
	}, listener.ListenerConfig{Async: false})

	const checks = 16
	errCh := make(chan error, checks)
	var checksWG sync.WaitGroup
	checksWG.Add(checks)
	for i := 0; i < checks; i++ {
		go func() {
			defer checksWG.Done()
			errCh <- mgr.CheckLogin(ctx, token)
		}()
	}
	checksWG.Wait()
	close(errCh)
	for checkErr := range errCh {
		if checkErr != nil {
			t.Fatalf("CheckLogin() error = %v", checkErr)
		}
	}

	mgr.asyncWG.Wait()
	eventMu.Lock()
	gotRenewEvents := renewEvents
	eventMu.Unlock()
	if gotRenewEvents != 1 {
		t.Fatalf("renew events = %d, want 1", gotRenewEvents)
	}
}

// TestManagerConcurrentChecksCoalesceActiveMaintenance verifies one token has only one in-flight active refresh. TestManagerConcurrentChecksCoalesceActiveMaintenance 验证同一 Token 同时只有一个活跃刷新任务。
func TestManagerConcurrentChecksCoalesceActiveMaintenance(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.AutoRenew = false
		cfg.ActiveTimeout = 30
	})

	token, err := mgr.Login(ctx, "coalesce-active", "web", "browser")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	baseStorage := requireManagerTestStorage(t, mgr).(*managerTestStorage)
	blockingStorage := &managerBlockingActiveStorage{
		managerTestStorage: baseStorage,
		activeKey:          mgr.getActiveKey(token),
		started:            make(chan struct{}),
		release:            make(chan struct{}),
	}
	defer blockingStorage.releaseMaintenance()
	mgr.storage = blockingStorage
	countingPool := &managerCountingMaintenancePool{}
	mgr.pool = countingPool

	if err = mgr.CheckLogin(ctx, token); err != nil {
		t.Fatalf("first CheckLogin() error = %v", err)
	}
	select {
	case <-blockingStorage.started:
	case <-time.After(time.Second):
		t.Fatal("active maintenance did not start")
	}

	const checks = 16
	var checksWG sync.WaitGroup
	errCh := make(chan error, checks)
	checksWG.Add(checks)
	for i := 0; i < checks; i++ {
		go func() {
			defer checksWG.Done()
			errCh <- mgr.CheckLogin(ctx, token)
		}()
	}
	checksWG.Wait()
	close(errCh)
	for checkErr := range errCh {
		if checkErr != nil {
			t.Fatalf("concurrent CheckLogin() error = %v", checkErr)
		}
	}
	if submissions := countingPool.submissionCount(); submissions != 1 {
		t.Fatalf("active maintenance submissions = %d, want 1", submissions)
	}

	blockingStorage.releaseMaintenance()
	mgr.asyncWG.Wait()
	if writes := blockingStorage.activeWrites(); writes == 0 {
		t.Fatal("active maintenance did not write the active marker")
	}
}

// TestManagerCanceledMaintenanceDoesNotTouchReusedToken verifies an old queued task cannot maintain a new lifecycle using the same token. TestManagerCanceledMaintenanceDoesNotTouchReusedToken 验证旧排队任务不会维护复用同值 Token 的新生命周期。
func TestManagerCanceledMaintenanceDoesNotTouchReusedToken(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.AutoRenew = false
		cfg.ActiveTimeout = 30
	})
	token := "reused-maintenance-token"

	if _, err := mgr.LoginWithOptions(ctx, LoginOptions{LoginID: "reused-maintenance", Device: "web", Token: token}); err != nil {
		t.Fatalf("first LoginWithOptions() error = %v", err)
	}
	queuedPool := &managerQueuedMaintenancePool{}
	defer queuedPool.runAll()
	mgr.pool = queuedPool
	if err := mgr.CheckLogin(ctx, token); err != nil {
		t.Fatalf("CheckLogin() error = %v", err)
	}
	if queuedPool.taskCount() != 1 {
		t.Fatalf("queued maintenance tasks = %d, want 1", queuedPool.taskCount())
	}

	if err := mgr.Logout(ctx, token); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if _, err := mgr.LoginWithOptions(ctx, LoginOptions{LoginID: "reused-maintenance", Device: "web", Token: token}); err != nil {
		t.Fatalf("second LoginWithOptions() error = %v", err)
	}
	const activeSentinel int64 = 12345
	if err := mgr.storage.Set(ctx, mgr.getActiveKey(token), activeSentinel, time.Minute); err != nil {
		t.Fatalf("Set(active sentinel) error = %v", err)
	}

	queuedPool.runAll()
	mgr.asyncWG.Wait()
	activeValue, err := mgr.storage.Get(ctx, mgr.getActiveKey(token))
	if err != nil {
		t.Fatalf("Get(active marker) error = %v", err)
	}
	if activeValue != activeSentinel {
		t.Fatalf("active marker after stale task = %#v, want %d", activeValue, activeSentinel)
	}
}

// TestManagerLoginByTokenMaintenanceDoesNotTouchReusedToken verifies manual renewal shares lifecycle invalidation. TestManagerLoginByTokenMaintenanceDoesNotTouchReusedToken 验证手动续期共享生命周期失效保护。
func TestManagerLoginByTokenMaintenanceDoesNotTouchReusedToken(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.AutoRenew = false
		cfg.ActiveTimeout = 30
	})
	token := "reused-manual-renew-token"

	if _, err := mgr.LoginWithOptions(ctx, LoginOptions{LoginID: "reused-manual-renew", Device: "web", Token: token}); err != nil {
		t.Fatalf("first LoginWithOptions() error = %v", err)
	}
	queuedPool := &managerQueuedMaintenancePool{}
	defer queuedPool.runAll()
	mgr.pool = queuedPool
	if err := mgr.LoginByToken(ctx, token); err != nil {
		t.Fatalf("LoginByToken() error = %v", err)
	}
	if queuedPool.taskCount() != 1 {
		t.Fatalf("queued maintenance tasks = %d, want 1", queuedPool.taskCount())
	}

	if err := mgr.Logout(ctx, token); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if _, err := mgr.LoginWithOptions(ctx, LoginOptions{LoginID: "reused-manual-renew", Device: "web", Token: token}); err != nil {
		t.Fatalf("second LoginWithOptions() error = %v", err)
	}
	const activeSentinel int64 = 23456
	if err := mgr.storage.Set(ctx, mgr.getActiveKey(token), activeSentinel, time.Minute); err != nil {
		t.Fatalf("Set(active sentinel) error = %v", err)
	}

	queuedPool.runAll()
	mgr.asyncWG.Wait()
	activeValue, err := mgr.storage.Get(ctx, mgr.getActiveKey(token))
	if err != nil {
		t.Fatalf("Get(active marker) error = %v", err)
	}
	if activeValue != activeSentinel {
		t.Fatalf("active marker after stale manual renewal = %#v, want %d", activeValue, activeSentinel)
	}
}

type managerBlockingActiveStorage struct {
	*managerTestStorage
	mu          sync.Mutex
	activeKey   string
	started     chan struct{}
	release     chan struct{}
	startOnce   sync.Once
	releaseOnce sync.Once
	writes      int
}

func (s *managerBlockingActiveStorage) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	if key == s.activeKey {
		s.mu.Lock()
		s.writes++
		s.mu.Unlock()
		s.startOnce.Do(func() { close(s.started) })
		<-s.release
	}
	return s.managerTestStorage.Set(ctx, key, value, expiration)
}

func (s *managerBlockingActiveStorage) activeWrites() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.writes
}

func (s *managerBlockingActiveStorage) releaseMaintenance() {
	s.releaseOnce.Do(func() { close(s.release) })
}

type managerQueuedMaintenancePool struct {
	mu    sync.Mutex
	tasks []func()
}

type managerCountingMaintenancePool struct {
	mu          sync.Mutex
	submissions int
}

func (p *managerCountingMaintenancePool) Submit(task func()) error {
	p.mu.Lock()
	p.submissions++
	p.mu.Unlock()
	go task()
	return nil
}

func (p *managerCountingMaintenancePool) Stop() {}

func (p *managerCountingMaintenancePool) Stats() (int, int, float64) {
	return 0, 0, 0
}

func (p *managerCountingMaintenancePool) submissionCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.submissions
}

func (p *managerQueuedMaintenancePool) Submit(task func()) error {
	p.mu.Lock()
	p.tasks = append(p.tasks, task)
	p.mu.Unlock()
	return nil
}

func (p *managerQueuedMaintenancePool) Stop() {}

func (p *managerQueuedMaintenancePool) Stats() (int, int, float64) {
	return 0, 0, 0
}

func (p *managerQueuedMaintenancePool) taskCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.tasks)
}

func (p *managerQueuedMaintenancePool) runAll() {
	p.mu.Lock()
	tasks := append([]func(){}, p.tasks...)
	p.tasks = nil
	p.mu.Unlock()
	for _, task := range tasks {
		task()
	}
}
