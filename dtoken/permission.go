// @Author daixk 2025/12/22 15:56:00
package dtoken

import "context"

// AddPermissions adds permissions to a user. AddPermissions 为用户添加权限。
func AddPermissions(ctx context.Context, loginID string, permissions []string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.AddPermissions(ctx, loginID, permissions)
}

// AddPermissionsByToken adds permissions by token. AddPermissionsByToken 按 token 添加权限。
func AddPermissionsByToken(ctx context.Context, tokenValue string, permissions []string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.AddPermissionsByToken(ctx, tokenValue, permissions)
}

// RemovePermissions removes permissions from a user. RemovePermissions 移除用户权限。
func RemovePermissions(ctx context.Context, loginID string, permissions []string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.RemovePermissions(ctx, loginID, permissions)
}

// RemovePermissionsByToken removes permissions by token. RemovePermissionsByToken 按 token 移除权限。
func RemovePermissionsByToken(ctx context.Context, tokenValue string, permissions []string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.RemovePermissionsByToken(ctx, tokenValue, permissions)
}

// GetPermissions returns user permissions. GetPermissions 获取用户权限列表。
func GetPermissions(ctx context.Context, loginID string, authType ...string) ([]string, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.GetPermissions(ctx, loginID)
}

// GetPermissionsByToken returns permissions by token. GetPermissionsByToken 按 token 获取权限列表。
func GetPermissionsByToken(ctx context.Context, tokenValue string, authType ...string) ([]string, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.GetPermissionsByToken(ctx, tokenValue)
}

// HasPermission reports whether a user has a permission. HasPermission 判断用户是否拥有权限。
func HasPermission(ctx context.Context, loginID string, permission string, authType ...string) bool {
	mgr, err := GetManager(authType...)
	if err != nil {
		return false
	}
	return mgr.HasPermission(ctx, loginID, permission)
}

// HasPermissionByToken reports whether a token has a permission. HasPermissionByToken 判断 token 是否拥有权限。
func HasPermissionByToken(ctx context.Context, tokenValue string, permission string, authType ...string) bool {
	mgr, err := GetManager(authType...)
	if err != nil {
		return false
	}
	return mgr.HasPermissionByToken(ctx, tokenValue, permission)
}

// HasPermissionsAnd reports whether all permissions are present. HasPermissionsAnd 判断是否拥有全部权限。
func HasPermissionsAnd(ctx context.Context, loginID string, permissions []string, authType ...string) bool {
	mgr, err := GetManager(authType...)
	if err != nil {
		return false
	}
	return mgr.HasPermissionsAnd(ctx, loginID, permissions)
}

// HasPermissionsAndByToken reports whether all permissions are present by token. HasPermissionsAndByToken 按 token 判断是否拥有全部权限。
func HasPermissionsAndByToken(ctx context.Context, tokenValue string, permissions []string, authType ...string) bool {
	mgr, err := GetManager(authType...)
	if err != nil {
		return false
	}
	return mgr.HasPermissionsAndByToken(ctx, tokenValue, permissions)
}

// HasPermissionsOr reports whether any permission is present. HasPermissionsOr 判断是否拥有任一权限。
func HasPermissionsOr(ctx context.Context, loginID string, permissions []string, authType ...string) bool {
	mgr, err := GetManager(authType...)
	if err != nil {
		return false
	}
	return mgr.HasPermissionsOr(ctx, loginID, permissions)
}

// HasPermissionsOrByToken reports whether any permission is present by token. HasPermissionsOrByToken 按 token 判断是否拥有任一权限。
func HasPermissionsOrByToken(ctx context.Context, tokenValue string, permissions []string, authType ...string) bool {
	mgr, err := GetManager(authType...)
	if err != nil {
		return false
	}
	return mgr.HasPermissionsOrByToken(ctx, tokenValue, permissions)
}

// CheckPermission validates a single permission. CheckPermission 校验单个权限。
func CheckPermission(ctx context.Context, loginID string, permission string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.CheckPermission(ctx, loginID, permission)
}

// CheckPermissionAnd validates all permissions. CheckPermissionAnd 校验全部权限。
func CheckPermissionAnd(ctx context.Context, loginID string, permissions []string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.CheckPermissionAnd(ctx, loginID, permissions)
}

// CheckPermissionOr validates at least one permission. CheckPermissionOr 校验任一权限。
func CheckPermissionOr(ctx context.Context, loginID string, permissions []string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.CheckPermissionOr(ctx, loginID, permissions)
}
