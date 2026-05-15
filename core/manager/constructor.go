// @Author daixk 2025/12/22 15:56:00
package manager

import (
	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/listener"
	"github.com/Zany2/dtoken-go/core/nonce"
	"github.com/Zany2/dtoken-go/core/oauth2"
)

// Option configures optional manager modules Option 配置 Manager 的可选模块
type Option func(m *Manager)

// WithNonceManager replaces the default nonce manager WithNonceManager 替换默认 Nonce 管理器
func WithNonceManager(nonceManager *nonce.NonceManager) Option {
	return func(m *Manager) {
		if nonceManager != nil {
			m.nonceManager = nonceManager
		}
	}
}

// WithOAuth2Manager replaces the default OAuth2 server WithOAuth2Manager 替换默认 OAuth2 服务端
func WithOAuth2Manager(oauth2Manager *oauth2.OAuth2Server) Option {
	return func(m *Manager) {
		if oauth2Manager != nil {
			m.oauth2Manager = oauth2Manager
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
		nonceManager:   nonce.NewDefaultNonceManager(cfg.AuthType, cfg.KeyPrefix, storage),
		oauth2Manager:  oauth2.NewDefaultOAuth2Server(cfg.AuthType, cfg.KeyPrefix, storage, serializer),
		eventManager:   listener.NewManager(logger),
		accessProvider: accessProvider,
	}

	// Apply optional module overrides after defaults are ready 默认模块初始化完成后应用外部覆盖
	for _, option := range options {
		if option != nil {
			option(mgr)
		}
	}

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
