// @Author daixk 2025/12/22 15:56:00
package hertz

import (
	"context"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/manager"
	"github.com/Zany2/dtoken-go/integrations/authcheck"
	hertzapp "github.com/cloudwego/hertz/pkg/app"
)

// GetTokenValueByContext gets token value from current Hertz context GetTokenValueByContext 从当前 Hertz 上下文获取 token 值。
func GetTokenValueByContext(ctx *hertzapp.RequestContext) (string, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return "", err
	}

	tokenValue := dCtx.GetTokenValue()
	if tokenValue == "" {
		return "", ErrNotLogin
	}
	return tokenValue, nil
}

// GetRequestContextByContext gets raw request context GetRequestContextByContext 获取原始请求上下文。
func GetRequestContextByContext(ctx *hertzapp.RequestContext) (adapter.RequestContext, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.GetRequestContext(), nil
}

// GetManagerByContext gets current DToken manager GetManagerByContext 获取当前 DToken 管理器。
func GetManagerByContext(ctx *hertzapp.RequestContext) (*manager.Manager, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.GetManager(), nil
}

// IsLoginByContext checks current request login state IsLoginByContext 检查当前请求登录状态。
func IsLoginByContext(ctx *hertzapp.RequestContext) bool {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return false
	}
	return dCtx.Auth().IsLogin(requestContext(ctx))
}

// CheckLoginByContext checks current request login state CheckLoginByContext 校验当前请求登录状态。
func CheckLoginByContext(ctx *hertzapp.RequestContext) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Auth().CheckLogin(requestContext(ctx))
}

// GetLoginIDByContext gets current login ID GetLoginIDByContext 获取当前登录 ID。
func GetLoginIDByContext(ctx *hertzapp.RequestContext) (string, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.Auth().GetLoginID(requestContext(ctx))
}

// LoginByTokenByContext renews current token login state LoginByTokenByContext 使用当前 token 续期登录态。
func LoginByTokenByContext(ctx *hertzapp.RequestContext) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Auth().LoginByToken(requestContext(ctx))
}

// LogoutByContext logs out current request token LogoutByContext 登出当前请求 token。
func LogoutByContext(ctx *hertzapp.RequestContext) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Auth().Logout(requestContext(ctx))
}

// KickoutByContext kicks out current request token KickoutByContext 踢出当前请求 token。
func KickoutByContext(ctx *hertzapp.RequestContext) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Auth().Kickout(requestContext(ctx))
}

// ReplaceByContext replaces current request token ReplaceByContext 顶替当前请求 token。
func ReplaceByContext(ctx *hertzapp.RequestContext) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Auth().Replace(requestContext(ctx))
}

// LogoutByDeviceByContext logs out current user by device LogoutByDeviceByContext 按设备登出当前用户。
func LogoutByDeviceByContext(ctx *hertzapp.RequestContext, device string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().LogoutByDevice(requestContext(ctx), device)
}

// LogoutByDeviceAndDeviceIdByContext logs out current user by device and id LogoutByDeviceAndDeviceIdByContext 按设备和设备 ID 登出当前用户。
func LogoutByDeviceAndDeviceIdByContext(ctx *hertzapp.RequestContext, deviceAndDeviceId ...string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().LogoutByDeviceAndDeviceId(requestContext(ctx), deviceAndDeviceId...)
}

// LogoutByLoginIDByContext logs out all terminals of current user LogoutByLoginIDByContext 登出当前用户所有终端。
func LogoutByLoginIDByContext(ctx *hertzapp.RequestContext) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().LogoutAll(requestContext(ctx))
}

// GetTokenInfoByContext gets current token info GetTokenInfoByContext 获取当前 token 信息。
func GetTokenInfoByContext(ctx *hertzapp.RequestContext) (*manager.TokenInfo, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Auth().GetTokenInfo(requestContext(ctx))
}

// IntrospectTokenByContext inspects current token without renewal side effects IntrospectTokenByContext 无续期副作用地检查当前 token 状态。
func IntrospectTokenByContext(ctx *hertzapp.RequestContext) (*manager.TokenIntrospection, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Auth().IntrospectToken(requestContext(ctx))
}

// GetDeviceByContext gets current token device GetDeviceByContext 获取当前 token 设备类型。
func GetDeviceByContext(ctx *hertzapp.RequestContext) (string, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.Auth().GetDevice(requestContext(ctx))
}

// GetDeviceIDByContext gets current token device id GetDeviceIDByContext 获取当前 token 设备 ID。
func GetDeviceIDByContext(ctx *hertzapp.RequestContext) (string, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.Auth().GetDeviceId(requestContext(ctx))
}

