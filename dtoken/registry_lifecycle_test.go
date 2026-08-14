package dtoken

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/manager"
)

// TestSetManagerClosesReplacedManager verifies same-type replacement releases the previous manager. TestSetManagerClosesReplacedManager 验证同类型替换会释放旧 Manager。
func TestSetManagerClosesReplacedManager(t *testing.T) {
	DeleteAllManager()
	t.Cleanup(DeleteAllManager)

	oldPool := &registryTestPool{}
	newPool := &registryTestPool{}
	oldManager := newRegistryTestManager("replace", oldPool)
	newManager := newRegistryTestManager("replace", newPool)

	SetManager(oldManager)
	SetManager(newManager)

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
	SetManager(mgr)
	SetManager(mgr)

	if mgr.IsClosed() {
		t.Fatal("manager is closed after duplicate registration")
	}
	if pool.stops() != 0 {
		t.Fatalf("pool stop count = %d, want 0", pool.stops())
	}
}

// TestSetManagerMovesInstanceToCurrentAuthType verifies one manager cannot remain registered under a stale auth type. TestSetManagerMovesInstanceToCurrentAuthType 验证同一 Manager 不会残留在旧认证类型下。
func TestSetManagerMovesInstanceToCurrentAuthType(t *testing.T) {
	DeleteAllManager()
	t.Cleanup(DeleteAllManager)

	mgr := newRegistryTestManager("original-registration", &registryTestPool{})
	SetManager(mgr)
	mgr.GetConfig().SetAuthType("moved-registration")
	SetManager(mgr)

	if _, err := GetManager("original-registration"); !errors.Is(err, derror.ErrManagerNotFound) {
		t.Fatalf("GetManager(original-registration) error = %v, want ErrManagerNotFound", err)
	}
	if got, err := GetManager("moved-registration"); err != nil || got != mgr {
		t.Fatalf("GetManager(moved-registration) = %v, %v, want moved manager", got, err)
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
	SetManager(oldManager)

	replaceDone := make(chan struct{})
	go func() {
		SetManager(newManager)
		close(replaceDone)
	}()

	select {
	case <-stopStarted:
	case <-time.After(time.Second):
		t.Fatal("replacement did not start closing the old manager")
	}

	SetManager(oldManager)
	if got, err := GetManager("retiring"); err != nil || got != newManager {
		t.Fatalf("GetManager(retiring) during cleanup = %v, %v, want new manager", got, err)
	}

	close(stopRelease)
	select {
	case <-replaceDone:
	case <-time.After(time.Second):
		t.Fatal("replacement did not finish")
	}

	SetManager(oldManager)
	if got, err := GetManager("retiring"); err != nil || got != newManager {
		t.Fatalf("GetManager(retiring) after cleanup = %v, %v, want new manager", got, err)
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
			SetManager(oldManager)

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

			SetManager(newManager)
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
		mgr, err := NewBuilder().
			AuthType(authType).
			IsPrintBanner(false).
			IsLog(true).
			AutoRenew(true).
			SetStorage(storage).
			SetLog(logger).
			SetPool(pool).
			UseManagerOption(manager.WithComponentOwnership(manager.ComponentOwnership{})).
			Build()
		if err != nil {
			t.Fatalf("Build(%s) error = %v", authType, err)
		}
		SetManager(mgr)
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

	mgr, err := NewBuilder().
		AuthType("manager-owned").
		IsPrintBanner(false).
		IsLog(true).
		AutoRenew(true).
		SetStorage(storage).
		SetLog(logger).
		SetPool(pool).
		Build()
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
	SetManager(mgr)

	auth, err := NewByAuthType("auth-close")
	if err != nil {
		t.Fatalf("NewByAuthType() error = %v", err)
	}
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

			mgr, err := auth.requireManager()
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

// TestAuthCloseUsesManagerIdentity verifies unregistering does not depend on mutable auth type configuration. TestAuthCloseUsesManagerIdentity 验证注销不依赖可变的认证类型配置。
func TestAuthCloseUsesManagerIdentity(t *testing.T) {
	DeleteAllManager()
	t.Cleanup(DeleteAllManager)

	mgr := newRegistryTestManager("stable-registration", &registryTestPool{})
	SetManager(mgr)
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
	SetManager(oldManager)
	oldAuth := New(oldManager)
	SetManager(newManager)

	oldAuth.Close()
	if got, err := GetManager("auth-replacement"); err != nil || got != newManager {
		t.Fatalf("GetManager(auth-replacement) = %v, %v, want replacement manager", got, err)
	}
}

// TestRegistryRejectsInvalidBuilderResults verifies invalid inputs do not panic or register unusable managers. TestRegistryRejectsInvalidBuilderResults 验证非法输入不会 panic 或注册不可用 Manager。
func TestRegistryRejectsInvalidBuilderResults(t *testing.T) {
	DeleteAllManager()
	t.Cleanup(DeleteAllManager)

	SetManager(nil)
	SetManager(&manager.Manager{})
	if _, err := GetManager(); !errors.Is(err, derror.ErrManagerNotFound) {
		t.Fatalf("GetManager() after invalid SetManager calls error = %v, want ErrManagerNotFound", err)
	}

	invalidBuilder := &registryTestBuilder{}
	if _, err := BuildAndSetManager(invalidBuilder); !errors.Is(err, derror.ErrManagerInvalidType) {
		t.Fatalf("BuildAndSetManager(nil result) error = %v, want ErrManagerInvalidType", err)
	}
	var nilCustomBuilder *registryTestBuilder
	if _, err := BuildAndSetManager(nilCustomBuilder); !errors.Is(err, derror.ErrManagerNotFound) {
		t.Fatalf("BuildAndSetManager(typed nil) error = %v, want ErrManagerNotFound", err)
	}

	closedManager := newRegistryTestManager("closed-builder", nil)
	closedManager.CloseManager()
	closedBuilder := &registryTestBuilder{manager: closedManager}
	if _, err := BuildAndSetManager(closedBuilder); !errors.Is(err, derror.ErrManagerInvalidType) {
		t.Fatalf("BuildAndSetManager(closed result) error = %v, want ErrManagerInvalidType", err)
	}

	validManager := newRegistryTestManager("custom-builder", nil)
	validBuilder := &registryTestBuilder{manager: validManager}
	if got, err := BuildAndSetManager(validBuilder); err != nil || got != validManager {
		t.Fatalf("BuildAndSetManager(custom builder) = %v, %v, want custom manager", got, err)
	}
	if validBuilder.builds != 1 {
		t.Fatalf("valid custom builder build count = %d, want 1", validBuilder.builds)
	}

	customManager := newRegistryTestManager("custom-override", nil)
	customBuilder := &registryTestBuilder{manager: customManager}
	if _, err := BuildAndSetManager(customBuilder, "override"); !errors.Is(err, derror.ErrInvalidParam) {
		t.Fatalf("BuildAndSetManager(custom override) error = %v, want ErrInvalidParam", err)
	}
	if customBuilder.builds != 0 {
		t.Fatalf("custom builder build count = %d, want 0", customBuilder.builds)
	}
	customManager.CloseManager()
}

func newRegistryTestManager(authType string, pool adapter.Pool) *manager.Manager {
	cfg := config.DefaultConfig()
	cfg.SetAuthType(authType)
	return manager.NewManager(cfg, nil, nil, nil, adapter.NewNopLogger(), pool, nil)
}

type registryTestBuilder struct {
	manager *manager.Manager
	builds  int
}

func (b *registryTestBuilder) Build() (*manager.Manager, error) {
	b.builds++
	return b.manager, nil
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
