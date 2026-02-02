package context

import (
	"context"
	"strings"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/manager"
)

const (
	bearerPrefix = "Bearer "
	authHeader   = "Authorization"
)

// DTokenContext 表示当前请求的 Sa-Token 上下文
type DTokenContext struct {
	reqCtx  adapter.RequestContext
	manager *manager.Manager
}

// NewContext 创建新的 DTokenContext 上下文
func NewContext(reqCtx adapter.RequestContext, mgr *manager.Manager) *DTokenContext {
	return &DTokenContext{
		reqCtx:  reqCtx,
		manager: mgr,
	}
}

// GetTokenValue 从当前请求中获取 Token 值，按 Header → Cookie → Query 顺序尝试
func (c *DTokenContext) GetTokenValue() string {
	cfg := c.manager.GetConfig()

	// 1. 尝试从 Header 获取
	if cfg.IsReadHeader {
		// 优先从配置的 Token 名称对应的 Header 中读取
		if token := strings.TrimSpace(c.reqCtx.GetHeader(cfg.TokenName)); token != "" {
			return token
		}

		// 其次尝试从 Authorization 头中提取 Bearer Token
		if auth := c.reqCtx.GetHeader(authHeader); auth != "" {
			if token := extractBearerToken(auth); token != "" {
				return token
			}
		}
	}

	// 2. 尝试从 Cookie 获取
	if cfg.IsReadCookie {
		if token := strings.TrimSpace(c.reqCtx.GetCookie(cfg.TokenName)); token != "" {
			return token
		}
	}

	// 3. 尝试从 URL 查询参数获取
	if token := strings.TrimSpace(c.reqCtx.GetQuery(cfg.TokenName)); token != "" {
		return token
	}

	return ""
}

// GetRequestContext 获取原始请求上下文
func (c *DTokenContext) GetRequestContext() adapter.RequestContext {
	return c.reqCtx
}

// GetManager 获取关联的认证管理器
func (c *DTokenContext) GetManager() *manager.Manager {
	return c.manager
}

// extractBearerToken 从 Authorization 头中提取 Bearer Token（忽略大小写）
func extractBearerToken(auth string) string {
	auth = strings.TrimSpace(auth)
	if auth == "" {
		return ""
	}

	// 检查是否以 "Bearer " 开头（不区分大小写）
	if len(auth) > 7 && strings.EqualFold(auth[:7], bearerPrefix) {
		return strings.TrimSpace(auth[7:])
	}

	// 若不符合 Bearer 格式，直接返回原值（兼容自定义格式）
	return auth
}

// ============================================================================
// Convenience Methods - 便捷方法
// ============================================================================

// ============================================================================
// 1. Authentication Methods - 基础认证方法
// ============================================================================

// IsLogin checks if the current token is logged in
// IsLogin 检查当前 token 是否已登录
func (c *DTokenContext) IsLogin(ctx context.Context) bool {
	token := c.GetTokenValue()
	if token == "" {
		return false
	}
	return c.manager.IsLogin(ctx, token)
}

// CheckLogin checks if the current token is logged in and returns an error if not
// CheckLogin 检查当前 token 是否已登录，未登录则返回错误
func (c *DTokenContext) CheckLogin(ctx context.Context) error {
	token := c.GetTokenValue()
	if token == "" {
		return derror.ErrNotLogin
	}
	return c.manager.CheckLogin(ctx, token)
}

// GetLoginID gets the login ID associated with the current token
// GetLoginID 获取当前 token 关联的登录 ID
func (c *DTokenContext) GetLoginID(ctx context.Context) (string, error) {
	token := c.GetTokenValue()
	if token == "" {
		return "", derror.ErrNotLogin
	}
	return c.manager.GetLoginID(ctx, token)
}

// LoginByToken logs in using the current token
// LoginByToken 使用当前 token 登录
func (c *DTokenContext) LoginByToken(ctx context.Context) error {
	token := c.GetTokenValue()
	if token == "" {
		return derror.ErrNotLogin
	}
	return c.manager.LoginByToken(ctx, token)
}

// Logout logs out the current token
// Logout 登出当前 token
func (c *DTokenContext) Logout(ctx context.Context) error {
	token := c.GetTokenValue()
	if token == "" {
		return derror.ErrNotLogin
	}
	return c.manager.Logout(ctx, token)
}

// ============================================================================
// 2. Token Information Methods - Token信息方法
// ============================================================================

// GetTokenInfo gets the token information for the current token
// GetTokenInfo 获取当前 token 的信息
func (c *DTokenContext) GetTokenInfo(ctx context.Context) (*manager.TokenInfo, error) {
	token := c.GetTokenValue()
	if token == "" {
		return nil, derror.ErrNotLogin
	}
	return c.manager.GetTokenInfo(ctx, token)
}

