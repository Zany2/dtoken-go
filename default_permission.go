package dtoken

import "context"

// CheckPermission delegates to the default authentication namespace. CheckPermission 委托默认认证体系执行。
func CheckPermission(ctx context.Context, opts PermissionOptions) error {
	return AuthType("").CheckPermission(ctx, opts)
}

// CheckPermissionsAnd delegates to the default authentication namespace. CheckPermissionsAnd 委托默认认证体系执行。
func CheckPermissionsAnd(ctx context.Context, opts PermissionOptions) error {
	return AuthType("").CheckPermissionsAnd(ctx, opts)
}

// CheckPermissionsOr delegates to the default authentication namespace. CheckPermissionsOr 委托默认认证体系执行。
func CheckPermissionsOr(ctx context.Context, opts PermissionOptions) error {
	return AuthType("").CheckPermissionsOr(ctx, opts)
}

// AddPermissions delegates to the default authentication namespace. AddPermissions 委托默认认证体系执行。
func AddPermissions(ctx context.Context, opts PermissionOptions) error {
	return AuthType("").AddPermissions(ctx, opts)
}

// AddPermissionsByToken delegates to the default authentication namespace. AddPermissionsByToken 委托默认认证体系执行。
func AddPermissionsByToken(ctx context.Context, token string, permissions []string) error {
	return AuthType("").AddPermissionsByToken(ctx, token, permissions)
}

// RemovePermissions delegates to the default authentication namespace. RemovePermissions 委托默认认证体系执行。
func RemovePermissions(ctx context.Context, opts PermissionOptions) error {
	return AuthType("").RemovePermissions(ctx, opts)
}

// RemovePermissionsByToken delegates to the default authentication namespace. RemovePermissionsByToken 委托默认认证体系执行。
func RemovePermissionsByToken(ctx context.Context, token string, permissions []string) error {
	return AuthType("").RemovePermissionsByToken(ctx, token, permissions)
}

// GetPermissions delegates to the default authentication namespace. GetPermissions 委托默认认证体系执行。
func GetPermissions(ctx context.Context, loginID string) ([]string, error) {
	return AuthType("").GetPermissions(ctx, loginID)
}

// GetPermissionsByToken delegates to the default authentication namespace. GetPermissionsByToken 委托默认认证体系执行。
func GetPermissionsByToken(ctx context.Context, token string) ([]string, error) {
	return AuthType("").GetPermissionsByToken(ctx, token)
}

// HasPermission delegates to the default authentication namespace. HasPermission 委托默认认证体系执行。
func HasPermission(ctx context.Context, loginID, permission string) bool {
	return AuthType("").HasPermission(ctx, loginID, permission)
}

// HasPermissionByToken delegates to the default authentication namespace. HasPermissionByToken 委托默认认证体系执行。
func HasPermissionByToken(ctx context.Context, token, permission string) bool {
	return AuthType("").HasPermissionByToken(ctx, token, permission)
}

// HasPermissionsAnd delegates to the default authentication namespace. HasPermissionsAnd 委托默认认证体系执行。
func HasPermissionsAnd(ctx context.Context, loginID string, permissions []string) bool {
	return AuthType("").HasPermissionsAnd(ctx, loginID, permissions)
}

// HasPermissionsAndByToken delegates to the default authentication namespace. HasPermissionsAndByToken 委托默认认证体系执行。
func HasPermissionsAndByToken(ctx context.Context, token string, permissions []string) bool {
	return AuthType("").HasPermissionsAndByToken(ctx, token, permissions)
}

// HasPermissionsOr delegates to the default authentication namespace. HasPermissionsOr 委托默认认证体系执行。
func HasPermissionsOr(ctx context.Context, loginID string, permissions []string) bool {
	return AuthType("").HasPermissionsOr(ctx, loginID, permissions)
}

// HasPermissionsOrByToken delegates to the default authentication namespace. HasPermissionsOrByToken 委托默认认证体系执行。
func HasPermissionsOrByToken(ctx context.Context, token string, permissions []string) bool {
	return AuthType("").HasPermissionsOrByToken(ctx, token, permissions)
}

// CheckPermissionByToken delegates to the default authentication namespace. CheckPermissionByToken 委托默认认证体系执行。
func CheckPermissionByToken(ctx context.Context, token, permission string) error {
	return AuthType("").CheckPermissionByToken(ctx, token, permission)
}

// CheckPermissionAnd delegates to the default authentication namespace. CheckPermissionAnd 委托默认认证体系执行。
func CheckPermissionAnd(ctx context.Context, loginID string, permissions []string) error {
	return AuthType("").CheckPermissionAnd(ctx, loginID, permissions)
}

// CheckPermissionAndByToken delegates to the default authentication namespace. CheckPermissionAndByToken 委托默认认证体系执行。
func CheckPermissionAndByToken(ctx context.Context, token string, permissions []string) error {
	return AuthType("").CheckPermissionAndByToken(ctx, token, permissions)
}

// CheckPermissionOr delegates to the default authentication namespace. CheckPermissionOr 委托默认认证体系执行。
func CheckPermissionOr(ctx context.Context, loginID string, permissions []string) error {
	return AuthType("").CheckPermissionOr(ctx, loginID, permissions)
}

// CheckPermissionOrByToken delegates to the default authentication namespace. CheckPermissionOrByToken 委托默认认证体系执行。
func CheckPermissionOrByToken(ctx context.Context, token string, permissions []string) error {
	return AuthType("").CheckPermissionOrByToken(ctx, token, permissions)
}
