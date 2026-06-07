// @Author daixk 2026/06/05
package hertz

import (
	"time"

	"github.com/Zany2/dtoken-go/core/manager"
	hertzapp "github.com/cloudwego/hertz/pkg/app"
)

// CheckDisableByContext checks current account disable state CheckDisableByContext ?
func CheckDisableByContext(ctx *hertzapp.RequestContext) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().CheckAccount(requestContext(ctx))
}

// DisableServiceByContext disables current account service DisableServiceByContext
func DisableServiceByContext(ctx *hertzapp.RequestContext, service string, duration time.Duration, reason ...string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().Service(requestContext(ctx), service, duration, reason...)
}

// DisableServiceLevelByContext disables current account service level DisableServiceLevelByContext ?
func DisableServiceLevelByContext(ctx *hertzapp.RequestContext, service string, level int, duration time.Duration, reason ...string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().ServiceLevel(requestContext(ctx), service, level, duration, reason...)
}

// UntieServiceByContext removes current account service disable state UntieServiceByContext
func UntieServiceByContext(ctx *hertzapp.RequestContext, service string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().UntieService(requestContext(ctx), service)
}

// IsDisableServiceByContext checks current account service disable state IsDisableServiceByContext ?
func IsDisableServiceByContext(ctx *hertzapp.RequestContext, service string) bool {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return false
	}
	return dCtx.Disable().IsService(requestContext(ctx), service)
}

// IsDisableServiceLevelByContext checks current account service level disable state IsDisableServiceLevelByContext ?
func IsDisableServiceLevelByContext(ctx *hertzapp.RequestContext, service string, level int) bool {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return false
	}
	return dCtx.Disable().IsServiceLevel(requestContext(ctx), service, level)
}

// CheckDisableServiceByContext checks current account service disable state CheckDisableServiceByContext ?
func CheckDisableServiceByContext(ctx *hertzapp.RequestContext, services ...string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().CheckService(requestContext(ctx), services...)
}

// CheckDisableServiceLevelByContext checks current account service level disable state CheckDisableServiceLevelByContext ?
func CheckDisableServiceLevelByContext(ctx *hertzapp.RequestContext, service string, level int) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().CheckServiceLevel(requestContext(ctx), service, level)
}

// GetDisableServiceInfoByContext gets current account service disable info GetDisableServiceInfoByContext
func GetDisableServiceInfoByContext(ctx *hertzapp.RequestContext, service string) (*manager.ServiceDisableInfo, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Disable().GetServiceInfo(requestContext(ctx), service)
}

// GetDisableServiceTTLByContext gets current account service disable TTL GetDisableServiceTTLByContext
func GetDisableServiceTTLByContext(ctx *hertzapp.RequestContext, service string) (int64, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return 0, err
	}
	return dCtx.Disable().GetServiceTTL(requestContext(ctx), service)
}

// DisableDeviceByContext disables current account device DisableDeviceByContext
func DisableDeviceByContext(ctx *hertzapp.RequestContext, device string, duration time.Duration, reason ...string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().Device(requestContext(ctx), device, duration, reason...)
}

// DisableDeviceAndDeviceIDByContext delegates to DToken context DisableDeviceAndDeviceIDByContext 转发到 DToken 上下文。
func DisableDeviceAndDeviceIDByContext(ctx *hertzapp.RequestContext, device, deviceId string, duration time.Duration, reason ...string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().DeviceAndDeviceId(requestContext(ctx), device, deviceId, duration, reason...)
}

// UntieDeviceByContext removes current account device disable state UntieDeviceByContext
func UntieDeviceByContext(ctx *hertzapp.RequestContext, device string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().UntieDevice(requestContext(ctx), device)
}

// UntieDeviceAndDeviceIDByContext removes current account device ID disable state UntieDeviceAndDeviceIDByContext  ID
func UntieDeviceAndDeviceIDByContext(ctx *hertzapp.RequestContext, device, deviceId string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().UntieDeviceAndDeviceId(requestContext(ctx), device, deviceId)
}

// IsDisableDeviceByContext checks current account device disable state IsDisableDeviceByContext ?
func IsDisableDeviceByContext(ctx *hertzapp.RequestContext, device string) bool {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return false
	}
	return dCtx.Disable().IsDevice(requestContext(ctx), device)
}

// IsDisableDeviceAndDeviceIDByContext checks current account device ID disable state IsDisableDeviceAndDeviceIDByContext ?ID ?
func IsDisableDeviceAndDeviceIDByContext(ctx *hertzapp.RequestContext, device, deviceId string) bool {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return false
	}
	return dCtx.Disable().IsDeviceAndDeviceId(requestContext(ctx), device, deviceId)
}

// CheckDisableDeviceByContext checks current account device disable state CheckDisableDeviceByContext ?
func CheckDisableDeviceByContext(ctx *hertzapp.RequestContext, device string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().CheckDevice(requestContext(ctx), device)
}

// CheckDisableDeviceAndDeviceIDByContext checks current account device ID disable state CheckDisableDeviceAndDeviceIDByContext  ID ?
func CheckDisableDeviceAndDeviceIDByContext(ctx *hertzapp.RequestContext, device, deviceId string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().CheckDeviceAndDeviceId(requestContext(ctx), device, deviceId)
}

// GetDisableDeviceInfoByContext gets current account device disable info GetDisableDeviceInfoByContext
func GetDisableDeviceInfoByContext(ctx *hertzapp.RequestContext, device string) (*manager.DeviceDisableInfo, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Disable().GetDeviceInfo(requestContext(ctx), device)
}

// GetDisableDeviceAndDeviceIDInfoByContext gets current account device ID disable info GetDisableDeviceAndDeviceIDInfoByContext  ID
func GetDisableDeviceAndDeviceIDInfoByContext(ctx *hertzapp.RequestContext, device, deviceId string) (*manager.DeviceDisableInfo, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Disable().GetDeviceAndDeviceIdInfo(requestContext(ctx), device, deviceId)
}

// GetDisableDeviceTTLByContext gets current account device disable TTL GetDisableDeviceTTLByContext
func GetDisableDeviceTTLByContext(ctx *hertzapp.RequestContext, device string) (int64, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return 0, err
	}
	return dCtx.Disable().GetDeviceTTL(requestContext(ctx), device)
}

// GetDisableDeviceAndDeviceIDTTLByContext gets current account device ID disable TTL GetDisableDeviceAndDeviceIDTTLByContext  ID
func GetDisableDeviceAndDeviceIDTTLByContext(ctx *hertzapp.RequestContext, device, deviceId string) (int64, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return 0, err
	}
	return dCtx.Disable().GetDeviceAndDeviceIdTTL(requestContext(ctx), device, deviceId)
}