// GetDevice gets the device type for the current token
// GetDevice 获取当前 token 的设备类型
func (c *DTokenContext) GetDevice(ctx context.Context) (string, error) {
	token := c.GetTokenValue()
	if token == "" {
		return "", derror.ErrNotLogin
	}
	return c.manager.GetDevice(ctx, token)
}

// GetDeviceId gets the device ID for the current token
// GetDeviceId 获取当前 token 的设备 ID
func (c *DTokenContext) GetDeviceId(ctx context.Context) (string, error) {
	token := c.GetTokenValue()
	if token == "" {
		return "", derror.ErrNotLogin
	}
	return c.manager.GetDeviceId(ctx, token)
}

// GetTokenCreateTime gets the creation time for the current token
// GetTokenCreateTime 获取当前 token 的创建时间
func (c *DTokenContext) GetTokenCreateTime(ctx context.Context) (int64, error) {
	token := c.GetTokenValue()
	if token == "" {
		return 0, derror.ErrNotLogin
	}
	return c.manager.GetTokenCreateTime(ctx, token)
}

// GetTokenTTL gets the remaining TTL for the current token
// GetTokenTTL 获取当前 token 的剩余有效期
func (c *DTokenContext) GetTokenTTL(ctx context.Context) (int64, error) {
	token := c.GetTokenValue()
	if token == "" {
		return 0, derror.ErrNotLogin
	}
	return c.manager.GetTokenTTL(ctx, token)
}

// ============================================================================
// 3. Token Management Methods - Token管理方法
// ============================================================================

// Kickout kicks out the current token
// Kickout 踢出当前 token
func (c *DTokenContext) Kickout(ctx context.Context) error {
	token := c.GetTokenValue()
	if token == "" {
		return derror.ErrNotLogin
	}
	return c.manager.Kickout(ctx, token)
}

// Replace replaces the current token
// Replace 顶替当前 token
func (c *DTokenContext) Replace(ctx context.Context) error {
	token := c.GetTokenValue()
	if token == "" {
		return derror.ErrNotLogin
	}
	return c.manager.Replace(ctx, token)
}

// ============================================================================
// 4. Device Management Methods - 设备管理方法
// ============================================================================

// LogoutByDevice logs out by device for the current user
// LogoutByDevice 按设备登出当前用户
func (c *DTokenContext) LogoutByDevice(ctx context.Context, device string) error {
	loginID, err := c.GetLoginID(ctx)
	if err != nil {
		return err
	}
	return c.manager.LogoutByDevice(ctx, loginID, device)
}

// LogoutByDeviceAndDeviceId logs out by device and device ID for the current user
// LogoutByDeviceAndDeviceId 按设备和设备ID登出当前用户
func (c *DTokenContext) LogoutByDeviceAndDeviceId(ctx context.Context, deviceAndDeviceId ...string) error {
	loginID, err := c.GetLoginID(ctx)
	if err != nil {
		return err
	}
	return c.manager.LogoutByDeviceAndDeviceId(ctx, loginID, deviceAndDeviceId...)
}

// KickoutByDevice kicks out by device for the current user
// KickoutByDevice 按设备踢出当前用户
func (c *DTokenContext) KickoutByDevice(ctx context.Context, device string) error {
	loginID, err := c.GetLoginID(ctx)
	if err != nil {
		return err
	}
	return c.manager.KickoutByDevice(ctx, loginID, device)
}

// KickoutByDeviceAndDeviceId kicks out by device and device ID for the current user
// KickoutByDeviceAndDeviceId 按设备和设备ID踢出当前用户
func (c *DTokenContext) KickoutByDeviceAndDeviceId(ctx context.Context, deviceAndDeviceId ...string) error {
	loginID, err := c.GetLoginID(ctx)
	if err != nil {
		return err
	}
	return c.manager.KickoutByDeviceAndDeviceId(ctx, loginID, deviceAndDeviceId...)
}

// ReplaceByDevice replaces by device for the current user
// ReplaceByDevice 按设备顶替当前用户
func (c *DTokenContext) ReplaceByDevice(ctx context.Context, device string) error {
	loginID, err := c.GetLoginID(ctx)
	if err != nil {
		return err
	}
	return c.manager.ReplaceByDevice(ctx, loginID, device)
}

// ReplaceByDeviceAndDeviceId replaces by device and device ID for the current user
// ReplaceByDeviceAndDeviceId 按设备和设备ID顶替当前用户
func (c *DTokenContext) ReplaceByDeviceAndDeviceId(ctx context.Context, deviceAndDeviceId ...string) error {
	loginID, err := c.GetLoginID(ctx)
	if err != nil {
		return err
	}
	return c.manager.ReplaceByDeviceAndDeviceId(ctx, loginID, deviceAndDeviceId...)
}

