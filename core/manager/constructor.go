package manager

import (
	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/listener"
	"github.com/Zany2/dtoken-go/core/nonce"
	"github.com/Zany2/dtoken-go/core/oauth2"
)

// NewManager creates a manager with the provided core components. NewManager 使用提供的核心组件创建管理器。
func NewManager(
	cfg *config.Config,
	generator adapter.Generator,
	storage adapter.Storage,
	serializer adapter.Codec,
	logger adapter.Log,
	pool adapter.Pool,
	accessProvider AccessProvider,
) *Manager {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	if logger == nil {
		logger = adapter.NewNopLogger()
	}

	return &Manager{
		config:         cfg,
		generator:      generator,
		storage:        storage,
		serializer:     serializer,
		logger:         logger,
		pool:           pool,
		nonceManager:   nonce.NewNonceManager(cfg.AuthType, cfg.KeyPrefix, storage, nonce.DefaultNonceTTL),
		oauth2Manager:  oauth2.NewOAuth2Server(cfg.AuthType, cfg.KeyPrefix, storage, serializer),
		eventManager:   listener.NewManager(logger),
		accessProvider: accessProvider,
	}
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
