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
