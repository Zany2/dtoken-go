// @Author daixk 2025/12/22 15:56:00
package manager

import (
	"sync"
	"time"
)

// managerBackground stores lifecycle state for manager background tasks. managerBackground 存储管理器后台任务生命周期状态。
type managerBackground struct {
	closeCh   chan struct{}  // closeCh stops background tasks. closeCh 停止后台任务。
	closeOnce sync.Once      // closeOnce closes closeCh once. closeOnce 确保 closeCh 只关闭一次。
	wg        sync.WaitGroup // wg waits background tasks. wg 等待后台任务退出。
}

// managerBackgrounds maps managers to background lifecycle. managerBackgrounds 映射管理器后台生命周期。
var managerBackgrounds sync.Map

// backgroundForManager returns lifecycle state for a manager. backgroundForManager 返回管理器生命周期状态。
func backgroundForManager(m *Manager) *managerBackground {
	// Build default background state 构建默认后台状态。
	background := &managerBackground{closeCh: make(chan struct{})}
	// Reuse existing state when present 存在时复用已有状态。
	value, _ := managerBackgrounds.LoadOrStore(m, background)
	// Return lifecycle state 返回生命周期状态。
	return value.(*managerBackground)
}

// StartRenewPoolStatusLogger starts renew pool status logging. StartRenewPoolStatusLogger 启动续期池状态日志。
func (m *Manager) StartRenewPoolStatusLogger(interval time.Duration) {
	// Validate logger preconditions 校验日志器启动条件。
	if m == nil || interval <= 0 || m.pool == nil || m.logger == nil {
		return
	}

	// Get background lifecycle 获取后台生命周期。
	background := backgroundForManager(m)
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
}

// stopBackgroundTasks stops manager background tasks. stopBackgroundTasks 停止管理器后台任务。
func (m *Manager) stopBackgroundTasks() {
	// Ignore nil manager 忽略空管理器。
	if m == nil {
		return
	}

	// Remove lifecycle state 移除生命周期状态。
	value, ok := managerBackgrounds.LoadAndDelete(m)
	if !ok {
		return
	}

	// Close stop channel once 只关闭一次停止通道。
	background := value.(*managerBackground)
	background.closeOnce.Do(func() {
		close(background.closeCh)
	})

	// Wait background tasks 等待后台任务退出。
	background.wg.Wait()
}
