package dtoken

import (
	"context"
	"time"

	manager "github.com/Zany2/dtoken-go/internal/engine"
)

// Disable delegates to the default authentication namespace. Disable 委托默认认证体系执行。
func Disable(ctx context.Context, opts DisableOptions) error {
	return AuthType("").Disable(ctx, opts)
}

// Untie delegates to the default authentication namespace. Untie 委托默认认证体系执行。
func Untie(ctx context.Context, loginID string) error {
	return AuthType("").Untie(ctx, loginID)
}

// IsDisable delegates to the default authentication namespace. IsDisable 委托默认认证体系执行。
func IsDisable(ctx context.Context, loginID string) bool {
	return AuthType("").IsDisable(ctx, loginID)
}

// CheckDisable delegates to the default authentication namespace. CheckDisable 委托默认认证体系执行。
func CheckDisable(ctx context.Context, loginID string) error {
	return AuthType("").CheckDisable(ctx, loginID)
}

// GetDisableInfo delegates to the default authentication namespace. GetDisableInfo 委托默认认证体系执行。
func GetDisableInfo(ctx context.Context, loginID string) (*manager.DisableInfo, error) {
	return AuthType("").GetDisableInfo(ctx, loginID)
}

// GetDisableTTL delegates to the default authentication namespace. GetDisableTTL 委托默认认证体系执行。
func GetDisableTTL(ctx context.Context, loginID string) (int64, error) {
	return AuthType("").GetDisableTTL(ctx, loginID)
}

// DisableService delegates to the default authentication namespace. DisableService 委托默认认证体系执行。
func DisableService(ctx context.Context, opts ServiceDisableOptions) error {
	return AuthType("").DisableService(ctx, opts)
}

// DisableServiceWithReason delegates to the default authentication namespace. DisableServiceWithReason 委托默认认证体系执行。
func DisableServiceWithReason(ctx context.Context, loginID, service string, duration time.Duration, reason string) error {
	return AuthType("").DisableServiceWithReason(ctx, loginID, service, duration, reason)
}

// DisableServiceLevel delegates to the default authentication namespace. DisableServiceLevel 委托默认认证体系执行。
func DisableServiceLevel(ctx context.Context, loginID, service string, level int, duration time.Duration) error {
	return AuthType("").DisableServiceLevel(ctx, loginID, service, level, duration)
}

// DisableServiceLevelWithReason delegates to the default authentication namespace. DisableServiceLevelWithReason 委托默认认证体系执行。
func DisableServiceLevelWithReason(ctx context.Context, loginID, service string, level int, duration time.Duration, reason string) error {
	return AuthType("").DisableServiceLevelWithReason(ctx, loginID, service, level, duration, reason)
}

// DisableDevice delegates to the default authentication namespace. DisableDevice 委托默认认证体系执行。
func DisableDevice(ctx context.Context, opts DeviceDisableOptions) error {
	return AuthType("").DisableDevice(ctx, opts)
}

// DisableDeviceWithReason delegates to the default authentication namespace. DisableDeviceWithReason 委托默认认证体系执行。
func DisableDeviceWithReason(ctx context.Context, loginID, device string, duration time.Duration, reason string) error {
	return AuthType("").DisableDeviceWithReason(ctx, loginID, device, duration, reason)
}

// DisableDeviceAndDeviceID delegates to the default authentication namespace. DisableDeviceAndDeviceID 委托默认认证体系执行。
func DisableDeviceAndDeviceID(ctx context.Context, loginID, device, deviceID string, duration time.Duration) error {
	return AuthType("").DisableDeviceAndDeviceID(ctx, loginID, device, deviceID, duration)
}

// DisableDeviceAndDeviceIDWithReason delegates to the default authentication namespace. DisableDeviceAndDeviceIDWithReason 委托默认认证体系执行。
func DisableDeviceAndDeviceIDWithReason(ctx context.Context, loginID, device, deviceID string, duration time.Duration, reason string) error {
	return AuthType("").DisableDeviceAndDeviceIDWithReason(ctx, loginID, device, deviceID, duration, reason)
}

// UntieService delegates to the default authentication namespace. UntieService 委托默认认证体系执行。
func UntieService(ctx context.Context, loginID, service string) error {
	return AuthType("").UntieService(ctx, loginID, service)
}

// IsDisableService delegates to the default authentication namespace. IsDisableService 委托默认认证体系执行。
func IsDisableService(ctx context.Context, loginID, service string) bool {
	return AuthType("").IsDisableService(ctx, loginID, service)
}

