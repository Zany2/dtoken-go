// @Author daixk 2025/12/22 15:56:00
package dtoken

import "context"

// AddRoles adds roles to a user. AddRoles 为用户添加角色。
func AddRoles(ctx context.Context, loginID string, roles []string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.AddRoles(ctx, loginID, roles)
}

// AddRolesByToken adds roles by token. AddRolesByToken 按 token 添加角色。
func AddRolesByToken(ctx context.Context, tokenValue string, roles []string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.AddRolesByToken(ctx, tokenValue, roles)
}

// RemoveRoles removes roles from a user. RemoveRoles 移除用户角色。
func RemoveRoles(ctx context.Context, loginID string, roles []string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.RemoveRoles(ctx, loginID, roles)
}

// RemoveRolesByToken removes roles by token. RemoveRolesByToken 按 token 移除角色。
func RemoveRolesByToken(ctx context.Context, tokenValue string, roles []string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.RemoveRolesByToken(ctx, tokenValue, roles)
}

// GetRoles returns user roles. GetRoles 获取用户角色列表。
func GetRoles(ctx context.Context, loginID string, authType ...string) ([]string, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.GetRoles(ctx, loginID)
}

// GetRolesByToken returns roles by token. GetRolesByToken 按 token 获取角色列表。
func GetRolesByToken(ctx context.Context, tokenValue string, authType ...string) ([]string, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.GetRolesByToken(ctx, tokenValue)
}

// HasRole reports whether a user has a role. HasRole 判断用户是否拥有角色。
func HasRole(ctx context.Context, loginID string, role string, authType ...string) bool {
	mgr, err := GetManager(authType...)
	if err != nil {
		return false
	}
	return mgr.HasRole(ctx, loginID, role)
}

// HasRoleByToken reports whether a token has a role. HasRoleByToken 判断 token 是否拥有角色。
func HasRoleByToken(ctx context.Context, tokenValue string, role string, authType ...string) bool {
	mgr, err := GetManager(authType...)
	if err != nil {
		return false
	}
	return mgr.HasRoleByToken(ctx, tokenValue, role)
}

// HasRolesAnd reports whether all roles are present. HasRolesAnd 判断是否拥有全部角色。
func HasRolesAnd(ctx context.Context, loginID string, roles []string, authType ...string) bool {
	mgr, err := GetManager(authType...)
	if err != nil {
		return false
	}
	return mgr.HasRolesAnd(ctx, loginID, roles)
}

// HasRolesAndByToken reports whether all roles are present by token. HasRolesAndByToken 按 token 判断是否拥有全部角色。
func HasRolesAndByToken(ctx context.Context, tokenValue string, roles []string, authType ...string) bool {
	mgr, err := GetManager(authType...)
	if err != nil {
		return false
	}
	return mgr.HasRolesAndByToken(ctx, tokenValue, roles)
}

// HasRolesOr reports whether any role is present. HasRolesOr 判断是否拥有任一角色。
func HasRolesOr(ctx context.Context, loginID string, roles []string, authType ...string) bool {
	mgr, err := GetManager(authType...)
	if err != nil {
		return false
	}
	return mgr.HasRolesOr(ctx, loginID, roles)
}

// HasRolesOrByToken reports whether any role is present by token. HasRolesOrByToken 按 token 判断是否拥有任一角色。
func HasRolesOrByToken(ctx context.Context, tokenValue string, roles []string, authType ...string) bool {
	mgr, err := GetManager(authType...)
	if err != nil {
		return false
	}
	return mgr.HasRolesOrByToken(ctx, tokenValue, roles)
}

// CheckRole validates a single role. CheckRole 校验单个角色。
func CheckRole(ctx context.Context, loginID string, role string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.CheckRole(ctx, loginID, role)
}

// CheckRoleByToken validates a single role by token. CheckRoleByToken 按 token 校验单个角色。
func CheckRoleByToken(ctx context.Context, tokenValue string, role string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.CheckRoleByToken(ctx, tokenValue, role)
}

// CheckRoleAnd validates all roles. CheckRoleAnd 校验全部角色。
func CheckRoleAnd(ctx context.Context, loginID string, roles []string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.CheckRoleAnd(ctx, loginID, roles)
}

// CheckRoleAndByToken validates all roles by token. CheckRoleAndByToken 按 token 校验全部角色。
func CheckRoleAndByToken(ctx context.Context, tokenValue string, roles []string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.CheckRoleAndByToken(ctx, tokenValue, roles)
}

// CheckRoleOr validates at least one role. CheckRoleOr 校验任一角色。
func CheckRoleOr(ctx context.Context, loginID string, roles []string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.CheckRoleOr(ctx, loginID, roles)
}

// CheckRoleOrByToken validates at least one role by token. CheckRoleOrByToken 按 token 校验任一角色。
func CheckRoleOrByToken(ctx context.Context, tokenValue string, roles []string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.CheckRoleOrByToken(ctx, tokenValue, roles)
}
