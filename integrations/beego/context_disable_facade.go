// @Author daixk 2026/06/06
package beego

import (
	"time"

	"github.com/Zany2/dtoken-go/core/manager"
	beegocontext "github.com/beego/beego/v2/server/web/context"
)

// IsDisableByContext checks whether current user is disabled IsDisableByContext 检查当前用户是否被封禁
func IsDisableByContext(c *beegocontext.Context) bool {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return false
	}
	return dCtx.Disable().IsAccount(requestContext(c))
}

// CheckDisableByContext checks current account disable state CheckDisableByContext 校验当前账号封禁状态
func CheckDisableByContext(c *beegocontext.Context) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Disable().CheckAccount(requestContext(c))
}

// GetDisableInfoByContext gets current user disable info GetDisableInfoByContext 获取当前用户封禁信息
func GetDisableInfoByContext(c *beegocontext.Context) (*manager.DisableInfo, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Disable().AccountInfo(requestContext(c))
}

// GetDisableTTLByContext gets current user disable TTL GetDisableTTLByContext 获取当前用户封禁剩余时间
func GetDisableTTLByContext(c *beegocontext.Context) (int64, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return 0, err
	}
	return dCtx.Disable().AccountTTL(requestContext(c))
}

// DisableByContext disables current user DisableByContext 封禁当前用户
func DisableByContext(c *beegocontext.Context, duration time.Duration, reason ...string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Disable().Account(requestContext(c), duration, reason...)
}

// UntieByContext removes current user disable state UntieByContext 解封当前用户
func UntieByContext(c *beegocontext.Context) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Disable().UntieAccount(requestContext(c))
}

// DisableServiceByContext disables current account service DisableServiceByContext 封禁当前账号服务
func DisableServiceByContext(c *beegocontext.Context, service string, duration time.Duration, reason ...string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Disable().Service(requestContext(c), service, duration, reason...)
}

// DisableServiceLevelByContext disables current account service level DisableServiceLevelByContext 按等级封禁当前账号服务
func DisableServiceLevelByContext(c *beegocontext.Context, service string, level int, duration time.Duration, reason ...string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Disable().ServiceLevel(requestContext(c), service, level, duration, reason...)
}

// UntieServiceByContext removes current account service disable state UntieServiceByContext 解封当前账号服务
func UntieServiceByContext(c *beegocontext.Context, service string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Disable().UntieService(requestContext(c), service)
}

// IsDisableServiceByContext checks current account service disable state IsDisableServiceByContext 检查当前账号服务封禁状态
func IsDisableServiceByContext(c *beegocontext.Context, service string) bool {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return false
	}
	return dCtx.Disable().IsService(requestContext(c), service)
}

// IsDisableServiceLevelByContext checks current account service level disable state IsDisableServiceLevelByContext 检查当前账号服务封禁等级
func IsDisableServiceLevelByContext(c *beegocontext.Context, service string, level int) bool {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return false
	}
	return dCtx.Disable().IsServiceLevel(requestContext(c), service, level)
}

// CheckDisableServiceByContext checks current account service disable state CheckDisableServiceByContext 校验当前账号服务封禁状态
func CheckDisableServiceByContext(c *beegocontext.Context, services ...string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Disable().CheckService(requestContext(c), services...)
}

// CheckDisableServiceLevelByContext checks current account service level disable state CheckDisableServiceLevelByContext 校验当前账号服务封禁等级
func CheckDisableServiceLevelByContext(c *beegocontext.Context, service string, level int) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Disable().CheckServiceLevel(requestContext(c), service, level)
}

// GetDisableServiceInfoByContext gets current account service disable info GetDisableServiceInfoByContext 获取当前账号服务封禁信息
func GetDisableServiceInfoByContext(c *beegocontext.Context, service string) (*manager.ServiceDisableInfo, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Disable().GetServiceInfo(requestContext(c), service)
}

// GetDisableServiceTTLByContext gets current account service disable TTL GetDisableServiceTTLByContext 获取当前账号服务封禁剩余时间
func GetDisableServiceTTLByContext(c *beegocontext.Context, service string) (int64, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return 0, err
	}
	return dCtx.Disable().GetServiceTTL(requestContext(c), service)
}

