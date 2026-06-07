// @Author daixk 2026/06/05
package context

import (
	"context"

	"github.com/Zany2/dtoken-go/core/derror"
)

// GetRoles gets current account roles GetRoles 获取当前账号角色列表
func (c *AccessContext) GetRoles(ctx context.Context) ([]string, error) {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return nil, err
	}
	return c.d.manager.GetRoles(ctx, loginID)
}

// GetRolesByToken gets roles by current token GetRolesByToken 根据当前 Token 获取角色列表
func (c *AccessContext) GetRolesByToken(ctx context.Context) ([]string, error) {
	token, err := c.d.requireToken()
	if err != nil {
		return nil, err
	}
	return c.d.manager.GetRolesByToken(ctx, token)
}

// HasRole checks one role HasRole 检查是否拥有指定角色
func (c *AccessContext) HasRole(ctx context.Context, role string) bool {
	token := c.d.GetTokenValue()
	if token == "" {
		return false
	}
	return c.d.manager.HasRoleByToken(ctx, token, role)
}

// HasRoles checks any role HasRoles 检查是否拥有任一角色
func (c *AccessContext) HasRoles(ctx context.Context, roles []string) bool {
	token := c.d.GetTokenValue()
	if token == "" {
		return false
	}
	return c.d.manager.HasRolesOrByToken(ctx, token, roles)
}

// HasRolesAnd checks all roles HasRolesAnd 检查是否拥有全部角色
func (c *AccessContext) HasRolesAnd(ctx context.Context, roles []string) bool {
	token := c.d.GetTokenValue()
	if token == "" {
		return false
	}
	return c.d.manager.HasRolesAndByToken(ctx, token, roles)
}

// CheckRole checks one role with error CheckRole 校验是否拥有指定角色
func (c *AccessContext) CheckRole(ctx context.Context, role string) error {
	token := c.d.GetTokenValue()
	if token == "" {
		return derror.ErrNotLogin
	}
	return c.d.manager.CheckRoleByToken(ctx, token, role)
}

// CheckRolesAnd checks all roles with error CheckRolesAnd 校验是否拥有全部角色
func (c *AccessContext) CheckRolesAnd(ctx context.Context, roles []string) error {
	token := c.d.GetTokenValue()
	if token == "" {
		return derror.ErrNotLogin
	}
	return c.d.manager.CheckRoleAndByToken(ctx, token, roles)
}

// CheckRolesOr checks any role with error CheckRolesOr 校验是否拥有任一角色
func (c *AccessContext) CheckRolesOr(ctx context.Context, roles []string) error {
	token := c.d.GetTokenValue()
	if token == "" {
		return derror.ErrNotLogin
	}
	return c.d.manager.CheckRoleOrByToken(ctx, token, roles)
}

// AddRoles adds roles to current account AddRoles 为当前账号添加角色
func (c *AccessContext) AddRoles(ctx context.Context, roles []string) error {
	token, err := c.d.requireToken()
	if err != nil {
		return err
	}
	return c.d.manager.AddRolesByToken(ctx, token, roles)
}

// RemoveRoles removes roles from current account RemoveRoles 从当前账号移除角色
func (c *AccessContext) RemoveRoles(ctx context.Context, roles []string) error {
	token, err := c.d.requireToken()
	if err != nil {
		return err
	}
	return c.d.manager.RemoveRolesByToken(ctx, token, roles)
}

// GetPermissions gets current account permissions GetPermissions 获取当前账号权限列表
func (c *AccessContext) GetPermissions(ctx context.Context) ([]string, error) {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return nil, err
	}
	return c.d.manager.GetPermissions(ctx, loginID)
}

// GetPermissionsByToken gets permissions by current token GetPermissionsByToken 根据当前 Token 获取权限列表
func (c *AccessContext) GetPermissionsByToken(ctx context.Context) ([]string, error) {
	token, err := c.d.requireToken()
	if err != nil {
		return nil, err
	}
	return c.d.manager.GetPermissionsByToken(ctx, token)
}

// HasPermission checks one permission HasPermission 检查是否拥有指定权限
func (c *AccessContext) HasPermission(ctx context.Context, permission string) bool {
	token := c.d.GetTokenValue()
	if token == "" {
		return false
	}
	return c.d.manager.HasPermissionByToken(ctx, token, permission)
}

// HasPermissions checks any permission HasPermissions 检查是否拥有任一权限
func (c *AccessContext) HasPermissions(ctx context.Context, permissions []string) bool {
	token := c.d.GetTokenValue()
	if token == "" {
		return false
	}
	return c.d.manager.HasPermissionsOrByToken(ctx, token, permissions)
}

// HasPermissionsAnd checks all permissions HasPermissionsAnd 检查是否拥有全部权限
func (c *AccessContext) HasPermissionsAnd(ctx context.Context, permissions []string) bool {
	token := c.d.GetTokenValue()
	if token == "" {
		return false
	}
	return c.d.manager.HasPermissionsAndByToken(ctx, token, permissions)
}

// CheckPermission checks one permission with error CheckPermission 校验是否拥有指定权限
func (c *AccessContext) CheckPermission(ctx context.Context, permission string) error {
	token := c.d.GetTokenValue()
	if token == "" {
		return derror.ErrNotLogin
	}
	return c.d.manager.CheckPermissionByToken(ctx, token, permission)
}

// CheckPermissionsAnd checks all permissions with error CheckPermissionsAnd 校验是否拥有全部权限
func (c *AccessContext) CheckPermissionsAnd(ctx context.Context, permissions []string) error {
	token := c.d.GetTokenValue()
	if token == "" {
		return derror.ErrNotLogin
	}
	return c.d.manager.CheckPermissionAndByToken(ctx, token, permissions)
}

// CheckPermissionsOr checks any permission with error CheckPermissionsOr 校验是否拥有任一权限
func (c *AccessContext) CheckPermissionsOr(ctx context.Context, permissions []string) error {
	token := c.d.GetTokenValue()
	if token == "" {
		return derror.ErrNotLogin
	}
	return c.d.manager.CheckPermissionOrByToken(ctx, token, permissions)
}

// AddPermissions adds permissions to current account AddPermissions 为当前账号添加权限
func (c *AccessContext) AddPermissions(ctx context.Context, permissions []string) error {
	token, err := c.d.requireToken()
	if err != nil {
		return err
	}
	return c.d.manager.AddPermissionsByToken(ctx, token, permissions)
}

// RemovePermissions removes permissions from current account RemovePermissions 从当前账号移除权限
func (c *AccessContext) RemovePermissions(ctx context.Context, permissions []string) error {
	token, err := c.d.requireToken()
	if err != nil {
		return err
	}
	return c.d.manager.RemovePermissionsByToken(ctx, token, permissions)
}
