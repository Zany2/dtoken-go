package gf

import (
	"context"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/builder"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/manager"
	"github.com/Zany2/dtoken-go/dtoken"
)

// ============================================================================
// Type Aliases - 类型别名
// ============================================================================

type (
	// Config represents the DToken configuration.
	// Config 表示 DToken 配置。
	Config = config.Config

	// Manager represents the DToken manager.
	// Manager 表示 DToken 管理器。
	Manager = manager.Manager

	// TokenInfo represents token information.
	// TokenInfo 表示 Token 信息。
	TokenInfo = manager.TokenInfo

	// DisableInfo represents account disable information.
	// DisableInfo 表示账号封禁信息。
	DisableInfo = manager.DisableInfo

	// Session represents user session information.
	// Session 表示用户会话信息。
	Session = manager.Session

	// Builder represents the DToken builder.
	// Builder 表示 DToken 构建器。
	Builder = builder.Builder

	// DTokenError represents a DToken error.
	// DTokenError 表示 DToken 错误。
	DTokenError = derror.DTokenError

	// TokenStyle represents token generation style.
	// TokenStyle 表示 Token 生成风格。
	TokenStyle = adapter.TokenStyle
)

// ============================================================================
// Error Codes - 错误码
// ============================================================================

const (
	// CodeSuccess indicates a successful operation.
	// CodeSuccess 表示操作成功。
	CodeSuccess = derror.CodeSuccess

	// CodeBadRequest indicates a bad request.
	// CodeBadRequest 表示请求参数错误。
	CodeBadRequest = derror.CodeBadRequest

	// CodeNotLogin indicates user is not logged in.
	// CodeNotLogin 表示用户未登录。
	CodeNotLogin = derror.CodeNotLogin

	// CodePermissionDenied indicates permission denied.
	// CodePermissionDenied 表示权限不足。
	CodePermissionDenied = derror.CodePermissionDenied

	// CodeNotFound indicates resource not found.
	// CodeNotFound 表示资源未找到。
	CodeNotFound = derror.CodeNotFound

	// CodeServerError indicates internal server error.
	// CodeServerError 表示服务器内部错误。
	CodeServerError = derror.CodeServerError

	// CodeTokenInvalid indicates invalid token.
	// CodeTokenInvalid 表示 Token 无效。
	CodeTokenInvalid = derror.CodeTokenInvalid

	// CodeTokenExpired indicates token expired.
	// CodeTokenExpired 表示 Token 已过期。
	CodeTokenExpired = derror.CodeTokenExpired

	// CodeAccountDisabled indicates account is disabled.
	// CodeAccountDisabled 表示账号已被封禁。
	CodeAccountDisabled = derror.CodeAccountDisabled
)

// ============================================================================
// Error Variables - 错误变量
// ============================================================================

var (
	// ErrNotLogin indicates user is not logged in.
	// ErrNotLogin 表示用户未登录。
	ErrNotLogin = derror.ErrNotLogin

	// ErrInvalidToken indicates token is invalid.
	// ErrInvalidToken 表示 Token 无效。
	ErrInvalidToken = derror.ErrInvalidToken

	// ErrTokenExpired indicates token has expired.
	// ErrTokenExpired 表示 Token 已过期。
	ErrTokenExpired = derror.ErrTokenExpired

	// ErrPermissionDenied indicates permission denied.
	// ErrPermissionDenied 表示权限不足。
	ErrPermissionDenied = derror.ErrPermissionDenied

	// ErrRoleDenied indicates role denied.
	// ErrRoleDenied 表示角色不足。
	ErrRoleDenied = derror.ErrRoleDenied

	// ErrAccountDisabled indicates account is disabled.
	// ErrAccountDisabled 表示账号已被封禁。
	ErrAccountDisabled = derror.ErrAccountDisabled
)

// ============================================================================
// Token Style Constants - Token 风格常量
// ============================================================================

const (
	// TokenStyleUUID represents UUID style token.
	// TokenStyleUUID 表示 UUID 风格 Token。
	TokenStyleUUID = adapter.TokenStyleUUID

	// TokenStyleSimple represents simple random string token.
	// TokenStyleSimple 表示简单随机字符串 Token。
	TokenStyleSimple = adapter.TokenStyleSimple

	// TokenStyleRandom32 represents 32-bit random string token.
	// TokenStyleRandom32 表示 32 位随机字符串 Token。
	TokenStyleRandom32 = adapter.TokenStyleRandom32

	// TokenStyleRandom64 represents 64-bit random string token.
	// TokenStyleRandom64 表示 64 位随机字符串 Token。
	TokenStyleRandom64 = adapter.TokenStyleRandom64

	// TokenStyleRandom128 represents 128-bit random string token.
	// TokenStyleRandom128 表示 128 位随机字符串 Token。
	TokenStyleRandom128 = adapter.TokenStyleRandom128

	// TokenStyleJWT represents JWT style token.
	// TokenStyleJWT 表示 JWT 风格 Token。
	TokenStyleJWT = adapter.TokenStyleJWT
)

