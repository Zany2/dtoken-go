package dtoken

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/config"
	adapter "github.com/Zany2/dtoken-go/contract"
	derror "github.com/Zany2/dtoken-go/errors"
	manager "github.com/Zany2/dtoken-go/internal/engine"
)

// TestSetManagerClosesReplacedManager verifies same-type replacement releases the previous manager. TestSetManagerClosesReplacedManager 验证同类型替换会释放旧 Manager。
func TestSetManagerClosesReplacedManager(t *testing.T) {
	DeleteAllManager()
	t.Cleanup(DeleteAllManager)

	oldPool := &registryTestPool{}
	newPool := &registryTestPool{}
	oldManager := newRegistryTestManager("replace", oldPool)
	newManager := newRegistryTestManager("replace", newPool)

	if err := SetManager(oldManager); err != nil {
		t.Errorf("register authentication: %v", err)
	}
	if err := SetManager(newManager); err != nil {
		t.Errorf("register authentication: %v", err)
	}

	if oldPool.stops() != 1 {
		t.Fatalf("old pool stop count = %d, want 1", oldPool.stops())
	}
	if newPool.stops() != 0 {
		t.Fatalf("new pool stop count = %d, want 0", newPool.stops())
	}
	if got, err := GetManager("replace"); err != nil || got != newManager {
		t.Fatalf("GetManager(replace) = %v, %v, want new manager", got, err)
	}
}

// TestSetManagerKeepsSameInstanceOpen verifies duplicate registration does not retire the current manager. TestSetManagerKeepsSameInstanceOpen 验证重复注册同一实例不会关闭当前 Manager。
func TestSetManagerKeepsSameInstanceOpen(t *testing.T) {
	DeleteAllManager()
	t.Cleanup(DeleteAllManager)

	pool := &registryTestPool{}
	mgr := newRegistryTestManager("same-instance", pool)
	if err := SetManager(mgr); err != nil {
		t.Errorf("register authentication: %v", err)
	}
	if err := SetManager(mgr); err != nil {
		t.Errorf("register authentication: %v", err)
	}

	if mgr.IsClosed() {
		t.Fatal("manager is closed after duplicate registration")
	}
	if pool.stops() != 0 {
		t.Fatalf("pool stop count = %d, want 0", pool.stops())
	}
}

// TestSetManagerKeepsBuiltNamespace verifies configuration snapshots cannot rename a registered engine. TestSetManagerKeepsBuiltNamespace 验证修改配置快照不能重命名已注册引擎。
func TestSetManagerKeepsBuiltNamespace(t *testing.T) {
	DeleteAllManager()
	t.Cleanup(DeleteAllManager)
	mgr := newRegistryTestManager("original-registration", &registryTestPool{})
	if err := SetManager(mgr); err != nil {
		t.Errorf("register authentication: %v", err)
	}
	mgr.GetConfig().SetAuthType("moved-registration")
	if err := SetManager(mgr); err != nil {
		t.Errorf("register authentication: %v", err)
	}
	if got, err := GetManager("original-registration"); err != nil || got != mgr {
		t.Fatalf("original registration = %v, %v", got, err)
	}
	if _, err := GetManager("moved-registration"); !errors.Is(err, derror.ErrManagerNotFound) {
		t.Fatalf("snapshot changed registration: %v", err)
	}
}

// TestSetManagerRejectsRetiringInstance verifies a manager cannot be registered again during or after replacement cleanup. TestSetManagerRejectsRetiringInstance 验证 Manager 在替换关闭期间及关闭后不能再次注册。
func TestSetManagerRejectsRetiringInstance(t *testing.T) {
	DeleteAllManager()
	t.Cleanup(DeleteAllManager)

	stopStarted := make(chan struct{})
	stopRelease := make(chan struct{})
	defer func() {
		select {
		case <-stopRelease:
		default:
			close(stopRelease)
		}
	}()

	oldManager := newRegistryTestManager("retiring", &registryTestBlockingPool{
		started: stopStarted,
		release: stopRelease,
	})
	newManager := newRegistryTestManager("retiring", &registryTestPool{})
	if err := SetManager(oldManager); err != nil {
		t.Errorf("register authentication: %v", err)
	}
	oldAuth, alias := New(oldManager), New(oldManager)

	replaceDone := make(chan struct{})
	go func() {
		if err := SetManager(newManager); err != nil {
			t.Errorf("register authentication: %v", err)
		}
		close(replaceDone)
	}()

	select {
	case <-stopStarted:
	case <-time.After(time.Second):
		t.Fatal("replacement did not start closing the old manager")
	}

	if !oldAuth.IsClosed() || oldAuth.Manager() != nil || alias.Manager() != nil {
		t.Fatal("fixed facades exposed the old engine during replacement cleanup")
	}

	if err := SetManager(oldManager); !errors.Is(err, derror.ErrManagerNotFound) {
		t.Fatalf("SetManager(retired) = %v", err)
	}
	if got, err := GetManager("retiring"); err != nil || got != newManager {
		t.Fatalf("GetManager(retiring) during cleanup = %v, %v, want new manager", got, err)
	}

	close(stopRelease)
	select {
	case <-replaceDone:
	case <-time.After(time.Second):
		t.Fatal("replacement did not finish")
	}

	if err := SetManager(oldManager); !errors.Is(err, derror.ErrManagerNotFound) {
		t.Fatalf("SetManager(retired) = %v", err)
	}
	if got, err := GetManager("retiring"); err != nil || got != newManager {
		t.Fatalf("GetManager(retiring) after cleanup = %v, %v, want new manager", got, err)
	}
}

