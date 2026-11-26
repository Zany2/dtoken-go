// @Author daixk 2026/06/06
package beego

import beegocontext "github.com/beego/beego/v2/server/web/context"

// HasRolesOrByContext checks whether current user has any role HasRolesOrByContext 检查当前用户是否拥有任一角色
func HasRolesOrByContext(c *beegocontext.Context, roles []string) bool {
	return HasRolesByContext(c, roles)
}

// HasPermissionsOrByContext checks whether current user has any permission HasPermissionsOrByContext 检查当前用户是否拥有任一权限
func HasPermissionsOrByContext(c *beegocontext.Context, permissions []string) bool {
	return HasPermissionsByContext(c, permissions)
}
