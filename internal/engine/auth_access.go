package engine

import (
	"context"
	"github.com/Zany2/dtoken-go/contract"
)

type (
	AccessSubject      = contract.AccessSubject
	AccessProvider     = contract.AccessProvider
	AccessProviderFunc = contract.AccessProviderFunc
)

// loadPermissionsByLoginID loads login ID permissions and prefers provider data. loadPermissionsByLoginID 按登录 ID 加载权限并优先使用提供器数据。
func (m *Manager) loadPermissionsByLoginID(ctx context.Context, loginID string) ([]string, error) {
	// Build account subject 构建账户主体。
	subject := AccessSubject{AuthType: m.config.AuthType, LoginID: loginID}

	// Prefer access provider 优先使用访问提供器。
	if m.accessProvider != nil {
		permissions, err := m.providerPermissions(ctx, nil, subject)
		if err != nil {
			return nil, err
		}

		// Return provider permissions 返回提供器权限。
		if permissions != nil {
			return permissions, nil
		}
	}

	// Load session fallback 加载会话回退值。
	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		return nil, err
	}

	// Return session permissions 返回会话权限。
	return sess.Permissions, nil
}

// loadRolesByLoginID loads login ID roles and prefers provider data. loadRolesByLoginID 按登录 ID 加载角色并优先使用提供器数据。
func (m *Manager) loadRolesByLoginID(ctx context.Context, loginID string) ([]string, error) {
	// Build account subject 构建账户主体。
	subject := AccessSubject{AuthType: m.config.AuthType, LoginID: loginID}

	// Prefer access provider 优先使用访问提供器。
	if m.accessProvider != nil {
		roles, err := m.providerRoles(ctx, nil, subject)
		if err != nil {
			return nil, err
		}

		// Return provider roles 返回提供器角色。
		if roles != nil {
			return roles, nil
		}
	}

	// Load session fallback 加载会话回退值。
	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		return nil, err
	}

	// Return session roles 返回会话角色。
	return sess.Roles, nil
}

// loadPermissions loads permissions from provider with fallback. loadPermissions 从提供器加载权限并支持回退值。
func (m *Manager) loadPermissions(ctx context.Context, fallback []string, subject AccessSubject) ([]string, error) {
	// Fill default auth type 填充默认认证类型
	if subject.AuthType == "" {
		subject.AuthType = m.config.AuthType
	}

	// Return fallback when provider is absent 提供器不存在时返回回退值
	if m.accessProvider == nil {
		return fallback, nil
	}

	// Resolve provider permissions 解析提供器权限。
	return m.providerPermissions(ctx, fallback, subject)
}

// providerPermissions resolves permissions from provider. providerPermissions 从提供器解析权限。
func (m *Manager) providerPermissions(ctx context.Context, fallback []string, subject AccessSubject) ([]string, error) {
	// Use fallback when provider is absent 提供器不存在时使用回退值。
	if m.accessProvider == nil {
		return fallback, nil
	}

	// Query access provider 查询访问提供器。
	permissions, err := m.accessProvider.Permissions(ctx, subject)
	if err != nil {
		return nil, err
	}

	// Use fallback when provider returns nil 提供器返回 nil 时使用回退值。
	if permissions == nil {
		return fallback, nil
	}

	// Return normalized provider permissions 返回规范化后的提供器权限。
	return normalizeProviderAccessValues(permissions), nil
}

// loadRoles loads roles from provider with fallback. loadRoles 从提供器加载角色并支持回退值。
func (m *Manager) loadRoles(ctx context.Context, fallback []string, subject AccessSubject) ([]string, error) {
	// Fill default auth type 填充默认认证类型
	if subject.AuthType == "" {
		subject.AuthType = m.config.AuthType
	}

	// Return fallback when provider is absent 提供器不存在时返回回退值
	if m.accessProvider == nil {
		return fallback, nil
	}

	// Resolve provider roles 解析提供器角色。
	return m.providerRoles(ctx, fallback, subject)
}

// providerRoles resolves roles from provider. providerRoles 从提供器解析角色。
func (m *Manager) providerRoles(ctx context.Context, fallback []string, subject AccessSubject) ([]string, error) {
	// Use fallback when provider is absent 提供器不存在时使用回退值。
	if m.accessProvider == nil {
		return fallback, nil
	}

	// Query access provider 查询访问提供器。
	roles, err := m.accessProvider.Roles(ctx, subject)
	if err != nil {
		return nil, err
	}

	// Use fallback when provider returns nil 提供器返回 nil 时使用回退值。
	if roles == nil {
		return fallback, nil
	}

	// Return normalized provider roles 返回规范化后的提供器角色。
	return normalizeProviderAccessValues(roles), nil
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