// ============================================================================
// Manager Management - Manager 管理
// ============================================================================

// SetManager stores the manager in the global map.
// SetManager 将管理器存储在全局 map 中。
func SetManager(mgr *manager.Manager) {
	dtoken.SetManager(mgr)
}

// GetManager retrieves the manager from the global map.
// GetManager 从全局 map 中获取管理器。
func GetManager(authType ...string) (*manager.Manager, error) {
	return dtoken.GetManager(authType...)
}

// DeleteManager deletes the specific manager and releases resources.
// DeleteManager 删除指定的管理器并释放资源。
func DeleteManager(authType ...string) error {
	return dtoken.DeleteManager(authType...)
}

// DeleteAllManager deletes all managers and releases resources.
// DeleteAllManager 删除所有管理器并释放资源。
func DeleteAllManager() {
	dtoken.DeleteAllManager()
}

// ============================================================================
// Builder & Config - 构建器和配置
// ============================================================================

// NewDefaultBuilder creates a new default builder.
// NewDefaultBuilder 创建默认构建器。
func NewDefaultBuilder() *builder.Builder {
	return builder.NewBuilder()
}

// NewDefaultConfig creates a new default config.
// NewDefaultConfig 创建默认配置。
func NewDefaultConfig() *config.Config {
	return config.DefaultConfig()
}

// ============================================================================
// Authentication - 登录认证
// ============================================================================

// Login performs user login.
// Login 用户登录。
func Login(ctx context.Context, loginID string, params ...string) (string, error) {
	return dtoken.Login(ctx, loginID, params...)
}

// LoginByToken performs login with specified token.
// LoginByToken 使用指定 Token 登录。
func LoginByToken(ctx context.Context, tokenValue string, authType ...string) error {
	return dtoken.LoginByToken(ctx, tokenValue, authType...)
}

// Logout performs user logout.
// Logout 用户登出。
func Logout(ctx context.Context, tokenValue string, authType ...string) error {
	return dtoken.Logout(ctx, tokenValue, authType...)
}

// LogoutByDevice logs out by device.
// LogoutByDevice 根据设备类型登出。
func LogoutByDevice(ctx context.Context, loginID string, device string, authType ...string) error {
	return dtoken.LogoutByDevice(ctx, loginID, device, authType...)
}

// Kickout kicks out a user session.
// Kickout 踢人下线。
func Kickout(ctx context.Context, tokenValue string, authType ...string) error {
	return dtoken.Kickout(ctx, tokenValue, authType...)
}

// Replace replaces user offline.
// Replace 顶人下线。
func Replace(ctx context.Context, tokenValue string, authType ...string) error {
	return dtoken.Replace(ctx, tokenValue, authType...)
}

// ============================================================================
// Token Validation - Token 验证
// ============================================================================

// IsLogin checks if the user is logged in.
// IsLogin 检查用户是否已登录。
func IsLogin(ctx context.Context, tokenValue string, authType ...string) bool {
	return dtoken.IsLogin(ctx, tokenValue, authType...)
}

// CheckLogin checks login status (throws error if not logged in).
// CheckLogin 检查登录状态（未登录抛出错误）。
func CheckLogin(ctx context.Context, tokenValue string, authType ...string) error {
	return dtoken.CheckLogin(ctx, tokenValue, authType...)
}

// GetLoginID gets the login ID from token.
// GetLoginID 从 Token 获取登录 ID。
func GetLoginID(ctx context.Context, tokenValue string, authType ...string) (string, error) {
	return dtoken.GetLoginID(ctx, tokenValue, authType...)
}

// GetTokenInfo gets token information.
// GetTokenInfo 获取 Token 信息。
func GetTokenInfo(ctx context.Context, tokenValue string, authType ...string) (*manager.TokenInfo, error) {
	return dtoken.GetTokenInfo(ctx, tokenValue, authType...)
}

// GetDevice gets device from token.
// GetDevice 从 Token 获取设备类型。
func GetDevice(ctx context.Context, tokenValue string, authType ...string) (string, error) {
	return dtoken.GetDevice(ctx, tokenValue, authType...)
}

// GetTokenTTL gets token TTL.
// GetTokenTTL 获取 Token 剩余时间。
func GetTokenTTL(ctx context.Context, tokenValue string, authType ...string) (int64, error) {
	return dtoken.GetTokenTTL(ctx, tokenValue, authType...)
}

// ============================================================================
// Account Disable - 账号封禁
// ============================================================================

// Disable disables an account for specified duration.
// Disable 封禁账号（指定时长）。
func Disable(ctx context.Context, loginID string, duration time.Duration, reason string, authType ...string) error {
	return dtoken.Disable(ctx, loginID, duration, reason, authType...)
}

// Untie re-enables a disabled account.
// Untie 解封账号。
func Untie(ctx context.Context, loginID string, authType ...string) error {
	return dtoken.Untie(ctx, loginID, authType...)
}

