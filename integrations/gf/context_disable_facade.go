// @Author daixk 2026/06/05
package gf

import (
	"context"
	"time"

	"github.com/Zany2/dtoken-go/core/manager"
)

// CheckDisableByCtx checks current account disable state CheckDisableByCtx 鏍￠獙褰撳墠璐﹀彿灏佺鐘舵€?
func CheckDisableByCtx(ctx context.Context) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().CheckAccount(ctx)
}

// DisableServiceByCtx disables current account service DisableServiceByCtx 灏佺褰撳墠璐﹀彿鏈嶅姟
func DisableServiceByCtx(ctx context.Context, service string, duration time.Duration, reason ...string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().Service(ctx, service, duration, reason...)
}

// DisableServiceLevelByCtx disables current account service level DisableServiceLevelByCtx 鎸夌瓑绾у皝绂佸綋鍓嶈处鍙锋湇鍔?
func DisableServiceLevelByCtx(ctx context.Context, service string, level int, duration time.Duration, reason ...string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().ServiceLevel(ctx, service, level, duration, reason...)
}

// UntieServiceByCtx removes current account service disable state UntieServiceByCtx 瑙ｅ皝褰撳墠璐﹀彿鏈嶅姟
func UntieServiceByCtx(ctx context.Context, service string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().UntieService(ctx, service)
}

// IsDisableServiceByCtx checks current account service disable state IsDisableServiceByCtx 妫€鏌ュ綋鍓嶈处鍙锋湇鍔″皝绂佺姸鎬?
func IsDisableServiceByCtx(ctx context.Context, service string) bool {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return false
	}
	return dCtx.Disable().IsService(ctx, service)
}

// IsDisableServiceLevelByCtx checks current account service level disable state IsDisableServiceLevelByCtx 妫€鏌ュ綋鍓嶈处鍙锋湇鍔＄瓑绾у皝绂佺姸鎬?
func IsDisableServiceLevelByCtx(ctx context.Context, service string, level int) bool {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return false
	}
	return dCtx.Disable().IsServiceLevel(ctx, service, level)
}

// CheckDisableServiceByCtx checks current account service disable state CheckDisableServiceByCtx 鏍￠獙褰撳墠璐﹀彿鏈嶅姟灏佺鐘舵€?
func CheckDisableServiceByCtx(ctx context.Context, services ...string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().CheckService(ctx, services...)
}

// CheckDisableServiceLevelByCtx checks current account service level disable state CheckDisableServiceLevelByCtx 鏍￠獙褰撳墠璐﹀彿鏈嶅姟绛夌骇灏佺鐘舵€?
func CheckDisableServiceLevelByCtx(ctx context.Context, service string, level int) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().CheckServiceLevel(ctx, service, level)
}

// GetDisableServiceInfoByCtx gets current account service disable info GetDisableServiceInfoByCtx 鑾峰彇褰撳墠璐﹀彿鏈嶅姟灏佺淇℃伅
func GetDisableServiceInfoByCtx(ctx context.Context, service string) (*manager.ServiceDisableInfo, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Disable().GetServiceInfo(ctx, service)
}

// GetDisableServiceTTLByCtx gets current account service disable TTL GetDisableServiceTTLByCtx 鑾峰彇褰撳墠璐﹀彿鏈嶅姟灏佺鍓╀綑鏃堕棿
func GetDisableServiceTTLByCtx(ctx context.Context, service string) (int64, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return 0, err
	}
	return dCtx.Disable().GetServiceTTL(ctx, service)
}

// DisableDeviceByCtx disables current account device DisableDeviceByCtx 灏佺褰撳墠璐﹀彿璁惧
func DisableDeviceByCtx(ctx context.Context, device string, duration time.Duration, reason ...string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().Device(ctx, device, duration, reason...)
}

// DisableDeviceAndDeviceIDByCtx disables current account device ID DisableDeviceAndDeviceIDByCtx 鎸夎澶囧拰璁惧 ID 灏佺褰撳墠璐﹀彿
func DisableDeviceAndDeviceIDByCtx(ctx context.Context, device, deviceId string, duration time.Duration, reason ...string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().DeviceAndDeviceId(ctx, device, deviceId, duration, reason...)
}

