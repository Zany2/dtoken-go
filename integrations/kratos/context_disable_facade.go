// @Author daixk 2026/06/05
package kratos

import (
	"context"
	"time"

	"github.com/Zany2/dtoken-go/core/manager"
)

// CheckDisableByCtx checks current account disable state CheckDisableByCtx ?
func CheckDisableByCtx(ctx context.Context) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().CheckAccount(ctx)
}

// DisableServiceByCtx disables current account service DisableServiceByCtx
func DisableServiceByCtx(ctx context.Context, service string, duration time.Duration, reason ...string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().Service(ctx, service, duration, reason...)
}

// DisableServiceLevelByCtx disables current account service level DisableServiceLevelByCtx ?
func DisableServiceLevelByCtx(ctx context.Context, service string, level int, duration time.Duration, reason ...string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().ServiceLevel(ctx, service, level, duration, reason...)
}

// UntieServiceByCtx removes current account service disable state UntieServiceByCtx
func UntieServiceByCtx(ctx context.Context, service string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().UntieService(ctx, service)
}

// IsDisableServiceByCtx checks current account service disable state IsDisableServiceByCtx ?
func IsDisableServiceByCtx(ctx context.Context, service string) bool {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return false
	}
	return dCtx.Disable().IsService(ctx, service)
}

// IsDisableServiceLevelByCtx checks current account service level disable state IsDisableServiceLevelByCtx ?
func IsDisableServiceLevelByCtx(ctx context.Context, service string, level int) bool {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return false
	}
	return dCtx.Disable().IsServiceLevel(ctx, service, level)
}

// CheckDisableServiceByCtx checks current account service disable state CheckDisableServiceByCtx ?
func CheckDisableServiceByCtx(ctx context.Context, services ...string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().CheckService(ctx, services...)
}

// CheckDisableServiceLevelByCtx checks current account service level disable state CheckDisableServiceLevelByCtx ?
func CheckDisableServiceLevelByCtx(ctx context.Context, service string, level int) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().CheckServiceLevel(ctx, service, level)
}

// GetDisableServiceInfoByCtx gets current account service disable info GetDisableServiceInfoByCtx
func GetDisableServiceInfoByCtx(ctx context.Context, service string) (*manager.ServiceDisableInfo, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Disable().GetServiceInfo(ctx, service)
}

// GetDisableServiceTTLByCtx gets current account service disable TTL GetDisableServiceTTLByCtx
func GetDisableServiceTTLByCtx(ctx context.Context, service string) (int64, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return 0, err
	}
	return dCtx.Disable().GetServiceTTL(ctx, service)
}

// DisableDeviceByCtx disables current account device DisableDeviceByCtx
func DisableDeviceByCtx(ctx context.Context, device string, duration time.Duration, reason ...string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().Device(ctx, device, duration, reason...)
}

// DisableDeviceAndDeviceIDByCtx delegates to DToken context DisableDeviceAndDeviceIDByCtx 转发到 DToken 上下文。
func DisableDeviceAndDeviceIDByCtx(ctx context.Context, device, deviceId string, duration time.Duration, reason ...string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().DeviceAndDeviceId(ctx, device, deviceId, duration, reason...)
}

// UntieDeviceByCtx removes current account device disable state UntieDeviceByCtx
func UntieDeviceByCtx(ctx context.Context, device string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().UntieDevice(ctx, device)
}

// UntieDeviceAndDeviceIDByCtx removes current account device ID disable state UntieDeviceAndDeviceIDByCtx  ID
func UntieDeviceAndDeviceIDByCtx(ctx context.Context, device, deviceId string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().UntieDeviceAndDeviceId(ctx, device, deviceId)
}

// IsDisableDeviceByCtx checks current account device disable state IsDisableDeviceByCtx ?
func IsDisableDeviceByCtx(ctx context.Context, device string) bool {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return false
	}
	return dCtx.Disable().IsDevice(ctx, device)
}

// IsDisableDeviceAndDeviceIDByCtx checks current account device ID disable state IsDisableDeviceAndDeviceIDByCtx ?ID ?
func IsDisableDeviceAndDeviceIDByCtx(ctx context.Context, device, deviceId string) bool {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return false
	}
	return dCtx.Disable().IsDeviceAndDeviceId(ctx, device, deviceId)
}

// CheckDisableDeviceByCtx checks current account device disable state CheckDisableDeviceByCtx ?
func CheckDisableDeviceByCtx(ctx context.Context, device string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().CheckDevice(ctx, device)
}

// CheckDisableDeviceAndDeviceIDByCtx checks current account device ID disable state CheckDisableDeviceAndDeviceIDByCtx  ID ?
func CheckDisableDeviceAndDeviceIDByCtx(ctx context.Context, device, deviceId string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().CheckDeviceAndDeviceId(ctx, device, deviceId)
}

// GetDisableDeviceInfoByCtx gets current account device disable info GetDisableDeviceInfoByCtx
func GetDisableDeviceInfoByCtx(ctx context.Context, device string) (*manager.DeviceDisableInfo, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Disable().GetDeviceInfo(ctx, device)
}

// GetDisableDeviceAndDeviceIDInfoByCtx gets current account device ID disable info GetDisableDeviceAndDeviceIDInfoByCtx  ID
func GetDisableDeviceAndDeviceIDInfoByCtx(ctx context.Context, device, deviceId string) (*manager.DeviceDisableInfo, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Disable().GetDeviceAndDeviceIdInfo(ctx, device, deviceId)
}

// GetDisableDeviceTTLByCtx gets current account device disable TTL GetDisableDeviceTTLByCtx
func GetDisableDeviceTTLByCtx(ctx context.Context, device string) (int64, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return 0, err
	}
	return dCtx.Disable().GetDeviceTTL(ctx, device)
}

// GetDisableDeviceAndDeviceIDTTLByCtx gets current account device ID disable TTL GetDisableDeviceAndDeviceIDTTLByCtx  ID
func GetDisableDeviceAndDeviceIDTTLByCtx(ctx context.Context, device, deviceId string) (int64, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return 0, err
	}
	return dCtx.Disable().GetDeviceAndDeviceIdTTL(ctx, device, deviceId)
}
