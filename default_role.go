package dtoken

import "context"

// CheckRole delegates to the default authentication namespace. CheckRole 委托默认认证体系执行。
func CheckRole(ctx context.Context, opts RoleOptions) error {
	return defaultAuth.CheckRole(ctx, opts)
}

// CheckRolesAnd delegates to the default authentication namespace. CheckRolesAnd 委托默认认证体系执行。
func CheckRolesAnd(ctx context.Context, opts RoleOptions) error {
	return defaultAuth.CheckRolesAnd(ctx, opts)
}

// CheckRolesOr delegates to the default authentication namespace. CheckRolesOr 委托默认认证体系执行。
func CheckRolesOr(ctx context.Context, opts RoleOptions) error {
	return defaultAuth.CheckRolesOr(ctx, opts)
}

// AddRoles delegates to the default authentication namespace. AddRoles 委托默认认证体系执行。
func AddRoles(ctx context.Context, opts RoleOptions) error {
	return defaultAuth.AddRoles(ctx, opts)
}

// AddRolesByToken delegates to the default authentication namespace. AddRolesByToken 委托默认认证体系执行。
func AddRolesByToken(ctx context.Context, token string, roles []string) error {
	return defaultAuth.AddRolesByToken(ctx, token, roles)
}

// RemoveRoles delegates to the default authentication namespace. RemoveRoles 委托默认认证体系执行。
func RemoveRoles(ctx context.Context, opts RoleOptions) error {
	return defaultAuth.RemoveRoles(ctx, opts)
}

// RemoveRolesByToken delegates to the default authentication namespace. RemoveRolesByToken 委托默认认证体系执行。
func RemoveRolesByToken(ctx context.Context, token string, roles []string) error {
	return defaultAuth.RemoveRolesByToken(ctx, token, roles)
}

// GetRoles delegates to the default authentication namespace. GetRoles 委托默认认证体系执行。
func GetRoles(ctx context.Context, loginID string) ([]string, error) {
	return defaultAuth.GetRoles(ctx, loginID)
}

// GetRolesByToken delegates to the default authentication namespace. GetRolesByToken 委托默认认证体系执行。
func GetRolesByToken(ctx context.Context, token string) ([]string, error) {
	return defaultAuth.GetRolesByToken(ctx, token)
}

// HasRole delegates to the default authentication namespace. HasRole 委托默认认证体系执行。
func HasRole(ctx context.Context, loginID, role string) bool {
	return defaultAuth.HasRole(ctx, loginID, role)
}

// HasRoleByToken delegates to the default authentication namespace. HasRoleByToken 委托默认认证体系执行。
func HasRoleByToken(ctx context.Context, token, role string) bool {
	return defaultAuth.HasRoleByToken(ctx, token, role)
}

// HasRolesAnd delegates to the default authentication namespace. HasRolesAnd 委托默认认证体系执行。
func HasRolesAnd(ctx context.Context, loginID string, roles []string) bool {
	return defaultAuth.HasRolesAnd(ctx, loginID, roles)
}

// HasRolesAndByToken delegates to the default authentication namespace. HasRolesAndByToken 委托默认认证体系执行。
func HasRolesAndByToken(ctx context.Context, token string, roles []string) bool {
	return defaultAuth.HasRolesAndByToken(ctx, token, roles)
}

// HasRolesOr delegates to the default authentication namespace. HasRolesOr 委托默认认证体系执行。
func HasRolesOr(ctx context.Context, loginID string, roles []string) bool {
	return defaultAuth.HasRolesOr(ctx, loginID, roles)
}

// HasRolesOrByToken delegates to the default authentication namespace. HasRolesOrByToken 委托默认认证体系执行。
func HasRolesOrByToken(ctx context.Context, token string, roles []string) bool {
	return defaultAuth.HasRolesOrByToken(ctx, token, roles)
}

// CheckRoleByToken delegates to the default authentication namespace. CheckRoleByToken 委托默认认证体系执行。
func CheckRoleByToken(ctx context.Context, token, role string) error {
	return defaultAuth.CheckRoleByToken(ctx, token, role)
}

// CheckRoleAnd delegates to the default authentication namespace. CheckRoleAnd 委托默认认证体系执行。
func CheckRoleAnd(ctx context.Context, loginID string, roles []string) error {
	return defaultAuth.CheckRoleAnd(ctx, loginID, roles)
}

// CheckRoleAndByToken delegates to the default authentication namespace. CheckRoleAndByToken 委托默认认证体系执行。
func CheckRoleAndByToken(ctx context.Context, token string, roles []string) error {
	return defaultAuth.CheckRoleAndByToken(ctx, token, roles)
}

// CheckRoleOr delegates to the default authentication namespace. CheckRoleOr 委托默认认证体系执行。
func CheckRoleOr(ctx context.Context, loginID string, roles []string) error {
	return defaultAuth.CheckRoleOr(ctx, loginID, roles)
}

// CheckRoleOrByToken delegates to the default authentication namespace. CheckRoleOrByToken 委托默认认证体系执行。
func CheckRoleOrByToken(ctx context.Context, token string, roles []string) error {
	return defaultAuth.CheckRoleOrByToken(ctx, token, roles)
}
