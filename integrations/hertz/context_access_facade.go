// @Author daixk 2026/06/05
package hertz

import (
	hertzapp "github.com/cloudwego/hertz/pkg/app"
)

// CheckRoleByContext checks current user role CheckRoleByContext
func CheckRoleByContext(ctx *hertzapp.RequestContext, role string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckRole(requestContext(ctx), role)
}

// CheckRolesAndByContext delegates to DToken context CheckRolesAndByContext 转发到 DToken 上下文。
func CheckRolesAndByContext(ctx *hertzapp.RequestContext, roles []string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckRolesAnd(requestContext(ctx), roles)
}

// CheckRolesOrByContext delegates to DToken context CheckRolesOrByContext 转发到 DToken 上下文。
func CheckRolesOrByContext(ctx *hertzapp.RequestContext, roles []string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckRolesOr(requestContext(ctx), roles)
}

// CheckPermissionByContext checks current user permission CheckPermissionByContext
func CheckPermissionByContext(ctx *hertzapp.RequestContext, permission string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckPermission(requestContext(ctx), permission)
}

// CheckPermissionsAndByContext delegates to DToken context CheckPermissionsAndByContext 转发到 DToken 上下文。
func CheckPermissionsAndByContext(ctx *hertzapp.RequestContext, permissions []string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckPermissionsAnd(requestContext(ctx), permissions)
}

// CheckPermissionsOrByContext delegates to DToken context CheckPermissionsOrByContext 转发到 DToken 上下文。
func CheckPermissionsOrByContext(ctx *hertzapp.RequestContext, permissions []string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckPermissionsOr(requestContext(ctx), permissions)
}
