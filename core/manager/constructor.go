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

// Option configures optional manager modules Option 配置 Manager 的可选模块
type Option func(m *Manager)

// WithNonceManager sets the optional nonce manager WithNonceManager 设置可选 Nonce 管理器
func WithNonceManager(nonceManager *nonce.NonceManager) Option {
	return func(m *Manager) {
		// Set nonce manager when provided 提供时设置 Nonce 管理器。
		if nonceManager != nil {
			m.nonceManager = nonceManager
		}
	}
}

// WithOAuth2Manager sets the optional OAuth2 server WithOAuth2Manager 设置可选 OAuth2 服务端
func WithOAuth2Manager(oauth2Manager *oauth2.OAuth2Server) Option {
	return func(m *Manager) {
		// Set OAuth2 manager when provided 提供时设置 OAuth2 管理器。
		if oauth2Manager != nil {
			m.oauth2Manager = oauth2Manager
		}
	}
}

// WithTicketManager sets the optional ticket manager. WithTicketManager 设置可选 Ticket 管理器。
func WithTicketManager(ticketManager *ticket.Manager) Option {
	return func(m *Manager) {
		// Set ticket manager when provided. 提供时设置 Ticket 管理器。
		if ticketManager != nil {
			m.ticketManager = ticketManager
		}
	}
}

// WithShortKeyManager sets the optional short key manager. WithShortKeyManager 设置可选短 Key 管理器。
func WithShortKeyManager(shortKeyManager *shortkey.Manager) Option {
	return func(m *Manager) {
		// Set short key manager when provided. 提供时设置短 Key 管理器。
		if shortKeyManager != nil {
			m.shortKeyManager = shortKeyManager
		}
	}
}

// NewManager creates a manager with the provided core components. NewManager 使用提供的核心组件创建管理器。
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
	// Use default config when absent 缺省时使用默认配置。
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	// Use no-op logger when absent 缺省时使用空日志器。
	if logger == nil {
		logger = adapter.NewNopLogger()
	}

	// Build manager with core components 构建只包含核心组件的管理器。
	mgr := &Manager{
		config:         cfg,
		generator:      generator,
		storage:        storage,
		serializer:     serializer,
		logger:         logger,
		pool:           pool,
		eventManager:   listener.NewManager(logger),
		accessProvider: accessProvider,
	}

	// Apply optional module assembly options 应用可选模块装配项。
	for _, option := range options {
		// Apply non-nil option 应用非空选项。
		if option != nil {
			option(mgr)
		}
	}

	// Return manager 返回管理器。
	return mgr
}

// CloseManager closes the manager and releases all resources. CloseManager 关闭管理器并释放全部资源。
func (m *Manager) CloseManager() {
	// Stop background tasks 停止后台任务。
	m.stopBackgroundTasks()

	// Stop async pool 停止异步池。
	if m.pool != nil {
		m.pool.Stop()
		m.pool = nil
	}
	// Wait event manager 等待事件管理器。
	if m.eventManager != nil {
		m.eventManager.Wait()
	}
	// Close storage adapter 关闭存储适配器。
	if storageCloser, ok := m.storage.(interface{ Close() error }); ok {
		if err := storageCloser.Close(); err != nil {
			m.logger.Errorf("manager.CloseManager: failed to close storage, error=%v", err)
		}
	}
	// Flush and close logger 刷新并关闭日志器。
	if logControl, ok := m.logger.(adapter.LogControl); ok {
		logControl.Flush()
		logControl.Close()
	}
}
