// @Author daixk 2025/12/22 15:56:00
package dtoken

import (
	"context"

	"github.com/Zany2/dtoken-go/core/manager"
)

// GetOnlineTerminalCount returns online terminal count. GetOnlineTerminalCount 获取在线终端数量。
func (a *Auth) GetOnlineTerminalCount(ctx context.Context, loginID string) (int, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return 0, err
	}
	return mgr.GetOnlineTerminalCount(ctx, loginID)
}

// GetOnlineTerminalCountByDevice returns online terminal count for a device. GetOnlineTerminalCountByDevice 获取指定设备在线终端数量。
func (a *Auth) GetOnlineTerminalCountByDevice(ctx context.Context, loginID, device string) (int, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return 0, err
	}
	return mgr.GetOnlineTerminalCountByDevice(ctx, loginID, device)
}

// GetOnlineTerminalCountByDeviceAndDeviceId returns online terminal count for a device ID. GetOnlineTerminalCountByDeviceAndDeviceId 获取指定设备 ID 在线终端数量。
func (a *Auth) GetOnlineTerminalCountByDeviceAndDeviceId(ctx context.Context, loginID, device, deviceID string) (int, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return 0, err
	}
	return mgr.GetOnlineTerminalCountByDeviceAndDeviceId(ctx, loginID, device, deviceID)
}

// ForEachTerminal visits each terminal for a login ID. ForEachTerminal 遍历指定账号的终端。
func (a *Auth) ForEachTerminal(ctx context.Context, loginID string, visitor manager.TerminalVisitor) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.ForEachTerminal(ctx, loginID, visitor)
}

// ForEachTerminalByDevice visits each terminal for a device. ForEachTerminalByDevice 遍历指定设备的终端。
func (a *Auth) ForEachTerminalByDevice(ctx context.Context, loginID, device string, visitor manager.TerminalVisitor) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.ForEachTerminalByDevice(ctx, loginID, device, visitor)
}

// GetSession gets session by login id. GetSession 根据登录 ID 获取会话。
func (a *Auth) GetSession(ctx context.Context, loginID string) (*manager.Session, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.GetSession(ctx, loginID)
}

// GetSessionByToken gets session by token. GetSessionByToken 根据 Token 获取会话。
func (a *Auth) GetSessionByToken(ctx context.Context, token string) (*manager.Session, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.GetSessionByToken(ctx, token)
}

// GetTokenValueListByLoginID returns token values for a login ID. GetTokenValueListByLoginID 获取账号的 Token 列表。
func (a *Auth) GetTokenValueListByLoginID(ctx context.Context, loginID string, checkAlive bool) ([]string, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.GetTokenValueListByLoginID(ctx, loginID, checkAlive)
}

// GetTokenValueListByDeviceAndDeviceId returns token values for a device ID. GetTokenValueListByDeviceAndDeviceId 获取指定设备 ID 的 Token 列表。
func (a *Auth) GetTokenValueListByDeviceAndDeviceId(ctx context.Context, loginID, device, deviceID string, checkAlive bool) ([]string, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.GetTokenValueListByDeviceAndDeviceId(ctx, loginID, device, deviceID, checkAlive)
}

// GetTokenValueListByDevice returns token values for a device. GetTokenValueListByDevice 获取指定设备的 Token 列表。
func (a *Auth) GetTokenValueListByDevice(ctx context.Context, loginID, device string, checkAlive bool) ([]string, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.GetTokenValueListByDevice(ctx, loginID, device, checkAlive)
}

// GetTerminalListByLoginID returns terminal list for a login ID. GetTerminalListByLoginID 获取账号终端列表。
func (a *Auth) GetTerminalListByLoginID(ctx context.Context, loginID string) ([]manager.TerminalInfo, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.GetTerminalListByLoginID(ctx, loginID)
}

// GetTerminalListByLoginIDAndDevice returns terminal list for a device. GetTerminalListByLoginIDAndDevice 获取指定设备的终端列表。
func (a *Auth) GetTerminalListByLoginIDAndDevice(ctx context.Context, loginID, device string) ([]manager.TerminalInfo, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.GetTerminalListByLoginID(ctx, loginID, device)
}

// GetTerminalInfoByToken gets terminal info by token. GetTerminalInfoByToken 根据 Token 获取终端信息。
func (a *Auth) GetTerminalInfoByToken(ctx context.Context, token string) (*manager.TerminalInfo, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.GetTerminalInfoByToken(ctx, token)
}

// SetSessionValue sets one session data value. SetSessionValue 设置一个会话扩展数据。
func (a *Auth) SetSessionValue(ctx context.Context, loginID, key string, value any) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.SetSessionValue(ctx, loginID, key, value)
}

// GetSessionValue gets one session data value. GetSessionValue 获取一个会话扩展数据。
func (a *Auth) GetSessionValue(ctx context.Context, loginID, key string) (any, bool, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, false, err
	}
	return mgr.GetSessionValue(ctx, loginID, key)
}

// DeleteSessionValue deletes one session data value. DeleteSessionValue 删除一个会话扩展数据。
func (a *Auth) DeleteSessionValue(ctx context.Context, loginID, key string) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.DeleteSessionValue(ctx, loginID, key)
}

// GetTokenValueByLoginID returns the latest token for a login ID. GetTokenValueByLoginID 获取账号最新 Token。
func (a *Auth) GetTokenValueByLoginID(ctx context.Context, loginID string) (string, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return "", err
	}
	return mgr.GetTokenValueByLoginID(ctx, loginID)
}

// GetTokenValueByLoginIDAndDevice returns the latest token for a device. GetTokenValueByLoginIDAndDevice 获取指定设备的最新 Token。
func (a *Auth) GetTokenValueByLoginIDAndDevice(ctx context.Context, loginID, device string) (string, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return "", err
	}
	return mgr.GetTokenValueByLoginID(ctx, loginID, device)
}

// SearchTokenValue searches token values by keyword. SearchTokenValue 按关键字搜索 Token。
func (a *Auth) SearchTokenValue(ctx context.Context, keyword string, start, size int) ([]string, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.SearchTokenValue(ctx, keyword, start, size)
}

// SearchSessionId searches session IDs by keyword. SearchSessionId 按关键字搜索会话 ID。
func (a *Auth) SearchSessionId(ctx context.Context, keyword string, start, size int) ([]string, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.SearchSessionId(ctx, keyword, start, size)
}
