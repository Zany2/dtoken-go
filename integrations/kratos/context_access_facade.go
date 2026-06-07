// @Author daixk 2026/06/05
package kratos

import (
	"context"
)

// CheckRoleByCtx checks current user role CheckRoleByCtx
func CheckRoleByCtx(ctx context.Context, role string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckRole(ctx, role)
}

// CheckRolesAndByCtx delegates to DToken context CheckRolesAndByCtx 转发到 DToken 上下文。
func CheckRolesAndByCtx(ctx context.Context, roles []string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckRolesAnd(ctx, roles)
}

// CheckRolesOrByCtx delegates to DToken context CheckRolesOrByCtx 转发到 DToken 上下文。
func CheckRolesOrByCtx(ctx context.Context, roles []string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckRolesOr(ctx, roles)
}

// CheckPermissionByCtx checks current user permission CheckPermissionByCtx
func CheckPermissionByCtx(ctx context.Context, permission string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckPermission(ctx, permission)
}

// CheckPermissionsAndByCtx delegates to DToken context CheckPermissionsAndByCtx 转发到 DToken 上下文。
func CheckPermissionsAndByCtx(ctx context.Context, permissions []string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckPermissionsAnd(ctx, permissions)
}

// CheckPermissionsOrByCtx delegates to DToken context CheckPermissionsOrByCtx 转发到 DToken 上下文。
func CheckPermissionsOrByCtx(ctx context.Context, permissions []string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckPermissionsOr(ctx, permissions)
}