// TestDeleteAllMarksEnginesBeforeCleanup verifies every removed engine is unavailable while the first resource cleanup is blocked. TestDeleteAllMarksEnginesBeforeCleanup 验证首个资源清理阻塞期间，全部被移除引擎均已不可用。
func TestDeleteAllMarksEnginesBeforeCleanup(t *testing.T) {
	DeleteAllManager()
	t.Cleanup(DeleteAllManager)

	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseCleanup := func() { releaseOnce.Do(func() { close(release) }) }
	defer releaseCleanup()
	firstStarted, secondStarted := make(chan struct{}), make(chan struct{})
	first := newRegistryTestManager("bulk-first", &registryTestBlockingPool{started: firstStarted, release: release})
	second := newRegistryTestManager("bulk-second", &registryTestBlockingPool{started: secondStarted, release: release})
	if err := SetManager(first); err != nil {
		t.Errorf("register authentication: %v", err)
	}
	if err := SetManager(second); err != nil {
		t.Errorf("register authentication: %v", err)
	}

	done := make(chan struct{})
	go func() {
		DeleteAllManager()
		close(done)
	}()
	select {
	case <-firstStarted:
	case <-secondStarted:
	case <-time.After(time.Second):
		t.Fatal("bulk cleanup did not start")
	}

	for _, mgr := range []*manager.Manager{first, second} {
		if !mgr.IsClosed() {
			t.Fatal("removed engine remains available while another engine is closing")
		}
		if err := SetManager(mgr); !errors.Is(err, derror.ErrManagerNotFound) {
			t.Fatalf("SetManager(removed) = %v, want ErrManagerNotFound", err)
		}
		if _, err := GetManager(mgr.GetConfig().AuthType); !errors.Is(err, derror.ErrManagerNotFound) {
			t.Fatalf("removed engine was registered again: %v", err)
		}
	}

	releaseCleanup()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("bulk cleanup did not finish")
	}
}

// TestRegistryRemovalHappensBeforeClose verifies registration during close survives single and bulk deletion. TestRegistryRemovalHappensBeforeClose 验证单个和批量删除关闭期间的新注册都能保留。
func TestRegistryRemovalHappensBeforeClose(t *testing.T) {
	tests := []struct {
		name   string
		remove func(string) error
	}{
		{name: "DeleteManager", remove: func(authType string) error { return DeleteManager(authType) }},
		{name: "DeleteAllManager", remove: func(string) error { DeleteAllManager(); return nil }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			DeleteAllManager()
			t.Cleanup(DeleteAllManager)

			stopStarted := make(chan struct{})
			stopRelease := make(chan struct{})
			defer func() {
				select {
				case <-stopRelease:
				default:
					close(stopRelease)
				}
			}()

			oldManager := newRegistryTestManager("concurrent-delete", &registryTestBlockingPool{
				started: stopStarted,
				release: stopRelease,
			})
			newManager := newRegistryTestManager("concurrent-delete", &registryTestPool{})
			if err := SetManager(oldManager); err != nil {
				t.Errorf("register authentication: %v", err)
			}

			deleteDone := make(chan error, 1)
			go func() {
				deleteDone <- tt.remove("concurrent-delete")
			}()

			select {
			case <-stopStarted:
			case <-time.After(time.Second):
				t.Fatalf("%s did not start closing the old manager", tt.name)
			}
			if _, err := GetManager("concurrent-delete"); !errors.Is(err, derror.ErrManagerNotFound) {
				t.Fatalf("GetManager during close error = %v, want ErrManagerNotFound", err)
			}

			if err := SetManager(newManager); err != nil {
				t.Errorf("register authentication: %v", err)
			}
			close(stopRelease)

			select {
			case err := <-deleteDone:
				if err != nil {
					t.Fatalf("%s error = %v", tt.name, err)
				}
			case <-time.After(time.Second):
				t.Fatalf("%s did not finish", tt.name)
			}
			if got, err := GetManager("concurrent-delete"); err != nil || got != newManager {
				t.Fatalf("GetManager after concurrent registration = %v, %v, want new manager", got, err)
			}
		})
	}
}

