package gin

import (
	"context"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/builder"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/manager"
	"github.com/Zany2/dtoken-go/core/oauth2"
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

// LogoutByDeviceAndDeviceId logs out by device type and device ID.
// LogoutByDeviceAndDeviceId 根据设备类型和设备ID登出。
func LogoutByDeviceAndDeviceId(ctx context.Context, loginID string, params ...string) error {
	return dtoken.LogoutByDeviceAndDeviceId(ctx, loginID, params...)
}

// LogoutByDevice logs out by device.
// LogoutByDevice 根据设备类型登出。
func LogoutByDevice(ctx context.Context, loginID string, device string, authType ...string) error {
	return dtoken.LogoutByDevice(ctx, loginID, device, authType...)
}

// LogoutByLoginID logs out all terminals for the specified loginID.
// LogoutByLoginID 登出指定 loginID 的所有终端。
func LogoutByLoginID(ctx context.Context, loginID string, authType ...string) error {
	return dtoken.LogoutByLoginID(ctx, loginID, authType...)
}

// Kickout kicks out a user session.
// Kickout 踢人下线。
func Kickout(ctx context.Context, tokenValue string, authType ...string) error {
	return dtoken.Kickout(ctx, tokenValue, authType...)
}

// KickoutByDeviceAndDeviceId kicks out by device type and device ID.
// KickoutByDeviceAndDeviceId 根据设备类型和设备ID踢人下线。
func KickoutByDeviceAndDeviceId(ctx context.Context, loginID string, params ...string) error {
	return dtoken.KickoutByDeviceAndDeviceId(ctx, loginID, params...)
}

// KickoutByDevice kicks out all terminals of a specific device type.
// KickoutByDevice 根据设备类型踢人下线。
func KickoutByDevice(ctx context.Context, loginID string, device string, authType ...string) error {
	return dtoken.KickoutByDevice(ctx, loginID, device, authType...)
}

// KickoutByLoginID kicks out all terminals for the specified loginID.
// KickoutByLoginID 踢出指定 loginID 的所有终端。
func KickoutByLoginID(ctx context.Context, loginID string, authType ...string) error {
	return dtoken.KickoutByLoginID(ctx, loginID, authType...)
}

// Replace replaces user offline.
// Replace 顶人下线。
func Replace(ctx context.Context, tokenValue string, authType ...string) error {
	return dtoken.Replace(ctx, tokenValue, authType...)
}

// ReplaceByDeviceAndDeviceId replaces by device type and device ID.
// ReplaceByDeviceAndDeviceId 根据设备类型和设备ID顶人下线。
func ReplaceByDeviceAndDeviceId(ctx context.Context, loginID string, params ...string) error {
	return dtoken.ReplaceByDeviceAndDeviceId(ctx, loginID, params...)
}

// ReplaceByDevice replaces all terminals of a specific device type.
// ReplaceByDevice 根据设备类型顶人下线。
func ReplaceByDevice(ctx context.Context, loginID string, device string, authType ...string) error {
	return dtoken.ReplaceByDevice(ctx, loginID, device, authType...)
}

// ReplaceByLoginID replaces all terminals for the specified loginID.
// ReplaceByLoginID 顶替指定 loginID 的所有终端。
func ReplaceByLoginID(ctx context.Context, loginID string, authType ...string) error {
	return dtoken.ReplaceByLoginID(ctx, loginID, authType...)
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

// GetDeviceId gets device ID from token.
// GetDeviceId 从 Token 获取设备 ID。
func GetDeviceId(ctx context.Context, tokenValue string, authType ...string) (string, error) {
	return dtoken.GetDeviceId(ctx, tokenValue, authType...)
}

// GetTokenCreateTime gets token creation time.
// GetTokenCreateTime 获取 Token 创建时间。
func GetTokenCreateTime(ctx context.Context, tokenValue string, authType ...string) (int64, error) {
	return dtoken.GetTokenCreateTime(ctx, tokenValue, authType...)
}

// GetTokenTTL gets token TTL.
// GetTokenTTL 获取 Token 剩余时间。
func GetTokenTTL(ctx context.Context, tokenValue string, authType ...string) (int64, error) {
	return dtoken.GetTokenTTL(ctx, tokenValue, authType...)
}

// GetOnlineTerminalCount gets total number of online terminals.
// GetOnlineTerminalCount 获取在线终端总数。
func GetOnlineTerminalCount(ctx context.Context, loginID string, authType ...string) (int, error) {
	return dtoken.GetOnlineTerminalCount(ctx, loginID, authType...)
}

// GetOnlineTerminalCountByDevice gets online terminal count by device.
// GetOnlineTerminalCountByDevice 获取指定设备类型的在线终端数。
func GetOnlineTerminalCountByDevice(ctx context.Context, loginID string, device string, authType ...string) (int, error) {
	return dtoken.GetOnlineTerminalCountByDevice(ctx, loginID, device, authType...)
}

// GetOnlineTerminalCountByDeviceAndDeviceId gets online terminal count by device and device ID.
// GetOnlineTerminalCountByDeviceAndDeviceId 获取指定设备类型和设备ID的在线终端数。
func GetOnlineTerminalCountByDeviceAndDeviceId(ctx context.Context, loginID string, device string, deviceId string, authType ...string) (int, error) {
	return dtoken.GetOnlineTerminalCountByDeviceAndDeviceId(ctx, loginID, device, deviceId, authType...)
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

// GetDisableTTL gets remaining disable time in seconds.
// GetDisableTTL 获取账号剩余封禁时间（秒）。
func GetDisableTTL(ctx context.Context, loginID string, authType ...string) (int64, error) {
	return dtoken.GetDisableTTL(ctx, loginID, authType...)
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

// GetTokenValueListByDevice gets all tokens for a specific device type.
// GetTokenValueListByDevice 获取指定设备类型的所有 Token。
func GetTokenValueListByDevice(ctx context.Context, loginID string, device string, checkAlive bool, authType ...string) ([]string, error) {
	return dtoken.GetTokenValueListByDevice(ctx, loginID, device, checkAlive, authType...)
}

// GetTokenValueListByDeviceAndDeviceId gets all tokens for a specific device type and device ID.
// GetTokenValueListByDeviceAndDeviceId 获取指定设备类型和设备 ID 的所有 Token。
func GetTokenValueListByDeviceAndDeviceId(ctx context.Context, loginID string, device string, deviceId string, checkAlive bool, authType ...string) ([]string, error) {
	return dtoken.GetTokenValueListByDeviceAndDeviceId(ctx, loginID, device, deviceId, checkAlive, authType...)
}

// ============================================================================
// Permission Verification - 权限验证
// ============================================================================

// AddPermissions adds permissions.
// AddPermissions 添加权限。
func AddPermissions(ctx context.Context, loginID string, permissions []string, authType ...string) error {
	return dtoken.AddPermissions(ctx, loginID, permissions, authType...)
}

// AddPermissionsByToken adds permissions by token.
// AddPermissionsByToken 根据 Token 添加权限。
func AddPermissionsByToken(ctx context.Context, tokenValue string, permissions []string, authType ...string) error {
	return dtoken.AddPermissionsByToken(ctx, tokenValue, permissions, authType...)
}

// RemovePermissions removes permissions.
// RemovePermissions 删除权限。
func RemovePermissions(ctx context.Context, loginID string, permissions []string, authType ...string) error {
	return dtoken.RemovePermissions(ctx, loginID, permissions, authType...)
}

// RemovePermissionsByToken removes permissions by token.
// RemovePermissionsByToken 根据 Token 删除权限。
func RemovePermissionsByToken(ctx context.Context, tokenValue string, permissions []string, authType ...string) error {
	return dtoken.RemovePermissionsByToken(ctx, tokenValue, permissions, authType...)
}

// GetPermissions gets permission list.
// GetPermissions 获取权限列表。
func GetPermissions(ctx context.Context, loginID string, authType ...string) ([]string, error) {
	return dtoken.GetPermissions(ctx, loginID, authType...)
}

// GetPermissionsByToken gets permission list by token.
// GetPermissionsByToken 根据 Token 获取权限列表。
func GetPermissionsByToken(ctx context.Context, tokenValue string, authType ...string) ([]string, error) {
	return dtoken.GetPermissionsByToken(ctx, tokenValue, authType...)
}

// HasPermission checks if has specified permission.
// HasPermission 检查是否拥有指定权限。
func HasPermission(ctx context.Context, loginID string, permission string, authType ...string) bool {
	return dtoken.HasPermission(ctx, loginID, permission, authType...)
}

// HasPermissionByToken checks if has specified permission by token.
// HasPermissionByToken 根据 Token 检查是否拥有指定权限。
func HasPermissionByToken(ctx context.Context, tokenValue string, permission string, authType ...string) bool {
	return dtoken.HasPermissionByToken(ctx, tokenValue, permission, authType...)
}

// HasPermissionsAnd checks if has all permissions (AND logic).
// HasPermissionsAnd 检查是否拥有所有权限（AND 逻辑）。
func HasPermissionsAnd(ctx context.Context, loginID string, permissions []string, authType ...string) bool {
	return dtoken.HasPermissionsAnd(ctx, loginID, permissions, authType...)
}

// HasPermissionsAndByToken checks if has all permissions by token (AND logic).
// HasPermissionsAndByToken 根据 Token 检查是否拥有所有权限（AND 逻辑）。
func HasPermissionsAndByToken(ctx context.Context, tokenValue string, permissions []string, authType ...string) bool {
	return dtoken.HasPermissionsAndByToken(ctx, tokenValue, permissions, authType...)
}

// HasPermissionsOr checks if has any permission (OR logic).
// HasPermissionsOr 检查是否拥有任一权限（OR 逻辑）。
func HasPermissionsOr(ctx context.Context, loginID string, permissions []string, authType ...string) bool {
	return dtoken.HasPermissionsOr(ctx, loginID, permissions, authType...)
}

// HasPermissionsOrByToken checks if has any permission by token (OR logic).
// HasPermissionsOrByToken 根据 Token 检查是否拥有任一权限（OR 逻辑）。
func HasPermissionsOrByToken(ctx context.Context, tokenValue string, permissions []string, authType ...string) bool {
	return dtoken.HasPermissionsOrByToken(ctx, tokenValue, permissions, authType...)
}

// ============================================================================
// Role Management - 角色管理
// ============================================================================

// AddRoles adds roles.
// AddRoles 添加角色。
func AddRoles(ctx context.Context, loginID string, roles []string, authType ...string) error {
	return dtoken.AddRoles(ctx, loginID, roles, authType...)
}

// AddRolesByToken adds roles by token.
// AddRolesByToken 根据 Token 添加角色。
func AddRolesByToken(ctx context.Context, tokenValue string, roles []string, authType ...string) error {
	return dtoken.AddRolesByToken(ctx, tokenValue, roles, authType...)
}

// RemoveRoles removes roles.
// RemoveRoles 删除角色。
func RemoveRoles(ctx context.Context, loginID string, roles []string, authType ...string) error {
	return dtoken.RemoveRoles(ctx, loginID, roles, authType...)
}

// RemoveRolesByToken removes roles by token.
// RemoveRolesByToken 根据 Token 删除角色。
func RemoveRolesByToken(ctx context.Context, tokenValue string, roles []string, authType ...string) error {
	return dtoken.RemoveRolesByToken(ctx, tokenValue, roles, authType...)
}

// GetRoles gets role list.
// GetRoles 获取角色列表。
func GetRoles(ctx context.Context, loginID string, authType ...string) ([]string, error) {
	return dtoken.GetRoles(ctx, loginID, authType...)
}

// GetRolesByToken gets role list by token.
// GetRolesByToken 根据 Token 获取角色列表。
func GetRolesByToken(ctx context.Context, tokenValue string, authType ...string) ([]string, error) {
	return dtoken.GetRolesByToken(ctx, tokenValue, authType...)
}

// HasRole checks if has specified role.
// HasRole 检查是否拥有指定角色。
func HasRole(ctx context.Context, loginID string, role string, authType ...string) bool {
	return dtoken.HasRole(ctx, loginID, role, authType...)
}

// HasRoleByToken checks if has specified role by token.
// HasRoleByToken 根据 Token 检查是否拥有指定角色。
func HasRoleByToken(ctx context.Context, tokenValue string, role string, authType ...string) bool {
	return dtoken.HasRoleByToken(ctx, tokenValue, role, authType...)
}

// HasRolesAnd checks if has all roles (AND logic).
// HasRolesAnd 检查是否拥有所有角色（AND 逻辑）。
func HasRolesAnd(ctx context.Context, loginID string, roles []string, authType ...string) bool {
	return dtoken.HasRolesAnd(ctx, loginID, roles, authType...)
}

// HasRolesAndByToken checks if has all roles by token (AND logic).
// HasRolesAndByToken 根据 Token 检查是否拥有所有角色（AND 逻辑）。
func HasRolesAndByToken(ctx context.Context, tokenValue string, roles []string, authType ...string) bool {
	return dtoken.HasRolesAndByToken(ctx, tokenValue, roles, authType...)
}

// HasRolesOr checks if has any role (OR logic).
// HasRolesOr 检查是否拥有任一角色（OR 逻辑）。
func HasRolesOr(ctx context.Context, loginID string, roles []string, authType ...string) bool {
	return dtoken.HasRolesOr(ctx, loginID, roles, authType...)
}

// HasRolesOrByToken checks if has any role by token (OR logic).
// HasRolesOrByToken 根据 Token 检查是否拥有任一角色（OR 逻辑）。
func HasRolesOrByToken(ctx context.Context, tokenValue string, roles []string, authType ...string) bool {
	return dtoken.HasRolesOrByToken(ctx, tokenValue, roles, authType...)
}

// ============================================================================
// Nonce Management - Nonce 管理
// ============================================================================

// GenerateNonce generates a new nonce.
// GenerateNonce 生成新的 nonce。
func GenerateNonce(ctx context.Context, authType ...string) (string, error) {
	return dtoken.GenerateNonce(ctx, authType...)
}

// VerifyNonce verifies and consumes a nonce (one-time use).
// VerifyNonce 验证并消费 nonce（一次性使用）。
func VerifyNonce(ctx context.Context, nonce string, authType ...string) bool {
	return dtoken.VerifyNonce(ctx, nonce, authType...)
}

// VerifyAndConsumeNonce verifies and consumes a nonce, returns error if invalid.
// VerifyAndConsumeNonce 验证并消费 nonce，无效时返回错误。
func VerifyAndConsumeNonce(ctx context.Context, nonce string, authType ...string) error {
	return dtoken.VerifyAndConsumeNonce(ctx, nonce, authType...)
}

// IsNonceValid checks if a nonce is valid without consuming it.
// IsNonceValid 检查 nonce 是否有效（不消费）。
func IsNonceValid(ctx context.Context, nonce string, authType ...string) bool {
	return dtoken.IsNonceValid(ctx, nonce, authType...)
}

// ============================================================================
// OAuth2 Management - OAuth2 管理
// ============================================================================

// RegisterOAuth2Client registers an OAuth2 client.
// RegisterOAuth2Client 注册 OAuth2 客户端。
func RegisterOAuth2Client(client *oauth2.Client, authType ...string) error {
	return dtoken.RegisterOAuth2Client(client, authType...)
}

// UnregisterOAuth2Client unregisters an OAuth2 client.
// UnregisterOAuth2Client 注销 OAuth2 客户端。
func UnregisterOAuth2Client(clientID string, authType ...string) error {
	return dtoken.UnregisterOAuth2Client(clientID, authType...)
}

// GetOAuth2Client gets an OAuth2 client by ID.
// GetOAuth2Client 根据 ID 获取 OAuth2 客户端。
func GetOAuth2Client(clientID string, authType ...string) (*oauth2.Client, error) {
	return dtoken.GetOAuth2Client(clientID, authType...)
}

// OAuth2Token unified token endpoint that dispatches to appropriate handler based on grant type.
// OAuth2Token 统一的令牌端点，根据授权类型分发到相应的处理逻辑。
func OAuth2Token(ctx context.Context, req *oauth2.TokenRequest, validateUser oauth2.UserValidator, authType ...string) (*oauth2.AccessToken, error) {
	return dtoken.OAuth2Token(ctx, req, validateUser, authType...)
}

// GenerateOAuth2AuthorizationCode generates an authorization code.
// GenerateOAuth2AuthorizationCode 生成授权码。
func GenerateOAuth2AuthorizationCode(ctx context.Context, clientID, userID, redirectURI string, scopes []string, authType ...string) (*oauth2.AuthorizationCode, error) {
	return dtoken.GenerateOAuth2AuthorizationCode(ctx, clientID, userID, redirectURI, scopes, authType...)
}

// ExchangeOAuth2CodeForToken exchanges authorization code for access token.
// ExchangeOAuth2CodeForToken 用授权码换取访问令牌。
func ExchangeOAuth2CodeForToken(ctx context.Context, code, clientID, clientSecret, redirectURI string, authType ...string) (*oauth2.AccessToken, error) {
	return dtoken.ExchangeOAuth2CodeForToken(ctx, code, clientID, clientSecret, redirectURI, authType...)
}

// OAuth2ClientCredentialsToken gets access token using client credentials grant.
// OAuth2ClientCredentialsToken 使用客户端凭证模式获取访问令牌。
func OAuth2ClientCredentialsToken(ctx context.Context, clientID, clientSecret string, scopes []string, authType ...string) (*oauth2.AccessToken, error) {
	return dtoken.OAuth2ClientCredentialsToken(ctx, clientID, clientSecret, scopes, authType...)
}

// OAuth2PasswordGrantToken gets access token using resource owner password credentials grant.
// OAuth2PasswordGrantToken 使用密码模式获取访问令牌。
func OAuth2PasswordGrantToken(ctx context.Context, clientID, clientSecret, username, password string, scopes []string, validateUser oauth2.UserValidator, authType ...string) (*oauth2.AccessToken, error) {
	return dtoken.OAuth2PasswordGrantToken(ctx, clientID, clientSecret, username, password, scopes, validateUser, authType...)
}

// RefreshOAuth2AccessToken refreshes access token using refresh token.
// RefreshOAuth2AccessToken 使用刷新令牌刷新访问令牌。
func RefreshOAuth2AccessToken(ctx context.Context, clientID, refreshToken, clientSecret string, authType ...string) (*oauth2.AccessToken, error) {
	return dtoken.RefreshOAuth2AccessToken(ctx, clientID, refreshToken, clientSecret, authType...)
}

// ValidateOAuth2AccessToken validates an access token.
// ValidateOAuth2AccessToken 验证访问令牌。
func ValidateOAuth2AccessToken(ctx context.Context, accessToken string, authType ...string) bool {
	return dtoken.ValidateOAuth2AccessToken(ctx, accessToken, authType...)
}

// ValidateOAuth2AccessTokenAndGetInfo validates access token and gets info.
// ValidateOAuth2AccessTokenAndGetInfo 验证访问令牌并获取信息。
func ValidateOAuth2AccessTokenAndGetInfo(ctx context.Context, accessToken string, authType ...string) (*oauth2.AccessToken, error) {
	return dtoken.ValidateOAuth2AccessTokenAndGetInfo(ctx, accessToken, authType...)
}

// RevokeOAuth2Token revokes an access token and its refresh token.
// RevokeOAuth2Token 撤销访问令牌及其刷新令牌。
func RevokeOAuth2Token(ctx context.Context, accessToken string, authType ...string) error {
	return dtoken.RevokeOAuth2Token(ctx, accessToken, authType...)
}
