// @Author daixk 2025/12/22 15:56:00
package dtoken

import (
	"context"

	"github.com/Zany2/dtoken-go/core/manager"
)

// GetOnlineTerminalCount returns the online terminal count. GetOnlineTerminalCount 获取在线终端数量。
func GetOnlineTerminalCount(ctx context.Context, loginID string, authType ...string) (int, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return 0, err
	}
	return mgr.GetOnlineTerminalCount(ctx, loginID)
}

// GetOnlineTerminalCountByDevice returns online terminal count for a device. GetOnlineTerminalCountByDevice 获取指定设备在线终端数量。
func GetOnlineTerminalCountByDevice(ctx context.Context, loginID string, device string, authType ...string) (int, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return 0, err
	}
	return mgr.GetOnlineTerminalCountByDevice(ctx, loginID, device)
}

// GetOnlineTerminalCountByDeviceAndDeviceId returns online terminal count for a device ID. GetOnlineTerminalCountByDeviceAndDeviceId 获取指定设备 ID 在线终端数量。
func GetOnlineTerminalCountByDeviceAndDeviceId(ctx context.Context, loginID string, device string, deviceId string, authType ...string) (int, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return 0, err
	}
	return mgr.GetOnlineTerminalCountByDeviceAndDeviceId(ctx, loginID, device, deviceId)
}

// ForEachTerminal visits each terminal for a login ID. ForEachTerminal 遍历指定账号的终端。
func ForEachTerminal(ctx context.Context, loginID string, visitor manager.TerminalVisitor, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.ForEachTerminal(ctx, loginID, visitor)
}

// ForEachTerminalByDevice visits each terminal for a device. ForEachTerminalByDevice 遍历指定设备的终端。
func ForEachTerminalByDevice(ctx context.Context, loginID, device string, visitor manager.TerminalVisitor, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.ForEachTerminalByDevice(ctx, loginID, device, visitor)
}

// GetSession returns a session by login ID. GetSession 按登录 ID 获取会话。
func GetSession(ctx context.Context, loginID string, authType ...string) (*manager.Session, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.GetSession(ctx, loginID)
}

// GetSessionByToken returns a session by token. GetSessionByToken 按 token 获取会话。
func GetSessionByToken(ctx context.Context, tokenValue string, authType ...string) (*manager.Session, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.GetSessionByToken(ctx, tokenValue)
}

// GetTokenValueListByLoginID returns token values for a login ID. GetTokenValueListByLoginID 获取账号的 token 列表。
func GetTokenValueListByLoginID(ctx context.Context, loginID string, checkAlive bool, authType ...string) ([]string, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.GetTokenValueListByLoginID(ctx, loginID, checkAlive)
}

// GetTokenValueListByDeviceAndDeviceId returns token values for a device ID. GetTokenValueListByDeviceAndDeviceId 获取指定设备 ID 的 token 列表。
func GetTokenValueListByDeviceAndDeviceId(ctx context.Context, loginID string, device string, deviceId string, checkAlive bool, authType ...string) ([]string, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.GetTokenValueListByDeviceAndDeviceId(ctx, loginID, device, deviceId, checkAlive)
}

// GetTokenValueListByDevice returns token values for a device. GetTokenValueListByDevice 获取指定设备的 token 列表。
func GetTokenValueListByDevice(ctx context.Context, loginID string, device string, checkAlive bool, authType ...string) ([]string, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.GetTokenValueListByDevice(ctx, loginID, device, checkAlive)
}

// GetTerminalListByLoginID returns terminal list for a login ID. GetTerminalListByLoginID 获取账号终端列表。
func GetTerminalListByLoginID(ctx context.Context, loginID string, authType ...string) ([]manager.TerminalInfo, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.GetTerminalListByLoginID(ctx, loginID)
}

// GetTerminalListByLoginIDAndDevice returns terminal list for a device. GetTerminalListByLoginIDAndDevice 获取指定设备的终端列表。
func GetTerminalListByLoginIDAndDevice(ctx context.Context, loginID string, device string, authType ...string) ([]manager.TerminalInfo, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.GetTerminalListByLoginID(ctx, loginID, device)
}

// GetTerminalInfoByToken returns terminal information by token. GetTerminalInfoByToken 按 token 获取终端信息。
func GetTerminalInfoByToken(ctx context.Context, tokenValue string, authType ...string) (*manager.TerminalInfo, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.GetTerminalInfoByToken(ctx, tokenValue)
}

// GetTokenValueByLoginID returns the latest token for a login ID. GetTokenValueByLoginID 获取账号最新 token。
func GetTokenValueByLoginID(ctx context.Context, loginID string, authType ...string) (string, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return "", err
	}
	return mgr.GetTokenValueByLoginID(ctx, loginID)
}

// GetTokenValueByLoginIDAndDevice returns the latest token for a device. GetTokenValueByLoginIDAndDevice 获取指定设备的最新 token。
func GetTokenValueByLoginIDAndDevice(ctx context.Context, loginID string, device string, authType ...string) (string, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return "", err
	}
	return mgr.GetTokenValueByLoginID(ctx, loginID, device)
}

// SearchTokenValue searches token values by keyword. SearchTokenValue 按关键字搜索 token。
func SearchTokenValue(ctx context.Context, keyword string, start, size int, authType ...string) ([]string, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.SearchTokenValue(ctx, keyword, start, size)
}

// SearchSessionId searches session IDs by keyword. SearchSessionId 按关键字搜索会话 ID。
func SearchSessionId(ctx context.Context, keyword string, start, size int, authType ...string) ([]string, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.SearchSessionId(ctx, keyword, start, size)
}
