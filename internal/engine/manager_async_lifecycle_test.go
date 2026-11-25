package engine

import (
	"sync"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/config"
)

// TestBeginCloseStopsAdmissionBeforeCleanup verifies shutdown becomes visible without releasing dependencies or starting new background work. TestBeginCloseStopsAdmissionBeforeCleanup 验证关闭标记立即生效，但不释放依赖或启动新的后台任务。
func TestBeginCloseStopsAdmissionBeforeCleanup(t *testing.T) {
	pool := &managerLifecycleTestPool{}
	logger := &managerLifecycleTestLogger{}
	mgr := newTestManagerWithRuntime(t, pool, logger)
	t.Cleanup(mgr.CloseManager)

	BeginClose(mgr)
	BeginClose(mgr)
	if !mgr.IsClosed() {
		t.Fatal("manager remains available after BeginClose")
	}
	if mgr.submitAsync("after shutdown begins", func() {}) {
		t.Fatal("manager accepted a task after BeginClose")
	}
	mgr.StartRenewPoolStatusLogger(time.Second)
	if mgr.background != nil {
		t.Fatal("manager started background work after BeginClose")
	}
	if pool.stopCount() != 0 || logger.closeCount() != 0 {
		t.Fatal("BeginClose released dependencies before cleanup")
	}

	mgr.CloseManager()
	mgr.CloseManager()
	if pool.stopCount() != 1 || logger.closeCount() != 1 {
		t.Fatal("CloseManager did not release dependencies exactly once")
	}
}

// TestCloseManagerWaitsForTasksWithCallerOwnedPool verifies mixed ownership keeps dependencies alive until accepted tasks finish. TestCloseManagerWaitsForTasksWithCallerOwnedPool 验证混合所有权下依赖会保持可用，直至已接收任务完成。
func TestCloseManagerWaitsForTasksWithCallerOwnedPool(t *testing.T) {
	storage := &managerAsyncCloseTestStorage{managerTestStorage: newManagerTestStorage()}
	logger := &managerLifecycleTestLogger{}
	pool := &managerLifecycleTestPool{}
	mgr := NewManager(
		config.DefaultConfig(),
		nil,
		storage,
		nil,
		logger,
		pool,
		nil,
		WithComponentOwnership(ComponentOwnership{
			Storage: true,
			Logger:  true,
			Pool:    false,
		}),
	)
	defer mgr.CloseManager()

	taskStarted := make(chan struct{})
	taskRelease := make(chan struct{})
	var releaseOnce sync.Once
	releaseTask := func() {
		releaseOnce.Do(func() {
			close(taskRelease)
		})
	}
	defer releaseTask()

	mgr.submitAsync("mixed ownership close", func() {
		close(taskStarted)
		<-taskRelease
	})
	select {
	case <-taskStarted:
	case <-time.After(time.Second):
		t.Fatal("async task did not start")
	}

	closeDone := make(chan struct{})
	go func() {
		mgr.CloseManager()
		close(closeDone)
	}()
	waitForManagerTest(t, time.Second, mgr.IsClosed)

	if storage.closes() != 0 || logger.closeCount() != 0 {
		t.Fatalf("dependencies closed before task completion: storage=%d logger=%d", storage.closes(), logger.closeCount())
	}
	select {
	case <-closeDone:
		t.Fatal("CloseManager returned before accepted task completion")
	default:
	}

	rejectedTaskRan := make(chan struct{}, 1)
	mgr.submitAsync("rejected during close", func() {
		rejectedTaskRan <- struct{}{}
	})
	releaseTask()

	select {
	case <-closeDone:
	case <-time.After(time.Second):
		t.Fatal("CloseManager did not return after accepted task completion")
	}
	if pool.stopCount() != 0 {
		t.Fatalf("caller-owned pool stop count = %d, want 0", pool.stopCount())
	}
	if storage.closes() != 1 || logger.flushCount() != 1 || logger.closeCount() != 1 {
		t.Fatalf(
			"owned dependency close counts = storage:%d logger flush/close:%d/%d, want 1 and 1/1",
			storage.closes(),
			logger.flushCount(),
			logger.closeCount(),
		)
	}
	select {
	case <-rejectedTaskRan:
		t.Fatal("task submitted during close was executed")
	default:
	}
}

type managerAsyncCloseTestStorage struct {
	*managerTestStorage
	mu         sync.Mutex
	closeCount int
}

func (s *managerAsyncCloseTestStorage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closeCount++
	return nil
}

func (s *managerAsyncCloseTestStorage) closes() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closeCount
}
