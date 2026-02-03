// @Author daixk 2026/1/22 16:54:00
package dtoken

import (
	"context"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/manager"
	"github.com/Zany2/dtoken-go/core/oauth2"
	"strings"
	"sync"
	"time"
)

var (
	globalManagerMap sync.Map
)

// ============================================================================
// Manager Lifecycle - 管理器生命周期
// ============================================================================

// SetManager stores a manager in the global map using the specified authentication type.
// SetManager 使用指定的认证类型将管理器存储在全局 map 中。
func SetManager(mgr *manager.Manager) {
	validAutoType := getAutoType(mgr.GetConfig().AuthType)
	globalManagerMap.Store(validAutoType, mgr)
}

// GetManager retrieves a manager from the global map by authentication type.
// GetManager 根据认证类型从全局 map 中获取管理器。
func GetManager(authType ...string) (*manager.Manager, error) {
	validAutoType := getAutoType(authType...)
	return loadManager(validAutoType)
}

// DeleteManager deletes the manager for the specified authentication type and releases resources.
// DeleteManager 删除指定认证类型的管理器并释放资源。
func DeleteManager(authType ...string) error {
	validAutoType := getAutoType(authType...)
	mgr, err := loadManager(validAutoType)
	if err != nil {
		return err
	}
	mgr.CloseManager()
	globalManagerMap.Delete(validAutoType)
	return nil
}

// DeleteAllManager closes and deletes all managers in the global map.
// DeleteAllManager 关闭并删除全局 map 中的所有管理器。
func DeleteAllManager() {
	globalManagerMap.Range(func(key, value interface{}) bool {
		if mgr, ok := value.(*manager.Manager); ok {
			mgr.CloseManager()
		}
		return true
	})
	globalManagerMap = sync.Map{}
}

// ============================================================================
// Login & Authentication - 登录与认证
// ============================================================================

// Login performs user login and returns a token.
// Login 执行用户登录并返回 token。
// params: [0]=device, [1]=deviceId, [2]=authType (all optional)
func Login(ctx context.Context, loginID string, params ...string) (string, error) {
	device, deviceId, authType := parseDeviceAndAuthType(params...)
	mgr, err := GetManager(authType)
	if err != nil {
		return "", err
	}
	return mgr.Login(ctx, loginID, device, deviceId)
}

// LoginByToken performs login renewal based on an existing token.
// LoginByToken 根据 Token 续期登录。
func LoginByToken(ctx context.Context, tokenValue string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.LoginByToken(ctx, tokenValue)
}

// Logout logs out a user by token.
// Logout 根据 Token 登出用户。
func Logout(ctx context.Context, tokenValue string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.Logout(ctx, tokenValue)
}

// LogoutByDeviceAndDeviceId logs out a user by device type and device ID.
// LogoutByDeviceAndDeviceId 根据设备类型和设备ID登出用户。
// params: [0]=device, [1]=deviceId, [2]=authType (all optional)
func LogoutByDeviceAndDeviceId(ctx context.Context, loginID string, params ...string) error {
	device, deviceId, authType := parseDeviceAndAuthType(params...)
	mgr, err := GetManager(authType)
	if err != nil {
		return err
	}
	return mgr.LogoutByDeviceAndDeviceId(ctx, loginID, device, deviceId)
}

// LogoutByDevice logs out all terminals of a specific device type.
// LogoutByDevice 根据设备类型登出所有该设备的终端。
func LogoutByDevice(ctx context.Context, loginID string, device string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.LogoutByDevice(ctx, loginID, device)
}

// LogoutByLoginID logs out all terminals for the specified loginID.
// LogoutByLoginID 登出指定 loginID 的所有终端。
func LogoutByLoginID(ctx context.Context, loginID string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.LogoutByLoginID(ctx, loginID)
}

// ============================================================================
// Online Status Management - 在线状态管理
// ============================================================================

// Kickout kicks out a user by token.
// Kickout 根据 Token 踢人下线。
func Kickout(ctx context.Context, tokenValue string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.Kickout(ctx, tokenValue)
}

