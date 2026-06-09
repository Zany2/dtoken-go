// @Author daixk 2025/12/22 15:56:00
package dtoken

import (
	"context"
	"time"

	"github.com/Zany2/dtoken-go/core/manager"
)

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

// CheckDisable returns an error when an account is disabled. CheckDisable 校验账号是否被封禁。
func (a *Auth) CheckDisable(ctx context.Context, loginID string) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.CheckDisable(ctx, loginID)
}

// GetDisableInfo returns account disable information. GetDisableInfo 获取账号封禁信息。
func (a *Auth) GetDisableInfo(ctx context.Context, loginID string) (*manager.DisableInfo, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.GetDisableInfo(ctx, loginID)
}

// GetDisableTTL returns account disable TTL in seconds. GetDisableTTL 获取账号封禁剩余秒数。
func (a *Auth) GetDisableTTL(ctx context.Context, loginID string) (int64, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return 0, err
	}
	return mgr.GetDisableTTL(ctx, loginID)
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

// DisableServiceWithReason disables an account service with reason. DisableServiceWithReason 带原因封禁账号的指定服务。
func (a *Auth) DisableServiceWithReason(ctx context.Context, loginID, service string, duration time.Duration, reason string) error {
	return a.DisableService(ctx, ServiceDisableOptions{
		LoginID:  loginID,
		Service:  service,
		Duration: duration,
		Reason:   reason,
	})
}

// DisableServiceLevel disables an account service at a level. DisableServiceLevel 按等级封禁账号服务。
func (a *Auth) DisableServiceLevel(ctx context.Context, loginID, service string, level int, duration time.Duration) error {
	return a.DisableService(ctx, ServiceDisableOptions{
		LoginID:  loginID,
		Service:  service,
		Level:    level,
		Duration: duration,
	})
}

// DisableServiceLevelWithReason disables an account service at a level with reason. DisableServiceLevelWithReason 带原因按等级封禁账号服务。
func (a *Auth) DisableServiceLevelWithReason(ctx context.Context, loginID, service string, level int, duration time.Duration, reason string) error {
	return a.DisableService(ctx, ServiceDisableOptions{
		LoginID:  loginID,
		Service:  service,
		Level:    level,
		Duration: duration,
		Reason:   reason,
	})
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

// DisableDeviceWithReason disables a device type with reason. DisableDeviceWithReason 带原因封禁账号的指定设备类型。
func (a *Auth) DisableDeviceWithReason(ctx context.Context, loginID, device string, duration time.Duration, reason string) error {
	return a.DisableDevice(ctx, DeviceDisableOptions{
		LoginID:  loginID,
		Device:   device,
		Duration: duration,
		Reason:   reason,
	})
}

// DisableDeviceAndDeviceId disables a concrete device. DisableDeviceAndDeviceId 封禁账号的具体设备。
func (a *Auth) DisableDeviceAndDeviceId(ctx context.Context, loginID, device, deviceID string, duration time.Duration) error {
	return a.DisableDevice(ctx, DeviceDisableOptions{
		LoginID:  loginID,
		Device:   device,
		DeviceID: deviceID,
		Duration: duration,
	})
}

// DisableDeviceAndDeviceIdWithReason disables a concrete device with reason. DisableDeviceAndDeviceIdWithReason 带原因封禁账号的具体设备。
func (a *Auth) DisableDeviceAndDeviceIdWithReason(ctx context.Context, loginID, device, deviceID string, duration time.Duration, reason string) error {
	return a.DisableDevice(ctx, DeviceDisableOptions{
		LoginID:  loginID,
		Device:   device,
		DeviceID: deviceID,
		Duration: duration,
		Reason:   reason,
	})
}

// UntieService removes service disable state. UntieService 解除服务封禁状态。
func (a *Auth) UntieService(ctx context.Context, loginID, service string) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.UntieService(ctx, loginID, service)
}