// GetTokenTTLByContext gets current token TTL GetTokenTTLByContext 获取当前 token 剩余有效期。
func GetTokenTTLByContext(ctx *hertzapp.RequestContext) (int64, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return 0, err
	}
	return dCtx.Auth().GetTokenTTL(requestContext(ctx))
}

// GetTokenCreateTimeByContext gets current token create time GetTokenCreateTimeByContext 获取当前 token 创建时间。
func GetTokenCreateTimeByContext(ctx *hertzapp.RequestContext) (int64, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return 0, err
	}
	return dCtx.Auth().GetTokenCreateTime(requestContext(ctx))
}

// RenewTimeoutByContext renews current token timeout RenewTimeoutByContext 续期当前 token 过期时间。
func RenewTimeoutByContext(ctx *hertzapp.RequestContext, timeout time.Duration) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Auth().RenewTimeout(requestContext(ctx), timeout)
}

// GetSessionByContext gets current user session GetSessionByContext 获取当前用户会话。
func GetSessionByContext(ctx *hertzapp.RequestContext) (*manager.Session, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Session().Get(requestContext(ctx))
}

// GetSessionByTokenByContext gets current token session GetSessionByTokenByContext 获取当前 token 会话。
func GetSessionByTokenByContext(ctx *hertzapp.RequestContext) (*manager.Session, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Session().GetByToken(requestContext(ctx))
}

// GetTokenValueListByContext gets current user token list GetTokenValueListByContext 获取当前用户 token 列表。
func GetTokenValueListByContext(ctx *hertzapp.RequestContext, checkAlive ...bool) ([]string, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().GetTokenValueList(requestContext(ctx), checkAlive...)
}

// GetOnlineTerminalCountByContext gets current user online terminal count GetOnlineTerminalCountByContext 获取当前用户在线终端数量。
func GetOnlineTerminalCountByContext(ctx *hertzapp.RequestContext) (int, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return 0, err
	}
	return dCtx.Terminal().GetOnlineTerminalCount(requestContext(ctx))
}

// GetRolesByContext gets current user roles GetRolesByContext 获取当前用户角色列表。
func GetRolesByContext(ctx *hertzapp.RequestContext) ([]string, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Access().GetRoles(requestContext(ctx))
}

// GetRolesByTokenByContext gets current token roles GetRolesByTokenByContext 使用当前 token 获取角色列表。
func GetRolesByTokenByContext(ctx *hertzapp.RequestContext) ([]string, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Access().GetRolesByToken(requestContext(ctx))
}

// HasRoleByContext checks current user role HasRoleByContext 检查当前用户角色。
func HasRoleByContext(ctx *hertzapp.RequestContext, role string) bool {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return false
	}
	return dCtx.Access().HasRole(requestContext(ctx), role)
}

// HasRolesByContext checks whether current user has any role HasRolesByContext 检查当前用户是否拥有任一角色。
func HasRolesByContext(ctx *hertzapp.RequestContext, roles []string) bool {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return false
	}
	return dCtx.Access().HasRoles(requestContext(ctx), roles)
}

// HasRolesOrByContext checks whether current user has any role HasRolesOrByContext 检查当前用户是否拥有任一角色。
func HasRolesOrByContext(ctx *hertzapp.RequestContext, roles []string) bool {
	return HasRolesByContext(ctx, roles)
}

// HasRolesAndByContext checks whether current user has all roles HasRolesAndByContext 检查当前用户是否拥有全部角色。
func HasRolesAndByContext(ctx *hertzapp.RequestContext, roles []string) bool {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return false
	}
	return dCtx.Access().HasRolesAnd(requestContext(ctx), roles)
}

// GetPermissionsByContext gets current user permissions GetPermissionsByContext 获取当前用户权限列表。
func GetPermissionsByContext(ctx *hertzapp.RequestContext) ([]string, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Access().GetPermissions(requestContext(ctx))
}

// GetPermissionsByTokenByContext gets current token permissions GetPermissionsByTokenByContext 使用当前 token 获取权限列表。
func GetPermissionsByTokenByContext(ctx *hertzapp.RequestContext) ([]string, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Access().GetPermissionsByToken(requestContext(ctx))
}

// HasPermissionByContext checks current user permission HasPermissionByContext 检查当前用户权限。
func HasPermissionByContext(ctx *hertzapp.RequestContext, permission string) bool {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return false
	}
	return dCtx.Access().HasPermission(requestContext(ctx), permission)
}

// HasPermissionsByContext checks whether current user has any permission HasPermissionsByContext 检查当前用户是否拥有任一权限。
func HasPermissionsByContext(ctx *hertzapp.RequestContext, permissions []string) bool {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return false
	}
	return dCtx.Access().HasPermissions(requestContext(ctx), permissions)
}

