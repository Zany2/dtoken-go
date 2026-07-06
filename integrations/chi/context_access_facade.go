// @Author daixk 2026/06/05
package chi

import (
	"context"
)

// CheckRoleByCtx checks current user role CheckRoleByCtx 校验当前用户角色
func CheckRoleByCtx(ctx context.Context, role string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckRole(ctx, role)
}

// CheckRolesAndByCtx checks all current user roles CheckRolesAndByCtx 校验当前用户是否拥有全部角色
func CheckRolesAndByCtx(ctx context.Context, roles []string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckRolesAnd(ctx, roles)
}

// CheckRolesOrByCtx checks any current user role CheckRolesOrByCtx 校验当前用户是否拥有任一角色
func CheckRolesOrByCtx(ctx context.Context, roles []string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckRolesOr(ctx, roles)
}

// CheckPermissionByCtx checks current user permission CheckPermissionByCtx 校验当前用户权限
func CheckPermissionByCtx(ctx context.Context, permission string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckPermission(ctx, permission)
}

// CheckPermissionsAndByCtx checks all current user permissions CheckPermissionsAndByCtx 校验当前用户是否拥有全部权限
func CheckPermissionsAndByCtx(ctx context.Context, permissions []string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckPermissionsAnd(ctx, permissions)
}

// CheckPermissionsOrByCtx checks any current user permission CheckPermissionsOrByCtx 校验当前用户是否拥有任一权限
func CheckPermissionsOrByCtx(ctx context.Context, permissions []string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckPermissionsOr(ctx, permissions)
}
