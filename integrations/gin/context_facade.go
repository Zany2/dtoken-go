// @Author daixk 2025/12/22 15:56:00
package gin

import (
	"context"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/manager"
	"github.com/Zany2/dtoken-go/integrations/authcheck"
	"github.com/gin-gonic/gin"
)

// GetTokenValueByContext gets token value from current Gin context GetTokenValueByContext 从当前 Gin 上下文获取 token 值
func GetTokenValueByContext(c *gin.Context) (string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return "", err
	}

	tokenValue := dCtx.GetTokenValue()
	if tokenValue == "" {
		return "", ErrNotLogin
	}
	return tokenValue, nil
}

// GetRequestContextByContext gets raw request context GetRequestContextByContext 获取原始请求上下文
func GetRequestContextByContext(c *gin.Context) (adapter.RequestContext, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.GetRequestContext(), nil
}

// GetManagerByContext gets current DToken manager GetManagerByContext 获取当前 DToken 管理器
func GetManagerByContext(c *gin.Context) (*manager.Manager, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.GetManager(), nil
}

// IsLoginByContext checks current request login state IsLoginByContext 检查当前请求登录状态
func IsLoginByContext(c *gin.Context) bool {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return false
	}
	return dCtx.Auth().IsLogin(requestContext(c))
}

// CheckLoginByContext checks current request login state CheckLoginByContext 校验当前请求登录状态
func CheckLoginByContext(c *gin.Context) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Auth().CheckLogin(requestContext(c))
}

// GetLoginIDByContext gets current login ID GetLoginIDByContext 获取当前登录 ID
func GetLoginIDByContext(c *gin.Context) (string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return "", err
	}
	return dCtx.Auth().GetLoginID(requestContext(c))
}

// LoginByTokenByContext renews current token login state LoginByTokenByContext 使用当前 token 续期登录态
func LoginByTokenByContext(c *gin.Context) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Auth().LoginByToken(requestContext(c))
}

// LogoutByContext logs out current request token LogoutByContext 登出当前请求 token
func LogoutByContext(c *gin.Context) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Auth().Logout(requestContext(c))
}

// KickoutByContext kicks out current request token KickoutByContext 踢出当前请求 token
func KickoutByContext(c *gin.Context) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Auth().Kickout(requestContext(c))
}

// ReplaceByContext replaces current request token ReplaceByContext 顶替当前请求 token
func ReplaceByContext(c *gin.Context) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Auth().Replace(requestContext(c))
}

// LogoutByDeviceByContext logs out current user by device LogoutByDeviceByContext 按设备登出当前用户
func LogoutByDeviceByContext(c *gin.Context, device string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Terminal().LogoutByDevice(requestContext(c), device)
}

// LogoutByDeviceAndDeviceIdByContext logs out current user by device and id LogoutByDeviceAndDeviceIdByContext 按设备和设备 ID 登出当前用户
func LogoutByDeviceAndDeviceIdByContext(c *gin.Context, deviceAndDeviceId ...string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Terminal().LogoutByDeviceAndDeviceId(requestContext(c), deviceAndDeviceId...)
}

// LogoutByLoginIDByContext logs out all terminals of current user LogoutByLoginIDByContext 登出当前用户所有终端
func LogoutByLoginIDByContext(c *gin.Context) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Terminal().LogoutAll(requestContext(c))
}

// GetTokenInfoByContext gets current token info GetTokenInfoByContext 获取当前 token 信息
func GetTokenInfoByContext(c *gin.Context) (*manager.TokenInfo, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Auth().GetTokenInfo(requestContext(c))
}

// IntrospectTokenByContext inspects current token without renewal side effects IntrospectTokenByContext 无续期副作用地检查当前 token 状态
func IntrospectTokenByContext(c *gin.Context) (*manager.TokenIntrospection, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Auth().IntrospectToken(requestContext(c))
}

// GetDeviceByContext gets current token device GetDeviceByContext 获取当前 token 设备类型
func GetDeviceByContext(c *gin.Context) (string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return "", err
	}
	return dCtx.Auth().GetDevice(requestContext(c))
}

// GetDeviceIDByContext gets current token device id GetDeviceIDByContext 获取当前 token 设备 ID
func GetDeviceIDByContext(c *gin.Context) (string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return "", err
	}
	return dCtx.Auth().GetDeviceId(requestContext(c))
}