// HasPermissionsOrByContext checks whether current user has any permission HasPermissionsOrByContext 检查当前用户是否拥有任一权限。
func HasPermissionsOrByContext(ctx *hertzapp.RequestContext, permissions []string) bool {
	return HasPermissionsByContext(ctx, permissions)
}

// HasPermissionsAndByContext checks whether current user has all permissions HasPermissionsAndByContext 检查当前用户是否拥有全部权限。
func HasPermissionsAndByContext(ctx *hertzapp.RequestContext, permissions []string) bool {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return false
	}
	return dCtx.Access().HasPermissionsAnd(requestContext(ctx), permissions)
}

// AddRolesByContext adds roles to current token AddRolesByContext 为当前 token 添加角色。
func AddRolesByContext(ctx *hertzapp.RequestContext, roles []string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Access().AddRoles(requestContext(ctx), roles)
}

// RemoveRolesByContext removes roles from current token RemoveRolesByContext 从当前 token 移除角色。
func RemoveRolesByContext(ctx *hertzapp.RequestContext, roles []string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Access().RemoveRoles(requestContext(ctx), roles)
}

// AddPermissionsByContext adds permissions to current token AddPermissionsByContext 为当前 token 添加权限。
func AddPermissionsByContext(ctx *hertzapp.RequestContext, permissions []string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Access().AddPermissions(requestContext(ctx), permissions)
}

// RemovePermissionsByContext removes permissions from current token RemovePermissionsByContext 从当前 token 移除权限。
func RemovePermissionsByContext(ctx *hertzapp.RequestContext, permissions []string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Access().RemovePermissions(requestContext(ctx), permissions)
}

// IsDisableByContext checks whether current user is disabled IsDisableByContext 检查当前用户是否被封禁。
func IsDisableByContext(ctx *hertzapp.RequestContext) bool {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return false
	}
	return dCtx.Disable().IsAccount(requestContext(ctx))
}

// GetDisableInfoByContext gets current user disable info GetDisableInfoByContext 获取当前用户封禁信息。
func GetDisableInfoByContext(ctx *hertzapp.RequestContext) (*manager.DisableInfo, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Disable().AccountInfo(requestContext(ctx))
}

// GetDisableTTLByContext gets current user disable ttl GetDisableTTLByContext 获取当前用户封禁剩余时间。
func GetDisableTTLByContext(ctx *hertzapp.RequestContext) (int64, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return 0, err
	}
	return dCtx.Disable().AccountTTL(requestContext(ctx))
}

// DisableByContext disables current user DisableByContext 封禁当前用户。
func DisableByContext(ctx *hertzapp.RequestContext, duration time.Duration, reason ...string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().Account(requestContext(ctx), duration, reason...)
}

// UntieByContext removes current user disable state UntieByContext 解封当前用户。
func UntieByContext(ctx *hertzapp.RequestContext) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().UntieAccount(requestContext(ctx))
}

// GenerateNonceByContext generates nonce with current manager GenerateNonceByContext 使用当前管理器生成 nonce。
func GenerateNonceByContext(ctx *hertzapp.RequestContext) (string, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.Nonce().Generate(requestContext(ctx))
}

// VerifyNonceByContext verifies nonce with current manager VerifyNonceByContext 使用当前管理器验证 nonce。
func VerifyNonceByContext(ctx *hertzapp.RequestContext, nonce string) bool {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return false
	}
	return dCtx.Nonce().Verify(requestContext(ctx), nonce)
}

// VerifyAndConsumeNonceByContext verifies and consumes nonce with current manager VerifyAndConsumeNonceByContext 使用当前管理器验证并消费 nonce。
func VerifyAndConsumeNonceByContext(ctx *hertzapp.RequestContext, nonce string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Nonce().VerifyAndConsume(requestContext(ctx), nonce)
}

// requestContext returns default execution context requestContext 返回默认执行上下文。
func requestContext(*hertzapp.RequestContext) context.Context {
	return context.Background()
}

// requireDTokenContextByContext gets required DToken context requireDTokenContextByContext 获取必需的 DToken 上下文。
func requireDTokenContextByContext(ctx *hertzapp.RequestContext) (*DTokenContext, error) {
	dCtx, ok := GetDTokenContext(ctx)
	if !ok {
		if ctx == nil {
			return nil, ErrNotLogin
		}
		mgr, err := authcheck.GetManager("")
		if err != nil {
			return nil, err
		}
		return getDTokenContext(ctx, mgr), nil
	}
	return dCtx, nil
}
