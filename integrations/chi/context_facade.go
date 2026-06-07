// @Author daixk 2025/12/22 15:56:00
package chi

import (
	"context"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/manager"
)

// GetTokenValueByCtx gets token value from current Chi context GetTokenValueByCtx 从当前 Chi 上下文获取 token 值。
func GetTokenValueByCtx(ctx context.Context) (string, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return "", err
	}

	tokenValue := dCtx.GetTokenValue()
	if tokenValue == "" {
		return "", ErrNotLogin
	}
	return tokenValue, nil
}

// GetRequestContextByCtx gets raw request context GetRequestContextByCtx 获取原始请求上下文。
func GetRequestContextByCtx(ctx context.Context) (adapter.RequestContext, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.GetRequestContext(), nil
}

// GetManagerByCtx gets current DToken manager GetManagerByCtx 获取当前 DToken 管理器。
func GetManagerByCtx(ctx context.Context) (*manager.Manager, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.GetManager(), nil
}

// IsLoginByCtx checks current request login state IsLoginByCtx 检查当前请求登录状态。
func IsLoginByCtx(ctx context.Context) bool {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return false
	}
	return dCtx.Auth().IsLogin(ctx)
}

// CheckLoginByCtx checks current request login state CheckLoginByCtx 校验当前请求登录状态。
func CheckLoginByCtx(ctx context.Context) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Auth().CheckLogin(ctx)
}

// LoginByTokenByCtx renews current token login state LoginByTokenByCtx 使用当前 token 续期登录态。
func LoginByTokenByCtx(ctx context.Context) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Auth().LoginByToken(ctx)
}

// LogoutByCtx logs out current request token LogoutByCtx 登出当前请求 token。
func LogoutByCtx(ctx context.Context) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Auth().Logout(ctx)
}

// KickoutByCtx kicks out current request token KickoutByCtx 踢出当前请求 token。
func KickoutByCtx(ctx context.Context) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Auth().Kickout(ctx)
}

// ReplaceByCtx replaces current request token ReplaceByCtx 顶替当前请求 token。
func ReplaceByCtx(ctx context.Context) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Auth().Replace(ctx)
}

// LogoutByDeviceByCtx logs out current user by device LogoutByDeviceByCtx 按设备登出当前用户。
func LogoutByDeviceByCtx(ctx context.Context, device string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().LogoutByDevice(ctx, device)
}

// LogoutByDeviceAndDeviceIdByCtx logs out current user by device and id LogoutByDeviceAndDeviceIdByCtx 按设备和设备 ID 登出当前用户。
func LogoutByDeviceAndDeviceIdByCtx(ctx context.Context, deviceAndDeviceId ...string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().LogoutByDeviceAndDeviceId(ctx, deviceAndDeviceId...)
}

// LogoutByLoginIDByCtx logs out all terminals of current user LogoutByLoginIDByCtx 登出当前用户所有终端。
func LogoutByLoginIDByCtx(ctx context.Context) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().LogoutAll(ctx)
}

// GetDeviceByCtx gets current token device GetDeviceByCtx 获取当前 token 设备类型。
func GetDeviceByCtx(ctx context.Context) (string, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.Auth().GetDevice(ctx)
}

// GetDeviceIDByCtx gets current token device id GetDeviceIDByCtx 获取当前 token 设备 ID。
func GetDeviceIDByCtx(ctx context.Context) (string, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.Auth().GetDeviceId(ctx)
}

// GetTokenTTLByCtx gets current token TTL GetTokenTTLByCtx 获取当前 token 剩余有效期。
func GetTokenTTLByCtx(ctx context.Context) (int64, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return 0, err
	}
	return dCtx.Auth().GetTokenTTL(ctx)
}

// GetTokenCreateTimeByCtx gets current token create time GetTokenCreateTimeByCtx 获取当前 token 创建时间。
func GetTokenCreateTimeByCtx(ctx context.Context) (int64, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return 0, err
	}
	return dCtx.Auth().GetTokenCreateTime(ctx)
}

