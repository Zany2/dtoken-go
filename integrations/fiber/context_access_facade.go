// @Author daixk 2026/06/05
package fiber

import (
	gofiber "github.com/gofiber/fiber/v2"
)

// CheckRoleByContext checks current user role CheckRoleByContext 校验当前用户角色。
func CheckRoleByContext(c *gofiber.Ctx, role string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckRole(requestContext(c), role)
}

// CheckRolesAndByContext checks all current user roles CheckRolesAndByContext 校验当前用户是否拥有全部角色。
func CheckRolesAndByContext(c *gofiber.Ctx, roles []string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckRolesAnd(requestContext(c), roles)
}

// CheckRolesOrByContext checks any current user role CheckRolesOrByContext 校验当前用户是否拥有任一角色。
func CheckRolesOrByContext(c *gofiber.Ctx, roles []string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckRolesOr(requestContext(c), roles)
}

// CheckPermissionByContext checks current user permission CheckPermissionByContext 校验当前用户权限。
func CheckPermissionByContext(c *gofiber.Ctx, permission string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckPermission(requestContext(c), permission)
}

// CheckPermissionsAndByContext checks all current user permissions CheckPermissionsAndByContext 校验当前用户是否拥有全部权限。
func CheckPermissionsAndByContext(c *gofiber.Ctx, permissions []string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckPermissionsAnd(requestContext(c), permissions)
}

// CheckPermissionsOrByContext checks any current user permission CheckPermissionsOrByContext 校验当前用户是否拥有任一权限。
func CheckPermissionsOrByContext(c *gofiber.Ctx, permissions []string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckPermissionsOr(requestContext(c), permissions)
}
