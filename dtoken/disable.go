// @Author daixk 2025/12/22 15:56:00
package dtoken

import (
	"context"
	"time"

	"github.com/Zany2/dtoken-go/core/manager"
)

// Disable disables an account for a duration. Disable 按时长封禁账号。
func Disable(ctx context.Context, loginID string, duration time.Duration, reason string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.Disable(ctx, loginID, duration, reason)
}

// Untie removes account disable state. Untie 解除账号封禁状态。
func Untie(ctx context.Context, loginID string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.Untie(ctx, loginID)
}

// IsDisable reports whether an account is disabled. IsDisable 判断账号是否被封禁。
func IsDisable(ctx context.Context, loginID string, authType ...string) bool {
	mgr, err := GetManager(authType...)
	if err != nil {
		return false
	}
	return mgr.IsDisable(ctx, loginID)
}

// CheckDisable returns an error when an account is disabled. CheckDisable 校验账号是否被封禁。
func CheckDisable(ctx context.Context, loginID string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.CheckDisable(ctx, loginID)
}

// GetDisableInfo returns account disable information. GetDisableInfo 获取账号封禁信息。
func GetDisableInfo(ctx context.Context, loginID string, authType ...string) (*manager.DisableInfo, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.GetDisableInfo(ctx, loginID)
}

// GetDisableTTL returns account disable TTL in seconds. GetDisableTTL 获取账号封禁剩余秒数。
func GetDisableTTL(ctx context.Context, loginID string, authType ...string) (int64, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return 0, err
	}
	return mgr.GetDisableTTL(ctx, loginID)
}

// DisableService disables an account service. DisableService 封禁账号的指定服务。
func DisableService(ctx context.Context, loginID, service string, duration time.Duration, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.DisableService(ctx, loginID, service, duration)
}

// DisableServiceWithReason disables an account service with reason. DisableServiceWithReason 带原因封禁账号的指定服务。
func DisableServiceWithReason(ctx context.Context, loginID, service string, duration time.Duration, reason string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.DisableService(ctx, loginID, service, duration, reason)
}

// DisableServiceLevel disables an account service at a level. DisableServiceLevel 按等级封禁账号服务。
func DisableServiceLevel(ctx context.Context, loginID, service string, level int, duration time.Duration, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.DisableServiceLevel(ctx, loginID, service, level, duration)
}

// DisableServiceLevelWithReason disables an account service at a level with reason. DisableServiceLevelWithReason 带原因按等级封禁账号服务。
func DisableServiceLevelWithReason(ctx context.Context, loginID, service string, level int, duration time.Duration, reason string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.DisableServiceLevel(ctx, loginID, service, level, duration, reason)
}

// UntieService removes service disable state. UntieService 解除服务封禁状态。
func UntieService(ctx context.Context, loginID, service string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.UntieService(ctx, loginID, service)
}

// IsDisableService reports whether a service is disabled. IsDisableService 判断服务是否被封禁。
func IsDisableService(ctx context.Context, loginID, service string, authType ...string) bool {
	mgr, err := GetManager(authType...)
	if err != nil {
		return false
	}
	return mgr.IsDisableService(ctx, loginID, service)
}

// IsDisableServiceLevel reports whether a service reaches a disable level. IsDisableServiceLevel 判断服务封禁是否达到指定等级。
func IsDisableServiceLevel(ctx context.Context, loginID, service string, level int, authType ...string) bool {
	mgr, err := GetManager(authType...)
	if err != nil {
		return false
	}
	return mgr.IsDisableServiceLevel(ctx, loginID, service, level)
}

// CheckDisableService validates service disable state. CheckDisableService 校验服务封禁状态。
func CheckDisableService(ctx context.Context, loginID string, services []string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.CheckDisableService(ctx, loginID, services...)
}

// CheckDisableServiceLevel validates service disable level. CheckDisableServiceLevel 校验服务封禁等级。
func CheckDisableServiceLevel(ctx context.Context, loginID, service string, level int, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.CheckDisableServiceLevel(ctx, loginID, service, level)
}

// GetDisableServiceInfo returns service disable information. GetDisableServiceInfo 获取服务封禁信息。
func GetDisableServiceInfo(ctx context.Context, loginID, service string, authType ...string) (*manager.ServiceDisableInfo, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.GetDisableServiceInfo(ctx, loginID, service)
}

// GetDisableServiceTTL returns service disable TTL in seconds. GetDisableServiceTTL 获取服务封禁剩余秒数。
func GetDisableServiceTTL(ctx context.Context, loginID, service string, authType ...string) (int64, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return 0, err
	}
	return mgr.GetDisableServiceTTL(ctx, loginID, service)
}

