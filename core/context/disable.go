// @Author daixk 2026/06/05
package context

import (
	"context"
	"time"

	"github.com/Zany2/dtoken-go/core/manager"
)

// Account disables current account Account 封禁当前账号
func (c *DisableContext) Account(ctx context.Context, duration time.Duration, reason ...string) error {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return err
	}
	return c.d.manager.Disable(ctx, loginID, duration, reason...)
}

// UntieAccount removes current account disable state UntieAccount 解封当前账号
func (c *DisableContext) UntieAccount(ctx context.Context) error {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return err
	}
	return c.d.manager.Untie(ctx, loginID)
}

// IsAccount checks current account disable state IsAccount 检查当前账号是否被封禁
func (c *DisableContext) IsAccount(ctx context.Context) bool {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return false
	}
	return c.d.manager.IsDisable(ctx, loginID)
}

// CheckAccount checks current account disable state with error CheckAccount 校验当前账号封禁状态
func (c *DisableContext) CheckAccount(ctx context.Context) error {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return err
	}
	return c.d.manager.CheckDisable(ctx, loginID)
}

// AccountInfo gets current account disable info AccountInfo 获取当前账号封禁信息
func (c *DisableContext) AccountInfo(ctx context.Context) (*manager.DisableInfo, error) {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return nil, err
	}
	return c.d.manager.GetDisableInfo(ctx, loginID)
}

// AccountTTL gets current account disable TTL AccountTTL 获取当前账号封禁剩余时间
func (c *DisableContext) AccountTTL(ctx context.Context) (int64, error) {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return 0, err
	}
	return c.d.manager.GetDisableTTL(ctx, loginID)
}

// Service disables a service for current account Service 封禁当前账号的指定服务
func (c *DisableContext) Service(ctx context.Context, service string, duration time.Duration, reason ...string) error {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return err
	}
	return c.d.manager.DisableService(ctx, loginID, service, duration, reason...)
}

// ServiceLevel disables a service with level ServiceLevel 按等级封禁当前账号的指定服务
func (c *DisableContext) ServiceLevel(ctx context.Context, service string, level int, duration time.Duration, reason ...string) error {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return err
	}
	return c.d.manager.DisableServiceLevel(ctx, loginID, service, level, duration, reason...)
}

// UntieService removes service disable state UntieService 解封当前账号的指定服务
func (c *DisableContext) UntieService(ctx context.Context, service string) error {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return err
	}
	return c.d.manager.UntieService(ctx, loginID, service)
}

// IsService checks service disable state IsService 检查当前账号指定服务是否被封禁
func (c *DisableContext) IsService(ctx context.Context, service string) bool {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return false
	}
	return c.d.manager.IsDisableService(ctx, loginID, service)
}

// IsServiceLevel checks service disable level IsServiceLevel 检查当前账号指定服务是否达到封禁等级
func (c *DisableContext) IsServiceLevel(ctx context.Context, service string, level int) bool {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return false
	}
	return c.d.manager.IsDisableServiceLevel(ctx, loginID, service, level)
}

// CheckService checks service disable state with error CheckService 校验当前账号指定服务封禁状态
func (c *DisableContext) CheckService(ctx context.Context, services ...string) error {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return err
	}
	return c.d.manager.CheckDisableService(ctx, loginID, services...)
}

// CheckServiceLevel checks service disable level with error CheckServiceLevel 校验当前账号指定服务封禁等级
func (c *DisableContext) CheckServiceLevel(ctx context.Context, service string, level int) error {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return err
	}
	return c.d.manager.CheckDisableServiceLevel(ctx, loginID, service, level)
}

// GetServiceInfo gets service disable info GetServiceInfo 获取当前账号指定服务封禁信息
func (c *DisableContext) GetServiceInfo(ctx context.Context, service string) (*manager.ServiceDisableInfo, error) {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return nil, err
	}
	return c.d.manager.GetDisableServiceInfo(ctx, loginID, service)
}

// GetServiceTTL gets service disable TTL GetServiceTTL 获取当前账号指定服务封禁剩余时间
func (c *DisableContext) GetServiceTTL(ctx context.Context, service string) (int64, error) {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return 0, err
	}
	return c.d.manager.GetDisableServiceTTL(ctx, loginID, service)
}