// ============================================================================
// 5. Token List Methods - Token列表方法
// ============================================================================

// GetTokenValueList gets all token values for the current logged-in user
// GetTokenValueList 获取当前登录用户的所有 token 列表
func (c *DTokenContext) GetTokenValueList(ctx context.Context, checkAlive ...bool) ([]string, error) {
	loginID, err := c.GetLoginID(ctx)
	if err != nil {
		return nil, err
	}
	return c.manager.GetTokenValueListByLoginID(ctx, loginID, checkAlive...)
}

// GetTokenValueListByDevice gets token list by device for the current user
// GetTokenValueListByDevice 按设备获取当前用户的token列表
func (c *DTokenContext) GetTokenValueListByDevice(ctx context.Context, device string, checkAlive ...bool) ([]string, error) {
	loginID, err := c.GetLoginID(ctx)
	if err != nil {
		return nil, err
	}
	return c.manager.GetTokenValueListByDevice(ctx, loginID, device, checkAlive...)
}

// GetTokenValueListByDeviceAndDeviceId gets token list by device and device ID for the current user
// GetTokenValueListByDeviceAndDeviceId 按设备和设备ID获取当前用户的token列表
func (c *DTokenContext) GetTokenValueListByDeviceAndDeviceId(ctx context.Context, device, deviceId string, checkAlive ...bool) ([]string, error) {
	loginID, err := c.GetLoginID(ctx)
	if err != nil {
		return nil, err
	}
	return c.manager.GetTokenValueListByDeviceAndDeviceId(ctx, loginID, device, deviceId, checkAlive...)
}

// ============================================================================
// 6. Online Terminal Methods - 在线终端方法
// ============================================================================

// GetOnlineTerminalCount gets the online terminal count for the current user
// GetOnlineTerminalCount 获取当前用户的在线终端数量
func (c *DTokenContext) GetOnlineTerminalCount(ctx context.Context) (int, error) {
	loginID, err := c.GetLoginID(ctx)
	if err != nil {
		return 0, err
	}
	return c.manager.GetOnlineTerminalCount(ctx, loginID)
}

// GetOnlineTerminalCountByDevice gets online terminal count by device for the current user
// GetOnlineTerminalCountByDevice 按设备获取当前用户的在线终端数量
func (c *DTokenContext) GetOnlineTerminalCountByDevice(ctx context.Context, device string) (int, error) {
	loginID, err := c.GetLoginID(ctx)
	if err != nil {
		return 0, err
	}
	return c.manager.GetOnlineTerminalCountByDevice(ctx, loginID, device)
}

// ============================================================================
// 7. Role Methods - 角色方法
// ============================================================================

// GetRoles gets the roles for the current logged-in user
// GetRoles 获取当前登录用户的角色列表
func (c *DTokenContext) GetRoles(ctx context.Context) ([]string, error) {
	loginID, err := c.GetLoginID(ctx)
	if err != nil {
		return nil, err
	}
	return c.manager.GetRoles(ctx, loginID)
}

// HasRole checks if the current user has the specified role
// HasRole 检查当前用户是否拥有指定角色
func (c *DTokenContext) HasRole(ctx context.Context, role string) bool {
	token := c.GetTokenValue()
	if token == "" {
		return false
	}
	return c.manager.HasRoleByToken(ctx, token, role)
}

// HasRoles checks if the current user has any of the specified roles (OR logic)
// HasRoles 检查当前用户是否拥有指定角色中的任意一个（OR 逻辑）
func (c *DTokenContext) HasRoles(ctx context.Context, roles []string) bool {
	token := c.GetTokenValue()
	if token == "" {
		return false
	}
	return c.manager.HasRolesOrByToken(ctx, token, roles)
}

// HasRolesAnd checks if the current user has all of the specified roles (AND logic)
// HasRolesAnd 检查当前用户是否拥有所有指定角色（AND 逻辑）
func (c *DTokenContext) HasRolesAnd(ctx context.Context, roles []string) bool {
	token := c.GetTokenValue()
	if token == "" {
		return false
	}
	return c.manager.HasRolesAndByToken(ctx, token, roles)
}

// AddRoles adds roles to the current user
// AddRoles 为当前用户添加角色
func (c *DTokenContext) AddRoles(ctx context.Context, roles []string) error {
	token := c.GetTokenValue()
	if token == "" {
		return derror.ErrNotLogin
	}
	return c.manager.AddRolesByToken(ctx, token, roles)
}

