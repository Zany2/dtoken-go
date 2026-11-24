package dtoken

import (
	"context"

	manager "github.com/Zany2/dtoken-go/internal/engine"
)

// GetOnlineTerminalCount delegates to the default authentication namespace. GetOnlineTerminalCount 委托默认认证体系执行。
func GetOnlineTerminalCount(ctx context.Context, loginID string) (int, error) {
	return AuthType("").GetOnlineTerminalCount(ctx, loginID)
}

// GetOnlineTerminalCountByDevice delegates to the default authentication namespace. GetOnlineTerminalCountByDevice 委托默认认证体系执行。
func GetOnlineTerminalCountByDevice(ctx context.Context, loginID, device string) (int, error) {
	return AuthType("").GetOnlineTerminalCountByDevice(ctx, loginID, device)
}

// GetOnlineTerminalCountByDeviceAndDeviceID delegates to the default authentication namespace. GetOnlineTerminalCountByDeviceAndDeviceID 委托默认认证体系执行。
func GetOnlineTerminalCountByDeviceAndDeviceID(ctx context.Context, loginID, device, deviceID string) (int, error) {
	return AuthType("").GetOnlineTerminalCountByDeviceAndDeviceID(ctx, loginID, device, deviceID)
}

// ForEachTerminal delegates to the default authentication namespace. ForEachTerminal 委托默认认证体系执行。
func ForEachTerminal(ctx context.Context, loginID string, visitor manager.TerminalVisitor) error {
	return AuthType("").ForEachTerminal(ctx, loginID, visitor)
}

// ForEachTerminalByDevice delegates to the default authentication namespace. ForEachTerminalByDevice 委托默认认证体系执行。
func ForEachTerminalByDevice(ctx context.Context, loginID, device string, visitor manager.TerminalVisitor) error {
	return AuthType("").ForEachTerminalByDevice(ctx, loginID, device, visitor)
}

// GetSession delegates to the default authentication namespace. GetSession 委托默认认证体系执行。
func GetSession(ctx context.Context, loginID string) (*manager.Session, error) {
	return AuthType("").GetSession(ctx, loginID)
}

// GetSessionByToken delegates to the default authentication namespace. GetSessionByToken 委托默认认证体系执行。
func GetSessionByToken(ctx context.Context, token string) (*manager.Session, error) {
	return AuthType("").GetSessionByToken(ctx, token)
}

// GetTokenValueListByLoginID delegates to the default authentication namespace. GetTokenValueListByLoginID 委托默认认证体系执行。
func GetTokenValueListByLoginID(ctx context.Context, loginID string, checkAlive bool) ([]string, error) {
	return AuthType("").GetTokenValueListByLoginID(ctx, loginID, checkAlive)
}

// GetTokenValueListByDeviceAndDeviceID delegates to the default authentication namespace. GetTokenValueListByDeviceAndDeviceID 委托默认认证体系执行。
func GetTokenValueListByDeviceAndDeviceID(ctx context.Context, loginID, device, deviceID string, checkAlive bool) ([]string, error) {
	return AuthType("").GetTokenValueListByDeviceAndDeviceID(ctx, loginID, device, deviceID, checkAlive)
}

// GetTokenValueListByDevice delegates to the default authentication namespace. GetTokenValueListByDevice 委托默认认证体系执行。
func GetTokenValueListByDevice(ctx context.Context, loginID, device string, checkAlive bool) ([]string, error) {
	return AuthType("").GetTokenValueListByDevice(ctx, loginID, device, checkAlive)
}

// GetTerminalListByLoginID delegates to the default authentication namespace. GetTerminalListByLoginID 委托默认认证体系执行。
func GetTerminalListByLoginID(ctx context.Context, loginID string) ([]manager.TerminalInfo, error) {
	return AuthType("").GetTerminalListByLoginID(ctx, loginID)
}

// GetTerminalListByLoginIDAndDevice delegates to the default authentication namespace. GetTerminalListByLoginIDAndDevice 委托默认认证体系执行。
func GetTerminalListByLoginIDAndDevice(ctx context.Context, loginID, device string) ([]manager.TerminalInfo, error) {
	return AuthType("").GetTerminalListByLoginIDAndDevice(ctx, loginID, device)
}

// GetTerminalInfoByToken delegates to the default authentication namespace. GetTerminalInfoByToken 委托默认认证体系执行。
func GetTerminalInfoByToken(ctx context.Context, token string) (*manager.TerminalInfo, error) {
	return AuthType("").GetTerminalInfoByToken(ctx, token)
}

// SetSessionValue delegates to the default authentication namespace. SetSessionValue 委托默认认证体系执行。
func SetSessionValue(ctx context.Context, loginID, key string, value any) error {
	return AuthType("").SetSessionValue(ctx, loginID, key, value)
}

// GetSessionValue delegates to the default authentication namespace. GetSessionValue 委托默认认证体系执行。
func GetSessionValue(ctx context.Context, loginID, key string) (any, bool, error) {
	return AuthType("").GetSessionValue(ctx, loginID, key)
}

// DeleteSessionValue delegates to the default authentication namespace. DeleteSessionValue 委托默认认证体系执行。
func DeleteSessionValue(ctx context.Context, loginID, key string) error {
	return AuthType("").DeleteSessionValue(ctx, loginID, key)
}

// SetSessionValueByToken delegates to the default authentication namespace. SetSessionValueByToken 委托默认认证体系执行。
func SetSessionValueByToken(ctx context.Context, token, key string, value any) error {
	return AuthType("").SetSessionValueByToken(ctx, token, key, value)
}

// GetSessionValueByToken delegates to the default authentication namespace. GetSessionValueByToken 委托默认认证体系执行。
func GetSessionValueByToken(ctx context.Context, token, key string) (any, bool, error) {
	return AuthType("").GetSessionValueByToken(ctx, token, key)
}

// DeleteSessionValueByToken delegates to the default authentication namespace. DeleteSessionValueByToken 委托默认认证体系执行。
func DeleteSessionValueByToken(ctx context.Context, token, key string) error {
	return AuthType("").DeleteSessionValueByToken(ctx, token, key)
}

// GetTokenValueByLoginID delegates to the default authentication namespace. GetTokenValueByLoginID 委托默认认证体系执行。
func GetTokenValueByLoginID(ctx context.Context, loginID string) (string, error) {
	return AuthType("").GetTokenValueByLoginID(ctx, loginID)
}

// GetTokenValueByLoginIDAndDevice delegates to the default authentication namespace. GetTokenValueByLoginIDAndDevice 委托默认认证体系执行。
func GetTokenValueByLoginIDAndDevice(ctx context.Context, loginID, device string) (string, error) {
	return AuthType("").GetTokenValueByLoginIDAndDevice(ctx, loginID, device)
}

// SearchTokenValue delegates to the default authentication namespace. SearchTokenValue 委托默认认证体系执行。
func SearchTokenValue(ctx context.Context, keyword string, start, size int) ([]string, error) {
	return AuthType("").SearchTokenValue(ctx, keyword, start, size)
}

// SearchSessionId delegates to the default authentication namespace. SearchSessionId 委托默认认证体系执行。
func SearchSessionId(ctx context.Context, keyword string, start, size int) ([]string, error) {
	return AuthType("").SearchSessionId(ctx, keyword, start, size)
}
