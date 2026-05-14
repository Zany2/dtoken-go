package dtoken

import "context"

// Disable disables an account. Disable 封禁账号。
func (a *Auth) Disable(ctx context.Context, opts DisableOptions) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.Disable(ctx, opts.LoginID, opts.Duration, opts.Reason)
}

// Untie removes account disable state. Untie 解除账号封禁状态。
func (a *Auth) Untie(ctx context.Context, loginID string) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.Untie(ctx, loginID)
}

// IsDisable checks account disable state. IsDisable 检查账号封禁状态。
func (a *Auth) IsDisable(ctx context.Context, loginID string) bool {
	mgr, err := a.requireManager()
	return err == nil && mgr.IsDisable(ctx, loginID)
}

// DisableService disables a service for an account. DisableService 封禁账号的指定服务。
func (a *Auth) DisableService(ctx context.Context, opts ServiceDisableOptions) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	if opts.Level > 0 {
		return mgr.DisableServiceLevel(ctx, opts.LoginID, opts.Service, opts.Level, opts.Duration, opts.Reason)
	}
	return mgr.DisableService(ctx, opts.LoginID, opts.Service, opts.Duration, opts.Reason)
}

// UntieService removes service disable state. UntieService 解除服务封禁状态。
func (a *Auth) UntieService(ctx context.Context, loginID, service string) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.UntieService(ctx, loginID, service)
}
