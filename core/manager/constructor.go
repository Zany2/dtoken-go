// @Author daixk 2025/12/22 15:56:00
package manager

import (
	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/listener"
	"github.com/Zany2/dtoken-go/core/nonce"
	"github.com/Zany2/dtoken-go/core/oauth2"
	"github.com/Zany2/dtoken-go/core/shortkey"
	"github.com/Zany2/dtoken-go/core/ticket"
)

// Option configures optional manager modules. Option 配置 Manager 的可选模块。
type Option func(m *Manager)

// ComponentOwnership declares which runtime components are owned and closed by Manager. ComponentOwnership 声明由 Manager 持有并负责关闭的运行时组件。
type ComponentOwnership struct {
	Storage bool // Storage marks the storage adapter as manager-owned. Storage 标记存储适配器由 Manager 持有。
	Logger  bool // Logger marks the logger as manager-owned. Logger 标记日志器由 Manager 持有。
	Pool    bool // Pool marks the task pool as manager-owned. Pool 标记任务池由 Manager 持有。
}

// WithNonceManager sets the optional nonce manager. WithNonceManager 设置可选 Nonce 管理器。
func WithNonceManager(nonceManager *nonce.NonceManager) Option {
	return func(m *Manager) {
		if nonceManager != nil {
			m.nonceManager = nonceManager
		}
	}
}

// WithOAuth2Manager sets the optional OAuth2 server. WithOAuth2Manager 设置可选 OAuth2 服务端。
func WithOAuth2Manager(oauth2Manager *oauth2.OAuth2Server) Option {
	return func(m *Manager) {
		if oauth2Manager != nil {
			m.oauth2Manager = oauth2Manager
		}
	}
}

// WithTicketManager sets the optional ticket manager. WithTicketManager 设置可选 Ticket 管理器。
func WithTicketManager(ticketManager *ticket.Manager) Option {
	return func(m *Manager) {
		if ticketManager != nil {
			m.ticketManager = ticketManager
		}
	}
}

// WithShortKeyManager sets the optional short key manager. WithShortKeyManager 设置可选短 Key 管理器。
func WithShortKeyManager(shortKeyManager *shortkey.Manager) Option {
	return func(m *Manager) {
		if shortKeyManager != nil {
			m.shortKeyManager = shortKeyManager
		}
	}
}

// WithStrategy sets replaceable manager algorithms. WithStrategy 设置可替换的管理器算法。
func WithStrategy(strategy *Strategy) Option {
	return func(m *Manager) {
		if strategy != nil {
			m.strategy = strategy.normalize()
		}
	}
}

// WithComponentOwnership configures the complete runtime ownership policy; false fields remain caller-owned. WithComponentOwnership 配置完整的运行时所有权策略，值为 false 的组件仍由调用方持有。
func WithComponentOwnership(ownership ComponentOwnership) Option {
	return func(m *Manager) {
		m.ownership = ownership
	}
}

// NewManager creates a manager with the provided core components and owns them by default. NewManager 使用提供的核心组件创建管理器，并默认持有这些组件。
func NewManager(
	cfg *config.Config,
	generator adapter.Generator,
	storage adapter.Storage,
	serializer adapter.Codec,
	logger adapter.Log,
	pool adapter.Pool,
	accessProvider AccessProvider,
	options ...Option,
) *Manager {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	if logger == nil {
		logger = adapter.NewNopLogger()
	}

	mgr := &Manager{
		config:       cfg,
		generator:    generator,
		storage:      storage,
		serializer:   serializer,
		logger:       logger,
		pool:         pool,
		eventManager: listener.NewManager(logger),
		ownership: ComponentOwnership{
			Storage: true,
			Logger:  true,
			Pool:    true,
		},
		accessProvider: accessProvider,
		strategy:       DefaultStrategy(),
	}

	for _, option := range options {
		if option != nil {
			option(mgr)
		}
	}
	mgr.strategy = mgr.strategy.normalize()

	return mgr
}

// CloseManager rejects new async work, waits for accepted tasks, and releases manager-owned resources. CloseManager 拒绝新的异步任务，等待已接收任务完成，并释放 Manager 自有资源。
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

		if m.pool != nil && m.ownership.Pool {
			m.pool.Stop()
		}
		m.pool = nil
		if m.eventManager != nil {
			m.eventManager.Wait()
		}
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