// Replace replaces a user session by token.
// Replace 根据 Token 顶人下线。
func Replace(ctx context.Context, tokenValue string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.Replace(ctx, tokenValue)
}

// KickoutByDeviceAndDeviceId kicks out a user by device type and device ID.
// KickoutByDeviceAndDeviceId 根据设备类型和设备ID踢人下线。
// params: [0]=device, [1]=deviceId, [2]=authType (all optional)
func KickoutByDeviceAndDeviceId(ctx context.Context, loginID string, params ...string) error {
	device, deviceId, authType := parseDeviceAndAuthType(params...)
	mgr, err := GetManager(authType)
	if err != nil {
		return err
	}
	return mgr.KickoutByDeviceAndDeviceId(ctx, loginID, device, deviceId)
}

// KickoutByDevice kicks out all terminals of a specific device type.
// KickoutByDevice 根据设备类型踢人下线（踢掉该设备类型的所有终端）。
func KickoutByDevice(ctx context.Context, loginID string, device string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.KickoutByDevice(ctx, loginID, device)
}

// KickoutByLoginID kicks out all terminals for the specified loginID.
// KickoutByLoginID 踢出指定 loginID 的所有终端。
func KickoutByLoginID(ctx context.Context, loginID string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.KickoutByLoginID(ctx, loginID)
}

// ReplaceByDeviceAndDeviceId replaces a user session by device type and device ID.
// ReplaceByDeviceAndDeviceId 根据设备类型和设备ID顶人下线。
// params: [0]=device, [1]=deviceId, [2]=authType (all optional)
func ReplaceByDeviceAndDeviceId(ctx context.Context, loginID string, params ...string) error {
	device, deviceId, authType := parseDeviceAndAuthType(params...)
	mgr, err := GetManager(authType)
	if err != nil {
		return err
	}
	return mgr.ReplaceByDeviceAndDeviceId(ctx, loginID, device, deviceId)
}

// ReplaceByDevice replaces all terminals of a specific device type.
// ReplaceByDevice 根据设备类型顶人下线（顶掉该设备类型的所有终端）。
func ReplaceByDevice(ctx context.Context, loginID string, device string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.ReplaceByDevice(ctx, loginID, device)
}

// ReplaceByLoginID replaces all terminals for the specified loginID.
// ReplaceByLoginID 顶替指定 loginID 的所有终端。
func ReplaceByLoginID(ctx context.Context, loginID string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.ReplaceByLoginID(ctx, loginID)
}

// ============================================================================
// Token Validation - Token 验证
// ============================================================================

// IsLogin checks if a user is logged in.
// IsLogin 检查用户是否登录。
func IsLogin(ctx context.Context, tokenValue string, authType ...string) bool {
	mgr, err := GetManager(authType...)
	if err != nil {
		return false
	}
	return mgr.IsLogin(ctx, tokenValue)
}

// CheckLogin checks if a user is logged in and returns an error if not.
// CheckLogin 检查用户是否登录，如果未登录则返回错误。
func CheckLogin(ctx context.Context, tokenValue string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.CheckLogin(ctx, tokenValue)
}

// ============================================================================
// Token Information - Token 信息
// ============================================================================

// GetLoginID retrieves the login ID from a token.
// GetLoginID 根据 Token 获取登录 ID。
func GetLoginID(ctx context.Context, tokenValue string, authType ...string) (string, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return "", err
	}
	return mgr.GetLoginID(ctx, tokenValue)
}

// GetTokenInfo retrieves token information.
// GetTokenInfo 根据 Token 获取 TokenInfo 信息。
func GetTokenInfo(ctx context.Context, tokenValue string, authType ...string) (*manager.TokenInfo, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.GetTokenInfo(ctx, tokenValue)
}

// GetDevice retrieves the device type from a token.
// GetDevice 根据 Token 获取设备类型。
func GetDevice(ctx context.Context, tokenValue string, authType ...string) (string, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return "", err
	}
	return mgr.GetDevice(ctx, tokenValue)
}

// GetDeviceId retrieves the device ID from a token.
// GetDeviceId 根据 Token 获取设备 ID。
func GetDeviceId(ctx context.Context, tokenValue string, authType ...string) (string, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return "", err
	}
	return mgr.GetDeviceId(ctx, tokenValue)
}

