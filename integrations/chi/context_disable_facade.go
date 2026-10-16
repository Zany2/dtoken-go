// @Author daixk 2026/06/05
package chi

import (
	"context"
	"time"

	"github.com/Zany2/dtoken-go/core/manager"
)

// CheckDisableByCtx checks current account disable state CheckDisableByCtx 校验当前账号封禁状态
func CheckDisableByCtx(ctx context.Context) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().CheckAccount(ctx)
}

// DisableServiceByCtx disables current account service DisableServiceByCtx 封禁当前账号服务
func DisableServiceByCtx(ctx context.Context, service string, duration time.Duration, reason ...string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().Service(ctx, service, duration, reason...)
}

// DisableServiceLevelByCtx disables current account service level DisableServiceLevelByCtx 按等级封禁当前账号服务
func DisableServiceLevelByCtx(ctx context.Context, service string, level int, duration time.Duration, reason ...string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().ServiceLevel(ctx, service, level, duration, reason...)
}

// UntieServiceByCtx removes current account service disable state UntieServiceByCtx 解除当前账号服务封禁状态
func UntieServiceByCtx(ctx context.Context, service string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().UntieService(ctx, service)
}

// IsDisableServiceByCtx checks current account service disable state IsDisableServiceByCtx 检查当前账号服务封禁状态
func IsDisableServiceByCtx(ctx context.Context, service string) bool {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return false
	}
	return dCtx.Disable().IsService(ctx, service)
}

// IsDisableServiceLevelByCtx checks current account service level disable state IsDisableServiceLevelByCtx 检查当前账号服务等级封禁状态
func IsDisableServiceLevelByCtx(ctx context.Context, service string, level int) bool {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return false
	}
	return dCtx.Disable().IsServiceLevel(ctx, service, level)
}

// CheckDisableServiceByCtx checks current account service disable state CheckDisableServiceByCtx 校验当前账号服务封禁状态
func CheckDisableServiceByCtx(ctx context.Context, services ...string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().CheckService(ctx, services...)
}

// CheckDisableServiceLevelByCtx checks current account service level disable state CheckDisableServiceLevelByCtx 校验当前账号服务等级封禁状态
func CheckDisableServiceLevelByCtx(ctx context.Context, service string, level int) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().CheckServiceLevel(ctx, service, level)
}

// GetDisableServiceInfoByCtx gets current account service disable info GetDisableServiceInfoByCtx 获取当前账号服务封禁信息
func GetDisableServiceInfoByCtx(ctx context.Context, service string) (*manager.ServiceDisableInfo, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Disable().GetServiceInfo(ctx, service)
}

// GetDisableServiceTTLByCtx gets current account service disable TTL GetDisableServiceTTLByCtx 获取当前账号服务封禁剩余时间
func GetDisableServiceTTLByCtx(ctx context.Context, service string) (int64, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return 0, err
	}
	return dCtx.Disable().GetServiceTTL(ctx, service)
}

// DisableDeviceByCtx disables current account device DisableDeviceByCtx 封禁当前账号设备
func DisableDeviceByCtx(ctx context.Context, device string, duration time.Duration, reason ...string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().Device(ctx, device, duration, reason...)
}

// DisableDeviceAndDeviceIDByCtx disables current account device ID DisableDeviceAndDeviceIDByCtx 按设备和设备 ID 封禁当前账号
func DisableDeviceAndDeviceIDByCtx(ctx context.Context, device, deviceID string, duration time.Duration, reason ...string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().DeviceAndDeviceID(ctx, device, deviceID, duration, reason...)
}

// UntieDeviceByCtx removes current account device disable state UntieDeviceByCtx 解除当前账号设备封禁状态
func UntieDeviceByCtx(ctx context.Context, device string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().UntieDevice(ctx, device)
}

// UntieDeviceAndDeviceIDByCtx removes current account device ID disable state UntieDeviceAndDeviceIDByCtx 解除当前账号指定设备 ID 的封禁状态
func UntieDeviceAndDeviceIDByCtx(ctx context.Context, device, deviceID string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().UntieDeviceAndDeviceID(ctx, device, deviceID)
}

// IsDisableDeviceByCtx checks current account device disable state IsDisableDeviceByCtx 检查当前账号设备封禁状态
func IsDisableDeviceByCtx(ctx context.Context, device string) bool {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return false
	}
	return dCtx.Disable().IsDevice(ctx, device)
}

// IsDisableDeviceAndDeviceIDByCtx checks current account device ID disable state IsDisableDeviceAndDeviceIDByCtx 检查当前账号设备 ID 封禁状态
func IsDisableDeviceAndDeviceIDByCtx(ctx context.Context, device, deviceID string) bool {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return false
	}
	return dCtx.Disable().IsDeviceAndDeviceID(ctx, device, deviceID)
}

// CheckDisableDeviceByCtx checks current account device disable state CheckDisableDeviceByCtx 校验当前账号设备封禁状态
func CheckDisableDeviceByCtx(ctx context.Context, device string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().CheckDevice(ctx, device)
}

// CheckDisableDeviceAndDeviceIDByCtx checks current account device ID disable state CheckDisableDeviceAndDeviceIDByCtx 校验当前账号设备 ID 封禁状态
func CheckDisableDeviceAndDeviceIDByCtx(ctx context.Context, device, deviceID string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().CheckDeviceAndDeviceID(ctx, device, deviceID)
}

// GetDisableDeviceInfoByCtx gets current account device disable info GetDisableDeviceInfoByCtx 获取当前账号设备封禁信息
func GetDisableDeviceInfoByCtx(ctx context.Context, device string) (*manager.DeviceDisableInfo, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Disable().GetDeviceInfo(ctx, device)
}

// GetDisableDeviceAndDeviceIDInfoByCtx gets current account device ID disable info GetDisableDeviceAndDeviceIDInfoByCtx 获取当前账号设备 ID 封禁信息
func GetDisableDeviceAndDeviceIDInfoByCtx(ctx context.Context, device, deviceID string) (*manager.DeviceDisableInfo, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Disable().GetDeviceAndDeviceIDInfo(ctx, device, deviceID)
}

// GetDisableDeviceTTLByCtx gets current account device disable TTL GetDisableDeviceTTLByCtx 获取当前账号设备封禁剩余时间
func GetDisableDeviceTTLByCtx(ctx context.Context, device string) (int64, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return 0, err
	}
	return dCtx.Disable().GetDeviceTTL(ctx, device)
}

// GetDisableDeviceAndDeviceIDTTLByCtx gets current account device ID disable TTL GetDisableDeviceAndDeviceIDTTLByCtx 获取当前账号设备 ID 封禁剩余时间
func GetDisableDeviceAndDeviceIDTTLByCtx(ctx context.Context, device, deviceID string) (int64, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return 0, err
	}
	return dCtx.Disable().GetDeviceAndDeviceIDTTL(ctx, device, deviceID)
}
