// @Author daixk 2025/12/12 11:55:00
package ants

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/panjf2000/ants/v2"
)

// RenewPoolManager manages a dynamic scaling goroutine pool for renewal tasks 续期任务协程池管理器
type RenewPoolManager struct {
	pool      *ants.Pool       // ants pool instance ants 协程池实例
	config    *RenewPoolConfig // Configuration object 池配置对象
	mu        sync.Mutex       // Synchronization lock 互斥锁
	stopCh    chan struct{}    // Stop signal channel 停止信号通道
	started   bool             // Indicates whether the pool manager is running 是否已启动
	closeOnce sync.Once        // Ensure Stop only executes once 确保 Stop 只执行一次
	active    atomic.Int64     // Tasks currently executing, excluding idle workers 当前执行中的任务数，不包含空闲协程
}

// Interface assertion keeps pool contract checked at compile time 接口断言在编译期检查协程池契约
var _ adapter.Pool = (*RenewPoolManager)(nil)

// NewRenewPoolManagerWithDefaultConfig creates a renew pool manager with default config 使用默认配置创建续期池管理器
func NewRenewPoolManagerWithDefaultConfig() *RenewPoolManager {
	mgr, err := NewRenewPoolManagerWithConfig(DefaultRenewPoolConfig())
	if err != nil {
		return &RenewPoolManager{config: DefaultRenewPoolConfig(), stopCh: make(chan struct{})}
	}
	return mgr
}

// NewRenewPoolManagerWithConfig creates a renew pool manager with config 使用配置创建续期池管理器
func NewRenewPoolManagerWithConfig(cfg *RenewPoolConfig) (*RenewPoolManager, error) {
	if cfg == nil {
		cfg = DefaultRenewPoolConfig()
	}
	cfg = cfg.Clone()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	mgr := &RenewPoolManager{
		config:  cfg,
		stopCh:  make(chan struct{}),
		started: true,
	}

	if err := mgr.initPool(); err != nil {
		return nil, err
	}

	// Start the auto scaling routine 启动自动扩缩容协程
	go mgr.autoScale()

	return mgr, nil
}

// initPool initializes the ants pool 初始化 ants 协程池
func (m *RenewPoolManager) initPool() error {
	p, err := ants.NewPool(
		m.config.MinSize,
		ants.WithExpiryDuration(m.config.Expiry),
		ants.WithPreAlloc(m.config.PreAlloc),
		ants.WithNonblocking(m.config.NonBlocking),
	)
	if err != nil {
		return err
	}

	m.pool = p
	return nil
}

// Submit submits a renewal task 提交续期任务
// Task panics are recovered and logged by ants. 任务 panic 由 ants 恢复并记录日志。
func (m *RenewPoolManager) Submit(task func()) error {
	if m == nil {
		return fmt.Errorf("renew pool not started")
	}
	if task == nil {
		return fmt.Errorf("renew pool task is nil")
	}

	m.mu.Lock()
	pool := m.pool
	started := m.started
	m.mu.Unlock()

	if !started || pool == nil {
		return fmt.Errorf("renew pool not started")
	}
	return pool.Submit(func() {
		// Count execution only, including cleanup when a task panics. 仅统计执行中的任务，并在任务 panic 时清理计数。
		m.active.Add(1)
		defer m.active.Add(-1)
		task()
	})
}

// Stop rejects new work, stops scaling, and waits up to DefaultStopTimeout for workers. Stop 拒绝新任务、停止扩缩容，并最多等待 DefaultStopTimeout 让协程退出。
// Tasks still running after the timeout continue until they return. 超时后仍在执行的任务会继续运行直至返回。
func (m *RenewPoolManager) Stop() {
	if m == nil {
		return
	}
	m.closeOnce.Do(func() {
		m.mu.Lock()
		if !m.started {
			m.mu.Unlock()
			return
		}
		close(m.stopCh)
		m.started = false
		pool := m.pool
		m.mu.Unlock()

		if pool != nil && !pool.IsClosed() {
			_ = pool.ReleaseTimeout(DefaultStopTimeout)
		}
	})
}

// Stats returns current pool statistics 返回当前池状态
func (m *RenewPoolManager) Stats() (running, capacity int, usage float64) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.pool == nil {
		return
	}
	running = int(m.active.Load()) // Active tasks 当前运行任务数
	capacity = m.pool.Cap()        // Pool capacity 当前池容量
	if capacity > 0 {
		usage = float64(running) / float64(capacity) // Usage ratio 当前使用率
		// Running tasks may temporarily exceed capacity after shrinking. 缩容后执行中的任务数可能暂时超过容量。
		if usage > 1.0 {
			usage = 1.0
		}
	}

	return
}

// autoScale runs automatic pool scale up and down logic 自动扩缩容逻辑
func (m *RenewPoolManager) autoScale() {
	if m == nil {
		return
	}
	m.mu.Lock()
	config := m.config
	m.mu.Unlock()
	if config == nil {
		return
	}
	ticker := time.NewTicker(config.CheckInterval) // Ticker for periodic usage checks 定时器，用于定期检测使用率
	defer ticker.Stop()                            // Stop ticker on exit 函数退出时停止定时器

	for {
		select {
		case <-ticker.C:
			m.mu.Lock() // Protect concurrent access 加锁防止并发冲突
			if !m.started || m.pool == nil || m.config == nil {
				m.mu.Unlock()
				continue
			}

			// Get current pool stats 获取当前运行状态
			running := int(m.active.Load()) // Exclude cached idle workers 不计入缓存的空闲协程
			capacity := m.pool.Cap()        // Current pool capacity 当前协程池容量

			// Skip when capacity is 0 to avoid division by zero 容量为 0 时跳过，避免除零
			if capacity <= 0 {
				m.mu.Unlock()
				continue
			}

			usage := float64(running) / float64(capacity) // Current usage ratio 当前使用率（运行数 ÷ 总容量）

			switch {
			// Expand when usage exceeds the threshold and capacity is below MaxSize 当使用率超过扩容阈值且容量小于最大值时扩容
			case usage > m.config.ScaleUpRate && capacity < m.config.MaxSize:
				// Clamp before converting to int to avoid overflow on 32-bit platforms. 转换为 int 前限制上界，避免在 32 位平台溢出。
				newCap := m.config.MaxSize
				if scaled := float64(capacity) * 1.5; scaled < float64(newCap) {
					newCap = int(scaled)
				}
				if newCap <= capacity {
					newCap = capacity + 1 // Ensure a scale-up for small capacities 确保小容量场景也能实际扩容
				}
				m.pool.Tune(newCap) // Apply new pool capacity 调整 ants 池容量

			// Reduce when usage is below the threshold and capacity is above MinSize 当使用率低于缩容阈值且容量大于最小值时缩容
			case usage < m.config.ScaleDownRate && capacity > m.config.MinSize:
				newCap := int(float64(capacity) * 0.7) // Reduce capacity to 70% 缩容为当前的 70%
				if newCap < m.config.MinSize {         // Ensure not below MinSize 限制最小值
					newCap = m.config.MinSize
				}
				m.pool.Tune(newCap) // Apply new pool capacity 调整 ants 池容量
			}

			m.mu.Unlock() // Unlock after adjustment 解锁

		case <-m.stopCh:
			// Exit the loop when a stop signal is received 收到停止信号，终止扩缩容协程
			return
		}
	}
}
