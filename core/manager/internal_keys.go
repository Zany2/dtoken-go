// @Author daixk 2025/12/22 15:56:00
package manager

import (
	"github.com/Zany2/dtoken-go/core/config"
)

// getTokenKey generates the storage key for a token. getTokenKey 获取 Token 存储键。
func (m *Manager) getTokenKey(tokenValue string) string {
	return m.config.KeyPrefix + m.config.AuthType + config.TokenKeyPrefix + tokenValue
}

// getLegacyTokenKey generates legacy token key before token namespace was added. getLegacyTokenKey 获取历史版本 Token 存储键。
func (m *Manager) getLegacyTokenKey(tokenValue string) string {
	return m.config.KeyPrefix + m.config.AuthType + tokenValue
}

// getTokenStorageKeys returns all token storage keys for cleanup. getTokenStorageKeys 返回 Token 清理需要覆盖的全部存储键。
func (m *Manager) getTokenStorageKeys(tokenValue string) []string {
	return []string{m.getTokenKey(tokenValue), m.getLegacyTokenKey(tokenValue)}
}

// getSessionKey generates the storage key for a session. getSessionKey 获取会话存储键。
func (m *Manager) getSessionKey(loginID string) string {
	return m.config.KeyPrefix + m.config.AuthType + SessionKeyPrefix + loginID
}

// getRenewKey generates the storage key for token renewal tracking. getRenewKey 获取 Token 续期追踪键。
func (m *Manager) getRenewKey(tokenValue string) string {
	return m.config.KeyPrefix + m.config.AuthType + RenewKeyPrefix + tokenValue
}

// getActiveKey generates the storage key for token activity tracking. getActiveKey 获取 Token 活跃时间追踪键。
func (m *Manager) getActiveKey(tokenValue string) string {
	return m.config.KeyPrefix + m.config.AuthType + ActivePrefix + tokenValue
}

// getDisableKey generates the storage key for account disable status. getDisableKey 获取账号禁用状态存储键。
func (m *Manager) getDisableKey(loginID string) string {
	return m.config.KeyPrefix + m.config.AuthType + DisableKeyPrefix + loginID
}

// getDisableServiceKey generates the storage key for service disable status. getDisableServiceKey 获取账号分类禁用状态存储键。
func (m *Manager) getDisableServiceKey(loginID, service string) string {
	return m.config.KeyPrefix + m.config.AuthType + DisableServiceKeyPrefix + loginID + ":" + service
}

// getDisableDeviceKey generates the storage key for device disable status. getDisableDeviceKey 获取设备封禁状态存储键。
func (m *Manager) getDisableDeviceKey(loginID, device string) string {
	return m.config.KeyPrefix + m.config.AuthType + DisableDeviceKeyPrefix + loginID + ":" + device
}

// getDisableDeviceAndDeviceIdKey generates the storage key for concrete device disable status. getDisableDeviceAndDeviceIdKey 获取具体设备封禁状态存储键。
func (m *Manager) getDisableDeviceAndDeviceIdKey(loginID, device, deviceId string) string {
	return m.config.KeyPrefix + m.config.AuthType + DisableDeviceIDKeyPrefix + loginID + ":" + device + ":" + deviceId
}
