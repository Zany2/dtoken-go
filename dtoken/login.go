// @Author daixk 2025/12/22 15:56:00
package dtoken

import (
	"context"
	"time"

	"github.com/Zany2/dtoken-go/core/manager"
)

// Login performs login and returns a token. Login 执行登录并返回 token。
func Login(ctx context.Context, loginID string, params ...string) (string, error) {
	device, deviceId, authType := parseDeviceAndAuthType(params...)
	mgr, err := GetManager(authType)
	if err != nil {
		return "", err
	}
	return mgr.Login(ctx, loginID, device, deviceId)
}

// LoginWithTimeout performs login with a custom token timeout. LoginWithTimeout 使用自定义过期时间登录。
func LoginWithTimeout(ctx context.Context, loginID string, timeout time.Duration, params ...string) (string, error) {
	device, deviceId, authType := parseDeviceAndAuthType(params...)
	mgr, err := GetManager(authType)
	if err != nil {
		return "", err
	}
	return mgr.LoginWithTimeout(ctx, loginID, timeout, device, deviceId)
}

// LoginWithRefreshToken logs in and returns access and refresh tokens. LoginWithRefreshToken 登录并返回访问令牌和刷新令牌。
func LoginWithRefreshToken(ctx context.Context, loginID string, params ...string) (*manager.RefreshTokenPair, error) {
	device, deviceId, authType := parseDeviceAndAuthType(params...)
	mgr, err := GetManager(authType)
	if err != nil {
		return nil, err
	}
	return mgr.LoginWithRefreshToken(ctx, loginID, device, deviceId)
}

// RefreshToken rotates a refresh token and returns a new token pair. RefreshToken 轮换刷新令牌并返回新的令牌对。
func RefreshToken(ctx context.Context, refreshToken string, authType ...string) (*manager.RefreshTokenPair, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.RefreshToken(ctx, refreshToken)
}

// RevokeRefreshToken revokes a refresh token and its related access token. RevokeRefreshToken 撤销刷新令牌及其关联访问令牌。
func RevokeRefreshToken(ctx context.Context, refreshToken string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.RevokeRefreshToken(ctx, refreshToken)
}

// GetRefreshTokenTTL returns refresh token remaining lifetime seconds. GetRefreshTokenTTL 返回刷新令牌剩余有效秒数。
func GetRefreshTokenTTL(ctx context.Context, refreshToken string, authType ...string) (int64, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return 0, err
	}
	return mgr.GetRefreshTokenTTL(ctx, refreshToken)
}

// IntrospectToken inspects token validity without renewal side effects. IntrospectToken 无续期副作用地检查令牌状态。
func IntrospectToken(ctx context.Context, tokenValue string, authType ...string) (*manager.TokenIntrospection, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.IntrospectToken(ctx, tokenValue)
}

// LoginByToken renews login state from an existing token. LoginByToken 基于已有 token 续期登录态。
func LoginByToken(ctx context.Context, tokenValue string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.LoginByToken(ctx, tokenValue)
}

// Logout logs out a token. Logout 注销指定 token。
func Logout(ctx context.Context, tokenValue string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.Logout(ctx, tokenValue)
}

// LogoutByDeviceAndDeviceId logs out a terminal by device and device ID. LogoutByDeviceAndDeviceId 按设备与设备 ID 注销终端。
func LogoutByDeviceAndDeviceId(ctx context.Context, loginID string, params ...string) error {
	device, deviceId, authType := parseDeviceAndAuthType(params...)
	mgr, err := GetManager(authType)
	if err != nil {
		return err
	}
	return mgr.LogoutByDeviceAndDeviceId(ctx, loginID, device, deviceId)
}

// LogoutByDevice logs out all terminals for a device. LogoutByDevice 注销指定设备下的所有终端。
func LogoutByDevice(ctx context.Context, loginID string, device string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.LogoutByDevice(ctx, loginID, device)
}

// LogoutByLoginID logs out all terminals for a login ID. LogoutByLoginID 注销指定账号的所有终端。
func LogoutByLoginID(ctx context.Context, loginID string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.LogoutByLoginID(ctx, loginID)
}

// Kickout marks a token as kicked out. Kickout 将指定 token 标记为踢下线。
func Kickout(ctx context.Context, tokenValue string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.Kickout(ctx, tokenValue)
}

// Replace marks a token as replaced. Replace 将指定 token 标记为顶下线。
func Replace(ctx context.Context, tokenValue string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.Replace(ctx, tokenValue)
}

