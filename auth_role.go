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

// AddRolesByToken adds roles by token. AddRolesByToken 按 Token 添加角色。
func (a *Auth) AddRolesByToken(ctx context.Context, token string, roles []string) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.AddRolesByToken(ctx, token, roles)
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

// RemoveRolesByToken removes roles by token. RemoveRolesByToken 按 Token 移除角色。
func (a *Auth) RemoveRolesByToken(ctx context.Context, token string, roles []string) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.RemoveRolesByToken(ctx, token, roles)
}

// GetRoles gets roles. GetRoles 获取角色列表。
func (a *Auth) GetRoles(ctx context.Context, loginID string) ([]string, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.GetRoles(ctx, loginID)
}

// GetRolesByToken gets roles by token. GetRolesByToken 按 Token 获取角色列表。
func (a *Auth) GetRolesByToken(ctx context.Context, token string) ([]string, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.GetRolesByToken(ctx, token)
}

// HasRole checks whether a login ID has one role. HasRole 判断账号是否拥有单个角色。
func (a *Auth) HasRole(ctx context.Context, loginID, role string) bool {
	mgr, err := a.requireManager()
	return err == nil && mgr.HasRole(ctx, loginID, role)
}

// HasRoleByToken checks whether a token has one role. HasRoleByToken 判断 Token 是否拥有单个角色。
func (a *Auth) HasRoleByToken(ctx context.Context, token, role string) bool {
	mgr, err := a.requireManager()
	return err == nil && mgr.HasRoleByToken(ctx, token, role)
}

// HasRolesAnd checks whether a login ID has all roles. HasRolesAnd 判断账号是否拥有全部角色。
func (a *Auth) HasRolesAnd(ctx context.Context, loginID string, roles []string) bool {
	mgr, err := a.requireManager()
	return err == nil && mgr.HasRolesAnd(ctx, loginID, roles)
}

// HasRolesAndByToken checks whether a token has all roles. HasRolesAndByToken 判断 Token 是否拥有全部角色。
func (a *Auth) HasRolesAndByToken(ctx context.Context, token string, roles []string) bool {
	mgr, err := a.requireManager()
	return err == nil && mgr.HasRolesAndByToken(ctx, token, roles)
}

// HasRolesOr checks whether a login ID has any role. HasRolesOr 判断账号是否拥有任一角色。
func (a *Auth) HasRolesOr(ctx context.Context, loginID string, roles []string) bool {
	mgr, err := a.requireManager()
	return err == nil && mgr.HasRolesOr(ctx, loginID, roles)
}

// HasRolesOrByToken checks whether a token has any role. HasRolesOrByToken 判断 Token 是否拥有任一角色。
func (a *Auth) HasRolesOrByToken(ctx context.Context, token string, roles []string) bool {
	mgr, err := a.requireManager()
	return err == nil && mgr.HasRolesOrByToken(ctx, token, roles)
}

// CheckRoleByToken validates one role by token. CheckRoleByToken 按 Token 校验单个角色。
func (a *Auth) CheckRoleByToken(ctx context.Context, token, role string) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.CheckRoleByToken(ctx, token, role)
}

// CheckRoleAnd validates all roles by login ID. CheckRoleAnd 按账号校验全部角色。
func (a *Auth) CheckRoleAnd(ctx context.Context, loginID string, roles []string) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.CheckRoleAnd(ctx, loginID, roles)
}

// CheckRoleAndByToken validates all roles by token. CheckRoleAndByToken 按 Token 校验全部角色。
func (a *Auth) CheckRoleAndByToken(ctx context.Context, token string, roles []string) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.CheckRoleAndByToken(ctx, token, roles)
}

// CheckRoleOr validates any role by login ID. CheckRoleOr 按账号校验任一角色。
func (a *Auth) CheckRoleOr(ctx context.Context, loginID string, roles []string) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.CheckRoleOr(ctx, loginID, roles)
}

// CheckRoleOrByToken validates any role by token. CheckRoleOrByToken 按 Token 校验任一角色。
func (a *Auth) CheckRoleOrByToken(ctx context.Context, token string, roles []string) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.CheckRoleOrByToken(ctx, token, roles)
}

// normalizeRoles merges single and multiple role options. normalizeRoles 合并单个和多个角色选项。
func normalizeRoles(opts RoleOptions) []string {
	if opts.Role == "" {
		return opts.Roles
	}
	return append([]string{opts.Role}, opts.Roles...)
}
