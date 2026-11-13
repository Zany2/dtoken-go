// @Author daixk 2025/12/22 15:56:00
package engine

import (
	"github.com/Zany2/dtoken-go/config"
	adapter "github.com/Zany2/dtoken-go/contract"
	listener "github.com/Zany2/dtoken-go/event"
)

// Option configures optional manager modules. Option 配置 Manager 的可选模块。
type Option func(m *Manager)

// ComponentOwnership declares which runtime components are owned and closed by Manager. ComponentOwnership 声明由 Manager 持有并负责关闭的运行时组件。
type ComponentOwnership struct {
	// Storage marks the storage adapter as manager-owned. Storage 标记存储适配器由 Manager 持有。
	Storage bool
	// Logger marks the logger as manager-owned. Logger 标记日志器由 Manager 持有。
	Logger bool
	// Pool marks the task pool as manager-owned. Pool 标记任务池由 Manager 持有。
	Pool bool
}

// WithNonceManager sets the optional nonce manager. WithNonceManager 设置可选 Nonce 管理器。

// WithOAuth2Manager sets the optional OAuth2 server. WithOAuth2Manager 设置可选 OAuth2 服务端。

// WithTicketManager sets the optional ticket manager. WithTicketManager 设置可选 Ticket 管理器。

// WithShortKeyManager sets the optional short key manager. WithShortKeyManager 设置可选短 Key 管理器。

// WithStrategy sets replaceable manager algorithms. WithStrategy 设置可替换的管理器算法。
func WithStrategy(strategy *Strategy) Option {
	return func(m *Manager) {
		if strategy != nil {
			// Snapshot caller-owned hooks before filling defaults. 补齐默认值前快照调用方持有的策略钩子。
			strategySnapshot := *strategy
			m.strategy = strategySnapshot.normalize()
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
		managerRuntime: managerRuntime{
			ownership: ComponentOwnership{Storage: true, Logger: true, Pool: true},
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