// GetTokenCreateTime retrieves the creation time of a token.
// GetTokenCreateTime 根据 Token 获取创建时间。
func GetTokenCreateTime(ctx context.Context, tokenValue string, authType ...string) (int64, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return 0, err
	}
	return mgr.GetTokenCreateTime(ctx, tokenValue)
}

// GetTokenTTL retrieves the remaining TTL (time to live) of a token in seconds.
// GetTokenTTL 根据 Token 获取剩余有效时间（秒）。
func GetTokenTTL(ctx context.Context, tokenValue string, authType ...string) (int64, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return 0, err
	}
	return mgr.GetTokenTTL(ctx, tokenValue)
}

// GetOnlineTerminalCount retrieves the total number of online terminals for a login ID.
// GetOnlineTerminalCount 获取指定登录 ID 的在线终端总数。
func GetOnlineTerminalCount(ctx context.Context, loginID string, authType ...string) (int, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return 0, err
	}
	return mgr.GetOnlineTerminalCount(ctx, loginID)
}

// GetOnlineTerminalCountByDevice retrieves the number of online terminals for a specific device type.
// GetOnlineTerminalCountByDevice 获取指定登录 ID 和设备类型的在线终端数。
func GetOnlineTerminalCountByDevice(ctx context.Context, loginID string, device string, authType ...string) (int, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return 0, err
	}
	return mgr.GetOnlineTerminalCountByDevice(ctx, loginID, device)
}

// GetOnlineTerminalCountByDeviceAndDeviceId retrieves the number of online terminals for a specific device type and device ID.
// GetOnlineTerminalCountByDeviceAndDeviceId 获取指定登录 ID、设备类型和设备ID的在线终端数。
func GetOnlineTerminalCountByDeviceAndDeviceId(ctx context.Context, loginID string, device string, deviceId string, authType ...string) (int, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return 0, err
	}
	return mgr.GetOnlineTerminalCountByDeviceAndDeviceId(ctx, loginID, device, deviceId)
}

// ============================================================================
// Account Disable Management - 账号封禁管理
// ============================================================================

// Disable disables an account for a specified duration.
// Disable 封禁账号指定时长。
func Disable(ctx context.Context, loginID string, duration time.Duration, reason string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.Disable(ctx, loginID, duration, reason)
}

// Untie removes the disable status from an account.
// Untie 解封账号。
func Untie(ctx context.Context, loginID string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.Untie(ctx, loginID)
}

// IsDisable checks if an account is disabled.
// IsDisable 检查账号是否被封禁。
func IsDisable(ctx context.Context, loginID string, authType ...string) bool {
	mgr, err := GetManager(authType...)
	if err != nil {
		return false
	}
	return mgr.IsDisable(ctx, loginID)
}

// GetDisableInfo retrieves disable information for an account.
// GetDisableInfo 获取账号的封禁信息。
func GetDisableInfo(ctx context.Context, loginID string, authType ...string) (*manager.DisableInfo, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.GetDisableInfo(ctx, loginID)
}

// GetDisableTTL retrieves the remaining disable time for an account in seconds.
// GetDisableTTL 获取账号剩余封禁时间（秒）。
func GetDisableTTL(ctx context.Context, loginID string, authType ...string) (int64, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return 0, err
	}
	return mgr.GetDisableTTL(ctx, loginID)
}

// ============================================================================
// Session Management - 会话管理
// ============================================================================

// GetSession retrieves session information for a login ID.
// GetSession 获取指定登录 ID 的会话信息。
func GetSession(ctx context.Context, loginID string, authType ...string) (*manager.Session, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.GetSession(ctx, loginID)
}

// GetSessionByToken retrieves session information by token.
// GetSessionByToken 通过 Token 值获取会话信息。
func GetSessionByToken(ctx context.Context, tokenValue string, authType ...string) (*manager.Session, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.GetSessionByToken(ctx, tokenValue)
}

// GetTokenValueListByLoginID retrieves all tokens for a login ID.
// GetTokenValueListByLoginID 获取指定登录 ID 的所有 Token。
func GetTokenValueListByLoginID(ctx context.Context, loginID string, checkAlive bool, authType ...string) ([]string, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.GetTokenValueListByLoginID(ctx, loginID, checkAlive)
}

