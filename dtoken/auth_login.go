// @Author daixk 2025/12/22 15:56:00
package dtoken

import (
	"context"
	"time"

	"github.com/Zany2/dtoken-go/core/manager"
)

// Login performs login with typed options. Login 使用类型化选项执行登录。
func (a *Auth) Login(ctx context.Context, opts LoginOptions) (string, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return "", err
	}
	return mgr.LoginWithOptions(ctx, manager.LoginOptions{
		LoginID:               opts.LoginID,
		Device:                opts.Device,
		DeviceID:              opts.DeviceID,
		Timeout:               opts.Timeout,
		ActiveTimeout:         opts.ActiveTimeout,
		Token:                 opts.Token,
		Extra:                 opts.Extra,
		TerminalExtra:         opts.TerminalExtra,
		IsConcurrent:          opts.IsConcurrent,
		IsShare:               opts.IsShare,
		MaxLoginCount:         opts.MaxLoginCount,
		ReplacedLoginExitMode: opts.ReplacedLoginExitMode,
		OverflowLogoutMode:    opts.OverflowLogoutMode,
	})
}

// LoginID logs in with only a subject id. LoginID 仅使用主体 ID 登录。
func (a *Auth) LoginID(ctx context.Context, loginID string) (string, error) {
	return a.Login(ctx, LoginOptions{LoginID: loginID})
}

// LoginWithTimeout logs in with a custom timeout. LoginWithTimeout 使用自定义过期时间登录。
func (a *Auth) LoginWithTimeout(ctx context.Context, loginID string, timeout time.Duration) (string, error) {
	return a.Login(ctx, LoginOptions{LoginID: loginID, Timeout: timeout})
}

// LoginWithRefreshToken logs in and returns access and refresh tokens. LoginWithRefreshToken 登录并返回访问令牌和刷新令牌。
func (a *Auth) LoginWithRefreshToken(ctx context.Context, loginID string) (*manager.RefreshTokenPair, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.LoginWithRefreshToken(ctx, loginID)
}

// LoginWithRefreshTokenOptions logs in with options and returns token pair. LoginWithRefreshTokenOptions 使用选项登录并返回令牌对。
func (a *Auth) LoginWithRefreshTokenOptions(ctx context.Context, opts RefreshTokenOptions) (*manager.RefreshTokenPair, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.LoginWithRefreshTokenOptions(ctx, manager.RefreshTokenOptions{
		LoginOptions: manager.LoginOptions{
			LoginID:               opts.LoginID,
			Device:                opts.Device,
			DeviceID:              opts.DeviceID,
			Timeout:               opts.Timeout,
			ActiveTimeout:         opts.ActiveTimeout,
			Token:                 opts.Token,
			Extra:                 opts.Extra,
			TerminalExtra:         opts.TerminalExtra,
			IsConcurrent:          opts.IsConcurrent,
			IsShare:               opts.IsShare,
			MaxLoginCount:         opts.MaxLoginCount,
			ReplacedLoginExitMode: opts.ReplacedLoginExitMode,
			OverflowLogoutMode:    opts.OverflowLogoutMode,
		},
		RefreshTimeout: opts.RefreshTimeout,
	})
}

// RefreshToken rotates a refresh token and returns a new token pair. RefreshToken 轮换刷新令牌并返回新的令牌对。
func (a *Auth) RefreshToken(ctx context.Context, refreshToken string) (*manager.RefreshTokenPair, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.RefreshToken(ctx, refreshToken)
}

// RevokeRefreshToken revokes a refresh token and its related access token. RevokeRefreshToken 撤销刷新令牌及其关联访问令牌。
func (a *Auth) RevokeRefreshToken(ctx context.Context, refreshToken string) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.RevokeRefreshToken(ctx, refreshToken)
}

// GetRefreshTokenTTL returns refresh token remaining lifetime seconds. GetRefreshTokenTTL 返回刷新令牌剩余有效秒数。
func (a *Auth) GetRefreshTokenTTL(ctx context.Context, refreshToken string) (int64, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return 0, err
	}
	return mgr.GetRefreshTokenTTL(ctx, refreshToken)
}

// IntrospectToken inspects token validity without renewal side effects. IntrospectToken 无续期副作用地检查令牌状态。
func (a *Auth) IntrospectToken(ctx context.Context, token string) (*manager.TokenIntrospection, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.IntrospectToken(ctx, token)
}

// LoginByToken renews login state by an existing token. LoginByToken 根据已有 Token 续期登录态。
func (a *Auth) LoginByToken(ctx context.Context, token string) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.LoginByToken(ctx, token)
}

// LogoutByToken logs out a terminal by token. LogoutByToken 根据 Token 登出终端。
func (a *Auth) LogoutByToken(ctx context.Context, token string) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.Logout(ctx, token)
}

// LogoutByDeviceAndDeviceId logs out a concrete terminal. LogoutByDeviceAndDeviceId 注销具体终端。
func (a *Auth) LogoutByDeviceAndDeviceId(ctx context.Context, loginID, device, deviceID string) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.LogoutByDeviceAndDeviceId(ctx, loginID, device, deviceID)
}

// LogoutByDevice logs out all terminals on a device type. LogoutByDevice 注销指定设备类型下的所有终端。
func (a *Auth) LogoutByDevice(ctx context.Context, loginID, device string) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.LogoutByDevice(ctx, loginID, device)
}

// LogoutByLoginID logs out all terminals for a login ID. LogoutByLoginID 注销指定账号的所有终端。
func (a *Auth) LogoutByLoginID(ctx context.Context, loginID string) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.LogoutByLoginID(ctx, loginID)
}

