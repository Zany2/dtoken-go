// @Author daixk 2025/12/22 15:56:00
package dtoken

import "context"

// CheckPermission checks one permission. CheckPermission 校验单个权限。
func (a *Auth) CheckPermission(ctx context.Context, opts PermissionOptions) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	if opts.Token != "" {
		return mgr.CheckPermissionByToken(ctx, opts.Token, opts.Permission)
	}
	return mgr.CheckPermission(ctx, opts.LoginID, opts.Permission)
}

// CheckPermissionsAnd checks all permissions. CheckPermissionsAnd 校验全部权限。
func (a *Auth) CheckPermissionsAnd(ctx context.Context, opts PermissionOptions) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	permissions := normalizePermissions(opts)
	if opts.Token != "" {
		return mgr.CheckPermissionAndByToken(ctx, opts.Token, permissions)
	}
	return mgr.CheckPermissionAnd(ctx, opts.LoginID, permissions)
}

// CheckPermissionsOr checks any permission. CheckPermissionsOr 校验任一权限。
func (a *Auth) CheckPermissionsOr(ctx context.Context, opts PermissionOptions) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	permissions := normalizePermissions(opts)
	if opts.Token != "" {
		return mgr.CheckPermissionOrByToken(ctx, opts.Token, permissions)
	}
	return mgr.CheckPermissionOr(ctx, opts.LoginID, permissions)
}

// AddPermissions adds permissions. AddPermissions 添加权限。
func (a *Auth) AddPermissions(ctx context.Context, opts PermissionOptions) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	permissions := normalizePermissions(opts)
	if opts.Token != "" {
		return mgr.AddPermissionsByToken(ctx, opts.Token, permissions)
	}
	return mgr.AddPermissions(ctx, opts.LoginID, permissions)
}

// AddPermissionsByToken adds permissions by token. AddPermissionsByToken 按 Token 添加权限。
func (a *Auth) AddPermissionsByToken(ctx context.Context, token string, permissions []string) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.AddPermissionsByToken(ctx, token, permissions)
}

// RemovePermissions removes permissions. RemovePermissions 移除权限。
func (a *Auth) RemovePermissions(ctx context.Context, opts PermissionOptions) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	permissions := normalizePermissions(opts)
	if opts.Token != "" {
		return mgr.RemovePermissionsByToken(ctx, opts.Token, permissions)
	}
	return mgr.RemovePermissions(ctx, opts.LoginID, permissions)
}

// RemovePermissionsByToken removes permissions by token. RemovePermissionsByToken 按 Token 移除权限。
func (a *Auth) RemovePermissionsByToken(ctx context.Context, token string, permissions []string) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.RemovePermissionsByToken(ctx, token, permissions)
}

// GetPermissions gets permissions. GetPermissions 获取权限列表。
func (a *Auth) GetPermissions(ctx context.Context, loginID string) ([]string, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.GetPermissions(ctx, loginID)
}

// GetPermissionsByToken gets permissions by token. GetPermissionsByToken 按 Token 获取权限列表。
func (a *Auth) GetPermissionsByToken(ctx context.Context, token string) ([]string, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.GetPermissionsByToken(ctx, token)
}

// HasPermission checks whether a login ID has one permission. HasPermission 判断账号是否拥有单个权限。
func (a *Auth) HasPermission(ctx context.Context, loginID, permission string) bool {
	mgr, err := a.requireManager()
	return err == nil && mgr.HasPermission(ctx, loginID, permission)
}

// HasPermissionByToken checks whether a token has one permission. HasPermissionByToken 判断 Token 是否拥有单个权限。
func (a *Auth) HasPermissionByToken(ctx context.Context, token, permission string) bool {
	mgr, err := a.requireManager()
	return err == nil && mgr.HasPermissionByToken(ctx, token, permission)
}

// HasPermissionsAnd checks whether a login ID has all permissions. HasPermissionsAnd 判断账号是否拥有全部权限。
func (a *Auth) HasPermissionsAnd(ctx context.Context, loginID string, permissions []string) bool {
	mgr, err := a.requireManager()
	return err == nil && mgr.HasPermissionsAnd(ctx, loginID, permissions)
}

// HasPermissionsAndByToken checks whether a token has all permissions. HasPermissionsAndByToken 判断 Token 是否拥有全部权限。
func (a *Auth) HasPermissionsAndByToken(ctx context.Context, token string, permissions []string) bool {
	mgr, err := a.requireManager()
	return err == nil && mgr.HasPermissionsAndByToken(ctx, token, permissions)
}

// HasPermissionsOr checks whether a login ID has any permission. HasPermissionsOr 判断账号是否拥有任一权限。
func (a *Auth) HasPermissionsOr(ctx context.Context, loginID string, permissions []string) bool {
	mgr, err := a.requireManager()
	return err == nil && mgr.HasPermissionsOr(ctx, loginID, permissions)
}

// HasPermissionsOrByToken checks whether a token has any permission. HasPermissionsOrByToken 判断 Token 是否拥有任一权限。
func (a *Auth) HasPermissionsOrByToken(ctx context.Context, token string, permissions []string) bool {
	mgr, err := a.requireManager()
	return err == nil && mgr.HasPermissionsOrByToken(ctx, token, permissions)
}

// CheckPermissionByToken validates one permission by token. CheckPermissionByToken 按 Token 校验单个权限。
func (a *Auth) CheckPermissionByToken(ctx context.Context, token, permission string) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.CheckPermissionByToken(ctx, token, permission)
}

// CheckPermissionAnd validates all permissions by login ID. CheckPermissionAnd 按账号校验全部权限。
func (a *Auth) CheckPermissionAnd(ctx context.Context, loginID string, permissions []string) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.CheckPermissionAnd(ctx, loginID, permissions)
}

// CheckPermissionAndByToken validates all permissions by token. CheckPermissionAndByToken 按 Token 校验全部权限。
func (a *Auth) CheckPermissionAndByToken(ctx context.Context, token string, permissions []string) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.CheckPermissionAndByToken(ctx, token, permissions)
}

// CheckPermissionOr validates any permission by login ID. CheckPermissionOr 按账号校验任一权限。
func (a *Auth) CheckPermissionOr(ctx context.Context, loginID string, permissions []string) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.CheckPermissionOr(ctx, loginID, permissions)
}

// CheckPermissionOrByToken validates any permission by token. CheckPermissionOrByToken 按 Token 校验任一权限。
func (a *Auth) CheckPermissionOrByToken(ctx context.Context, token string, permissions []string) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.CheckPermissionOrByToken(ctx, token, permissions)
}

// normalizePermissions merges single and multiple permission options. normalizePermissions 合并单个和多个权限选项。
func normalizePermissions(opts PermissionOptions) []string {
	if opts.Permission == "" {
		return opts.Permissions
	}
	return append([]string{opts.Permission}, opts.Permissions...)
}
