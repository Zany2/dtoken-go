// @Author daixk 2025/12/22 15:56:00
package manager

import (
	"context"
	"errors"

	"github.com/Zany2/dtoken-go/core/derror"
)

// TokenIntrospection describes token introspection result. TokenIntrospection 描述令牌自省结果。
type TokenIntrospection struct {
	Active        bool           `json:"active"`                  // Active reports whether token can access protected resources. Active 表示令牌是否可访问受保护资源。
	AuthType      string         `json:"authType,omitempty"`      // AuthType stores auth namespace. AuthType 存储认证命名空间。
	LoginID       string         `json:"loginId,omitempty"`       // LoginID stores subject identifier. LoginID 存储主体标识。
	Device        string         `json:"device,omitempty"`        // Device stores device type. Device 存储设备类型。
	DeviceID      string         `json:"deviceId,omitempty"`      // DeviceID stores concrete device id. DeviceID 存储具体设备 ID。
	CreateTime    int64          `json:"createTime,omitempty"`    // CreateTime stores token creation time. CreateTime 存储令牌创建时间。
	ExpiresIn     int64          `json:"expiresIn,omitempty"`     // ExpiresIn stores remaining token lifetime seconds. ExpiresIn 存储令牌剩余有效秒数。
	Timeout       int64          `json:"timeout,omitempty"`       // Timeout stores configured token timeout seconds. Timeout 存储令牌配置超时秒数。
	ActiveTimeout int64          `json:"activeTimeout,omitempty"` // ActiveTimeout stores inactive timeout seconds. ActiveTimeout 存储不活跃超时秒数。
	Permissions   []string       `json:"permissions,omitempty"`   // Permissions stores resolved permissions. Permissions 存储解析后的权限列表。
	Roles         []string       `json:"roles,omitempty"`         // Roles stores resolved roles. Roles 存储解析后的角色列表。
	Extra         map[string]any `json:"extra,omitempty"`         // Extra stores token extension data. Extra 存储令牌扩展数据。
	Error         string         `json:"error,omitempty"`         // Error stores inactive reason. Error 存储非活跃原因。
}

// IntrospectToken inspects token validity without renewal side effects. IntrospectToken 无续期副作用地检查令牌状态。
func (m *Manager) IntrospectToken(ctx context.Context, tokenValue string) (*TokenIntrospection, error) {
	result := &TokenIntrospection{Active: false}
	if tokenValue == "" {
		result.Error = "invalid_token"
		return result, nil
	}

	tokenInfo, err := m.getTokenInfo(ctx, tokenValue)
	if err != nil {
		if isTokenInactiveError(err) {
			result.Error = err.Error()
			return result, nil
		}
		return nil, err
	}

	sess, sessErr := m.getSession(ctx, tokenInfo.LoginID)
	if sessErr != nil {
		if errors.Is(sessErr, derror.ErrSessionNotFound) {
			result.Error = "inactive_token"
			return result, nil
		}
		return nil, sessErr
	}

	alive, err := m.checkTerminalTokenAliveWithContext(ctx, tokenValue, tokenInfo, sess)
	if err != nil {
		return nil, err
	}
	if !alive {
		result.Error = "inactive_token"
		return result, nil
	}

	// Check account and device disable status 检查账号及设备封禁状态
	if m.isDisable(ctx, tokenInfo.LoginID) {
		result.Error = derror.ErrAccountDisabled.Error()
		return result, nil
	}
	if m.isDisableDeviceMatch(ctx, tokenInfo.LoginID, tokenInfo.Device, tokenInfo.DeviceId) {
		result.Error = derror.ErrDeviceDisabled.Error()
		return result, nil
	}

	ttl, err := m.GetTokenTTL(ctx, tokenValue)
	if err != nil {
		return nil, err
	}

	subject := AccessSubject{
		AuthType: m.config.AuthType,
		LoginID:  tokenInfo.LoginID,
		Device:   tokenInfo.Device,
		DeviceID: tokenInfo.DeviceId,
		Token:    tokenValue,
	}

	// Reuse session for permission/role fallback 复用会话以获取权限/角色回退值
	var sessionPermissions, sessionRoles []string
	if sess != nil {
		sessionPermissions = sess.Permissions
		sessionRoles = sess.Roles
	}

	permissions, err := m.loadPermissions(ctx, sessionPermissions, subject)
	if err != nil {
		return nil, err
	}
	roles, err := m.loadRoles(ctx, sessionRoles, subject)
	if err != nil {
		return nil, err
	}

	result.Active = true
	result.AuthType = tokenInfo.AuthType
	result.LoginID = tokenInfo.LoginID
	result.Device = tokenInfo.Device
	result.DeviceID = tokenInfo.DeviceId
	result.CreateTime = tokenInfo.CreateTime
	result.ExpiresIn = ttl
	result.Timeout = tokenInfo.Timeout
	result.ActiveTimeout = tokenInfo.ActiveTimeout
	result.Permissions = permissions
	result.Roles = roles
	result.Extra = tokenInfo.Extra
	return result, nil
}

// isTokenInactiveError reports whether an error should become inactive introspection. isTokenInactiveError 判断错误是否应映射为非活跃自省结果。
func isTokenInactiveError(err error) bool {
	return errors.Is(err, derror.ErrInvalidToken) ||
		errors.Is(err, derror.ErrTokenExpired) ||
		errors.Is(err, derror.ErrActiveTimeout) ||
		errors.Is(err, derror.ErrTokenKickout) ||
		errors.Is(err, derror.ErrTokenReplaced)
}