// KickoutByDeviceAndDeviceId kicks out a terminal by device and device ID. KickoutByDeviceAndDeviceId 按设备与设备 ID 踢下线。
func KickoutByDeviceAndDeviceId(ctx context.Context, loginID string, params ...string) error {
	device, deviceId, authType := parseDeviceAndAuthType(params...)
	mgr, err := GetManager(authType)
	if err != nil {
		return err
	}
	return mgr.KickoutByDeviceAndDeviceId(ctx, loginID, device, deviceId)
}

// KickoutByDevice kicks out all terminals for a device. KickoutByDevice 踢下指定设备下的所有终端。
func KickoutByDevice(ctx context.Context, loginID string, device string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.KickoutByDevice(ctx, loginID, device)
}

// KickoutByLoginID kicks out all terminals for a login ID. KickoutByLoginID 踢下指定账号的所有终端。
func KickoutByLoginID(ctx context.Context, loginID string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.KickoutByLoginID(ctx, loginID)
}

// ReplaceByDeviceAndDeviceId replaces a terminal by device and device ID. ReplaceByDeviceAndDeviceId 按设备与设备 ID 顶下线。
func ReplaceByDeviceAndDeviceId(ctx context.Context, loginID string, params ...string) error {
	device, deviceId, authType := parseDeviceAndAuthType(params...)
	mgr, err := GetManager(authType)
	if err != nil {
		return err
	}
	return mgr.ReplaceByDeviceAndDeviceId(ctx, loginID, device, deviceId)
}

// ReplaceByDevice replaces all terminals for a device. ReplaceByDevice 顶下指定设备下的所有终端。
func ReplaceByDevice(ctx context.Context, loginID string, device string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.ReplaceByDevice(ctx, loginID, device)
}

// ReplaceByLoginID replaces all terminals for a login ID. ReplaceByLoginID 顶下指定账号的所有终端。
func ReplaceByLoginID(ctx context.Context, loginID string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.ReplaceByLoginID(ctx, loginID)
}

// IsLogin reports whether the token is logged in. IsLogin 判断 token 是否已登录。
func IsLogin(ctx context.Context, tokenValue string, authType ...string) bool {
	mgr, err := GetManager(authType...)
	if err != nil {
		return false
	}
	return mgr.IsLogin(ctx, tokenValue)
}

// CheckLogin validates login state. CheckLogin 校验登录态。
func CheckLogin(ctx context.Context, tokenValue string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.CheckLogin(ctx, tokenValue)
}

// GetLoginID returns the login ID bound to a token. GetLoginID 获取 token 绑定的登录 ID。
func GetLoginID(ctx context.Context, tokenValue string, authType ...string) (string, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return "", err
	}
	return mgr.GetLoginID(ctx, tokenValue)
}

// GetTokenInfo returns token metadata. GetTokenInfo 获取 token 元数据。
func GetTokenInfo(ctx context.Context, tokenValue string, authType ...string) (*manager.TokenInfo, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.GetTokenInfo(ctx, tokenValue)
}

// GetDevice returns the device bound to a token. GetDevice 获取 token 绑定的设备。
func GetDevice(ctx context.Context, tokenValue string, authType ...string) (string, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return "", err
	}
	return mgr.GetDevice(ctx, tokenValue)
}

// GetDeviceId returns the device ID bound to a token. GetDeviceId 获取 token 绑定的设备 ID。
func GetDeviceId(ctx context.Context, tokenValue string, authType ...string) (string, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return "", err
	}
	return mgr.GetDeviceId(ctx, tokenValue)
}

// GetTokenCreateTime returns token creation time. GetTokenCreateTime 获取 token 创建时间。
func GetTokenCreateTime(ctx context.Context, tokenValue string, authType ...string) (int64, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return 0, err
	}
	return mgr.GetTokenCreateTime(ctx, tokenValue)
}

// GetTokenTTL returns token TTL in seconds. GetTokenTTL 获取 token 剩余有效期秒数。
func GetTokenTTL(ctx context.Context, tokenValue string, authType ...string) (int64, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return 0, err
	}
	return mgr.GetTokenTTL(ctx, tokenValue)
}

// RenewTimeout renews token timeout. RenewTimeout 手动续期 token。
func RenewTimeout(ctx context.Context, tokenValue string, timeout time.Duration, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.RenewTimeout(ctx, tokenValue, timeout)
}