// GetTokenValueListByDeviceAndDeviceId retrieves all tokens for a specific device type and device ID.
// GetTokenValueListByDeviceAndDeviceId 获取指定设备类型和设备 ID 的所有 Token。
func GetTokenValueListByDeviceAndDeviceId(ctx context.Context, loginID string, device string, deviceId string, checkAlive bool, authType ...string) ([]string, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.GetTokenValueListByDeviceAndDeviceId(ctx, loginID, device, deviceId, checkAlive)
}

// GetTokenValueListByDevice retrieves all tokens for a specific device type.
// GetTokenValueListByDevice 获取指定设备类型的所有 Token。
func GetTokenValueListByDevice(ctx context.Context, loginID string, device string, checkAlive bool, authType ...string) ([]string, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.GetTokenValueListByDevice(ctx, loginID, device, checkAlive)
}

// ============================================================================
// Permission Management - 权限管理
// ============================================================================

// AddPermissions adds permissions to a user.
// AddPermissions 为用户添加权限。
func AddPermissions(ctx context.Context, loginID string, permissions []string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.AddPermissions(ctx, loginID, permissions)
}

// AddPermissionsByToken adds permissions to a user by token.
// AddPermissionsByToken 根据 Token 为用户添加权限。
func AddPermissionsByToken(ctx context.Context, tokenValue string, permissions []string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.AddPermissionsByToken(ctx, tokenValue, permissions)
}

// RemovePermissions removes permissions from a user.
// RemovePermissions 删除用户的指定权限。
func RemovePermissions(ctx context.Context, loginID string, permissions []string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.RemovePermissions(ctx, loginID, permissions)
}

// RemovePermissionsByToken removes permissions from a user by token.
// RemovePermissionsByToken 根据 Token 删除用户的指定权限。
func RemovePermissionsByToken(ctx context.Context, tokenValue string, permissions []string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.RemovePermissionsByToken(ctx, tokenValue, permissions)
}

// GetPermissions retrieves the permission list for a user.
// GetPermissions 获取用户的权限列表。
func GetPermissions(ctx context.Context, loginID string, authType ...string) ([]string, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.GetPermissions(ctx, loginID)
}

// GetPermissionsByToken retrieves the permission list by token.
// GetPermissionsByToken 根据 Token 获取权限列表。
func GetPermissionsByToken(ctx context.Context, tokenValue string, authType ...string) ([]string, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.GetPermissionsByToken(ctx, tokenValue)
}

// HasPermission checks if a user has a specific permission.
// HasPermission 检查用户是否拥有指定权限。
func HasPermission(ctx context.Context, loginID string, permission string, authType ...string) bool {
	mgr, err := GetManager(authType...)
	if err != nil {
		return false
	}
	return mgr.HasPermission(ctx, loginID, permission)
}

// HasPermissionByToken checks if a user has a specific permission by token.
// HasPermissionByToken 根据 Token 检查用户是否拥有指定权限。
func HasPermissionByToken(ctx context.Context, tokenValue string, permission string, authType ...string) bool {
	mgr, err := GetManager(authType...)
	if err != nil {
		return false
	}
	return mgr.HasPermissionByToken(ctx, tokenValue, permission)
}

// HasPermissionsAnd checks if a user has all specified permissions (AND logic).
// HasPermissionsAnd 检查用户是否拥有所有指定权限（AND 逻辑）。
func HasPermissionsAnd(ctx context.Context, loginID string, permissions []string, authType ...string) bool {
	mgr, err := GetManager(authType...)
	if err != nil {
		return false
	}
	return mgr.HasPermissionsAnd(ctx, loginID, permissions)
}

// HasPermissionsAndByToken checks if a user has all specified permissions by token (AND logic).
// HasPermissionsAndByToken 根据 Token 检查用户是否拥有所有指定权限（AND 逻辑）。
func HasPermissionsAndByToken(ctx context.Context, tokenValue string, permissions []string, authType ...string) bool {
	mgr, err := GetManager(authType...)
	if err != nil {
		return false
	}
	return mgr.HasPermissionsAndByToken(ctx, tokenValue, permissions)
}

