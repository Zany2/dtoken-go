// @Author daixk 2026/06/05
package fiber

import (
	"time"

	"github.com/Zany2/dtoken-go/core/manager"
	gofiber "github.com/gofiber/fiber/v2"
)

// CheckDisableByContext checks current account disable state CheckDisableByContext 鏍￠獙褰撳墠璐﹀彿灏佺鐘舵€?
func CheckDisableByContext(c *gofiber.Ctx) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Disable().CheckAccount(requestContext(c))
}

// DisableServiceByContext disables current account service DisableServiceByContext 灏佺褰撳墠璐﹀彿鏈嶅姟
func DisableServiceByContext(c *gofiber.Ctx, service string, duration time.Duration, reason ...string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Disable().Service(requestContext(c), service, duration, reason...)
}

// DisableServiceLevelByContext disables current account service level DisableServiceLevelByContext 鎸夌瓑绾у皝绂佸綋鍓嶈处鍙锋湇鍔?
func DisableServiceLevelByContext(c *gofiber.Ctx, service string, level int, duration time.Duration, reason ...string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Disable().ServiceLevel(requestContext(c), service, level, duration, reason...)
}

// UntieServiceByContext removes current account service disable state UntieServiceByContext 瑙ｅ皝褰撳墠璐﹀彿鏈嶅姟
func UntieServiceByContext(c *gofiber.Ctx, service string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Disable().UntieService(requestContext(c), service)
}

// IsDisableServiceByContext checks current account service disable state IsDisableServiceByContext 妫€鏌ュ綋鍓嶈处鍙锋湇鍔″皝绂佺姸鎬?
func IsDisableServiceByContext(c *gofiber.Ctx, service string) bool {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return false
	}
	return dCtx.Disable().IsService(requestContext(c), service)
}

// IsDisableServiceLevelByContext checks current account service level disable state IsDisableServiceLevelByContext 妫€鏌ュ綋鍓嶈处鍙锋湇鍔＄瓑绾у皝绂佺姸鎬?
func IsDisableServiceLevelByContext(c *gofiber.Ctx, service string, level int) bool {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return false
	}
	return dCtx.Disable().IsServiceLevel(requestContext(c), service, level)
}

// CheckDisableServiceByContext checks current account service disable state CheckDisableServiceByContext 鏍￠獙褰撳墠璐﹀彿鏈嶅姟灏佺鐘舵€?
func CheckDisableServiceByContext(c *gofiber.Ctx, services ...string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Disable().CheckService(requestContext(c), services...)
}

// CheckDisableServiceLevelByContext checks current account service level disable state CheckDisableServiceLevelByContext 鏍￠獙褰撳墠璐﹀彿鏈嶅姟绛夌骇灏佺鐘舵€?
func CheckDisableServiceLevelByContext(c *gofiber.Ctx, service string, level int) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Disable().CheckServiceLevel(requestContext(c), service, level)
}

// GetDisableServiceInfoByContext gets current account service disable info GetDisableServiceInfoByContext 鑾峰彇褰撳墠璐﹀彿鏈嶅姟灏佺淇℃伅
func GetDisableServiceInfoByContext(c *gofiber.Ctx, service string) (*manager.ServiceDisableInfo, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Disable().GetServiceInfo(requestContext(c), service)
}

// GetDisableServiceTTLByContext gets current account service disable TTL GetDisableServiceTTLByContext 鑾峰彇褰撳墠璐﹀彿鏈嶅姟灏佺鍓╀綑鏃堕棿
func GetDisableServiceTTLByContext(c *gofiber.Ctx, service string) (int64, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return 0, err
	}
	return dCtx.Disable().GetServiceTTL(requestContext(c), service)
}

// DisableDeviceByContext disables current account device DisableDeviceByContext 灏佺褰撳墠璐﹀彿璁惧
func DisableDeviceByContext(c *gofiber.Ctx, device string, duration time.Duration, reason ...string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Disable().Device(requestContext(c), device, duration, reason...)
}

// DisableDeviceAndDeviceIDByContext disables current account device ID DisableDeviceAndDeviceIDByContext 鎸夎澶囧拰璁惧 ID 灏佺褰撳墠璐﹀彿
func DisableDeviceAndDeviceIDByContext(c *gofiber.Ctx, device, deviceId string, duration time.Duration, reason ...string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Disable().DeviceAndDeviceId(requestContext(c), device, deviceId, duration, reason...)
}

