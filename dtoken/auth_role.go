package dtoken

import "context"

// CheckRole checks one role. CheckRole 校验单个角色。
func (a *Auth) CheckRole(ctx context.Context, opts RoleOptions) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.CheckRole(ctx, opts.LoginID, opts.Role)
}

// CheckRolesAnd checks all roles. CheckRolesAnd 校验全部角色。
func (a *Auth) CheckRolesAnd(ctx context.Context, opts RoleOptions) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.CheckRoleAnd(ctx, opts.LoginID, opts.Roles)
}

// CheckRolesOr checks any role. CheckRolesOr 校验任一角色。
func (a *Auth) CheckRolesOr(ctx context.Context, opts RoleOptions) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.CheckRoleOr(ctx, opts.LoginID, opts.Roles)
}

// AddRoles adds roles. AddRoles 添加角色。
func (a *Auth) AddRoles(ctx context.Context, opts RoleOptions) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	if opts.Token != "" {
		return mgr.AddRolesByToken(ctx, opts.Token, opts.Roles)
	}
	return mgr.AddRoles(ctx, opts.LoginID, opts.Roles)
}

// RemoveRoles removes roles. RemoveRoles 移除角色。
func (a *Auth) RemoveRoles(ctx context.Context, opts RoleOptions) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	if opts.Token != "" {
		return mgr.RemoveRolesByToken(ctx, opts.Token, opts.Roles)
	}
	return mgr.RemoveRoles(ctx, opts.LoginID, opts.Roles)
}

// GetRoles gets roles. GetRoles 获取角色列表。
func (a *Auth) GetRoles(ctx context.Context, loginID string) ([]string, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.GetRoles(ctx, loginID)
}