// TestInjectedComponentsRemainCallerOwned verifies shared injected components survive manager deletion. TestInjectedComponentsRemainCallerOwned 验证共享注入组件在 Manager 删除后仍由调用方持有。
func TestInjectedComponentsRemainCallerOwned(t *testing.T) {
	DeleteAllManager()
	t.Cleanup(DeleteAllManager)

	storage := &registryTestStorage{}
	logger := &registryTestLogger{NopLogger: adapter.NewNopLogger()}
	pool := &registryTestPool{}

	for _, authType := range []string{"shared-user", "shared-admin"} {
		mgr, err := buildTestManager(NewBuilder().
			AuthType(authType).
			IsPrintBanner(false).
			IsLog(true).
			AutoRenew(true).
			Storage(storage).
			SetLog(logger).
			SetPool(pool).
			UseManagerOption(manager.WithComponentOwnership(manager.ComponentOwnership{})),
		)
		if err != nil {
			t.Fatalf("Build(%s) error = %v", authType, err)
		}
		if err := SetManager(mgr); err != nil {
			t.Errorf("register authentication: %v", err)
		}
	}

	if err := DeleteManager("shared-user"); err != nil {
		t.Fatalf("DeleteManager(shared-user) error = %v", err)
	}
	if storage.closes() != 0 || logger.closes() != 0 || pool.stops() != 0 {
		t.Fatalf("shared component close counts = storage:%d logger:%d pool:%d, want all zero", storage.closes(), logger.closes(), pool.stops())
	}
	if _, err := GetManager("shared-admin"); err != nil {
		t.Fatalf("GetManager(shared-admin) error = %v", err)
	}

	DeleteAllManager()
	if storage.closes() != 0 || logger.closes() != 0 || pool.stops() != 0 {
		t.Fatalf("shared component close counts after DeleteAllManager = storage:%d logger:%d pool:%d, want all zero", storage.closes(), logger.closes(), pool.stops())
	}
}

// TestComponentsAreManagerOwnedByDefault verifies configured components are released unless marked borrowed. TestComponentsAreManagerOwnedByDefault 验证配置的组件默认随 Manager 释放。
func TestComponentsAreManagerOwnedByDefault(t *testing.T) {
	storage := &registryTestStorage{}
	logger := &registryTestLogger{NopLogger: adapter.NewNopLogger()}
	pool := &registryTestPool{}

	mgr, err := buildTestManager(NewBuilder().
		AuthType("manager-owned").
		IsPrintBanner(false).
		IsLog(true).
		AutoRenew(true).
		Storage(storage).
		SetLog(logger).
		SetPool(pool),
	)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	mgr.CloseManager()
	if storage.closes() != 1 || logger.closes() != 1 || pool.stops() != 1 {
		t.Fatalf("component close counts = storage:%d logger:%d pool:%d, want all one", storage.closes(), logger.closes(), pool.stops())
	}
}

// TestAuthCloseUnregistersManager verifies a registry-backed facade cannot leave a closed manager registered. TestAuthCloseUnregistersManager 验证注册表门面关闭后不会遗留已关闭 Manager。
func TestAuthCloseUnregistersManager(t *testing.T) {
	DeleteAllManager()
	t.Cleanup(DeleteAllManager)

	pool := &registryTestPool{}
	mgr := newRegistryTestManager("auth-close", pool)
	if err := SetManager(mgr); err != nil {
		t.Errorf("register authentication: %v", err)
	}

	selected, err := GetManager("auth-close")
	if err != nil {
		t.Fatalf("GetManager() error = %v", err)
	}
	auth := New(selected)
	auth.Close()

	if auth.Manager() != nil {
		t.Fatal("Auth.Manager() after Close = non-nil, want nil")
	}
	if _, err = GetManager("auth-close"); !errors.Is(err, derror.ErrManagerNotFound) {
		t.Fatalf("GetManager() after Auth.Close error = %v, want ErrManagerNotFound", err)
	}
	if pool.stops() != 1 {
		t.Fatalf("pool stop count = %d, want 1", pool.stops())
	}
}