// Device disables current account device type Device 封禁当前账号的设备类型
func (c *DisableContext) Device(ctx context.Context, device string, duration time.Duration, reason ...string) error {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return err
	}
	return c.d.manager.DisableDevice(ctx, loginID, device, duration, reason...)
}

// DeviceAndDeviceId disables current account concrete device DeviceAndDeviceId 封禁当前账号的具体设备
func (c *DisableContext) DeviceAndDeviceId(ctx context.Context, device, deviceId string, duration time.Duration, reason ...string) error {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return err
	}
	return c.d.manager.DisableDeviceAndDeviceId(ctx, loginID, device, deviceId, duration, reason...)
}

// UntieDevice removes current account device type disable state UntieDevice 解封当前账号的设备类型
func (c *DisableContext) UntieDevice(ctx context.Context, device string) error {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return err
	}
	return c.d.manager.UntieDevice(ctx, loginID, device)
}

// UntieDeviceAndDeviceId removes current account concrete device disable state UntieDeviceAndDeviceId 解封当前账号的具体设备
func (c *DisableContext) UntieDeviceAndDeviceId(ctx context.Context, device, deviceId string) error {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return err
	}
	return c.d.manager.UntieDeviceAndDeviceId(ctx, loginID, device, deviceId)
}

// IsDevice checks current account device type disable state IsDevice 检查当前账号设备类型封禁状态
func (c *DisableContext) IsDevice(ctx context.Context, device string) bool {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return false
	}
	return c.d.manager.IsDisableDevice(ctx, loginID, device)
}

// IsDeviceAndDeviceId checks current account concrete device disable state IsDeviceAndDeviceId 检查当前账号具体设备封禁状态
func (c *DisableContext) IsDeviceAndDeviceId(ctx context.Context, device, deviceId string) bool {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return false
	}
	return c.d.manager.IsDisableDeviceAndDeviceId(ctx, loginID, device, deviceId)
}

// CheckDevice checks current account device type disable state CheckDevice 校验当前账号设备类型封禁状态
func (c *DisableContext) CheckDevice(ctx context.Context, device string) error {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return err
	}
	return c.d.manager.CheckDisableDevice(ctx, loginID, device)
}

// CheckDeviceAndDeviceId checks current account concrete device disable state CheckDeviceAndDeviceId 校验当前账号具体设备封禁状态
func (c *DisableContext) CheckDeviceAndDeviceId(ctx context.Context, device, deviceId string) error {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return err
	}
	return c.d.manager.CheckDisableDeviceAndDeviceId(ctx, loginID, device, deviceId)
}

// GetDeviceInfo gets current account device type disable info GetDeviceInfo 获取当前账号设备类型封禁信息
func (c *DisableContext) GetDeviceInfo(ctx context.Context, device string) (*manager.DeviceDisableInfo, error) {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return nil, err
	}
	return c.d.manager.GetDisableDeviceInfo(ctx, loginID, device)
}

// GetDeviceAndDeviceIdInfo gets current account concrete device disable info GetDeviceAndDeviceIdInfo 获取当前账号具体设备封禁信息
func (c *DisableContext) GetDeviceAndDeviceIdInfo(ctx context.Context, device, deviceId string) (*manager.DeviceDisableInfo, error) {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return nil, err
	}
	return c.d.manager.GetDisableDeviceAndDeviceIdInfo(ctx, loginID, device, deviceId)
}

// GetDeviceTTL gets current account device type disable TTL GetDeviceTTL 获取当前账号设备类型封禁剩余时间
func (c *DisableContext) GetDeviceTTL(ctx context.Context, device string) (int64, error) {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return 0, err
	}
	return c.d.manager.GetDisableDeviceTTL(ctx, loginID, device)
}

// GetDeviceAndDeviceIdTTL gets current account concrete device disable TTL GetDeviceAndDeviceIdTTL 获取当前账号具体设备封禁剩余时间
func (c *DisableContext) GetDeviceAndDeviceIdTTL(ctx context.Context, device, deviceId string) (int64, error) {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return 0, err
	}
	return c.d.manager.GetDisableDeviceAndDeviceIdTTL(ctx, loginID, device, deviceId)
}
