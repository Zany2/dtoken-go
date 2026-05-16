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
	if f.PermissionFunc == nil {
		return nil, nil
	}
	return f.PermissionFunc(ctx, subject)
}

// Roles resolves roles through RoleFunc. Roles 通过 RoleFunc 解析角色。
func (f AccessProviderFunc) Roles(ctx context.Context, subject AccessSubject) ([]string, error) {
	if f.RoleFunc == nil {
		return nil, nil
	}
	return f.RoleFunc(ctx, subject)
}

// legacyAccessProvider adapts the previous callback-style API. legacyAccessProvider 适配旧版回调式 API。
type legacyAccessProvider struct {
	permissionFunc    func(loginID, authType string) ([]string, error)                   // permissionFunc resolves legacy permissions. permissionFunc 解析旧版权限。
	roleFunc          func(loginID, authType string) ([]string, error)                   // roleFunc resolves legacy roles. roleFunc 解析旧版角色。
	permissionExtFunc func(loginID, device, deviceId, authType string) ([]string, error) // permissionExtFunc resolves terminal permissions. permissionExtFunc 解析终端维度权限。
	roleExtFunc       func(loginID, device, deviceId, authType string) ([]string, error) // roleExtFunc resolves terminal roles. roleExtFunc 解析终端维度角色。
}

// empty reports whether no legacy callback exists. empty 判断是否没有任何旧版回调。
func (p *legacyAccessProvider) empty() bool {
	return p == nil || (p.permissionFunc == nil && p.roleFunc == nil && p.permissionExtFunc == nil && p.roleExtFunc == nil)
}

// Permissions resolves permissions through legacy callbacks. Permissions 通过旧版回调解析权限。
func (p *legacyAccessProvider) Permissions(_ context.Context, subject AccessSubject) ([]string, error) {
	if p == nil || subject.LoginID == "" {
		return nil, nil
	}
	if p.permissionExtFunc != nil && (subject.Device != "" || subject.DeviceID != "" || subject.Token != "") {
		return p.permissionExtFunc(subject.LoginID, subject.Device, subject.DeviceID, subject.AuthType)
	}
	if p.permissionFunc != nil {
		return p.permissionFunc(subject.LoginID, subject.AuthType)
	}
	return nil, nil
}

// Roles resolves roles through legacy callbacks. Roles 通过旧版回调解析角色。
func (p *legacyAccessProvider) Roles(_ context.Context, subject AccessSubject) ([]string, error) {
	if p == nil || subject.LoginID == "" {
		return nil, nil
	}
	if p.roleExtFunc != nil && (subject.Device != "" || subject.DeviceID != "" || subject.Token != "") {
		return p.roleExtFunc(subject.LoginID, subject.Device, subject.DeviceID, subject.AuthType)
	}
	if p.roleFunc != nil {
		return p.roleFunc(subject.LoginID, subject.AuthType)
	}
	return nil, nil
}

// NewLegacyAccessProvider adapts the previous callback-style API into AccessProvider. NewLegacyAccessProvider 将旧版回调式 API 适配为 AccessProvider。
func NewLegacyAccessProvider(
	permissionFunc, roleFunc func(loginID, authType string) ([]string, error),
	permissionExtFunc, roleExtFunc func(loginID, device, deviceId, authType string) ([]string, error),
) AccessProvider {
	provider := &legacyAccessProvider{
		permissionFunc:    permissionFunc,
		roleFunc:          roleFunc,
		permissionExtFunc: permissionExtFunc,
		roleExtFunc:       roleExtFunc,
	}
	if provider.empty() {
		return nil
	}
	return provider
}

// loadPermissionsByLoginID loads login ID permissions and prefers provider data. loadPermissionsByLoginID 按登录 ID 加载权限并优先使用提供器数据。
func (m *Manager) loadPermissionsByLoginID(ctx context.Context, loginID string) ([]string, error) {
	subject := AccessSubject{AuthType: m.config.AuthType, LoginID: loginID}
	if m.accessProvider != nil {
		permissions, err := m.providerPermissions(ctx, nil, subject)
		if err != nil {
			return nil, err
		}
		if permissions != nil {
			return permissions, nil
		}
	}

	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		return nil, err
	}
	return sess.Permissions, nil
}

// loadRolesByLoginID loads login ID roles and prefers provider data. loadRolesByLoginID 按登录 ID 加载角色并优先使用提供器数据。
func (m *Manager) loadRolesByLoginID(ctx context.Context, loginID string) ([]string, error) {
	subject := AccessSubject{AuthType: m.config.AuthType, LoginID: loginID}
	if m.accessProvider != nil {
		roles, err := m.providerRoles(ctx, nil, subject)
		if err != nil {
			return nil, err
		}
		if roles != nil {
			return roles, nil
		}
	}

	sess, err := m.getSession(ctx, loginID)
	if err != nil {
		return nil, err
	}
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
	return m.providerPermissions(ctx, fallback, subject)
}

// providerPermissions resolves permissions from provider. providerPermissions 从提供器解析权限。
func (m *Manager) providerPermissions(ctx context.Context, fallback []string, subject AccessSubject) ([]string, error) {
	permissions, err := m.accessProvider.Permissions(ctx, subject)
	if err != nil {
		return nil, err
	}
	if permissions == nil {
		return fallback, nil
	}
	return permissions, nil
}

// resolvePermissions resolves permissions and fails closed on provider errors. resolvePermissions 解析权限并在提供器出错时安全拒绝。
func (m *Manager) resolvePermissions(ctx context.Context, fallback []string, subject AccessSubject) []string {
	permissions, err := m.loadPermissions(ctx, fallback, subject)
	if err != nil {
		m.logger.Errorf("manager.resolvePermissions: failed to resolve permissions, loginID=%s, error=%v", subject.LoginID, err)
		return []string{}
	}
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
	return m.providerRoles(ctx, fallback, subject)
}

// providerRoles resolves roles from provider. providerRoles 从提供器解析角色。
func (m *Manager) providerRoles(ctx context.Context, fallback []string, subject AccessSubject) ([]string, error) {
	roles, err := m.accessProvider.Roles(ctx, subject)
	if err != nil {
		return nil, err
	}
	if roles == nil {
		return fallback, nil
	}
	return roles, nil
}

// resolveRoles resolves roles and fails closed on provider errors. resolveRoles 解析角色并在提供器出错时安全拒绝。
func (m *Manager) resolveRoles(ctx context.Context, fallback []string, subject AccessSubject) []string {
	roles, err := m.loadRoles(ctx, fallback, subject)
	if err != nil {
		m.logger.Errorf("manager.resolveRoles: failed to resolve roles, loginID=%s, error=%v", subject.LoginID, err)
		return []string{}
	}
	return roles
}
