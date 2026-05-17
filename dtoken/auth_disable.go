// @Author daixk 2025/12/22 15:56:00
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

// DisableDevice disables a device type or concrete device. DisableDevice 封禁设备类型或具体设备。
func (a *Auth) DisableDevice(ctx context.Context, opts DeviceDisableOptions) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	if opts.DeviceID != "" {
		return mgr.DisableDeviceAndDeviceId(ctx, opts.LoginID, opts.Device, opts.DeviceID, opts.Duration, opts.Reason)
	}
	return mgr.DisableDevice(ctx, opts.LoginID, opts.Device, opts.Duration, opts.Reason)
}

// UntieService removes service disable state. UntieService 解除服务封禁状态。
func (a *Auth) UntieService(ctx context.Context, loginID, service string) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.UntieService(ctx, loginID, service)
}

// UntieDevice removes device type disable state. UntieDevice 解除设备类型封禁状态。
func (a *Auth) UntieDevice(ctx context.Context, loginID, device string) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.UntieDevice(ctx, loginID, device)
}

// UntieDeviceAndDeviceId removes concrete device disable state. UntieDeviceAndDeviceId 解除具体设备封禁状态。
func (a *Auth) UntieDeviceAndDeviceId(ctx context.Context, loginID, device, deviceId string) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.UntieDeviceAndDeviceId(ctx, loginID, device, deviceId)
}

// IsDisableDevice checks device type disable state. IsDisableDevice 检查设备类型封禁状态。
func (a *Auth) IsDisableDevice(ctx context.Context, loginID, device string) bool {
	mgr, err := a.requireManager()
	return err == nil && mgr.IsDisableDevice(ctx, loginID, device)
}

// IsDisableDeviceAndDeviceId checks concrete device disable state. IsDisableDeviceAndDeviceId 检查具体设备封禁状态。
func (a *Auth) IsDisableDeviceAndDeviceId(ctx context.Context, loginID, device, deviceId string) bool {
	mgr, err := a.requireManager()
	return err == nil && mgr.IsDisableDeviceAndDeviceId(ctx, loginID, device, deviceId)
}