// TestAuthCloseConcurrentAccess verifies concurrent reads never observe a nil manager without an error. TestAuthCloseConcurrentAccess 验证并发读取不会在无错误时取得空 Manager。
func TestAuthCloseConcurrentAccess(t *testing.T) {
	pool := &registryTestPool{}
	auth := New(newRegistryTestManager("auth-close-concurrent", pool))

	const workers = 32
	start := make(chan struct{})
	invalid := make(chan struct{}, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start

			mgr, err := requireTestManager(auth)
			if err == nil && mgr == nil {
				invalid <- struct{}{}
			}
			_ = auth.Manager()
			_ = auth.EventManager()
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-start
		auth.Close()
	}()

	close(start)
	wg.Wait()

	if len(invalid) != 0 {
		t.Fatal("requireManager() returned nil manager without an error")
	}
	if auth.Manager() != nil {
		t.Fatal("Auth.Manager() after concurrent Close = non-nil, want nil")
	}
	if pool.stops() != 1 {
		t.Fatalf("pool stop count = %d, want 1", pool.stops())
	}
}

// TestFixedAuthCloseWaitsForSharedCleanup verifies repeated and alias closure wait for the engine's ongoing cleanup. TestFixedAuthCloseWaitsForSharedCleanup 验证固定门面的重复关闭与别名关闭均等待引擎正在进行的清理。
func TestFixedAuthCloseWaitsForSharedCleanup(t *testing.T) {
	for _, shared := range []bool{false, true} {
		name := "same-auth"
		if shared {
			name = "shared-engine"
		}
		t.Run(name, func(t *testing.T) {
			started, release := make(chan struct{}), make(chan struct{})
			var releaseOnce sync.Once
			releaseCleanup := func() { releaseOnce.Do(func() { close(release) }) }
			defer releaseCleanup()
			mgr := newRegistryTestManager("fixed-close-wait", &registryTestBlockingPool{started: started, release: release})
			auth := New(mgr)
			other := auth
			if shared {
				other = New(mgr)
			}

			firstDone := make(chan struct{})
			go func() { auth.Close(); close(firstDone) }()
			select {
			case <-started:
			case <-time.After(time.Second):
				t.Fatal("first Close did not start resource cleanup")
			}
			if auth.Manager() != nil || other.Manager() != nil {
				t.Fatal("closing engine remains available through a fixed facade")
			}

			secondStarted, secondDone := make(chan struct{}), make(chan struct{})
			go func() {
				close(secondStarted)
				other.Close()
				close(secondDone)
			}()
			<-secondStarted
			select {
			case <-secondDone:
				t.Fatal("second Close returned before resource cleanup finished")
			case <-time.After(20 * time.Millisecond):
			}

			releaseCleanup()
			for _, done := range []chan struct{}{firstDone, secondDone} {
				select {
				case <-done:
				case <-time.After(time.Second):
					t.Fatal("Close did not return after resource cleanup finished")
				}
			}
		})
	}
}

// TestAuthCloseUsesManagerIdentity verifies changing a configuration snapshot cannot prevent unregistering. TestAuthCloseUsesManagerIdentity 验证修改配置快照不影响注销。
func TestAuthCloseUsesManagerIdentity(t *testing.T) {
	DeleteAllManager()
	t.Cleanup(DeleteAllManager)

	mgr := newRegistryTestManager("stable-registration", &registryTestPool{})
	if err := SetManager(mgr); err != nil {
		t.Errorf("register authentication: %v", err)
	}
	auth := New(mgr)
	mgr.GetConfig().SetAuthType("renamed-after-registration")

	auth.Close()
	if _, err := GetManager("stable-registration"); !errors.Is(err, derror.ErrManagerNotFound) {
		t.Fatalf("GetManager(stable-registration) after Auth.Close error = %v, want ErrManagerNotFound", err)
	}
}

// TestAuthClosePreservesReplacement verifies an old facade cannot unregister the current manager. TestAuthClosePreservesReplacement 验证旧门面无法注销当前 Manager。
func TestAuthClosePreservesReplacement(t *testing.T) {
	DeleteAllManager()
	t.Cleanup(DeleteAllManager)

	oldManager := newRegistryTestManager("auth-replacement", &registryTestPool{})
	newManager := newRegistryTestManager("auth-replacement", &registryTestPool{})
	if err := SetManager(oldManager); err != nil {
		t.Errorf("register authentication: %v", err)
	}
	oldAuth := New(oldManager)
	if err := SetManager(newManager); err != nil {
		t.Errorf("register authentication: %v", err)
	}

	oldAuth.Close()
	if got, err := GetManager("auth-replacement"); err != nil || got != newManager {
		t.Fatalf("GetManager(auth-replacement) = %v, %v, want replacement manager", got, err)
	}
}

