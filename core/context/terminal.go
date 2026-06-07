// @Author daixk 2026/06/05
package context

import (
	"context"

	"github.com/Zany2/dtoken-go/core/manager"
)

// Logout logs out current token Logout 注销当前 Token
func (c *TerminalContext) Logout(ctx context.Context) error {
	return c.d.Auth().Logout(ctx)
}

// Kickout kicks out current token Kickout 踢出当前 Token
func (c *TerminalContext) Kickout(ctx context.Context) error {
	return c.d.Auth().Kickout(ctx)
}

// Replace replaces current token Replace 顶替当前 Token
func (c *TerminalContext) Replace(ctx context.Context) error {
	return c.d.Auth().Replace(ctx)
}

// LogoutByDevice logs out current account terminals by device LogoutByDevice 按设备注销当前账号终端
func (c *TerminalContext) LogoutByDevice(ctx context.Context, device string) error {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return err
	}
	return c.d.manager.LogoutByDevice(ctx, loginID, device)
}

// LogoutByDeviceAndDeviceId logs out current account terminals by device ID LogoutByDeviceAndDeviceId 按设备和设备 ID 注销当前账号终端
func (c *TerminalContext) LogoutByDeviceAndDeviceId(ctx context.Context, deviceAndDeviceId ...string) error {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return err
	}
	return c.d.manager.LogoutByDeviceAndDeviceId(ctx, loginID, deviceAndDeviceId...)
}

// KickoutByDevice kicks out current account terminals by device KickoutByDevice 按设备踢出当前账号终端
func (c *TerminalContext) KickoutByDevice(ctx context.Context, device string) error {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return err
	}
	return c.d.manager.KickoutByDevice(ctx, loginID, device)
}

// KickoutByDeviceAndDeviceId kicks out current account terminals by device ID KickoutByDeviceAndDeviceId 按设备和设备 ID 踢出当前账号终端
func (c *TerminalContext) KickoutByDeviceAndDeviceId(ctx context.Context, deviceAndDeviceId ...string) error {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return err
	}
	return c.d.manager.KickoutByDeviceAndDeviceId(ctx, loginID, deviceAndDeviceId...)
}

// ReplaceByDevice replaces current account terminals by device ReplaceByDevice 按设备顶替当前账号终端
func (c *TerminalContext) ReplaceByDevice(ctx context.Context, device string) error {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return err
	}
	return c.d.manager.ReplaceByDevice(ctx, loginID, device)
}

// ReplaceByDeviceAndDeviceId replaces current account terminals by device ID ReplaceByDeviceAndDeviceId 按设备和设备 ID 顶替当前账号终端
func (c *TerminalContext) ReplaceByDeviceAndDeviceId(ctx context.Context, deviceAndDeviceId ...string) error {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return err
	}
	return c.d.manager.ReplaceByDeviceAndDeviceId(ctx, loginID, deviceAndDeviceId...)
}

// LogoutAll logs out all current account terminals LogoutAll 注销当前账号全部终端
func (c *TerminalContext) LogoutAll(ctx context.Context) error {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return err
	}
	return c.d.manager.LogoutByLoginID(ctx, loginID)
}

// KickoutAll kicks out all current account terminals KickoutAll 踢出当前账号全部终端
func (c *TerminalContext) KickoutAll(ctx context.Context) error {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return err
	}
	return c.d.manager.KickoutByLoginID(ctx, loginID)
}

// ReplaceAll replaces all current account terminals ReplaceAll 顶替当前账号全部终端
func (c *TerminalContext) ReplaceAll(ctx context.Context) error {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return err
	}
	return c.d.manager.ReplaceByLoginID(ctx, loginID)
}

// Terminate applies one terminal operation Terminate 按选项执行一次终端下线操作
func (c *TerminalContext) Terminate(ctx context.Context, opts manager.TerminateOptions) error {
	if opts.Token == "" && opts.LoginID == "" {
		loginID, err := c.d.currentLoginID(ctx)
		if err != nil {
			return err
		}
		opts.LoginID = loginID
	}
	return c.d.manager.Terminate(ctx, opts)
}

// GetTokenValueList gets current account token list GetTokenValueList 获取当前账号 Token 列表
func (c *TerminalContext) GetTokenValueList(ctx context.Context, checkAlive ...bool) ([]string, error) {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return nil, err
	}
	return c.d.manager.GetTokenValueListByLoginID(ctx, loginID, checkAlive...)
}

