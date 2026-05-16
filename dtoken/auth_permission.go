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

// GetPermissions gets permissions. GetPermissions 获取权限列表。
func (a *Auth) GetPermissions(ctx context.Context, loginID string) ([]string, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.GetPermissions(ctx, loginID)
}

// normalizePermissions merges single and multiple permission options. normalizePermissions 合并单个和多个权限选项。
func normalizePermissions(opts PermissionOptions) []string {
	if opts.Permission == "" {
		return opts.Permissions
	}
	return append([]string{opts.Permission}, opts.Permissions...)
}
