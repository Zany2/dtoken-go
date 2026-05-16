// @Author daixk 2025/12/22 15:56:00
package dtoken

import (
	"context"

	"github.com/Zany2/dtoken-go/core/derror"
)

// Default returns the default auth instance. Default 返回默认鉴权实例。
func Default() (*Auth, error) {
	mgr, err := GetManager()
	if err != nil {
		return nil, err
	}
	return New(mgr), nil
}

// NewByAuthType returns an auth instance by auth type. NewByAuthType 根据认证类型返回鉴权实例。
func NewByAuthType(authType string) (*Auth, error) {
	mgr, err := GetManager(authType)
	if err != nil {
		return nil, err
	}
	return New(mgr), nil
}

// LoginWithOptions performs login with typed options. LoginWithOptions 使用类型化选项执行登录。
func LoginWithOptions(ctx context.Context, opts LoginOptions, authType ...string) (string, error) {
	mgr, err := GetManager(resolveAuthType(opts.AuthType, authType...))
	if err != nil {
		return "", err
	}
	return New(mgr).Login(ctx, opts)
}

// LogoutWithOptions logs out by typed terminal options. LogoutWithOptions 根据类型化终端选项登出。
func LogoutWithOptions(ctx context.Context, opts LogoutOptions, authType ...string) error {
	mgr, err := GetManager(resolveAuthType(opts.AuthType, authType...))
	if err != nil {
		return err
	}
	return New(mgr).Logout(ctx, opts)
}

// KickoutWithOptions kicks out by typed terminal options. KickoutWithOptions 根据类型化终端选项踢人下线。
func KickoutWithOptions(ctx context.Context, opts LogoutOptions, authType ...string) error {
	mgr, err := GetManager(resolveAuthType(opts.AuthType, authType...))
	if err != nil {
		return err
	}
	return New(mgr).Kickout(ctx, opts)
}

// ReplaceWithOptions replaces by typed terminal options. ReplaceWithOptions 根据类型化终端选项顶人下线。
func ReplaceWithOptions(ctx context.Context, opts LogoutOptions, authType ...string) error {
	mgr, err := GetManager(resolveAuthType(opts.AuthType, authType...))
	if err != nil {
		return err
	}
	return New(mgr).Replace(ctx, opts)
}

// DisableWithOptions disables an account with typed options. DisableWithOptions 使用类型化选项封禁账号。
func DisableWithOptions(ctx context.Context, opts DisableOptions, authType ...string) error {
	mgr, err := GetManager(resolveAuthType(opts.AuthType, authType...))
	if err != nil {
		return err
	}
	return New(mgr).Disable(ctx, opts)
}

// DisableServiceWithOptions disables a service with typed options. DisableServiceWithOptions 使用类型化选项封禁服务。
func DisableServiceWithOptions(ctx context.Context, opts ServiceDisableOptions, authType ...string) error {
	mgr, err := GetManager(resolveAuthType(opts.AuthType, authType...))
	if err != nil {
		return err
	}
	return New(mgr).DisableService(ctx, opts)
}

// DisableDeviceWithOptions disables a device with typed options. DisableDeviceWithOptions 使用类型化选项封禁设备。
func DisableDeviceWithOptions(ctx context.Context, opts DeviceDisableOptions, authType ...string) error {
	mgr, err := GetManager(resolveAuthType(opts.AuthType, authType...))
	if err != nil {
		return err
	}
	return New(mgr).DisableDevice(ctx, opts)
}

// AddPermissionsWithOptions adds permissions with typed options. AddPermissionsWithOptions 使用类型化选项添加权限。
func AddPermissionsWithOptions(ctx context.Context, opts PermissionOptions, authType ...string) error {
	mgr, err := GetManager(resolveAuthType(opts.AuthType, authType...))
	if err != nil {
		return err
	}
	return New(mgr).AddPermissions(ctx, opts)
}

// RemovePermissionsWithOptions removes permissions with typed options. RemovePermissionsWithOptions 使用类型化选项移除权限。
func RemovePermissionsWithOptions(ctx context.Context, opts PermissionOptions, authType ...string) error {
	mgr, err := GetManager(resolveAuthType(opts.AuthType, authType...))
	if err != nil {
		return err
	}
	return New(mgr).RemovePermissions(ctx, opts)
}