// GetTokenTTLByContext gets current token TTL GetTokenTTLByContext 获取当前 token 剩余有效期
func GetTokenTTLByContext(c *gin.Context) (int64, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return 0, err
	}
	return dCtx.Auth().GetTokenTTL(requestContext(c))
}

// GetTokenCreateTimeByContext gets current token create time GetTokenCreateTimeByContext 获取当前 token 创建时间
func GetTokenCreateTimeByContext(c *gin.Context) (int64, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return 0, err
	}
	return dCtx.Auth().GetTokenCreateTime(requestContext(c))
}

// RenewTimeoutByContext renews current token timeout RenewTimeoutByContext 续期当前 token 过期时间
func RenewTimeoutByContext(c *gin.Context, timeout time.Duration) error {
	tokenValue, err := GetTokenValueByContext(c)
	if err != nil {
		return err
	}
	return RenewTimeout(requestContext(c), tokenValue, timeout)
}

// GetSessionByContext gets current user session GetSessionByContext 获取当前用户会话
func GetSessionByContext(c *gin.Context) (*manager.Session, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Session().Get(requestContext(c))
}

// GetSessionByTokenByContext gets current token session GetSessionByTokenByContext 获取当前 token 会话
func GetSessionByTokenByContext(c *gin.Context) (*manager.Session, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Session().GetByToken(requestContext(c))
}

// GetTokenValueListByContext gets current user token list GetTokenValueListByContext 获取当前用户 token 列表
func GetTokenValueListByContext(c *gin.Context, checkAlive ...bool) ([]string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().GetTokenValueList(requestContext(c), checkAlive...)
}

// GetOnlineTerminalCountByContext gets current user online terminal count GetOnlineTerminalCountByContext 获取当前用户在线终端数量
func GetOnlineTerminalCountByContext(c *gin.Context) (int, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return 0, err
	}
	return dCtx.Terminal().GetOnlineTerminalCount(requestContext(c))
}

// GetRolesByContext gets current user roles GetRolesByContext 获取当前用户角色列表
func GetRolesByContext(c *gin.Context) ([]string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Access().GetRoles(requestContext(c))
}

// GetRolesByTokenByContext gets current token roles GetRolesByTokenByContext 使用当前 token 获取角色列表
func GetRolesByTokenByContext(c *gin.Context) ([]string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Access().GetRolesByToken(requestContext(c))
}

// HasRoleByContext checks current user role HasRoleByContext 检查当前用户角色
func HasRoleByContext(c *gin.Context, role string) bool {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return false
	}
	return dCtx.Access().HasRole(requestContext(c), role)
}

// HasRolesByContext checks whether current user has any role HasRolesByContext 检查当前用户是否拥有任一角色
func HasRolesByContext(c *gin.Context, roles []string) bool {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return false
	}
	return dCtx.Access().HasRoles(requestContext(c), roles)
}

// HasRolesOrByContext checks whether current user has any role HasRolesOrByContext 检查当前用户是否拥有任一角色
func HasRolesOrByContext(c *gin.Context, roles []string) bool {
	return HasRolesByContext(c, roles)
}

// HasRolesAndByContext checks whether current user has all roles HasRolesAndByContext 检查当前用户是否拥有全部角色
func HasRolesAndByContext(c *gin.Context, roles []string) bool {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return false
	}
	return dCtx.Access().HasRolesAnd(requestContext(c), roles)
}

// GetPermissionsByContext gets current user permissions GetPermissionsByContext 获取当前用户权限列表
func GetPermissionsByContext(c *gin.Context) ([]string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Access().GetPermissions(requestContext(c))
}

// GetPermissionsByTokenByContext gets current token permissions GetPermissionsByTokenByContext 使用当前 token 获取权限列表
func GetPermissionsByTokenByContext(c *gin.Context) ([]string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Access().GetPermissionsByToken(requestContext(c))
}

// HasPermissionByContext checks current user permission HasPermissionByContext 检查当前用户权限
func HasPermissionByContext(c *gin.Context, permission string) bool {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return false
	}
	return dCtx.Access().HasPermission(requestContext(c), permission)
}

// HasPermissionsByContext checks whether current user has any permission HasPermissionsByContext 检查当前用户是否拥有任一权限
func HasPermissionsByContext(c *gin.Context, permissions []string) bool {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return false
	}
	return dCtx.Access().HasPermissions(requestContext(c), permissions)
}