// IsDisableService checks service disable state. IsDisableService 检查服务封禁状态。
func (a *Auth) IsDisableService(ctx context.Context, loginID, service string) bool {
	mgr, err := a.requireManager()
	return err == nil && mgr.IsDisableService(ctx, loginID, service)
}

// IsDisableServiceLevel checks service disable level. IsDisableServiceLevel 检查服务封禁等级。
func (a *Auth) IsDisableServiceLevel(ctx context.Context, loginID, service string, level int) bool {
	mgr, err := a.requireManager()
	return err == nil && mgr.IsDisableServiceLevel(ctx, loginID, service, level)
}

// CheckDisableService validates service disable state. CheckDisableService 校验服务封禁状态。
func (a *Auth) CheckDisableService(ctx context.Context, loginID string, services ...string) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.CheckDisableService(ctx, loginID, services...)
}

// CheckDisableServiceLevel validates service disable level. CheckDisableServiceLevel 校验服务封禁等级。
func (a *Auth) CheckDisableServiceLevel(ctx context.Context, loginID, service string, level int) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.CheckDisableServiceLevel(ctx, loginID, service, level)
}

// GetDisableServiceInfo returns service disable information. GetDisableServiceInfo 获取服务封禁信息。
func (a *Auth) GetDisableServiceInfo(ctx context.Context, loginID, service string) (*manager.ServiceDisableInfo, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.GetDisableServiceInfo(ctx, loginID, service)
}

// GetDisableServiceTTL returns service disable TTL in seconds. GetDisableServiceTTL 获取服务封禁剩余秒数。
func (a *Auth) GetDisableServiceTTL(ctx context.Context, loginID, service string) (int64, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return 0, err
	}
	return mgr.GetDisableServiceTTL(ctx, loginID, service)
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

// CheckDisableDevice validates device type disable state. CheckDisableDevice 校验设备类型封禁状态。
func (a *Auth) CheckDisableDevice(ctx context.Context, loginID, device string) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.CheckDisableDevice(ctx, loginID, device)
}

// CheckDisableDeviceAndDeviceId validates concrete device disable state. CheckDisableDeviceAndDeviceId 校验具体设备封禁状态。
func (a *Auth) CheckDisableDeviceAndDeviceId(ctx context.Context, loginID, device, deviceID string) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.CheckDisableDeviceAndDeviceId(ctx, loginID, device, deviceID)
}

// GetDisableDeviceInfo returns device type disable information. GetDisableDeviceInfo 获取设备类型封禁信息。
func (a *Auth) GetDisableDeviceInfo(ctx context.Context, loginID, device string) (*manager.DeviceDisableInfo, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.GetDisableDeviceInfo(ctx, loginID, device)
}

// GetDisableDeviceAndDeviceIdInfo returns concrete device disable information. GetDisableDeviceAndDeviceIdInfo 获取具体设备封禁信息。
func (a *Auth) GetDisableDeviceAndDeviceIdInfo(ctx context.Context, loginID, device, deviceID string) (*manager.DeviceDisableInfo, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.GetDisableDeviceAndDeviceIdInfo(ctx, loginID, device, deviceID)
}

// GetDisableDeviceTTL returns device type disable TTL in seconds. GetDisableDeviceTTL 获取设备类型封禁剩余秒数。
func (a *Auth) GetDisableDeviceTTL(ctx context.Context, loginID, device string) (int64, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return 0, err
	}
	return mgr.GetDisableDeviceTTL(ctx, loginID, device)
}

// GetDisableDeviceAndDeviceIdTTL returns concrete device disable TTL in seconds. GetDisableDeviceAndDeviceIdTTL 获取具体设备封禁剩余秒数。
func (a *Auth) GetDisableDeviceAndDeviceIdTTL(ctx context.Context, loginID, device, deviceID string) (int64, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return 0, err
	}
	return mgr.GetDisableDeviceAndDeviceIdTTL(ctx, loginID, device, deviceID)
}
