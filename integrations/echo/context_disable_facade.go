// @Author daixk 2026/06/05
package echo

import (
	"time"

	"github.com/Zany2/dtoken-go/core/manager"
	echo4 "github.com/labstack/echo/v4"
)

// CheckDisableByContext checks current account disable state CheckDisableByContext 校验当前账号封禁状态
func CheckDisableByContext(c echo4.Context) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Disable().CheckAccount(requestContext(c))
}

// DisableServiceByContext disables current account service DisableServiceByContext 封禁当前账号服务
func DisableServiceByContext(c echo4.Context, service string, duration time.Duration, reason ...string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Disable().Service(requestContext(c), service, duration, reason...)
}

// DisableServiceLevelByContext disables current account service level DisableServiceLevelByContext 按等级封禁当前账号服务
func DisableServiceLevelByContext(c echo4.Context, service string, level int, duration time.Duration, reason ...string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Disable().ServiceLevel(requestContext(c), service, level, duration, reason...)
}

// UntieServiceByContext removes current account service disable state UntieServiceByContext 瑙ｅ皝褰撳墠璐﹀彿鏈嶅姟
func UntieServiceByContext(c echo4.Context, service string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Disable().UntieService(requestContext(c), service)
}

// IsDisableServiceByContext checks current account service disable state IsDisableServiceByContext 检查当前账号服务封禁状态
func IsDisableServiceByContext(c echo4.Context, service string) bool {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return false
	}
	return dCtx.Disable().IsService(requestContext(c), service)
}

// IsDisableServiceLevelByContext checks current account service level disable state IsDisableServiceLevelByContext 检查当前账号服务等级封禁状态
func IsDisableServiceLevelByContext(c echo4.Context, service string, level int) bool {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return false
	}
	return dCtx.Disable().IsServiceLevel(requestContext(c), service, level)
}

// CheckDisableServiceByContext checks current account service disable state CheckDisableServiceByContext 校验当前账号服务封禁状态
func CheckDisableServiceByContext(c echo4.Context, services ...string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Disable().CheckService(requestContext(c), services...)
}

// CheckDisableServiceLevelByContext checks current account service level disable state CheckDisableServiceLevelByContext 校验当前账号服务等级封禁状态
func CheckDisableServiceLevelByContext(c echo4.Context, service string, level int) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Disable().CheckServiceLevel(requestContext(c), service, level)
}

// GetDisableServiceInfoByContext gets current account service disable info GetDisableServiceInfoByContext 获取当前账号服务封禁信息
func GetDisableServiceInfoByContext(c echo4.Context, service string) (*manager.ServiceDisableInfo, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Disable().GetServiceInfo(requestContext(c), service)
}

// GetDisableServiceTTLByContext gets current account service disable TTL GetDisableServiceTTLByContext 获取当前账号服务封禁剩余时间
func GetDisableServiceTTLByContext(c echo4.Context, service string) (int64, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return 0, err
	}
	return dCtx.Disable().GetServiceTTL(requestContext(c), service)
}

// DisableDeviceByContext disables current account device DisableDeviceByContext 封禁当前账号设备
func DisableDeviceByContext(c echo4.Context, device string, duration time.Duration, reason ...string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Disable().Device(requestContext(c), device, duration, reason...)
}

// DisableDeviceAndDeviceIDByContext disables current account device ID DisableDeviceAndDeviceIDByContext 按设备和设备 ID 封禁当前账号
func DisableDeviceAndDeviceIDByContext(c echo4.Context, device, deviceID string, duration time.Duration, reason ...string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Disable().DeviceAndDeviceID(requestContext(c), device, deviceID, duration, reason...)
}

// UntieDeviceByContext removes current account device disable state UntieDeviceByContext 瑙ｅ皝褰撳墠璐﹀彿璁惧
func UntieDeviceByContext(c echo4.Context, device string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Disable().UntieDevice(requestContext(c), device)
}

// UntieDeviceAndDeviceIDByContext removes current account device ID disable state UntieDeviceAndDeviceIDByContext 瑙ｅ皝褰撳墠璐﹀彿璁惧 ID
func UntieDeviceAndDeviceIDByContext(c echo4.Context, device, deviceID string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Disable().UntieDeviceAndDeviceID(requestContext(c), device, deviceID)
}

// IsDisableDeviceByContext checks current account device disable state IsDisableDeviceByContext 检查当前账号设备封禁状态
func IsDisableDeviceByContext(c echo4.Context, device string) bool {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return false
	}
	return dCtx.Disable().IsDevice(requestContext(c), device)
}

// IsDisableDeviceAndDeviceIDByContext checks current account device ID disable state IsDisableDeviceAndDeviceIDByContext 检查当前账号设备 ID 封禁状态
func IsDisableDeviceAndDeviceIDByContext(c echo4.Context, device, deviceID string) bool {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return false
	}
	return dCtx.Disable().IsDeviceAndDeviceID(requestContext(c), device, deviceID)
}

// CheckDisableDeviceByContext checks current account device disable state CheckDisableDeviceByContext 校验当前账号设备封禁状态
func CheckDisableDeviceByContext(c echo4.Context, device string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Disable().CheckDevice(requestContext(c), device)
}

// CheckDisableDeviceAndDeviceIDByContext checks current account device ID disable state CheckDisableDeviceAndDeviceIDByContext 校验当前账号设备 ID 封禁状态
func CheckDisableDeviceAndDeviceIDByContext(c echo4.Context, device, deviceID string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Disable().CheckDeviceAndDeviceID(requestContext(c), device, deviceID)
}

// GetDisableDeviceInfoByContext gets current account device disable info GetDisableDeviceInfoByContext 获取当前账号设备封禁信息
func GetDisableDeviceInfoByContext(c echo4.Context, device string) (*manager.DeviceDisableInfo, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Disable().GetDeviceInfo(requestContext(c), device)
}

// GetDisableDeviceAndDeviceIDInfoByContext gets current account device ID disable info GetDisableDeviceAndDeviceIDInfoByContext 获取当前账号设备 ID 封禁信息
func GetDisableDeviceAndDeviceIDInfoByContext(c echo4.Context, device, deviceID string) (*manager.DeviceDisableInfo, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Disable().GetDeviceAndDeviceIDInfo(requestContext(c), device, deviceID)
}

// GetDisableDeviceTTLByContext gets current account device disable TTL GetDisableDeviceTTLByContext 获取当前账号设备封禁剩余时间
func GetDisableDeviceTTLByContext(c echo4.Context, device string) (int64, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return 0, err
	}
	return dCtx.Disable().GetDeviceTTL(requestContext(c), device)
}

// GetDisableDeviceAndDeviceIDTTLByContext gets current account device ID disable TTL GetDisableDeviceAndDeviceIDTTLByContext 获取当前账号设备 ID 封禁剩余时间
func GetDisableDeviceAndDeviceIDTTLByContext(c echo4.Context, device, deviceID string) (int64, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return 0, err
	}
	return dCtx.Disable().GetDeviceAndDeviceIDTTL(requestContext(c), device, deviceID)
}
