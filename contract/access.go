// @Author daixk 2025/12/22 15:56:00
package contract

import "context"

// AccessSubject describes the subject used to resolve permissions and roles. AccessSubject 描述用于解析权限和角色的主体。
type AccessSubject struct {
	// AuthType stores auth namespace. AuthType 存储认证命名空间。
	AuthType string
	// LoginID stores subject identifier. LoginID 存储主体标识。
	LoginID string
	// Device stores device type. Device 存储设备类型。
	Device string
	// DeviceID stores concrete device ID. DeviceID 存储具体设备 ID。
	DeviceID string
	// Token stores terminal token. Token 存储终端 Token。
	Token string
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
	// PermissionFunc resolves permissions. PermissionFunc 解析权限。
	PermissionFunc func(ctx context.Context, subject AccessSubject) ([]string, error)
	// RoleFunc resolves roles. RoleFunc 解析角色。
	RoleFunc func(ctx context.Context, subject AccessSubject) ([]string, error)
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