// DisableDeviceByContext disables current account device DisableDeviceByContext 封禁当前账号设备
func DisableDeviceByContext(c *beegocontext.Context, device string, duration time.Duration, reason ...string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Disable().Device(requestContext(c), device, duration, reason...)
}

// DisableDeviceAndDeviceIDByContext disables current account device ID DisableDeviceAndDeviceIDByContext 封禁当前账号指定设备 ID
func DisableDeviceAndDeviceIDByContext(c *beegocontext.Context, device, deviceId string, duration time.Duration, reason ...string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Disable().DeviceAndDeviceId(requestContext(c), device, deviceId, duration, reason...)
}

// UntieDeviceByContext removes current account device disable state UntieDeviceByContext 解封当前账号设备
func UntieDeviceByContext(c *beegocontext.Context, device string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Disable().UntieDevice(requestContext(c), device)
}

// UntieDeviceAndDeviceIDByContext removes current account device ID disable state UntieDeviceAndDeviceIDByContext 解封当前账号指定设备 ID
func UntieDeviceAndDeviceIDByContext(c *beegocontext.Context, device, deviceId string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Disable().UntieDeviceAndDeviceId(requestContext(c), device, deviceId)
}

// IsDisableDeviceByContext checks current account device disable state IsDisableDeviceByContext 检查当前账号设备封禁状态
func IsDisableDeviceByContext(c *beegocontext.Context, device string) bool {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return false
	}
	return dCtx.Disable().IsDevice(requestContext(c), device)
}

// IsDisableDeviceAndDeviceIDByContext checks current account device ID disable state IsDisableDeviceAndDeviceIDByContext 检查当前账号指定设备 ID 封禁状态
func IsDisableDeviceAndDeviceIDByContext(c *beegocontext.Context, device, deviceId string) bool {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return false
	}
	return dCtx.Disable().IsDeviceAndDeviceId(requestContext(c), device, deviceId)
}

// CheckDisableDeviceByContext checks current account device disable state CheckDisableDeviceByContext 校验当前账号设备封禁状态
func CheckDisableDeviceByContext(c *beegocontext.Context, device string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Disable().CheckDevice(requestContext(c), device)
}

// CheckDisableDeviceAndDeviceIDByContext checks current account device ID disable state CheckDisableDeviceAndDeviceIDByContext 校验当前账号指定设备 ID 封禁状态
func CheckDisableDeviceAndDeviceIDByContext(c *beegocontext.Context, device, deviceId string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Disable().CheckDeviceAndDeviceId(requestContext(c), device, deviceId)
}

// GetDisableDeviceInfoByContext gets current account device disable info GetDisableDeviceInfoByContext 获取当前账号设备封禁信息
func GetDisableDeviceInfoByContext(c *beegocontext.Context, device string) (*manager.DeviceDisableInfo, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Disable().GetDeviceInfo(requestContext(c), device)
}

// GetDisableDeviceAndDeviceIDInfoByContext gets current account device ID disable info GetDisableDeviceAndDeviceIDInfoByContext 获取当前账号指定设备 ID 封禁信息
func GetDisableDeviceAndDeviceIDInfoByContext(c *beegocontext.Context, device, deviceId string) (*manager.DeviceDisableInfo, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Disable().GetDeviceAndDeviceIdInfo(requestContext(c), device, deviceId)
}

// GetDisableDeviceTTLByContext gets current account device disable TTL GetDisableDeviceTTLByContext 获取当前账号设备封禁剩余时间
func GetDisableDeviceTTLByContext(c *beegocontext.Context, device string) (int64, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return 0, err
	}
	return dCtx.Disable().GetDeviceTTL(requestContext(c), device)
}

// GetDisableDeviceAndDeviceIDTTLByContext gets current account device ID disable TTL GetDisableDeviceAndDeviceIDTTLByContext 获取当前账号指定设备 ID 封禁剩余时间
func GetDisableDeviceAndDeviceIDTTLByContext(c *beegocontext.Context, device, deviceId string) (int64, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return 0, err
	}
	return dCtx.Disable().GetDeviceAndDeviceIdTTL(requestContext(c), device, deviceId)
}
