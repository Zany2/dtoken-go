// @Author daixk 2026/06/05
package echo

import (
	echo4 "github.com/labstack/echo/v4"
)

// CheckRoleByContext checks current user role CheckRoleByContext 校验当前用户角色。
func CheckRoleByContext(c echo4.Context, role string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckRole(requestContext(c), role)
}

// CheckRolesAndByContext checks all current user roles CheckRolesAndByContext 校验当前用户是否拥有全部角色。
func CheckRolesAndByContext(c echo4.Context, roles []string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckRolesAnd(requestContext(c), roles)
}

// CheckRolesOrByContext checks any current user role CheckRolesOrByContext 校验当前用户是否拥有任一角色。
func CheckRolesOrByContext(c echo4.Context, roles []string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckRolesOr(requestContext(c), roles)
}

// CheckPermissionByContext checks current user permission CheckPermissionByContext 校验当前用户权限。
func CheckPermissionByContext(c echo4.Context, permission string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckPermission(requestContext(c), permission)
}

// CheckPermissionsAndByContext checks all current user permissions CheckPermissionsAndByContext 校验当前用户是否拥有全部权限。
func CheckPermissionsAndByContext(c echo4.Context, permissions []string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckPermissionsAnd(requestContext(c), permissions)
}

// CheckPermissionsOrByContext checks any current user permission CheckPermissionsOrByContext 校验当前用户是否拥有任一权限。
func CheckPermissionsOrByContext(c echo4.Context, permissions []string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckPermissionsOr(requestContext(c), permissions)
}
