// @Author daixk 2026/06/05
package hertz

import (
	hertzapp "github.com/cloudwego/hertz/pkg/app"
)

// CheckRoleByContext checks current user role CheckRoleByContext 校验当前用户角色
func CheckRoleByContext(ctx *hertzapp.RequestContext, role string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckRole(requestContext(ctx), role)
}

// CheckRolesAndByContext checks all current user roles CheckRolesAndByContext 校验当前用户是否拥有全部角色
func CheckRolesAndByContext(ctx *hertzapp.RequestContext, roles []string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckRolesAnd(requestContext(ctx), roles)
}

// CheckRolesOrByContext checks any current user role CheckRolesOrByContext 校验当前用户是否拥有任一角色
func CheckRolesOrByContext(ctx *hertzapp.RequestContext, roles []string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckRolesOr(requestContext(ctx), roles)
}

// CheckPermissionByContext checks current user permission CheckPermissionByContext 校验当前用户权限
func CheckPermissionByContext(ctx *hertzapp.RequestContext, permission string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckPermission(requestContext(ctx), permission)
}

// CheckPermissionsAndByContext checks all current user permissions CheckPermissionsAndByContext 校验当前用户是否拥有全部权限
func CheckPermissionsAndByContext(ctx *hertzapp.RequestContext, permissions []string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckPermissionsAnd(requestContext(ctx), permissions)
}

// CheckPermissionsOrByContext checks any current user permission CheckPermissionsOrByContext 校验当前用户是否拥有任一权限
func CheckPermissionsOrByContext(ctx *hertzapp.RequestContext, permissions []string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckPermissionsOr(requestContext(ctx), permissions)
}