// CheckPermissionWithOptions checks one permission with typed options. CheckPermissionWithOptions 使用类型化选项校验单个权限。
func CheckPermissionWithOptions(ctx context.Context, opts PermissionOptions, authType ...string) error {
	mgr, err := GetManager(resolveAuthType(opts.AuthType, authType...))
	if err != nil {
		return err
	}
	return New(mgr).CheckPermission(ctx, opts)
}

// CheckPermissionsAndWithOptions checks all permissions with typed options. CheckPermissionsAndWithOptions 使用类型化选项校验全部权限。
func CheckPermissionsAndWithOptions(ctx context.Context, opts PermissionOptions, authType ...string) error {
	mgr, err := GetManager(resolveAuthType(opts.AuthType, authType...))
	if err != nil {
		return err
	}
	return New(mgr).CheckPermissionsAnd(ctx, opts)
}

// CheckPermissionsOrWithOptions checks any permission with typed options. CheckPermissionsOrWithOptions 使用类型化选项校验任一权限。
func CheckPermissionsOrWithOptions(ctx context.Context, opts PermissionOptions, authType ...string) error {
	mgr, err := GetManager(resolveAuthType(opts.AuthType, authType...))
	if err != nil {
		return err
	}
	return New(mgr).CheckPermissionsOr(ctx, opts)
}

// AddRolesWithOptions adds roles with typed options. AddRolesWithOptions 使用类型化选项添加角色。
func AddRolesWithOptions(ctx context.Context, opts RoleOptions, authType ...string) error {
	mgr, err := GetManager(resolveAuthType(opts.AuthType, authType...))
	if err != nil {
		return err
	}
	return New(mgr).AddRoles(ctx, opts)
}

// RemoveRolesWithOptions removes roles with typed options. RemoveRolesWithOptions 使用类型化选项移除角色。
func RemoveRolesWithOptions(ctx context.Context, opts RoleOptions, authType ...string) error {
	mgr, err := GetManager(resolveAuthType(opts.AuthType, authType...))
	if err != nil {
		return err
	}
	return New(mgr).RemoveRoles(ctx, opts)
}

// CheckRoleWithOptions checks one role with typed options. CheckRoleWithOptions 使用类型化选项校验单个角色。
func CheckRoleWithOptions(ctx context.Context, opts RoleOptions, authType ...string) error {
	mgr, err := GetManager(resolveAuthType(opts.AuthType, authType...))
	if err != nil {
		return err
	}
	return New(mgr).CheckRole(ctx, opts)
}

// CheckRolesAndWithOptions checks all roles with typed options. CheckRolesAndWithOptions 使用类型化选项校验全部角色。
func CheckRolesAndWithOptions(ctx context.Context, opts RoleOptions, authType ...string) error {
	mgr, err := GetManager(resolveAuthType(opts.AuthType, authType...))
	if err != nil {
		return err
	}
	return New(mgr).CheckRolesAnd(ctx, opts)
}

// CheckRolesOrWithOptions checks any role with typed options. CheckRolesOrWithOptions 使用类型化选项校验任一角色。
func CheckRolesOrWithOptions(ctx context.Context, opts RoleOptions, authType ...string) error {
	mgr, err := GetManager(resolveAuthType(opts.AuthType, authType...))
	if err != nil {
		return err
	}
	return New(mgr).CheckRolesOr(ctx, opts)
}

// MustDefault returns the default auth instance or panics. MustDefault 返回默认鉴权实例，失败时 panic。
func MustDefault() *Auth {
	auth, err := Default()
	if err != nil {
		panic(derror.ErrManagerNotFound)
	}
	return auth
}

// resolveAuthType keeps global helper signatures explicit. resolveAuthType 保持全局辅助函数的认证类型解析显式。
func resolveAuthType(optionAuthType string, authType ...string) string {
	if len(authType) == 0 {
		return optionAuthType
	}
	return authType[0]
}