// IsDisableServiceLevel delegates to the default authentication namespace. IsDisableServiceLevel 委托默认认证体系执行。
func IsDisableServiceLevel(ctx context.Context, loginID, service string, level int) bool {
	return AuthType("").IsDisableServiceLevel(ctx, loginID, service, level)
}

// CheckDisableService delegates to the default authentication namespace. CheckDisableService 委托默认认证体系执行。
func CheckDisableService(ctx context.Context, loginID string, services ...string) error {
	return AuthType("").CheckDisableService(ctx, loginID, services...)
}

// CheckDisableServiceLevel delegates to the default authentication namespace. CheckDisableServiceLevel 委托默认认证体系执行。
func CheckDisableServiceLevel(ctx context.Context, loginID, service string, level int) error {
	return AuthType("").CheckDisableServiceLevel(ctx, loginID, service, level)
}

// GetDisableServiceInfo delegates to the default authentication namespace. GetDisableServiceInfo 委托默认认证体系执行。
func GetDisableServiceInfo(ctx context.Context, loginID, service string) (*manager.ServiceDisableInfo, error) {
	return AuthType("").GetDisableServiceInfo(ctx, loginID, service)
}

// GetDisableServiceTTL delegates to the default authentication namespace. GetDisableServiceTTL 委托默认认证体系执行。
func GetDisableServiceTTL(ctx context.Context, loginID, service string) (int64, error) {
	return AuthType("").GetDisableServiceTTL(ctx, loginID, service)
}

// UntieDevice delegates to the default authentication namespace. UntieDevice 委托默认认证体系执行。
func UntieDevice(ctx context.Context, loginID, device string) error {
	return AuthType("").UntieDevice(ctx, loginID, device)
}

// UntieDeviceAndDeviceID delegates to the default authentication namespace. UntieDeviceAndDeviceID 委托默认认证体系执行。
func UntieDeviceAndDeviceID(ctx context.Context, loginID, device, deviceID string) error {
	return AuthType("").UntieDeviceAndDeviceID(ctx, loginID, device, deviceID)
}

// IsDisableDevice delegates to the default authentication namespace. IsDisableDevice 委托默认认证体系执行。
func IsDisableDevice(ctx context.Context, loginID, device string) bool {
	return AuthType("").IsDisableDevice(ctx, loginID, device)
}

// IsDisableDeviceAndDeviceID delegates to the default authentication namespace. IsDisableDeviceAndDeviceID 委托默认认证体系执行。
func IsDisableDeviceAndDeviceID(ctx context.Context, loginID, device, deviceID string) bool {
	return AuthType("").IsDisableDeviceAndDeviceID(ctx, loginID, device, deviceID)
}

// CheckDisableDevice delegates to the default authentication namespace. CheckDisableDevice 委托默认认证体系执行。
func CheckDisableDevice(ctx context.Context, loginID, device string) error {
	return AuthType("").CheckDisableDevice(ctx, loginID, device)
}

// CheckDisableDeviceAndDeviceID delegates to the default authentication namespace. CheckDisableDeviceAndDeviceID 委托默认认证体系执行。
func CheckDisableDeviceAndDeviceID(ctx context.Context, loginID, device, deviceID string) error {
	return AuthType("").CheckDisableDeviceAndDeviceID(ctx, loginID, device, deviceID)
}

// GetDisableDeviceInfo delegates to the default authentication namespace. GetDisableDeviceInfo 委托默认认证体系执行。
func GetDisableDeviceInfo(ctx context.Context, loginID, device string) (*manager.DeviceDisableInfo, error) {
	return AuthType("").GetDisableDeviceInfo(ctx, loginID, device)
}

// GetDisableDeviceAndDeviceIDInfo delegates to the default authentication namespace. GetDisableDeviceAndDeviceIDInfo 委托默认认证体系执行。
func GetDisableDeviceAndDeviceIDInfo(ctx context.Context, loginID, device, deviceID string) (*manager.DeviceDisableInfo, error) {
	return AuthType("").GetDisableDeviceAndDeviceIDInfo(ctx, loginID, device, deviceID)
}

// GetDisableDeviceTTL delegates to the default authentication namespace. GetDisableDeviceTTL 委托默认认证体系执行。
func GetDisableDeviceTTL(ctx context.Context, loginID, device string) (int64, error) {
	return AuthType("").GetDisableDeviceTTL(ctx, loginID, device)
}

// GetDisableDeviceAndDeviceIDTTL delegates to the default authentication namespace. GetDisableDeviceAndDeviceIDTTL 委托默认认证体系执行。
func GetDisableDeviceAndDeviceIDTTL(ctx context.Context, loginID, device, deviceID string) (int64, error) {
	return AuthType("").GetDisableDeviceAndDeviceIDTTL(ctx, loginID, device, deviceID)
}
