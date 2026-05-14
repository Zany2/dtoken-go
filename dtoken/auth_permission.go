package dtoken

import "context"

// CheckPermission checks one permission. CheckPermission 校验单个权限。
func (a *Auth) CheckPermission(ctx context.Context, opts PermissionOptions) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.CheckPermission(ctx, opts.LoginID, opts.Permission)
}

// CheckPermissionsAnd checks all permissions. CheckPermissionsAnd 校验全部权限。
func (a *Auth) CheckPermissionsAnd(ctx context.Context, opts PermissionOptions) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.CheckPermissionAnd(ctx, opts.LoginID, opts.Permissions)
}

// CheckPermissionsOr checks any permission. CheckPermissionsOr 校验任一权限。
func (a *Auth) CheckPermissionsOr(ctx context.Context, opts PermissionOptions) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.CheckPermissionOr(ctx, opts.LoginID, opts.Permissions)
}

// AddPermissions adds permissions. AddPermissions 添加权限。
func (a *Auth) AddPermissions(ctx context.Context, opts PermissionOptions) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	if opts.Token != "" {
		return mgr.AddPermissionsByToken(ctx, opts.Token, opts.Permissions)
	}
	return mgr.AddPermissions(ctx, opts.LoginID, opts.Permissions)
}

// RemovePermissions removes permissions. RemovePermissions 移除权限。
func (a *Auth) RemovePermissions(ctx context.Context, opts PermissionOptions) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	if opts.Token != "" {
		return mgr.RemovePermissionsByToken(ctx, opts.Token, opts.Permissions)
	}
	return mgr.RemovePermissions(ctx, opts.LoginID, opts.Permissions)
}

// GetPermissions gets permissions. GetPermissions 获取权限列表。
func (a *Auth) GetPermissions(ctx context.Context, loginID string) ([]string, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.GetPermissions(ctx, loginID)
}