// Logout logs out by typed terminal options. Logout 根据类型化终端选项登出。
func (a *Auth) Logout(ctx context.Context, opts LogoutOptions) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.Terminate(ctx, manager.TerminateOptions{
		Action:   manager.TerminateActionLogout,
		LoginID:  opts.LoginID,
		Token:    opts.Token,
		Device:   opts.Device,
		DeviceID: opts.DeviceID,
	})
}

// KickoutByToken kicks out a terminal by token. KickoutByToken 根据 Token 踢人下线。
func (a *Auth) KickoutByToken(ctx context.Context, token string) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.Kickout(ctx, token)
}

// KickoutByDeviceAndDeviceId kicks out a concrete terminal. KickoutByDeviceAndDeviceId 踢下具体终端。
func (a *Auth) KickoutByDeviceAndDeviceId(ctx context.Context, loginID, device, deviceID string) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.KickoutByDeviceAndDeviceId(ctx, loginID, device, deviceID)
}

// KickoutByDevice kicks out all terminals on a device type. KickoutByDevice 踢下指定设备类型下的所有终端。
func (a *Auth) KickoutByDevice(ctx context.Context, loginID, device string) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.KickoutByDevice(ctx, loginID, device)
}

// KickoutByLoginID kicks out all terminals for a login ID. KickoutByLoginID 踢下指定账号的所有终端。
func (a *Auth) KickoutByLoginID(ctx context.Context, loginID string) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.KickoutByLoginID(ctx, loginID)
}

// Kickout kicks out by typed terminal options. Kickout 根据类型化终端选项踢人下线。
func (a *Auth) Kickout(ctx context.Context, opts LogoutOptions) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.Terminate(ctx, manager.TerminateOptions{
		Action:   manager.TerminateActionKickout,
		LoginID:  opts.LoginID,
		Token:    opts.Token,
		Device:   opts.Device,
		DeviceID: opts.DeviceID,
	})
}

// ReplaceByToken replaces a terminal by token. ReplaceByToken 根据 Token 顶人下线。
func (a *Auth) ReplaceByToken(ctx context.Context, token string) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.Replace(ctx, token)
}

// ReplaceByDeviceAndDeviceId replaces a concrete terminal. ReplaceByDeviceAndDeviceId 顶下具体终端。
func (a *Auth) ReplaceByDeviceAndDeviceId(ctx context.Context, loginID, device, deviceID string) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.ReplaceByDeviceAndDeviceId(ctx, loginID, device, deviceID)
}

// ReplaceByDevice replaces all terminals on a device type. ReplaceByDevice 顶下指定设备类型下的所有终端。
func (a *Auth) ReplaceByDevice(ctx context.Context, loginID, device string) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.ReplaceByDevice(ctx, loginID, device)
}

// ReplaceByLoginID replaces all terminals for a login ID. ReplaceByLoginID 顶下指定账号的所有终端。
func (a *Auth) ReplaceByLoginID(ctx context.Context, loginID string) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.ReplaceByLoginID(ctx, loginID)
}

// Replace replaces login state by typed terminal options. Replace 根据类型化终端选项顶人下线。
func (a *Auth) Replace(ctx context.Context, opts LogoutOptions) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.Terminate(ctx, manager.TerminateOptions{
		Action:   manager.TerminateActionReplace,
		LoginID:  opts.LoginID,
		Token:    opts.Token,
		Device:   opts.Device,
		DeviceID: opts.DeviceID,
	})
}

// IsLogin checks login status by token. IsLogin 根据 Token 检查登录状态。
func (a *Auth) IsLogin(ctx context.Context, token string) bool {
	mgr, err := a.requireManager()
	return err == nil && mgr.IsLogin(ctx, token)
}

// CheckLogin checks login status by token. CheckLogin 根据 Token 校验登录状态。
func (a *Auth) CheckLogin(ctx context.Context, token string) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.CheckLogin(ctx, token)
}

// GetLoginID resolves login id from token. GetLoginID 根据 Token 解析登录 ID。
func (a *Auth) GetLoginID(ctx context.Context, token string) (string, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return "", err
	}
	return mgr.GetLoginID(ctx, token)
}

// GetTokenInfo resolves token metadata. GetTokenInfo 获取 Token 元信息。
func (a *Auth) GetTokenInfo(ctx context.Context, token string) (*manager.TokenInfo, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.GetTokenInfo(ctx, token)
}

// GetDevice resolves device from token. GetDevice 根据 Token 解析设备类型。
func (a *Auth) GetDevice(ctx context.Context, token string) (string, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return "", err
	}
	return mgr.GetDevice(ctx, token)
}

// GetDeviceId resolves device id from token. GetDeviceId 根据 Token 解析设备 ID。
func (a *Auth) GetDeviceId(ctx context.Context, token string) (string, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return "", err
	}
	return mgr.GetDeviceId(ctx, token)
}

// GetTokenCreateTime resolves token creation time. GetTokenCreateTime 获取 Token 创建时间。
func (a *Auth) GetTokenCreateTime(ctx context.Context, token string) (int64, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return 0, err
	}
	return mgr.GetTokenCreateTime(ctx, token)
}

// GetTokenTTL gets the remaining token lifetime in seconds. GetTokenTTL 获取 Token 剩余有效秒数。
func (a *Auth) GetTokenTTL(ctx context.Context, token string) (int64, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return 0, err
	}
	return mgr.GetTokenTTL(ctx, token)
}

// RenewTimeout renews token timeout. RenewTimeout 续期 Token 过期时间。
func (a *Auth) RenewTimeout(ctx context.Context, token string, timeout time.Duration) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.RenewTimeout(ctx, token, timeout)
}