// GetTokenValueListByDevice gets current account token list by device GetTokenValueListByDevice 按设备获取当前账号 Token 列表
func (c *TerminalContext) GetTokenValueListByDevice(ctx context.Context, device string, checkAlive ...bool) ([]string, error) {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return nil, err
	}
	return c.d.manager.GetTokenValueListByDevice(ctx, loginID, device, checkAlive...)
}

// GetTokenValueListByDeviceAndDeviceId gets current account token list by device ID GetTokenValueListByDeviceAndDeviceId 按设备和设备 ID 获取当前账号 Token 列表
func (c *TerminalContext) GetTokenValueListByDeviceAndDeviceId(ctx context.Context, device, deviceId string, checkAlive ...bool) ([]string, error) {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return nil, err
	}
	return c.d.manager.GetTokenValueListByDeviceAndDeviceId(ctx, loginID, device, deviceId, checkAlive...)
}

// GetOnlineTerminalCount gets current account online count GetOnlineTerminalCount 获取当前账号在线终端数
func (c *TerminalContext) GetOnlineTerminalCount(ctx context.Context) (int, error) {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return 0, err
	}
	return c.d.manager.GetOnlineTerminalCount(ctx, loginID)
}

// GetOnlineTerminalCountByDevice gets current account online count by device GetOnlineTerminalCountByDevice 按设备获取当前账号在线终端数
func (c *TerminalContext) GetOnlineTerminalCountByDevice(ctx context.Context, device string) (int, error) {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return 0, err
	}
	return c.d.manager.GetOnlineTerminalCountByDevice(ctx, loginID, device)
}

// GetOnlineTerminalCountByDeviceAndDeviceId gets current account online count by device ID GetOnlineTerminalCountByDeviceAndDeviceId 按设备和设备 ID 获取当前账号在线终端数
func (c *TerminalContext) GetOnlineTerminalCountByDeviceAndDeviceId(ctx context.Context, device, deviceId string) (int, error) {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return 0, err
	}
	return c.d.manager.GetOnlineTerminalCountByDeviceAndDeviceId(ctx, loginID, device, deviceId)
}

// GetTerminalInfo gets current terminal info GetTerminalInfo 获取当前终端信息
func (c *TerminalContext) GetTerminalInfo(ctx context.Context) (*manager.TerminalInfo, error) {
	token, err := c.d.requireToken()
	if err != nil {
		return nil, err
	}
	return c.d.manager.GetTerminalInfoByToken(ctx, token)
}

// GetTerminalList gets current account terminal list GetTerminalList 获取当前账号终端列表
func (c *TerminalContext) GetTerminalList(ctx context.Context, device ...string) ([]manager.TerminalInfo, error) {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return nil, err
	}
	return c.d.manager.GetTerminalListByLoginID(ctx, loginID, device...)
}

// GetLatestTokenValue gets latest current account token GetLatestTokenValue 获取当前账号最新 Token
func (c *TerminalContext) GetLatestTokenValue(ctx context.Context, device ...string) (string, error) {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return "", err
	}
	return c.d.manager.GetTokenValueByLoginID(ctx, loginID, device...)
}

// SearchTokenValue searches token values SearchTokenValue 搜索 Token 值
func (c *TerminalContext) SearchTokenValue(ctx context.Context, keyword string, start, size int) ([]string, error) {
	return c.d.manager.SearchTokenValue(ctx, keyword, start, size)
}

// SearchSessionId searches session ids SearchSessionId 搜索 Session ID
func (c *TerminalContext) SearchSessionId(ctx context.Context, keyword string, start, size int) ([]string, error) {
	return c.d.manager.SearchSessionId(ctx, keyword, start, size)
}

// ForEachTerminal visits current account terminals ForEachTerminal 遍历当前账号终端
func (c *TerminalContext) ForEachTerminal(ctx context.Context, visitor manager.TerminalVisitor) error {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return err
	}
	return c.d.manager.ForEachTerminal(ctx, loginID, visitor)
}

// ForEachTerminalByDevice visits current account terminals by device ForEachTerminalByDevice 按设备遍历当前账号终端
func (c *TerminalContext) ForEachTerminalByDevice(ctx context.Context, device string, visitor manager.TerminalVisitor) error {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return err
	}
	return c.d.manager.ForEachTerminalByDevice(ctx, loginID, device, visitor)
}
