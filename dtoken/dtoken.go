// @Author daixk 2026/1/22 16:54:00
package dtoken

import (
	"context"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/manager"
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

// ReplaceByDeviceAndDeviceId replaces a user session by device type and device ID.
// ReplaceByDeviceAndDeviceId 根据设备类型和设备ID顶人下线。
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
