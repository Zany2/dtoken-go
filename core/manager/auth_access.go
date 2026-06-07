// @Author daixk 2025/12/22 15:56:00
package manager

import "context"

// AccessSubject describes the subject used to resolve permissions and roles. AccessSubject 描述用于解析权限和角色的主体。
type AccessSubject struct {
	AuthType string // AuthType stores auth namespace. AuthType 存储认证命名空间。
	LoginID  string // LoginID stores subject identifier. LoginID 存储主体标识。
	Device   string // Device stores device type. Device 存储设备类型。
	DeviceID string // DeviceID stores concrete device ID. DeviceID 存储具体设备 ID。
	Token    string // Token stores terminal token. Token 存储终端 Token。
}

// AccessProvider resolves permissions and roles for a subject. AccessProvider 为主体解析权限和角色。
type AccessProvider interface {
	// Permissions resolves subject permissions. Permissions 解析主体权限。
	Permissions(ctx context.Context, subject AccessSubject) ([]string, error)
	// Roles resolves subject roles. Roles 解析主体角色。
	Roles(ctx context.Context, subject AccessSubject) ([]string, error)
}

// AccessProviderFunc adapts functions into an AccessProvider. AccessProviderFunc 将函数适配为 AccessProvider。
type AccessProviderFunc struct {
	PermissionFunc func(ctx context.Context, subject AccessSubject) ([]string, error) // PermissionFunc resolves permissions. PermissionFunc 解析权限。
	RoleFunc       func(ctx context.Context, subject AccessSubject) ([]string, error) // RoleFunc resolves roles. RoleFunc 解析角色。
}

// Permissions resolves permissions through PermissionFunc. Permissions 通过 PermissionFunc 解析权限。
func (f AccessProviderFunc) Permissions(ctx context.Context, subject AccessSubject) ([]string, error) {
	// Check permission callback 检查权限回调。
	if f.PermissionFunc == nil {
		return nil, nil
	}
	// Execute permission callback 执行权限回调。
	return f.PermissionFunc(ctx, subject)
}

// Roles resolves roles through RoleFunc. Roles 通过 RoleFunc 解析角色。
func (f AccessProviderFunc) Roles(ctx context.Context, subject AccessSubject) ([]string, error) {
	// Check role callback 检查角色回调。
	if f.RoleFunc == nil {
		return nil, nil
	}
	// Execute role callback 执行角色回调。
	return f.RoleFunc(ctx, subject)
}

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
	// Query access provider 查询访问提供器。
	permissions, err := m.accessProvider.Permissions(ctx, subject)
	if err != nil {
		return nil, err
	}
	// Use fallback when provider returns nil 提供器返回 nil 时使用回退值。
	if permissions == nil {
		return fallback, nil
	}
	// Return provider permissions 返回提供器权限。
	return permissions, nil
}

// resolvePermissions resolves permissions and fails closed on provider errors. resolvePermissions 解析权限并在提供器出错时安全拒绝。
func (m *Manager) resolvePermissions(ctx context.Context, fallback []string, subject AccessSubject) []string {
	// Load permissions safely 安全加载权限。
	permissions, err := m.loadPermissions(ctx, fallback, subject)
	if err != nil {
		m.logger.Errorf("manager.resolvePermissions: failed to resolve permissions, loginID=%s, error=%v", subject.LoginID, err)
		return []string{}
	}
	// Return resolved permissions 返回解析后的权限。
	return permissions
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
	// Query access provider 查询访问提供器。
	roles, err := m.accessProvider.Roles(ctx, subject)
	if err != nil {
		return nil, err
	}
	// Use fallback when provider returns nil 提供器返回 nil 时使用回退值。
	if roles == nil {
		return fallback, nil
	}
	// Return provider roles 返回提供器角色。
	return roles, nil
}

// resolveRoles resolves roles and fails closed on provider errors. resolveRoles 解析角色并在提供器出错时安全拒绝。
func (m *Manager) resolveRoles(ctx context.Context, fallback []string, subject AccessSubject) []string {
	// Load roles safely 安全加载角色。
	roles, err := m.loadRoles(ctx, fallback, subject)
	if err != nil {
		m.logger.Errorf("manager.resolveRoles: failed to resolve roles, loginID=%s, error=%v", subject.LoginID, err)
		return []string{}
	}
	// Return resolved roles 返回解析后的角色。
	return roles
}
