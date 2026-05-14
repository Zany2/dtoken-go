package manager

import (
	"sync"
	"time"
)

// managerBackground stores lifecycle state for manager background tasks managerBackground 存储管理器后台任务生命周期状态
type managerBackground struct {
	closeCh   chan struct{}  // closeCh stops background tasks closeCh 停止后台任务
	closeOnce sync.Once      // closeOnce closes closeCh once closeOnce 确保 closeCh 只关闭一次
	wg        sync.WaitGroup // wg waits background tasks wg 等待后台任务退出
}

var managerBackgrounds sync.Map // managerBackgrounds maps managers to background lifecycle managerBackgrounds 映射管理器后台生命周期

// backgroundForManager returns lifecycle state for a manager backgroundForManager 返回管理器生命周期状态
func backgroundForManager(m *Manager) *managerBackground {
	background := &managerBackground{closeCh: make(chan struct{})}
	value, _ := managerBackgrounds.LoadOrStore(m, background)
	return value.(*managerBackground)
}

// StartRenewPoolStatusLogger starts renew pool status logging StartRenewPoolStatusLogger 启动续期池状态日志
func (m *Manager) StartRenewPoolStatusLogger(interval time.Duration) {
	if m == nil || interval <= 0 || m.pool == nil || m.logger == nil {
		return
	}

	background := backgroundForManager(m)
	background.wg.Add(1)
	go func() {
		defer background.wg.Done()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if m.pool == nil || m.logger == nil {
					continue
				}
				running, capacity, usage := m.pool.Stats()
				m.logger.Infof(
					"manager.StartRenewPoolStatusLogger: renew pool status, capacity=%d, running=%d, usage=%.2f%%",
					capacity, running, usage*100,
				)
			case <-background.closeCh:
				return
			}
		}
	}()
}

// stopBackgroundTasks stops manager background tasks stopBackgroundTasks 停止管理器后台任务
func (m *Manager) stopBackgroundTasks() {
	if m == nil {
		return
	}

	value, ok := managerBackgrounds.LoadAndDelete(m)
	if !ok {
		return
	}

	background := value.(*managerBackground)
	background.closeOnce.Do(func() {
		close(background.closeCh)
	})

	background.wg.Wait()
}