// DisableDevice disables a device type for an account. DisableDevice 封禁账号的指定设备类型。
func DisableDevice(ctx context.Context, loginID, device string, duration time.Duration, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.DisableDevice(ctx, loginID, device, duration)
}

// DisableDeviceWithReason disables a device type with reason. DisableDeviceWithReason 带原因封禁账号的指定设备类型。
func DisableDeviceWithReason(ctx context.Context, loginID, device string, duration time.Duration, reason string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.DisableDevice(ctx, loginID, device, duration, reason)
}

// DisableDeviceAndDeviceId disables a concrete device for an account. DisableDeviceAndDeviceId 封禁账号的具体设备。
func DisableDeviceAndDeviceId(ctx context.Context, loginID, device, deviceId string, duration time.Duration, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.DisableDeviceAndDeviceId(ctx, loginID, device, deviceId, duration)
}

// DisableDeviceAndDeviceIdWithReason disables a concrete device with reason. DisableDeviceAndDeviceIdWithReason 带原因封禁账号的具体设备。
func DisableDeviceAndDeviceIdWithReason(ctx context.Context, loginID, device, deviceId string, duration time.Duration, reason string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.DisableDeviceAndDeviceId(ctx, loginID, device, deviceId, duration, reason)
}

// UntieDevice removes device type disable state. UntieDevice 解除设备类型封禁状态。
func UntieDevice(ctx context.Context, loginID, device string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.UntieDevice(ctx, loginID, device)
}

// UntieDeviceAndDeviceId removes concrete device disable state. UntieDeviceAndDeviceId 解除具体设备封禁状态。
func UntieDeviceAndDeviceId(ctx context.Context, loginID, device, deviceId string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.UntieDeviceAndDeviceId(ctx, loginID, device, deviceId)
}

// IsDisableDevice reports whether a device type is disabled. IsDisableDevice 判断设备类型是否被封禁。
func IsDisableDevice(ctx context.Context, loginID, device string, authType ...string) bool {
	mgr, err := GetManager(authType...)
	if err != nil {
		return false
	}
	return mgr.IsDisableDevice(ctx, loginID, device)
}

// IsDisableDeviceAndDeviceId reports whether a concrete device is disabled. IsDisableDeviceAndDeviceId 判断具体设备是否被封禁。
func IsDisableDeviceAndDeviceId(ctx context.Context, loginID, device, deviceId string, authType ...string) bool {
	mgr, err := GetManager(authType...)
	if err != nil {
		return false
	}
	return mgr.IsDisableDeviceAndDeviceId(ctx, loginID, device, deviceId)
}

// CheckDisableDevice validates device type disable state. CheckDisableDevice 校验设备类型封禁状态。
func CheckDisableDevice(ctx context.Context, loginID, device string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.CheckDisableDevice(ctx, loginID, device)
}

// CheckDisableDeviceAndDeviceId validates concrete device disable state. CheckDisableDeviceAndDeviceId 校验具体设备封禁状态。
func CheckDisableDeviceAndDeviceId(ctx context.Context, loginID, device, deviceId string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.CheckDisableDeviceAndDeviceId(ctx, loginID, device, deviceId)
}

// GetDisableDeviceInfo returns device type disable information. GetDisableDeviceInfo 获取设备类型封禁信息。
func GetDisableDeviceInfo(ctx context.Context, loginID, device string, authType ...string) (*manager.DeviceDisableInfo, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.GetDisableDeviceInfo(ctx, loginID, device)
}

// GetDisableDeviceAndDeviceIdInfo returns concrete device disable information. GetDisableDeviceAndDeviceIdInfo 获取具体设备封禁信息。
func GetDisableDeviceAndDeviceIdInfo(ctx context.Context, loginID, device, deviceId string, authType ...string) (*manager.DeviceDisableInfo, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.GetDisableDeviceAndDeviceIdInfo(ctx, loginID, device, deviceId)
}

// GetDisableDeviceTTL returns device type disable TTL in seconds. GetDisableDeviceTTL 获取设备类型封禁剩余秒数。
func GetDisableDeviceTTL(ctx context.Context, loginID, device string, authType ...string) (int64, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return 0, err
	}
	return mgr.GetDisableDeviceTTL(ctx, loginID, device)
}

// GetDisableDeviceAndDeviceIdTTL returns concrete device disable TTL in seconds. GetDisableDeviceAndDeviceIdTTL 获取具体设备封禁剩余秒数。
func GetDisableDeviceAndDeviceIdTTL(ctx context.Context, loginID, device, deviceId string, authType ...string) (int64, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return 0, err
	}
	return mgr.GetDisableDeviceAndDeviceIdTTL(ctx, loginID, device, deviceId)
}