// RenewTimeoutByCtx renews current token timeout RenewTimeoutByCtx 续期当前 token 过期时间。
func RenewTimeoutByCtx(ctx context.Context, timeout time.Duration) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Auth().RenewTimeout(ctx, timeout)
}

// GetSessionByCtx gets current user session GetSessionByCtx 获取当前用户会话。
func GetSessionByCtx(ctx context.Context) (*manager.Session, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Session().Get(ctx)
}

// GetSessionByTokenByCtx gets current token session GetSessionByTokenByCtx 获取当前 token 会话。
func GetSessionByTokenByCtx(ctx context.Context) (*manager.Session, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Session().GetByToken(ctx)
}

// GetTokenValueListByCtx gets current user token list GetTokenValueListByCtx 获取当前用户 token 列表。
func GetTokenValueListByCtx(ctx context.Context, checkAlive ...bool) ([]string, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().GetTokenValueList(ctx, checkAlive...)
}

// GetOnlineTerminalCountByCtx gets current user online terminal count GetOnlineTerminalCountByCtx 获取当前用户在线终端数量。
func GetOnlineTerminalCountByCtx(ctx context.Context) (int, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return 0, err
	}
	return dCtx.Terminal().GetOnlineTerminalCount(ctx)
}

// GetRolesByCtx gets current user roles GetRolesByCtx 获取当前用户角色列表。
func GetRolesByCtx(ctx context.Context) ([]string, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Access().GetRoles(ctx)
}

// GetRolesByTokenByCtx gets current token roles GetRolesByTokenByCtx 使用当前 token 获取角色列表。
func GetRolesByTokenByCtx(ctx context.Context) ([]string, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Access().GetRolesByToken(ctx)
}

// HasRoleByCtx checks current user role HasRoleByCtx 检查当前用户角色。
func HasRoleByCtx(ctx context.Context, role string) bool {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return false
	}
	return dCtx.Access().HasRole(ctx, role)
}

// HasRolesByCtx checks whether current user has any role HasRolesByCtx 检查当前用户是否拥有任一角色。
func HasRolesByCtx(ctx context.Context, roles []string) bool {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return false
	}
	return dCtx.Access().HasRoles(ctx, roles)
}

// HasRolesOrByCtx checks whether current user has any role HasRolesOrByCtx 检查当前用户是否拥有任一角色。
func HasRolesOrByCtx(ctx context.Context, roles []string) bool {
	return HasRolesByCtx(ctx, roles)
}

// HasRolesAndByCtx checks whether current user has all roles HasRolesAndByCtx 检查当前用户是否拥有全部角色。
func HasRolesAndByCtx(ctx context.Context, roles []string) bool {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return false
	}
	return dCtx.Access().HasRolesAnd(ctx, roles)
}

// GetPermissionsByCtx gets current user permissions GetPermissionsByCtx 获取当前用户权限列表。
func GetPermissionsByCtx(ctx context.Context) ([]string, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Access().GetPermissions(ctx)
}

// GetPermissionsByTokenByCtx gets current token permissions GetPermissionsByTokenByCtx 使用当前 token 获取权限列表。
func GetPermissionsByTokenByCtx(ctx context.Context) ([]string, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Access().GetPermissionsByToken(ctx)
}

// HasPermissionByCtx checks current user permission HasPermissionByCtx 检查当前用户权限。
func HasPermissionByCtx(ctx context.Context, permission string) bool {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return false
	}
	return dCtx.Access().HasPermission(ctx, permission)
}

// HasPermissionsByCtx checks whether current user has any permission HasPermissionsByCtx 检查当前用户是否拥有任一权限。
func HasPermissionsByCtx(ctx context.Context, permissions []string) bool {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return false
	}
	return dCtx.Access().HasPermissions(ctx, permissions)
}

