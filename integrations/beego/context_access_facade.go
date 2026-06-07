// @Author daixk 2026/06/06
package beego

import beegocontext "github.com/beego/beego/v2/server/web/context"

// CheckRoleByContext checks current user role CheckRoleByContext 校验当前用户角色
func CheckRoleByContext(c *beegocontext.Context, role string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckRole(requestContext(c), role)
}

// CheckRolesAndByContext checks all current user roles CheckRolesAndByContext 校验当前用户是否拥有全部角色
func CheckRolesAndByContext(c *beegocontext.Context, roles []string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckRolesAnd(requestContext(c), roles)
}

// CheckRolesOrByContext checks any current user role CheckRolesOrByContext 校验当前用户是否拥有任一角色
func CheckRolesOrByContext(c *beegocontext.Context, roles []string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckRolesOr(requestContext(c), roles)
}

// CheckPermissionByContext checks current user permission CheckPermissionByContext 校验当前用户权限
func CheckPermissionByContext(c *beegocontext.Context, permission string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckPermission(requestContext(c), permission)
}

// CheckPermissionsAndByContext checks all current user permissions CheckPermissionsAndByContext 校验当前用户是否拥有全部权限
func CheckPermissionsAndByContext(c *beegocontext.Context, permissions []string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckPermissionsAnd(requestContext(c), permissions)
}

// CheckPermissionsOrByContext checks any current user permission CheckPermissionsOrByContext 校验当前用户是否拥有任一权限
func CheckPermissionsOrByContext(c *beegocontext.Context, permissions []string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Access().CheckPermissionsOr(requestContext(c), permissions)
}

// GetRolesByContext gets current user roles GetRolesByContext 获取当前用户角色列表
func GetRolesByContext(c *beegocontext.Context) ([]string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Access().GetRoles(requestContext(c))
}

// GetRolesByTokenByContext gets current token roles GetRolesByTokenByContext 使用当前 token 获取角色列表
func GetRolesByTokenByContext(c *beegocontext.Context) ([]string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Access().GetRolesByToken(requestContext(c))
}

// HasRoleByContext checks current user role HasRoleByContext 检查当前用户角色
func HasRoleByContext(c *beegocontext.Context, role string) bool {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return false
	}
	return dCtx.Access().HasRole(requestContext(c), role)
}

// HasRolesByContext checks whether current user has any role HasRolesByContext 检查当前用户是否拥有任一角色
func HasRolesByContext(c *beegocontext.Context, roles []string) bool {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return false
	}
	return dCtx.Access().HasRoles(requestContext(c), roles)
}

// HasRolesOrByContext checks whether current user has any role HasRolesOrByContext 检查当前用户是否拥有任一角色
func HasRolesOrByContext(c *beegocontext.Context, roles []string) bool {
	return HasRolesByContext(c, roles)
}

// HasRolesAndByContext checks whether current user has all roles HasRolesAndByContext 检查当前用户是否拥有全部角色
func HasRolesAndByContext(c *beegocontext.Context, roles []string) bool {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return false
	}
	return dCtx.Access().HasRolesAnd(requestContext(c), roles)
}

// GetPermissionsByContext gets current user permissions GetPermissionsByContext 获取当前用户权限列表
func GetPermissionsByContext(c *beegocontext.Context) ([]string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Access().GetPermissions(requestContext(c))
}

// GetPermissionsByTokenByContext gets current token permissions GetPermissionsByTokenByContext 使用当前 token 获取权限列表
func GetPermissionsByTokenByContext(c *beegocontext.Context) ([]string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Access().GetPermissionsByToken(requestContext(c))
}

// HasPermissionByContext checks current user permission HasPermissionByContext 检查当前用户权限
func HasPermissionByContext(c *beegocontext.Context, permission string) bool {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return false
	}
	return dCtx.Access().HasPermission(requestContext(c), permission)
}

// HasPermissionsByContext checks whether current user has any permission HasPermissionsByContext 检查当前用户是否拥有任一权限
func HasPermissionsByContext(c *beegocontext.Context, permissions []string) bool {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return false
	}
	return dCtx.Access().HasPermissions(requestContext(c), permissions)
}

// HasPermissionsOrByContext checks whether current user has any permission HasPermissionsOrByContext 检查当前用户是否拥有任一权限
func HasPermissionsOrByContext(c *beegocontext.Context, permissions []string) bool {
	return HasPermissionsByContext(c, permissions)
}

// HasPermissionsAndByContext checks whether current user has all permissions HasPermissionsAndByContext 检查当前用户是否拥有全部权限
func HasPermissionsAndByContext(c *beegocontext.Context, permissions []string) bool {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return false
	}
	return dCtx.Access().HasPermissionsAnd(requestContext(c), permissions)
}

// AddRolesByContext adds roles to current token AddRolesByContext 为当前 token 添加角色
func AddRolesByContext(c *beegocontext.Context, roles []string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Access().AddRoles(requestContext(c), roles)
}

// RemoveRolesByContext removes roles from current token RemoveRolesByContext 从当前 token 移除角色
func RemoveRolesByContext(c *beegocontext.Context, roles []string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Access().RemoveRoles(requestContext(c), roles)
}

// AddPermissionsByContext adds permissions to current token AddPermissionsByContext 为当前 token 添加权限
func AddPermissionsByContext(c *beegocontext.Context, permissions []string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Access().AddPermissions(requestContext(c), permissions)
}

// RemovePermissionsByContext removes permissions from current token RemovePermissionsByContext 从当前 token 移除权限
func RemovePermissionsByContext(c *beegocontext.Context, permissions []string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Access().RemovePermissions(requestContext(c), permissions)
}
