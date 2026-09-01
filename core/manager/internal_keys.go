// @Author daixk 2025/12/22 15:56:00
package manager

import (
	"strings"

	"github.com/Zany2/dtoken-go/core/config"
)

// getTokenKey generates the storage key for a token. getTokenKey 获取 Token 存储键。
func (m *Manager) getTokenKey(tokenValue string) string {
	return m.config.KeyPrefix + m.config.AuthType + config.TokenKeyPrefix + tokenValue
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

// getRefreshTokenKey generates the storage key for a refresh token. getRefreshTokenKey 获取刷新令牌存储键。
func (m *Manager) getRefreshTokenKey(refreshToken string) string {
	return m.config.KeyPrefix + m.config.AuthType + RefreshTokenKeyPrefix + refreshToken
}

// getTokenRefreshKey generates the storage key for token refresh mapping. getTokenRefreshKey 获取访问令牌刷新映射键。
func (m *Manager) getTokenRefreshKey(tokenValue string) string {
	return m.config.KeyPrefix + m.config.AuthType + TokenRefreshKeyPrefix + tokenValue
}

// getDisableKey generates the storage key for account disable status. getDisableKey 获取账号禁用状态存储键。
func (m *Manager) getDisableKey(loginID string) string {
	return m.config.KeyPrefix + m.config.AuthType + DisableKeyPrefix + loginID
}

// getDisableServiceKey generates the storage key for service disable status. getDisableServiceKey 获取账号分类禁用状态存储键。
func (m *Manager) getDisableServiceKey(loginID, service string) string {
	return m.config.KeyPrefix + m.config.AuthType + DisableServiceKeyPrefix + escapeStorageKeyComponent(loginID) + ":" + escapeStorageKeyComponent(service)
}

// getLegacyDisableServiceKey returns the pre-escaping service key for migration reads. getLegacyDisableServiceKey 返回转义改造前的服务封禁键，用于兼容读取。
func (m *Manager) getLegacyDisableServiceKey(loginID, service string) string {
	return m.config.KeyPrefix + m.config.AuthType + DisableServiceKeyPrefix + loginID + ":" + service
}

// getDisableDeviceKey generates the storage key for device disable status. getDisableDeviceKey 获取设备封禁状态存储键。
func (m *Manager) getDisableDeviceKey(loginID, device string) string {
	return m.config.KeyPrefix + m.config.AuthType + DisableDeviceKeyPrefix + escapeStorageKeyComponent(loginID) + ":" + escapeStorageKeyComponent(device)
}

// getLegacyDisableDeviceKey returns the pre-escaping device key for migration reads. getLegacyDisableDeviceKey 返回转义改造前的设备封禁键，用于兼容读取。
func (m *Manager) getLegacyDisableDeviceKey(loginID, device string) string {
	return m.config.KeyPrefix + m.config.AuthType + DisableDeviceKeyPrefix + loginID + ":" + device
}

// getDisableDeviceAndDeviceIDKey generates the storage key for concrete device disable status. getDisableDeviceAndDeviceIDKey 获取具体设备封禁状态存储键。
func (m *Manager) getDisableDeviceAndDeviceIDKey(loginID, device, deviceID string) string {
	return m.config.KeyPrefix + m.config.AuthType + DisableDeviceIDKeyPrefix + escapeStorageKeyComponent(loginID) + ":" + escapeStorageKeyComponent(device) + ":" + escapeStorageKeyComponent(deviceID)
}

// getLegacyDisableDeviceAndDeviceIDKey returns the pre-escaping concrete device key for migration reads. getLegacyDisableDeviceAndDeviceIDKey 返回转义改造前的具体设备封禁键，用于兼容读取。
func (m *Manager) getLegacyDisableDeviceAndDeviceIDKey(loginID, device, deviceID string) string {
	return m.config.KeyPrefix + m.config.AuthType + DisableDeviceIDKeyPrefix + loginID + ":" + device + ":" + deviceID
}

// escapeStorageKeyComponent escapes composite-key separators while preserving ordinary keys. escapeStorageKeyComponent 转义复合键分隔符，同时保持普通键格式不变。
func escapeStorageKeyComponent(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	return strings.ReplaceAll(value, ":", "\\:")
}