// HasPermissionsOr checks if a user has any of the specified permissions (OR logic).
// HasPermissionsOr 检查用户是否拥有任一指定权限（OR 逻辑）。
func HasPermissionsOr(ctx context.Context, loginID string, permissions []string, authType ...string) bool {
	mgr, err := GetManager(authType...)
	if err != nil {
		return false
	}
	return mgr.HasPermissionsOr(ctx, loginID, permissions)
}

// HasPermissionsOrByToken checks if a user has any of the specified permissions by token (OR logic).
// HasPermissionsOrByToken 根据 Token 检查用户是否拥有任一指定权限（OR 逻辑）。
func HasPermissionsOrByToken(ctx context.Context, tokenValue string, permissions []string, authType ...string) bool {
	mgr, err := GetManager(authType...)
	if err != nil {
		return false
	}
	return mgr.HasPermissionsOrByToken(ctx, tokenValue, permissions)
}

// ============================================================================
// Role Management - 角色管理
// ============================================================================

// AddRoles adds roles to a user.
// AddRoles 为用户添加角色。
func AddRoles(ctx context.Context, loginID string, roles []string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.AddRoles(ctx, loginID, roles)
}

// AddRolesByToken adds roles to a user by token.
// AddRolesByToken 根据 Token 为用户添加角色。
func AddRolesByToken(ctx context.Context, tokenValue string, roles []string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.AddRolesByToken(ctx, tokenValue, roles)
}

// RemoveRoles removes roles from a user.
// RemoveRoles 删除用户的指定角色。
func RemoveRoles(ctx context.Context, loginID string, roles []string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.RemoveRoles(ctx, loginID, roles)
}

// RemoveRolesByToken removes roles from a user by token.
// RemoveRolesByToken 根据 Token 删除用户的指定角色。
func RemoveRolesByToken(ctx context.Context, tokenValue string, roles []string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.RemoveRolesByToken(ctx, tokenValue, roles)
}

// GetRoles retrieves the role list for a user.
// GetRoles 获取用户的角色列表。
func GetRoles(ctx context.Context, loginID string, authType ...string) ([]string, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.GetRoles(ctx, loginID)
}

// GetRolesByToken retrieves the role list by token.
// GetRolesByToken 根据 Token 获取角色列表。
func GetRolesByToken(ctx context.Context, tokenValue string, authType ...string) ([]string, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.GetRolesByToken(ctx, tokenValue)
}

// HasRole checks if a user has a specific role.
// HasRole 检查用户是否拥有指定角色。
func HasRole(ctx context.Context, loginID string, role string, authType ...string) bool {
	mgr, err := GetManager(authType...)
	if err != nil {
		return false
	}
	return mgr.HasRole(ctx, loginID, role)
}

// HasRoleByToken checks if a user has a specific role by token.
// HasRoleByToken 根据 Token 检查用户是否拥有指定角色。
func HasRoleByToken(ctx context.Context, tokenValue string, role string, authType ...string) bool {
	mgr, err := GetManager(authType...)
	if err != nil {
		return false
	}
	return mgr.HasRoleByToken(ctx, tokenValue, role)
}

// HasRolesAnd checks if a user has all specified roles (AND logic).
// HasRolesAnd 检查用户是否拥有所有指定角色（AND 逻辑）。
func HasRolesAnd(ctx context.Context, loginID string, roles []string, authType ...string) bool {
	mgr, err := GetManager(authType...)
	if err != nil {
		return false
	}
	return mgr.HasRolesAnd(ctx, loginID, roles)
}

// HasRolesAndByToken checks if a user has all specified roles by token (AND logic).
// HasRolesAndByToken 根据 Token 检查用户是否拥有所有指定角色（AND 逻辑）。
func HasRolesAndByToken(ctx context.Context, tokenValue string, roles []string, authType ...string) bool {
	mgr, err := GetManager(authType...)
	if err != nil {
		return false
	}
	return mgr.HasRolesAndByToken(ctx, tokenValue, roles)
}