// HasPermissionsOrByCtx checks whether current user has any permission HasPermissionsOrByCtx 检查当前用户是否拥有任一权限。
func HasPermissionsOrByCtx(ctx context.Context, permissions []string) bool {
	return HasPermissionsByCtx(ctx, permissions)
}

// HasPermissionsAndByCtx checks whether current user has all permissions HasPermissionsAndByCtx 检查当前用户是否拥有全部权限。
func HasPermissionsAndByCtx(ctx context.Context, permissions []string) bool {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return false
	}
	return dCtx.Access().HasPermissionsAnd(ctx, permissions)
}

// AddRolesByCtx adds roles to current token AddRolesByCtx 为当前 token 添加角色。
func AddRolesByCtx(ctx context.Context, roles []string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Access().AddRoles(ctx, roles)
}

// RemoveRolesByCtx removes roles from current token RemoveRolesByCtx 从当前 token 移除角色。
func RemoveRolesByCtx(ctx context.Context, roles []string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Access().RemoveRoles(ctx, roles)
}

// AddPermissionsByCtx adds permissions to current token AddPermissionsByCtx 为当前 token 添加权限。
func AddPermissionsByCtx(ctx context.Context, permissions []string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Access().AddPermissions(ctx, permissions)
}

// RemovePermissionsByCtx removes permissions from current token RemovePermissionsByCtx 从当前 token 移除权限。
func RemovePermissionsByCtx(ctx context.Context, permissions []string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Access().RemovePermissions(ctx, permissions)
}

// IsDisableByCtx checks whether current user is disabled IsDisableByCtx 检查当前用户是否被封禁。
func IsDisableByCtx(ctx context.Context) bool {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return false
	}
	return dCtx.Disable().IsAccount(ctx)
}

// GetDisableInfoByCtx gets current user disable info GetDisableInfoByCtx 获取当前用户封禁信息。
func GetDisableInfoByCtx(ctx context.Context) (*manager.DisableInfo, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Disable().AccountInfo(ctx)
}

// GetDisableTTLByCtx gets current user disable ttl GetDisableTTLByCtx 获取当前用户封禁剩余时间。
func GetDisableTTLByCtx(ctx context.Context) (int64, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return 0, err
	}
	return dCtx.Disable().AccountTTL(ctx)
}

// DisableByCtx disables current user DisableByCtx 封禁当前用户。
func DisableByCtx(ctx context.Context, duration time.Duration, reason ...string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().Account(ctx, duration, reason...)
}

// UntieByCtx removes current user disable state UntieByCtx 解封当前用户。
func UntieByCtx(ctx context.Context) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().UntieAccount(ctx)
}

// GenerateNonceByCtx generates nonce with current manager GenerateNonceByCtx 使用当前管理器生成 nonce。
func GenerateNonceByCtx(ctx context.Context) (string, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.Nonce().Generate(ctx)
}

// VerifyNonceByCtx verifies nonce with current manager VerifyNonceByCtx 使用当前管理器验证 nonce。
func VerifyNonceByCtx(ctx context.Context, nonce string) bool {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return false
	}
	return dCtx.Nonce().Verify(ctx, nonce)
}

// VerifyAndConsumeNonceByCtx verifies and consumes nonce with current manager VerifyAndConsumeNonceByCtx 使用当前管理器验证并消费 nonce。
func VerifyAndConsumeNonceByCtx(ctx context.Context, nonce string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Nonce().VerifyAndConsume(ctx, nonce)
}

// requireDTokenContextByCtx gets required DToken context requireDTokenContextByCtx 获取必需的 DToken 上下文。
func requireDTokenContextByCtx(ctx context.Context) (*DTokenContext, error) {
	dCtx, ok := GetDTokenContextByCtx(ctx)
	if !ok {
		return nil, ErrNotLogin
	}
	return dCtx, nil
}
