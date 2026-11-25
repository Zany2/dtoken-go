// @Author daixk 2025/12/22 15:56:00
package engine

import (
	"github.com/Zany2/dtoken-go/config"
	adapter "github.com/Zany2/dtoken-go/contract"
	listener "github.com/Zany2/dtoken-go/event"
)

// Manager is the core auth facade; business methods return ErrManagerNotFound or false once shutdown starts. Manager 是鉴权核心门面；开始关闭后业务方法返回 ErrManagerNotFound 或 false。
type Manager struct {
	config             *config.Config    // config stores runtime configuration. config 存储运行时配置。
	generator          adapter.Generator // generator creates token values. generator 负责生成 Token 值。
	storage            adapter.Storage   // storage persists auth data. storage 持久化鉴权数据。
	serializer         adapter.Codec     // serializer encodes and decodes storage payloads. serializer 编解码存储数据。
	logger             adapter.Log       // logger writes framework logs. logger 写入框架日志。
	pool               adapter.Pool      // pool runs asynchronous tasks. pool 执行异步任务。
	eventManager       *listener.Manager // eventManager dispatches auth events. eventManager 分发鉴权事件。
	managerAccounts                      // managerAccounts owns account write coordination. managerAccounts 持有账号写入协调状态。
	managerMaintenance                   // managerMaintenance owns token maintenance generations. managerMaintenance 持有 Token 维护代次。
	managerRuntime                       // managerRuntime owns task admission and shutdown. managerRuntime 持有任务接收与关闭状态。
	accessProvider     AccessProvider    // accessProvider resolves roles and permissions. accessProvider 解析角色和权限。
	strategy           *Strategy         // strategy stores replaceable algorithms. strategy 存储可替换算法。
}

// GetConfig returns an isolated configuration snapshot; changes never affect the running engine. GetConfig 返回独立配置快照，修改不会影响运行中的引擎。
func (m *Manager) GetConfig() *config.Config {
	if m == nil || m.config == nil {
		return nil
	}
	return m.config.Clone()
}

// RuntimeConfig borrows immutable configuration for internal request paths; callers must never modify it. RuntimeConfig 供内部请求路径借用不可变配置，调用方不得修改。
func RuntimeConfig(m *Manager) *config.Config {
	if m == nil {
		return nil
	}
	return m.config
}

// IsClosed reports whether manager shutdown has started. IsClosed 返回 Manager 是否已开始关闭。
func (m *Manager) IsClosed() bool {
	return m == nil || m.closed.Load()
}

// GetGenerator retrieves the token generator. GetGenerator 获取 Token 生成器。
func (m *Manager) GetGenerator() adapter.Generator {
	return m.generator
}

// GetStorage retrieves the storage adapter. GetStorage 获取存储适配器。
func (m *Manager) GetStorage() adapter.Storage {
	return m.storage
}

// GetSerializer retrieves the serializer adapter. GetSerializer 获取序列化适配器。
func (m *Manager) GetSerializer() adapter.Codec {
	return m.serializer
}

// GetLogger retrieves the logger adapter. GetLogger 获取日志适配器。
func (m *Manager) GetLogger() adapter.Log {
	return m.logger
}

// GetPool retrieves the goroutine pool. GetPool 获取协程池。
func (m *Manager) GetPool() adapter.Pool {
	return m.pool
}

// GetAccessProvider retrieves the access provider. GetAccessProvider 获取访问权限提供器。
func (m *Manager) GetAccessProvider() AccessProvider {
	return m.accessProvider
}

// GetStrategy returns a strategy snapshot; callback closures remain caller-managed. GetStrategy 返回策略快照，回调闭包内部状态仍由调用方管理。
func (m *Manager) GetStrategy() *Strategy {
	if m == nil || m.strategy == nil {
		return nil
	}
	snapshot := *m.strategy
	return &snapshot
}

// GetEventManager retrieves the event manager. GetEventManager 获取事件监听管理器。
func (m *Manager) GetEventManager() *listener.Manager {
	return m.eventManager
}