// HasPermissionsOrByContext checks whether current user has any permission HasPermissionsOrByContext 检查当前用户是否拥有任一权限
func HasPermissionsOrByContext(c *gin.Context, permissions []string) bool {
	return HasPermissionsByContext(c, permissions)
}

// HasPermissionsAndByContext checks whether current user has all permissions HasPermissionsAndByContext 检查当前用户是否拥有全部权限
func HasPermissionsAndByContext(c *gin.Context, permissions []string) bool {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return false
	}
	return dCtx.Access().HasPermissionsAnd(requestContext(c), permissions)
}

// AddRolesByContext adds roles to current token AddRolesByContext 为当前 token 添加角色
func AddRolesByContext(c *gin.Context, roles []string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Access().AddRoles(requestContext(c), roles)
}

// RemoveRolesByContext removes roles from current token RemoveRolesByContext 从当前 token 移除角色
func RemoveRolesByContext(c *gin.Context, roles []string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Access().RemoveRoles(requestContext(c), roles)
}

// AddPermissionsByContext adds permissions to current token AddPermissionsByContext 为当前 token 添加权限
func AddPermissionsByContext(c *gin.Context, permissions []string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Access().AddPermissions(requestContext(c), permissions)
}

// RemovePermissionsByContext removes permissions from current token RemovePermissionsByContext 从当前 token 移除权限
func RemovePermissionsByContext(c *gin.Context, permissions []string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Access().RemovePermissions(requestContext(c), permissions)
}

// IsDisableByContext checks whether current user is disabled IsDisableByContext 检查当前用户是否被封禁
func IsDisableByContext(c *gin.Context) bool {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return false
	}
	return dCtx.Disable().IsAccount(requestContext(c))
}

// GetDisableInfoByContext gets current user disable info GetDisableInfoByContext 获取当前用户封禁信息
func GetDisableInfoByContext(c *gin.Context) (*manager.DisableInfo, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Disable().AccountInfo(requestContext(c))
}

// GetDisableTTLByContext gets current user disable ttl GetDisableTTLByContext 获取当前用户封禁剩余时间
func GetDisableTTLByContext(c *gin.Context) (int64, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return 0, err
	}
	return dCtx.Disable().AccountTTL(requestContext(c))
}

// DisableByContext disables current user DisableByContext 封禁当前用户
func DisableByContext(c *gin.Context, duration time.Duration, reason ...string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Disable().Account(requestContext(c), duration, reason...)
}

// UntieByContext removes current user disable state UntieByContext 解封当前用户
func UntieByContext(c *gin.Context) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Disable().UntieAccount(requestContext(c))
}

// GenerateNonceByContext generates nonce with current manager GenerateNonceByContext 使用当前管理器生成 nonce
func GenerateNonceByContext(c *gin.Context) (string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return "", err
	}
	return dCtx.Nonce().Generate(requestContext(c))
}

// VerifyNonceByContext verifies nonce with current manager VerifyNonceByContext 使用当前管理器验证 nonce
func VerifyNonceByContext(c *gin.Context, nonce string) bool {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return false
	}
	return dCtx.Nonce().Verify(requestContext(c), nonce)
}

// VerifyAndConsumeNonceByContext verifies and consumes nonce with current manager VerifyAndConsumeNonceByContext 使用当前管理器验证并消费 nonce
func VerifyAndConsumeNonceByContext(c *gin.Context, nonce string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Nonce().VerifyAndConsume(requestContext(c), nonce)
}

// requestContext gets standard context from Gin request requestContext 从 Gin 请求获取标准上下文
func requestContext(c *gin.Context) context.Context {
	if c != nil && c.Request != nil {
		return c.Request.Context()
	}
	return context.Background()
}

// requireDTokenContextByContext gets required DToken context requireDTokenContextByContext 获取必需的 DToken 上下文
func requireDTokenContextByContext(c *gin.Context) (*DTokenContext, error) {
	dCtx, ok := GetDTokenContext(c)
	if !ok {
		if c == nil {
			return nil, ErrNotLogin
		}
		mgr, err := authcheck.GetManager("")
		if err != nil {
			return nil, err
		}
		return getDContext(c, mgr), nil
	}
	return dCtx, nil
}