// RemoveRoles removes roles from the current user
// RemoveRoles 从当前用户移除角色
func (c *DTokenContext) RemoveRoles(ctx context.Context, roles []string) error {
	token := c.GetTokenValue()
	if token == "" {
		return derror.ErrNotLogin
	}
	return c.manager.RemoveRolesByToken(ctx, token, roles)
}

// ============================================================================
// 8. Permission Methods - 权限方法
// ============================================================================

// GetPermissions gets the permissions for the current logged-in user
// GetPermissions 获取当前登录用户的权限列表
func (c *DTokenContext) GetPermissions(ctx context.Context) ([]string, error) {
	loginID, err := c.GetLoginID(ctx)
	if err != nil {
		return nil, err
	}
	return c.manager.GetPermissions(ctx, loginID)
}

// HasPermission checks if the current user has the specified permission
// HasPermission 检查当前用户是否拥有指定权限
func (c *DTokenContext) HasPermission(ctx context.Context, permission string) bool {
	token := c.GetTokenValue()
	if token == "" {
		return false
	}
	return c.manager.HasPermissionByToken(ctx, token, permission)
}

// HasPermissions checks if the current user has any of the specified permissions (OR logic)
// HasPermissions 检查当前用户是否拥有指定权限中的任意一个（OR 逻辑）
func (c *DTokenContext) HasPermissions(ctx context.Context, permissions []string) bool {
	token := c.GetTokenValue()
	if token == "" {
		return false
	}
	return c.manager.HasPermissionsOrByToken(ctx, token, permissions)
}

// HasPermissionsAnd checks if the current user has all of the specified permissions (AND logic)
// HasPermissionsAnd 检查当前用户是否拥有所有指定权限（AND 逻辑）
func (c *DTokenContext) HasPermissionsAnd(ctx context.Context, permissions []string) bool {
	token := c.GetTokenValue()
	if token == "" {
		return false
	}
	return c.manager.HasPermissionsAndByToken(ctx, token, permissions)
}

// AddPermissions adds permissions to the current user
// AddPermissions 为当前用户添加权限
func (c *DTokenContext) AddPermissions(ctx context.Context, permissions []string) error {
	token := c.GetTokenValue()
	if token == "" {
		return derror.ErrNotLogin
	}
	return c.manager.AddPermissionsByToken(ctx, token, permissions)
}

// RemovePermissions removes permissions from the current user
// RemovePermissions 从当前用户移除权限
func (c *DTokenContext) RemovePermissions(ctx context.Context, permissions []string) error {
	token := c.GetTokenValue()
	if token == "" {
		return derror.ErrNotLogin
	}
	return c.manager.RemovePermissionsByToken(ctx, token, permissions)
}

// ============================================================================
// 9. Disable Management Methods - 封禁管理方法
// ============================================================================

// IsDisable checks if the current user is disabled
// IsDisable 检查当前用户是否被封禁
func (c *DTokenContext) IsDisable(ctx context.Context) bool {
	loginID, err := c.GetLoginID(ctx)
	if err != nil {
		return false
	}
	return c.manager.IsDisable(ctx, loginID)
}

// GetDisableInfo gets the disable information for the current user
// GetDisableInfo 获取当前用户的封禁信息
func (c *DTokenContext) GetDisableInfo(ctx context.Context) (*manager.DisableInfo, error) {
	loginID, err := c.GetLoginID(ctx)
	if err != nil {
		return nil, err
	}
	return c.manager.GetDisableInfo(ctx, loginID)
}

// GetDisableTTL gets the remaining disable TTL for the current user
// GetDisableTTL 获取当前用户的封禁剩余时间
func (c *DTokenContext) GetDisableTTL(ctx context.Context) (int64, error) {
	loginID, err := c.GetLoginID(ctx)
	if err != nil {
		return 0, err
	}
	return c.manager.GetDisableTTL(ctx, loginID)
}

// ============================================================================
// 10. Session Methods - Session方法
// ============================================================================

// GetSession gets the session for the current logged-in user
// GetSession 获取当前登录用户的 Session
func (c *DTokenContext) GetSession(ctx context.Context) (*manager.Session, error) {
	loginID, err := c.GetLoginID(ctx)
	if err != nil {
		return nil, err
	}
	return c.manager.GetSession(ctx, loginID)
}

// GetSessionByToken gets the session using the current token
// GetSessionByToken 使用当前 token 获取 Session
func (c *DTokenContext) GetSessionByToken(ctx context.Context) (*manager.Session, error) {
	token := c.GetTokenValue()
	if token == "" {
		return nil, derror.ErrNotLogin
	}
	return c.manager.GetSessionByToken(ctx, token)
}