// HasRolesOr checks if a user has any of the specified roles (OR logic).
// HasRolesOr 检查用户是否拥有任一指定角色（OR 逻辑）。
func HasRolesOr(ctx context.Context, loginID string, roles []string, authType ...string) bool {
	mgr, err := GetManager(authType...)
	if err != nil {
		return false
	}
	return mgr.HasRolesOr(ctx, loginID, roles)
}

// HasRolesOrByToken checks if a user has any of the specified roles by token (OR logic).
// HasRolesOrByToken 根据 Token 检查用户是否拥有任一指定角色（OR 逻辑）。
func HasRolesOrByToken(ctx context.Context, tokenValue string, roles []string, authType ...string) bool {
	mgr, err := GetManager(authType...)
	if err != nil {
		return false
	}
	return mgr.HasRolesOrByToken(ctx, tokenValue, roles)
}

// ============================================================================
// Nonce Management - Nonce 管理
// ============================================================================

// GenerateNonce generates a new nonce.
// GenerateNonce 生成新的 nonce。
func GenerateNonce(ctx context.Context, authType ...string) (string, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return "", err
	}
	return mgr.GenerateNonce(ctx)
}

// VerifyNonce verifies and consumes a nonce (one-time use).
// VerifyNonce 验证并消费 nonce（一次性使用）。
func VerifyNonce(ctx context.Context, nonce string, authType ...string) bool {
	mgr, err := GetManager(authType...)
	if err != nil {
		return false
	}
	return mgr.VerifyNonce(ctx, nonce)
}

// VerifyAndConsumeNonce verifies and consumes a nonce, returns error if invalid.
// VerifyAndConsumeNonce 验证并消费 nonce，无效时返回错误。
func VerifyAndConsumeNonce(ctx context.Context, nonce string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.VerifyAndConsumeNonce(ctx, nonce)
}

// IsNonceValid checks if a nonce is valid without consuming it.
// IsNonceValid 检查 nonce 是否有效（不消费）。
func IsNonceValid(ctx context.Context, nonce string, authType ...string) bool {
	mgr, err := GetManager(authType...)
	if err != nil {
		return false
	}
	return mgr.IsNonceValid(ctx, nonce)
}

// ============================================================================
// OAuth2 Management - OAuth2 管理
// ============================================================================

// RegisterOAuth2Client registers an OAuth2 client.
// RegisterOAuth2Client 注册 OAuth2 客户端。
func RegisterOAuth2Client(client *oauth2.Client, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.RegisterOAuth2Client(client)
}

// UnregisterOAuth2Client unregisters an OAuth2 client.
// UnregisterOAuth2Client 注销 OAuth2 客户端。
func UnregisterOAuth2Client(clientID string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	mgr.UnregisterOAuth2Client(clientID)
	return nil
}

// GetOAuth2Client gets an OAuth2 client by ID.
// GetOAuth2Client 根据 ID 获取 OAuth2 客户端。
func GetOAuth2Client(clientID string, authType ...string) (*oauth2.Client, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.GetOAuth2Client(clientID)
}

// OAuth2Token unified token endpoint that dispatches to appropriate handler based on grant type.
// OAuth2Token 统一的令牌端点，根据授权类型分发到相应的处理逻辑。
func OAuth2Token(ctx context.Context, req *oauth2.TokenRequest, validateUser oauth2.UserValidator, authType ...string) (*oauth2.AccessToken, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.OAuth2Token(ctx, req, validateUser)
}

// GenerateOAuth2AuthorizationCode generates an authorization code.
// GenerateOAuth2AuthorizationCode 生成授权码。
func GenerateOAuth2AuthorizationCode(ctx context.Context, clientID, userID, redirectURI string, scopes []string, authType ...string) (*oauth2.AuthorizationCode, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.GenerateOAuth2AuthorizationCode(ctx, clientID, userID, redirectURI, scopes)
}

// ExchangeOAuth2CodeForToken exchanges authorization code for access token.
// ExchangeOAuth2CodeForToken 用授权码换取访问令牌。
func ExchangeOAuth2CodeForToken(ctx context.Context, code, clientID, clientSecret, redirectURI string, authType ...string) (*oauth2.AccessToken, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.ExchangeOAuth2CodeForToken(ctx, code, clientID, clientSecret, redirectURI)
}

