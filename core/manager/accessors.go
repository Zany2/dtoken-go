// @Author daixk 2025/12/22 15:56:00
package manager

import (
	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/listener"
	"github.com/Zany2/dtoken-go/core/nonce"
	"github.com/Zany2/dtoken-go/core/oauth2"
)

// GetConfig retrieves the manager configuration. GetConfig 获取管理器配置。
func (m *Manager) GetConfig() *config.Config {
	return m.config
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

// GetNonceManager retrieves the nonce manager. GetNonceManager 获取 nonce 管理器。
func (m *Manager) GetNonceManager() *nonce.NonceManager {
	return m.nonceManager
}

// GetOAuth2Manager retrieves the OAuth2 manager. GetOAuth2Manager 获取 OAuth2 管理器。
func (m *Manager) GetOAuth2Manager() *oauth2.OAuth2Server {
	return m.oauth2Manager
}

// GetEventManager retrieves the event manager. GetEventManager 获取事件监听管理器。
func (m *Manager) GetEventManager() *listener.Manager {
	return m.eventManager
}
