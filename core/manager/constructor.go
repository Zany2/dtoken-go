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
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	if logger == nil {
		logger = adapter.NewNopLogger()
	}

	mgr := &Manager{
		config:         cfg,
		generator:      generator,
		storage:        storage,
		serializer:     serializer,
		logger:         logger,
		pool:           pool,
		eventManager:   listener.NewManager(logger),
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

// CloseManager closes the manager and releases all resources. CloseManager 关闭管理器并释放全部资源。
func (m *Manager) CloseManager() {
	m.stopBackgroundTasks()

	if m.pool != nil {
		m.pool.Stop()
		m.pool = nil
	}
	if m.eventManager != nil {
		m.eventManager.Wait()
	}
	if storageCloser, ok := m.storage.(interface{ Close() error }); ok {
		if err := storageCloser.Close(); err != nil {
			m.logger.Errorf("manager.CloseManager: failed to close storage, error=%v", err)
		}
	}
	if logControl, ok := m.logger.(adapter.LogControl); ok {
		logControl.Flush()
		logControl.Close()
	}
}