// OAuth2ClientCredentialsToken gets access token using client credentials grant.
// OAuth2ClientCredentialsToken 使用客户端凭证模式获取访问令牌。
func OAuth2ClientCredentialsToken(ctx context.Context, clientID, clientSecret string, scopes []string, authType ...string) (*oauth2.AccessToken, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.OAuth2ClientCredentialsToken(ctx, clientID, clientSecret, scopes)
}

// OAuth2PasswordGrantToken gets access token using resource owner password credentials grant.
// OAuth2PasswordGrantToken 使用密码模式获取访问令牌。
func OAuth2PasswordGrantToken(ctx context.Context, clientID, clientSecret, username, password string, scopes []string, validateUser oauth2.UserValidator, authType ...string) (*oauth2.AccessToken, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.OAuth2PasswordGrantToken(ctx, clientID, clientSecret, username, password, scopes, validateUser)
}

// RefreshOAuth2AccessToken refreshes access token using refresh token.
// RefreshOAuth2AccessToken 使用刷新令牌刷新访问令牌。
func RefreshOAuth2AccessToken(ctx context.Context, clientID, refreshToken, clientSecret string, authType ...string) (*oauth2.AccessToken, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.RefreshOAuth2AccessToken(ctx, clientID, refreshToken, clientSecret)
}

// ValidateOAuth2AccessToken validates an access token.
// ValidateOAuth2AccessToken 验证访问令牌。
func ValidateOAuth2AccessToken(ctx context.Context, accessToken string, authType ...string) bool {
	mgr, err := GetManager(authType...)
	if err != nil {
		return false
	}
	return mgr.ValidateOAuth2AccessToken(ctx, accessToken)
}

// ValidateOAuth2AccessTokenAndGetInfo validates access token and gets info.
// ValidateOAuth2AccessTokenAndGetInfo 验证访问令牌并获取信息。
func ValidateOAuth2AccessTokenAndGetInfo(ctx context.Context, accessToken string, authType ...string) (*oauth2.AccessToken, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.ValidateOAuth2AccessTokenAndGetInfo(ctx, accessToken)
}

// RevokeOAuth2Token revokes an access token and its refresh token.
// RevokeOAuth2Token 撤销访问令牌及其刷新令牌。
func RevokeOAuth2Token(ctx context.Context, accessToken string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.RevokeOAuth2Token(ctx, accessToken)
}

// ============================================================================
// Internal Helper Methods - 内部辅助方法
// ============================================================================

// getAutoType retrieves the valid authentication type prefix, ensuring it ends with a colon. Returns default if not provided.
// getAutoType 获取有效的认证类型前缀，确保以冒号结尾。若未提供则返回默认值。
func getAutoType(authType ...string) string {
	if len(authType) > 0 && strings.TrimSpace(authType[0]) != "" {
		trimmed := strings.TrimSpace(authType[0])
		if !strings.HasSuffix(trimmed, ":") {
			trimmed += ":"
		}
		return trimmed
	}
	return config.DefaultAuthType
}

// loadManager loads the manager for the specified authentication type from the global map.
// loadManager 从全局 map 中加载指定认证类型的管理器。
func loadManager(authType string) (*manager.Manager, error) {
	value, ok := globalManagerMap.Load(authType)
	if !ok {
		return nil, derror.ErrManagerNotFound
	}
	mgr, ok := value.(*manager.Manager)
	if !ok {
		return nil, derror.ErrManagerInvalidType
	}
	return mgr, nil
}

// parseDeviceAndAuthType parses optional parameters: [0]=device, [1]=deviceId, [2]=authType
// parseDeviceAndAuthType 解析可选参数：[0]=device, [1]=deviceId, [2]=authType
func parseDeviceAndAuthType(params ...string) (device, deviceId, authType string) {
	if len(params) > 0 {
		device = params[0]
	}
	if len(params) > 1 {
		deviceId = params[1]
	}
	if len(params) > 2 {
		authType = params[2]
	}
	return
}
