package manager

import (
	"sync"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/config"
)

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
	waitForManagerTest(t, time.Second, func() bool {
		mgr.asyncMu.Lock()
		defer mgr.asyncMu.Unlock()
		return mgr.asyncClosed
	})

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
