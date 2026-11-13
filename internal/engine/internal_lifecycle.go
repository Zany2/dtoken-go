// @Author daixk 2025/12/22 15:56:00
package engine

import (
	"sync"
	"sync/atomic"
	"time"

	adapter "github.com/Zany2/dtoken-go/contract"
)

// managerBackground stores lifecycle state for manager background tasks. managerBackground 存储管理器后台任务生命周期状态。
type managerBackground struct {
	closeCh          chan struct{}  // closeCh stops background tasks. closeCh 停止后台任务。
	closeOnce        sync.Once      // closeOnce closes closeCh once. closeOnce 确保 closeCh 只关闭一次。
	statusLoggerOnce sync.Once      // statusLoggerOnce starts status logger once. statusLoggerOnce 确保状态日志只启动一次。
	wg               sync.WaitGroup // wg waits background tasks. wg 等待后台任务退出。
}

// managerRuntime holds shutdown ownership and per-instance background work. managerRuntime 持有关闭所有权及实例级后台任务。
type managerRuntime struct {
	asyncMu     sync.Mutex         // asyncMu serializes async task admission and shutdown. asyncMu 串行化异步任务接收与关闭。
	asyncWG     sync.WaitGroup     // asyncWG waits for accepted async tasks. asyncWG 等待已接收的异步任务完成。
	asyncClosed bool               // asyncClosed rejects new async tasks during shutdown. asyncClosed 在关闭期间拒绝新的异步任务。
	closed      atomic.Bool        // closed reports whether manager shutdown has started. closed 标记 Manager 是否已开始关闭。
	closeOnce   sync.Once          // closeOnce releases manager resources once. closeOnce 确保资源只释放一次。
	ownership   ComponentOwnership // ownership records which runtime components are manager-owned. ownership 记录由 Manager 持有的运行时组件。
	background  *managerBackground // background is guarded by asyncMu. background 由 asyncMu 保护。
}

// StartRenewPoolStatusLogger starts renew pool status logging. StartRenewPoolStatusLogger 启动续期池状态日志。
func (m *Manager) StartRenewPoolStatusLogger(interval time.Duration) {
	// Validate immutable preconditions 校验不可变启动条件。
	if m == nil || interval <= 0 {
		return
	}

	// Serialize logger startup with manager shutdown. 将状态日志启动与 Manager 关闭流程串行化。
	m.asyncMu.Lock()
	defer m.asyncMu.Unlock()
	if m.asyncClosed || m.pool == nil || m.logger == nil {
		return
	}

	// Get background lifecycle 获取后台生命周期。
	if m.background == nil {
		m.background = &managerBackground{closeCh: make(chan struct{})}
	}
	background := m.background

	// Start only one logger per manager lifecycle 每个管理器生命周期只启动一个日志协程。
	background.statusLoggerOnce.Do(func() {
		// Track logger goroutine 跟踪日志协程。
		background.wg.Add(1)
		go func() {
			// Mark goroutine done 标记协程结束。
			defer background.wg.Done()

			// Create interval ticker 创建间隔定时器。
			ticker := time.NewTicker(interval)
			defer ticker.Stop()

			// Run until stopped 运行直到停止。
			for {
				select {
				case <-ticker.C:
					// Skip when dependencies are gone 依赖缺失时跳过。
					if m.pool == nil || m.logger == nil {
						continue
					}

					// Read pool status 读取协程池状态。
					running, capacity, usage := m.pool.Stats()
					m.logger.Infof(
						"manager.StartRenewPoolStatusLogger: renew pool status, capacity=%d, running=%d, usage=%.2f%%",
						capacity, running, usage*100,
					)
				case <-background.closeCh:
					// Stop logger goroutine 停止日志协程。
					return
				}
			}
		}()
	})
}

// stopBackgroundTasks stops manager background tasks. stopBackgroundTasks 停止管理器后台任务。
func (m *Manager) stopBackgroundTasks() {
	// Ignore nil manager 忽略空管理器。
	if m == nil {
		return
	}

	// Detach background state while startup remains serialized. 在启动同步锁内分离后台状态。
	m.asyncMu.Lock()
	background := m.background
	m.background = nil
	m.asyncMu.Unlock()
	if background == nil {
		return
	}

	// Close stop channel once 只关闭一次停止通道。
	background.closeOnce.Do(func() {
		close(background.closeCh)
	})

	// Wait background tasks 等待后台任务退出。
	background.wg.Wait()
}

// CloseManager rejects new async work, waits for accepted tasks and releases owned resources; stop request producers first and call outside manager tasks and listeners to avoid self-deadlock. CloseManager 拒绝新异步任务，等待已接收任务并释放自有资源；须先停止请求生产方，并在 Manager 任务和监听器之外调用，避免等待自身。
func (m *Manager) CloseManager() {
	if m == nil {
		return
	}
	m.closed.Store(true)

	m.closeOnce.Do(func() {
		// Reject new async tasks before releasing runtime dependencies. 释放运行时依赖前拒绝新的异步任务。
		m.asyncMu.Lock()
		m.asyncClosed = true
		m.asyncMu.Unlock()

		m.stopBackgroundTasks()

		// Wait for this manager's accepted tasks even when the pool is caller-owned. 即使协程池由调用方持有，也等待当前 Manager 已接收的任务。
		m.asyncWG.Wait()

		// Listeners may still use the pool, storage, and logger after event dispatch returns. 事件分发返回后，监听器仍可能使用协程池、存储和日志器。
		if m.eventManager != nil {
			m.eventManager.Wait()
		}

		if m.pool != nil && m.ownership.Pool {
			m.pool.Stop()
		}
		m.pool = nil
		if storageCloser, ok := m.storage.(interface{ Close() error }); ok && m.ownership.Storage {
			if err := storageCloser.Close(); err != nil {
				m.logger.Errorf("manager.CloseManager: failed to close storage, error=%v", err)
			}
		}
		if logControl, ok := m.logger.(adapter.LogControl); ok && m.ownership.Logger {
			logControl.Flush()
			logControl.Close()
		}
	})
}

// submitAsync submits async work with a goroutine fallback and reports admission. submitAsync 提交异步任务并在池不可用时回退到 goroutine，同时返回是否已接收。
func (m *Manager) submitAsync(name string, task func()) bool {
	// Register the task unless manager shutdown has started. Manager 开始关闭后不再接收新任务。
	m.asyncMu.Lock()
	if m.asyncClosed {
		m.asyncMu.Unlock()
		return false
	}
	m.asyncWG.Add(1)
	pool := m.pool
	m.asyncMu.Unlock()

	// Track task completion independently from pool ownership. 独立于协程池所有权跟踪任务完成状态。
	trackedTask := func() {
		defer m.asyncWG.Done()

		// Keep goroutine fallback as safe as pool execution when a component panics. 组件发生 panic 时，确保 goroutine 回退与协程池执行一样不会终止进程。
		defer func() {
			if recovered := recover(); recovered != nil {
				m.logger.Errorf("manager.submitAsync: task panic recovered, task=%s, panic=%v", name, recovered)
			}
		}()
		task()
	}

	// Fallback when pool is absent 协程池不存在时回退。
	if pool == nil {
		go trackedTask()
		return true
	}

	// Submit task to pool 提交任务到协程池。
	if err := pool.Submit(trackedTask); err != nil {
		m.logger.Errorf("manager.submitAsync: failed to submit async task, task=%s, error=%v", name, err)

		// Fallback when submit fails 提交失败时回退。
		go trackedTask()
	}
	return true
}