// IsDisable checks if an account is disabled.
// IsDisable 检查账号是否被封禁。
func IsDisable(ctx context.Context, loginID string, authType ...string) bool {
	return dtoken.IsDisable(ctx, loginID, authType...)
}

// GetDisableInfo gets disable info.
// GetDisableInfo 获取封禁信息。
func GetDisableInfo(ctx context.Context, loginID string, authType ...string) (*manager.DisableInfo, error) {
	return dtoken.GetDisableInfo(ctx, loginID, authType...)
}

// ============================================================================
// Session Management - Session 管理
// ============================================================================

// GetSession gets session by login ID.
// GetSession 根据登录 ID 获取 Session。
func GetSession(ctx context.Context, loginID string, authType ...string) (*manager.Session, error) {
	return dtoken.GetSession(ctx, loginID, authType...)
}

// GetSessionByToken gets session by token.
// GetSessionByToken 根据 Token 获取 Session。
func GetSessionByToken(ctx context.Context, tokenValue string, authType ...string) (*manager.Session, error) {
	return dtoken.GetSessionByToken(ctx, tokenValue, authType...)
}

// GetTokenValueListByLoginID gets all tokens for a login ID.
// GetTokenValueListByLoginID 获取指定账号的所有 Token。
func GetTokenValueListByLoginID(ctx context.Context, loginID string, checkAlive bool, authType ...string) ([]string, error) {
	return dtoken.GetTokenValueListByLoginID(ctx, loginID, checkAlive, authType...)
}

// ============================================================================
// Permission Verification - 权限验证
// ============================================================================

// AddPermissions adds permissions.
// AddPermissions 添加权限。
func AddPermissions(ctx context.Context, loginID string, permissions []string, authType ...string) error {
	return dtoken.AddPermissions(ctx, loginID, permissions, authType...)
}

// RemovePermissions removes permissions.
// RemovePermissions 删除权限。
func RemovePermissions(ctx context.Context, loginID string, permissions []string, authType ...string) error {
	return dtoken.RemovePermissions(ctx, loginID, permissions, authType...)
}

// GetPermissions gets permission list.
// GetPermissions 获取权限列表。
func GetPermissions(ctx context.Context, loginID string, authType ...string) ([]string, error) {
	return dtoken.GetPermissions(ctx, loginID, authType...)
}

// HasPermission checks if has specified permission.
// HasPermission 检查是否拥有指定权限。
func HasPermission(ctx context.Context, loginID string, permission string, authType ...string) bool {
	return dtoken.HasPermission(ctx, loginID, permission, authType...)
}

// HasPermissionsAnd checks if has all permissions (AND logic).
// HasPermissionsAnd 检查是否拥有所有权限（AND 逻辑）。
func HasPermissionsAnd(ctx context.Context, loginID string, permissions []string, authType ...string) bool {
	return dtoken.HasPermissionsAnd(ctx, loginID, permissions, authType...)
}

// HasPermissionsOr checks if has any permission (OR logic).
// HasPermissionsOr 检查是否拥有任一权限（OR 逻辑）。
func HasPermissionsOr(ctx context.Context, loginID string, permissions []string, authType ...string) bool {
	return dtoken.HasPermissionsOr(ctx, loginID, permissions, authType...)
}

// ============================================================================
// Role Management - 角色管理
// ============================================================================

// AddRoles adds roles.
// AddRoles 添加角色。
func AddRoles(ctx context.Context, loginID string, roles []string, authType ...string) error {
	return dtoken.AddRoles(ctx, loginID, roles, authType...)
}

// RemoveRoles removes roles.
// RemoveRoles 删除角色。
func RemoveRoles(ctx context.Context, loginID string, roles []string, authType ...string) error {
	return dtoken.RemoveRoles(ctx, loginID, roles, authType...)
}

// GetRoles gets role list.
// GetRoles 获取角色列表。
func GetRoles(ctx context.Context, loginID string, authType ...string) ([]string, error) {
	return dtoken.GetRoles(ctx, loginID, authType...)
}

// HasRole checks if has specified role.
// HasRole 检查是否拥有指定角色。
func HasRole(ctx context.Context, loginID string, role string, authType ...string) bool {
	return dtoken.HasRole(ctx, loginID, role, authType...)
}

// HasRolesAnd checks if has all roles (AND logic).
// HasRolesAnd 检查是否拥有所有角色（AND 逻辑）。
func HasRolesAnd(ctx context.Context, loginID string, roles []string, authType ...string) bool {
	return dtoken.HasRolesAnd(ctx, loginID, roles, authType...)
}

// HasRolesOr checks if has any role (OR logic).
// HasRolesOr 检查是否拥有任一角色（OR 逻辑）。
func HasRolesOr(ctx context.Context, loginID string, roles []string, authType ...string) bool {
	return dtoken.HasRolesOr(ctx, loginID, roles, authType...)
}
