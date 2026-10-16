// @Author daixk 2026/06/05
package hertz

import (
	"time"

	"github.com/Zany2/dtoken-go/core/manager"
	hertzapp "github.com/cloudwego/hertz/pkg/app"
)

// CheckDisableByContext checks current account disable state CheckDisableByContext 校验当前账号封禁状态
func CheckDisableByContext(ctx *hertzapp.RequestContext) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().CheckAccount(requestContext(ctx))
}

// DisableServiceByContext disables current account service DisableServiceByContext 封禁当前账号服务
func DisableServiceByContext(ctx *hertzapp.RequestContext, service string, duration time.Duration, reason ...string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().Service(requestContext(ctx), service, duration, reason...)
}

// DisableServiceLevelByContext disables current account service level DisableServiceLevelByContext 按等级封禁当前账号服务
func DisableServiceLevelByContext(ctx *hertzapp.RequestContext, service string, level int, duration time.Duration, reason ...string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().ServiceLevel(requestContext(ctx), service, level, duration, reason...)
}

// UntieServiceByContext removes current account service disable state UntieServiceByContext 解除当前账号服务封禁状态
func UntieServiceByContext(ctx *hertzapp.RequestContext, service string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().UntieService(requestContext(ctx), service)
}

// IsDisableServiceByContext checks current account service disable state IsDisableServiceByContext 检查当前账号服务封禁状态
func IsDisableServiceByContext(ctx *hertzapp.RequestContext, service string) bool {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return false
	}
	return dCtx.Disable().IsService(requestContext(ctx), service)
}

// IsDisableServiceLevelByContext checks current account service level disable state IsDisableServiceLevelByContext 检查当前账号服务等级封禁状态
func IsDisableServiceLevelByContext(ctx *hertzapp.RequestContext, service string, level int) bool {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return false
	}
	return dCtx.Disable().IsServiceLevel(requestContext(ctx), service, level)
}

// CheckDisableServiceByContext checks current account service disable state CheckDisableServiceByContext 校验当前账号服务封禁状态
func CheckDisableServiceByContext(ctx *hertzapp.RequestContext, services ...string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().CheckService(requestContext(ctx), services...)
}

// CheckDisableServiceLevelByContext checks current account service level disable state CheckDisableServiceLevelByContext 校验当前账号服务等级封禁状态
func CheckDisableServiceLevelByContext(ctx *hertzapp.RequestContext, service string, level int) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().CheckServiceLevel(requestContext(ctx), service, level)
}

// GetDisableServiceInfoByContext gets current account service disable info GetDisableServiceInfoByContext 获取当前账号服务封禁信息
func GetDisableServiceInfoByContext(ctx *hertzapp.RequestContext, service string) (*manager.ServiceDisableInfo, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Disable().GetServiceInfo(requestContext(ctx), service)
}

// GetDisableServiceTTLByContext gets current account service disable TTL GetDisableServiceTTLByContext 获取当前账号服务封禁剩余时间
func GetDisableServiceTTLByContext(ctx *hertzapp.RequestContext, service string) (int64, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return 0, err
	}
	return dCtx.Disable().GetServiceTTL(requestContext(ctx), service)
}

// DisableDeviceByContext disables current account device DisableDeviceByContext 封禁当前账号设备
func DisableDeviceByContext(ctx *hertzapp.RequestContext, device string, duration time.Duration, reason ...string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().Device(requestContext(ctx), device, duration, reason...)
}

// DisableDeviceAndDeviceIDByContext disables current account device ID DisableDeviceAndDeviceIDByContext 按设备和设备 ID 封禁当前账号
func DisableDeviceAndDeviceIDByContext(ctx *hertzapp.RequestContext, device, deviceID string, duration time.Duration, reason ...string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().DeviceAndDeviceID(requestContext(ctx), device, deviceID, duration, reason...)
}

// UntieDeviceByContext removes current account device disable state UntieDeviceByContext 解除当前账号设备封禁状态
func UntieDeviceByContext(ctx *hertzapp.RequestContext, device string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().UntieDevice(requestContext(ctx), device)
}

// UntieDeviceAndDeviceIDByContext removes current account device ID disable state UntieDeviceAndDeviceIDByContext 解除当前账号指定设备 ID 的封禁状态
func UntieDeviceAndDeviceIDByContext(ctx *hertzapp.RequestContext, device, deviceID string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().UntieDeviceAndDeviceID(requestContext(ctx), device, deviceID)
}

// IsDisableDeviceByContext checks current account device disable state IsDisableDeviceByContext 检查当前账号设备封禁状态
func IsDisableDeviceByContext(ctx *hertzapp.RequestContext, device string) bool {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return false
	}
	return dCtx.Disable().IsDevice(requestContext(ctx), device)
}

// IsDisableDeviceAndDeviceIDByContext checks current account device ID disable state IsDisableDeviceAndDeviceIDByContext 检查当前账号设备 ID 封禁状态
func IsDisableDeviceAndDeviceIDByContext(ctx *hertzapp.RequestContext, device, deviceID string) bool {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return false
	}
	return dCtx.Disable().IsDeviceAndDeviceID(requestContext(ctx), device, deviceID)
}

// CheckDisableDeviceByContext checks current account device disable state CheckDisableDeviceByContext 校验当前账号设备封禁状态
func CheckDisableDeviceByContext(ctx *hertzapp.RequestContext, device string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().CheckDevice(requestContext(ctx), device)
}

// CheckDisableDeviceAndDeviceIDByContext checks current account device ID disable state CheckDisableDeviceAndDeviceIDByContext 校验当前账号设备 ID 封禁状态
func CheckDisableDeviceAndDeviceIDByContext(ctx *hertzapp.RequestContext, device, deviceID string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().CheckDeviceAndDeviceID(requestContext(ctx), device, deviceID)
}

// GetDisableDeviceInfoByContext gets current account device disable info GetDisableDeviceInfoByContext 获取当前账号设备封禁信息
func GetDisableDeviceInfoByContext(ctx *hertzapp.RequestContext, device string) (*manager.DeviceDisableInfo, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Disable().GetDeviceInfo(requestContext(ctx), device)
}

// GetDisableDeviceAndDeviceIDInfoByContext gets current account device ID disable info GetDisableDeviceAndDeviceIDInfoByContext 获取当前账号设备 ID 封禁信息
func GetDisableDeviceAndDeviceIDInfoByContext(ctx *hertzapp.RequestContext, device, deviceID string) (*manager.DeviceDisableInfo, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Disable().GetDeviceAndDeviceIDInfo(requestContext(ctx), device, deviceID)
}

// GetDisableDeviceTTLByContext gets current account device disable TTL GetDisableDeviceTTLByContext 获取当前账号设备封禁剩余时间
func GetDisableDeviceTTLByContext(ctx *hertzapp.RequestContext, device string) (int64, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return 0, err
	}
	return dCtx.Disable().GetDeviceTTL(requestContext(ctx), device)
}

// GetDisableDeviceAndDeviceIDTTLByContext gets current account device ID disable TTL GetDisableDeviceAndDeviceIDTTLByContext 获取当前账号设备 ID 封禁剩余时间
func GetDisableDeviceAndDeviceIDTTLByContext(ctx *hertzapp.RequestContext, device, deviceID string) (int64, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return 0, err
	}
	return dCtx.Disable().GetDeviceAndDeviceIDTTL(requestContext(ctx), device, deviceID)
}