// UntieDeviceByCtx removes current account device disable state UntieDeviceByCtx 瑙ｅ皝褰撳墠璐﹀彿璁惧
func UntieDeviceByCtx(ctx context.Context, device string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().UntieDevice(ctx, device)
}

// UntieDeviceAndDeviceIDByCtx removes current account device ID disable state UntieDeviceAndDeviceIDByCtx 瑙ｅ皝褰撳墠璐﹀彿璁惧 ID
func UntieDeviceAndDeviceIDByCtx(ctx context.Context, device, deviceId string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().UntieDeviceAndDeviceId(ctx, device, deviceId)
}

// IsDisableDeviceByCtx checks current account device disable state IsDisableDeviceByCtx 妫€鏌ュ綋鍓嶈处鍙疯澶囧皝绂佺姸鎬?
func IsDisableDeviceByCtx(ctx context.Context, device string) bool {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return false
	}
	return dCtx.Disable().IsDevice(ctx, device)
}

// IsDisableDeviceAndDeviceIDByCtx checks current account device ID disable state IsDisableDeviceAndDeviceIDByCtx 妫€鏌ュ綋鍓嶈处鍙疯澶?ID 灏佺鐘舵€?
func IsDisableDeviceAndDeviceIDByCtx(ctx context.Context, device, deviceId string) bool {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return false
	}
	return dCtx.Disable().IsDeviceAndDeviceId(ctx, device, deviceId)
}

// CheckDisableDeviceByCtx checks current account device disable state CheckDisableDeviceByCtx 鏍￠獙褰撳墠璐﹀彿璁惧灏佺鐘舵€?
func CheckDisableDeviceByCtx(ctx context.Context, device string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().CheckDevice(ctx, device)
}

// CheckDisableDeviceAndDeviceIDByCtx checks current account device ID disable state CheckDisableDeviceAndDeviceIDByCtx 鏍￠獙褰撳墠璐﹀彿璁惧 ID 灏佺鐘舵€?
func CheckDisableDeviceAndDeviceIDByCtx(ctx context.Context, device, deviceId string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Disable().CheckDeviceAndDeviceId(ctx, device, deviceId)
}

// GetDisableDeviceInfoByCtx gets current account device disable info GetDisableDeviceInfoByCtx 鑾峰彇褰撳墠璐﹀彿璁惧灏佺淇℃伅
func GetDisableDeviceInfoByCtx(ctx context.Context, device string) (*manager.DeviceDisableInfo, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Disable().GetDeviceInfo(ctx, device)
}

// GetDisableDeviceAndDeviceIDInfoByCtx gets current account device ID disable info GetDisableDeviceAndDeviceIDInfoByCtx 鑾峰彇褰撳墠璐﹀彿璁惧 ID 灏佺淇℃伅
func GetDisableDeviceAndDeviceIDInfoByCtx(ctx context.Context, device, deviceId string) (*manager.DeviceDisableInfo, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Disable().GetDeviceAndDeviceIdInfo(ctx, device, deviceId)
}

// GetDisableDeviceTTLByCtx gets current account device disable TTL GetDisableDeviceTTLByCtx 鑾峰彇褰撳墠璐﹀彿璁惧灏佺鍓╀綑鏃堕棿
func GetDisableDeviceTTLByCtx(ctx context.Context, device string) (int64, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return 0, err
	}
	return dCtx.Disable().GetDeviceTTL(ctx, device)
}

// GetDisableDeviceAndDeviceIDTTLByCtx gets current account device ID disable TTL GetDisableDeviceAndDeviceIDTTLByCtx 鑾峰彇褰撳墠璐﹀彿璁惧 ID 灏佺鍓╀綑鏃堕棿
func GetDisableDeviceAndDeviceIDTTLByCtx(ctx context.Context, device, deviceId string) (int64, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return 0, err
	}
	return dCtx.Disable().GetDeviceAndDeviceIdTTL(ctx, device, deviceId)
}