// TestRegistryRejectsInvalidInstances verifies failed registration reports errors and preserves the active engine. TestRegistryRejectsInvalidInstances 验证注册失败返回错误且保留当前引擎。
func TestRegistryRejectsInvalidInstances(t *testing.T) {
	DeleteAllManager()
	t.Cleanup(DeleteAllManager)
	current := newRegistryTestManager("invalid-registration", nil)
	if err := SetManager(current); err != nil {
		t.Errorf("register authentication: %v", err)
	}
	closed := newRegistryTestManager("invalid-registration", nil)
	closed.CloseManager()
	for _, tc := range []struct {
		name     string
		register func() error
		want     error
	}{
		{"nil manager", func() error { return SetManager(nil) }, derror.ErrManagerNotFound},
		{"zero manager", func() error { return SetManager(&manager.Manager{}) }, derror.ErrManagerInvalidType},
		{"closed manager", func() error { return SetManager(closed) }, derror.ErrManagerNotFound},
		{"nil auth", func() error { return SetAuth(nil) }, derror.ErrManagerNotFound},
		{"zero auth", func() error { return SetAuth(&Auth{}) }, derror.ErrManagerNotFound},
		{"closed auth", func() error { return SetAuth(New(closed)) }, derror.ErrManagerNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.register(); !errors.Is(err, tc.want) {
				t.Fatalf("register = %v, want %v", err, tc.want)
			}
			if got, err := GetManager("invalid-registration"); err != nil || got != current || current.IsClosed() {
				t.Fatalf("failed registration changed current engine: %v, %v", got, err)
			}
		})
	}
}

func newRegistryTestManager(authType string, pool adapter.Pool) *manager.Manager {
	cfg := config.DefaultConfig()
	cfg.SetAuthType(authType)
	return manager.NewManager(cfg, nil, nil, nil, adapter.NewNopLogger(), pool, nil)
}

type registryTestPool struct {
	mu        sync.Mutex
	stopCount int
}

func (p *registryTestPool) Submit(task func()) error {
	if task != nil {
		task()
	}
	return nil
}

func (p *registryTestPool) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stopCount++
}

func (p *registryTestPool) Stats() (running, capacity int, usage float64) {
	return 0, 1, 0
}

func (p *registryTestPool) stops() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.stopCount
}

type registryTestBlockingPool struct {
	started chan<- struct{}
	release <-chan struct{}
}

func (p *registryTestBlockingPool) Submit(task func()) error {
	return nil
}

func (p *registryTestBlockingPool) Stop() {
	close(p.started)
	<-p.release
}

func (p *registryTestBlockingPool) Stats() (running, capacity int, usage float64) {
	return 0, 1, 0
}

type registryTestStorage struct {
	mu         sync.Mutex
	closeCount int
}

func (s *registryTestStorage) Set(context.Context, string, any, time.Duration) error {
	return nil
}

func (s *registryTestStorage) Get(context.Context, string) (any, error) {
	return nil, nil
}

func (s *registryTestStorage) Delete(context.Context, ...string) error {
	return nil
}

func (s *registryTestStorage) Exists(context.Context, string) bool {
	return false
}

func (s *registryTestStorage) Expire(context.Context, string, time.Duration) error {
	return nil
}

func (s *registryTestStorage) TTL(context.Context, string) (time.Duration, error) {
	return adapter.TTLNotFound, nil
}

func (s *registryTestStorage) Ping(context.Context) error {
	return nil
}

func (s *registryTestStorage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closeCount++
	return nil
}

func (s *registryTestStorage) closes() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closeCount
}

type registryTestLogger struct {
	*adapter.NopLogger
	mu         sync.Mutex
	closeCount int
}

func (l *registryTestLogger) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.closeCount++
}

func (l *registryTestLogger) Flush()                    {}
func (l *registryTestLogger) SetLevel(adapter.LogLevel) {}
func (l *registryTestLogger) SetPrefix(string)          {}
func (l *registryTestLogger) SetStdout(bool)            {}
func (l *registryTestLogger) LogPath() string           { return "" }
func (l *registryTestLogger) DropCount() uint64         { return 0 }

func (l *registryTestLogger) closes() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.closeCount
}
