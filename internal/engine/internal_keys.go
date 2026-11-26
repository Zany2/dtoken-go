// @Author daixk 2025/12/22 15:56:00
package engine

import (
	"strings"

	"github.com/Zany2/dtoken-go/config"
)

// storageNamespace keeps namespace boundaries independent of business key suffixes. storageNamespace 保持命名空间边界独立于业务键后缀。
func (m *Manager) storageNamespace() string {
	return m.namespace
}

// getTokenKey generates the storage key for a token. getTokenKey 获取 Token 存储键。
func (m *Manager) getTokenKey(tokenValue string) string {
	return m.storageNamespace() + config.TokenKeyPrefix + tokenValue
}

// getTokenExtraKey binds extension data to an individual credential lifecycle. getTokenExtraKey 将扩展数据绑定到独立的凭据生命周期。
func (m *Manager) getTokenExtraKey(accessID string) string {
	return m.storageNamespace() + "token-extra:" + accessID
}

// getSessionKey generates the storage key for a session. getSessionKey 获取会话存储键。
func (m *Manager) getSessionKey(loginID string) string {
	return m.storageNamespace() + SessionKeyPrefix + loginID
}

// getRenewKey generates the storage key for token renewal tracking. getRenewKey 获取 Token 续期追踪键。
func (m *Manager) getRenewKey(tokenValue string) string {
	return m.storageNamespace() + RenewKeyPrefix + tokenValue
}

// getActiveKey generates the storage key for token activity tracking. getActiveKey 获取 Token 活跃时间追踪键。
func (m *Manager) getActiveKey(tokenValue string) string {
	return m.storageNamespace() + ActivePrefix + tokenValue
}

// getRefreshTokenKey generates the storage key for a refresh token. getRefreshTokenKey 获取刷新令牌存储键。
func (m *Manager) getRefreshTokenKey(refreshToken string) string {
	return m.storageNamespace() + RefreshTokenKeyPrefix + refreshToken
}

// getTokenRefreshKey generates the storage key for token refresh mapping. getTokenRefreshKey 获取访问令牌刷新映射键。
func (m *Manager) getTokenRefreshKey(tokenValue string) string {
	return m.storageNamespace() + TokenRefreshKeyPrefix + tokenValue
}

// getDisableKey generates the storage key for account disable status. getDisableKey 获取账号禁用状态存储键。
func (m *Manager) getDisableKey(loginID string) string {
	return m.storageNamespace() + DisableKeyPrefix + escapeStorageKeyComponent(loginID)
}

// getDisableServiceKey generates the storage key for service disable status. getDisableServiceKey 获取账号分类禁用状态存储键。
func (m *Manager) getDisableServiceKey(loginID, service string) string {
	return m.storageNamespace() + DisableServiceKeyPrefix + escapeStorageKeyComponent(loginID) + ":" + escapeStorageKeyComponent(service)
}

// getDisableDeviceKey generates the storage key for device disable status. getDisableDeviceKey 获取设备封禁状态存储键。
func (m *Manager) getDisableDeviceKey(loginID, device string) string {
	return m.storageNamespace() + DisableDeviceKeyPrefix + escapeStorageKeyComponent(loginID) + ":" + escapeStorageKeyComponent(device)
}

// getDisableDeviceAndDeviceIDKey generates the storage key for concrete device disable status. getDisableDeviceAndDeviceIDKey 获取具体设备封禁状态存储键。
func (m *Manager) getDisableDeviceAndDeviceIDKey(loginID, device, deviceID string) string {
	return m.storageNamespace() + DisableDeviceIDKeyPrefix + escapeStorageKeyComponent(loginID) + ":" + escapeStorageKeyComponent(device) + ":" + escapeStorageKeyComponent(deviceID)
}

// escapeStorageKeyComponent escapes composite-key separators while preserving ordinary keys. escapeStorageKeyComponent 转义复合键分隔符，同时保持普通键格式不变。
func escapeStorageKeyComponent(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	return strings.ReplaceAll(value, ":", "\\:")
}
