package engine

import (
	"context"
	"errors"

	"github.com/Zany2/dtoken-go/contract"
	derror "github.com/Zany2/dtoken-go/errors"
)

type (
	AccessSubject      = contract.AccessSubject
	AccessProvider     = contract.AccessProvider
	AccessProviderFunc = contract.AccessProviderFunc
)

// loadPermissionsByLoginID loads login ID permissions and prefers provider data. loadPermissionsByLoginID 按登录 ID 加载权限并优先使用提供器数据。
func (m *Manager) loadPermissionsByLoginID(ctx context.Context, loginID string) ([]string, error) {
	return m.loadPermissions(ctx, nil, AccessSubject{AuthType: m.config.AuthType, LoginID: loginID})
}

// loadRolesByLoginID loads login ID roles and prefers provider data. loadRolesByLoginID 按登录 ID 加载角色并优先使用提供器数据。
func (m *Manager) loadRolesByLoginID(ctx context.Context, loginID string) ([]string, error) {
	return m.loadRoles(ctx, nil, AccessSubject{AuthType: m.config.AuthType, LoginID: loginID})
}

// loadPermissions prefers provider values and loads Session only for nil fallback. loadPermissions 优先使用提供器值，仅在 nil 回退时加载 Session。
func (m *Manager) loadPermissions(ctx context.Context, cached **Session, subject AccessSubject) ([]string, error) {
	// Fill default auth type 填充默认认证类型
	if subject.AuthType == "" {
		subject.AuthType = m.config.AuthType
	}

	// Only nil provider data requests Session fallback; empty data and failures are final. 仅提供器返回 nil 时回退 Session，空列表和错误均为最终结果。
	if m.accessProvider != nil {
		values, err := m.accessProvider.Permissions(ctx, subject)
		if err != nil {
			return nil, err
		}
		if values != nil {
			return normalizeProviderAccessValues(values), nil
		}
	}

	sess, err := m.loadAccessSession(ctx, cached, subject)
	if err != nil {
		return nil, err
	}
	return sess.Permissions, nil
}

// loadRoles prefers provider values and loads Session only for nil fallback. loadRoles 优先使用提供器值，仅在 nil 回退时加载 Session。
func (m *Manager) loadRoles(ctx context.Context, cached **Session, subject AccessSubject) ([]string, error) {
	// Fill default auth type 填充默认认证类型
	if subject.AuthType == "" {
		subject.AuthType = m.config.AuthType
	}

	// Only nil provider data requests Session fallback; empty data and failures are final. 仅提供器返回 nil 时回退 Session，空列表和错误均为最终结果。
	if m.accessProvider != nil {
		values, err := m.accessProvider.Roles(ctx, subject)
		if err != nil {
			return nil, err
		}
		if values != nil {
			return normalizeProviderAccessValues(values), nil
		}
	}

	sess, err := m.loadAccessSession(ctx, cached, subject)
	if err != nil {
		return nil, err
	}
	return sess.Roles, nil
}

// loadAccessSession loads fallback data once per operation; nil cached disables reuse. loadAccessSession 在单次操作内按需加载一次回退数据，cached 为 nil 时不缓存。
func (m *Manager) loadAccessSession(ctx context.Context, cached **Session, subject AccessSubject) (*Session, error) {
	if cached != nil && *cached != nil {
		return *cached, nil
	}
	sess, err := m.getSession(ctx, subject.LoginID)
	if err != nil {
		if subject.Token != "" && errors.Is(err, derror.ErrSessionNotFound) {
			return nil, derror.ErrInvalidToken
		}
		return nil, err
	}
	if cached != nil {
		*cached = sess
	}
	return sess, nil
}

// normalizeProviderAccessValues normalizes provider values while preserving non-nil empty meaning. normalizeProviderAccessValues 规范化提供器返回值，并保留非 nil 空列表语义。
func normalizeProviderAccessValues(values []string) []string {
	// Preserve nil as fallback signal 保留 nil 作为回退信号。
	if values == nil {
		return nil
	}

	// Normalize non-nil provider values 规范化非 nil 提供器返回值。
	normalized := normalizeAccessValues(values)
	if normalized == nil {
		return []string{}
	}

	return normalized
}
