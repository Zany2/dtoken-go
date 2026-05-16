// @Author daixk 2025/12/22 15:56:00
package dtoken

import "context"

// CheckRole checks one role. CheckRole 校验单个角色。
func (a *Auth) CheckRole(ctx context.Context, opts RoleOptions) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	if opts.Token != "" {
		return mgr.CheckRoleByToken(ctx, opts.Token, opts.Role)
	}
	return mgr.CheckRole(ctx, opts.LoginID, opts.Role)
}

// CheckRolesAnd checks all roles. CheckRolesAnd 校验全部角色。
func (a *Auth) CheckRolesAnd(ctx context.Context, opts RoleOptions) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	roles := normalizeRoles(opts)
	if opts.Token != "" {
		return mgr.CheckRoleAndByToken(ctx, opts.Token, roles)
	}
	return mgr.CheckRoleAnd(ctx, opts.LoginID, roles)
}

// CheckRolesOr checks any role. CheckRolesOr 校验任一角色。
func (a *Auth) CheckRolesOr(ctx context.Context, opts RoleOptions) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	roles := normalizeRoles(opts)
	if opts.Token != "" {
		return mgr.CheckRoleOrByToken(ctx, opts.Token, roles)
	}
	return mgr.CheckRoleOr(ctx, opts.LoginID, roles)
}

// AddRoles adds roles. AddRoles 添加角色。
func (a *Auth) AddRoles(ctx context.Context, opts RoleOptions) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	roles := normalizeRoles(opts)
	if opts.Token != "" {
		return mgr.AddRolesByToken(ctx, opts.Token, roles)
	}
	return mgr.AddRoles(ctx, opts.LoginID, roles)
}

// RemoveRoles removes roles. RemoveRoles 移除角色。
func (a *Auth) RemoveRoles(ctx context.Context, opts RoleOptions) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	roles := normalizeRoles(opts)
	if opts.Token != "" {
		return mgr.RemoveRolesByToken(ctx, opts.Token, roles)
	}
	return mgr.RemoveRoles(ctx, opts.LoginID, roles)
}

// GetRoles gets roles. GetRoles 获取角色列表。
func (a *Auth) GetRoles(ctx context.Context, loginID string) ([]string, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.GetRoles(ctx, loginID)
}

// normalizeRoles merges single and multiple role options. normalizeRoles 合并单个和多个角色选项。
func normalizeRoles(opts RoleOptions) []string {
	if opts.Role == "" {
		return opts.Roles
	}
	return append([]string{opts.Role}, opts.Roles...)
}