// UntieDeviceByContext removes current account device disable state UntieDeviceByContext 瑙ｅ皝褰撳墠璐﹀彿璁惧
func UntieDeviceByContext(c *gofiber.Ctx, device string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Disable().UntieDevice(requestContext(c), device)
}

// UntieDeviceAndDeviceIDByContext removes current account device ID disable state UntieDeviceAndDeviceIDByContext 瑙ｅ皝褰撳墠璐﹀彿璁惧 ID
func UntieDeviceAndDeviceIDByContext(c *gofiber.Ctx, device, deviceId string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Disable().UntieDeviceAndDeviceId(requestContext(c), device, deviceId)
}

// IsDisableDeviceByContext checks current account device disable state IsDisableDeviceByContext 妫€鏌ュ綋鍓嶈处鍙疯澶囧皝绂佺姸鎬?
func IsDisableDeviceByContext(c *gofiber.Ctx, device string) bool {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return false
	}
	return dCtx.Disable().IsDevice(requestContext(c), device)
}

// IsDisableDeviceAndDeviceIDByContext checks current account device ID disable state IsDisableDeviceAndDeviceIDByContext 妫€鏌ュ綋鍓嶈处鍙疯澶?ID 灏佺鐘舵€?
func IsDisableDeviceAndDeviceIDByContext(c *gofiber.Ctx, device, deviceId string) bool {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return false
	}
	return dCtx.Disable().IsDeviceAndDeviceId(requestContext(c), device, deviceId)
}

// CheckDisableDeviceByContext checks current account device disable state CheckDisableDeviceByContext 鏍￠獙褰撳墠璐﹀彿璁惧灏佺鐘舵€?
func CheckDisableDeviceByContext(c *gofiber.Ctx, device string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Disable().CheckDevice(requestContext(c), device)
}

// CheckDisableDeviceAndDeviceIDByContext checks current account device ID disable state CheckDisableDeviceAndDeviceIDByContext 鏍￠獙褰撳墠璐﹀彿璁惧 ID 灏佺鐘舵€?
func CheckDisableDeviceAndDeviceIDByContext(c *gofiber.Ctx, device, deviceId string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Disable().CheckDeviceAndDeviceId(requestContext(c), device, deviceId)
}

// GetDisableDeviceInfoByContext gets current account device disable info GetDisableDeviceInfoByContext 鑾峰彇褰撳墠璐﹀彿璁惧灏佺淇℃伅
func GetDisableDeviceInfoByContext(c *gofiber.Ctx, device string) (*manager.DeviceDisableInfo, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Disable().GetDeviceInfo(requestContext(c), device)
}

// GetDisableDeviceAndDeviceIDInfoByContext gets current account device ID disable info GetDisableDeviceAndDeviceIDInfoByContext 鑾峰彇褰撳墠璐﹀彿璁惧 ID 灏佺淇℃伅
func GetDisableDeviceAndDeviceIDInfoByContext(c *gofiber.Ctx, device, deviceId string) (*manager.DeviceDisableInfo, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Disable().GetDeviceAndDeviceIdInfo(requestContext(c), device, deviceId)
}

// GetDisableDeviceTTLByContext gets current account device disable TTL GetDisableDeviceTTLByContext 鑾峰彇褰撳墠璐﹀彿璁惧灏佺鍓╀綑鏃堕棿
func GetDisableDeviceTTLByContext(c *gofiber.Ctx, device string) (int64, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return 0, err
	}
	return dCtx.Disable().GetDeviceTTL(requestContext(c), device)
}

// GetDisableDeviceAndDeviceIDTTLByContext gets current account device ID disable TTL GetDisableDeviceAndDeviceIDTTLByContext 鑾峰彇褰撳墠璐﹀彿璁惧 ID 灏佺鍓╀綑鏃堕棿
func GetDisableDeviceAndDeviceIDTTLByContext(c *gofiber.Ctx, device, deviceId string) (int64, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return 0, err
	}
	return dCtx.Disable().GetDeviceAndDeviceIdTTL(requestContext(c), device, deviceId)
}
