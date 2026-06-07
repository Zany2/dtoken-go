// @Author daixk 2026/06/06
package beego

import (
	"github.com/Zany2/dtoken-go/core/manager"
	beegocontext "github.com/beego/beego/v2/server/web/context"
)

// LogoutByDeviceByContext logs out current user by device LogoutByDeviceByContext 按设备登出当前用户
func LogoutByDeviceByContext(c *beegocontext.Context, device string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Terminal().LogoutByDevice(requestContext(c), device)
}

// LogoutByDeviceAndDeviceIdByContext logs out current user by device and id LogoutByDeviceAndDeviceIdByContext 按设备和设备 ID 登出当前用户
func LogoutByDeviceAndDeviceIdByContext(c *beegocontext.Context, deviceAndDeviceId ...string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Terminal().LogoutByDeviceAndDeviceId(requestContext(c), deviceAndDeviceId...)
}

// LogoutByLoginIDByContext logs out all terminals of current user LogoutByLoginIDByContext 登出当前用户全部终端
func LogoutByLoginIDByContext(c *beegocontext.Context) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Terminal().LogoutAll(requestContext(c))
}

// KickoutByDeviceByContext kicks out current user by device KickoutByDeviceByContext 按设备踢出当前用户
func KickoutByDeviceByContext(c *beegocontext.Context, device string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Terminal().KickoutByDevice(requestContext(c), device)
}

// KickoutByDeviceAndDeviceIDByContext kicks out current user by device ID KickoutByDeviceAndDeviceIDByContext 按设备 ID 踢出当前用户
func KickoutByDeviceAndDeviceIDByContext(c *beegocontext.Context, deviceAndDeviceId ...string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Terminal().KickoutByDeviceAndDeviceId(requestContext(c), deviceAndDeviceId...)
}

// KickoutByLoginIDByContext kicks out all terminals of current user KickoutByLoginIDByContext 踢出当前用户全部终端
func KickoutByLoginIDByContext(c *beegocontext.Context) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Terminal().KickoutAll(requestContext(c))
}

// ReplaceByDeviceByContext replaces current user by device ReplaceByDeviceByContext 按设备顶替当前用户
func ReplaceByDeviceByContext(c *beegocontext.Context, device string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Terminal().ReplaceByDevice(requestContext(c), device)
}

// ReplaceByDeviceAndDeviceIDByContext replaces current user by device ID ReplaceByDeviceAndDeviceIDByContext 按设备 ID 顶替当前用户
func ReplaceByDeviceAndDeviceIDByContext(c *beegocontext.Context, deviceAndDeviceId ...string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Terminal().ReplaceByDeviceAndDeviceId(requestContext(c), deviceAndDeviceId...)
}

// ReplaceByLoginIDByContext replaces all terminals of current user ReplaceByLoginIDByContext 顶替当前用户全部终端
func ReplaceByLoginIDByContext(c *beegocontext.Context) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Terminal().ReplaceAll(requestContext(c))
}

// TerminateByContext terminates current or specified terminal TerminateByContext 下线当前或指定终端
func TerminateByContext(c *beegocontext.Context, opts manager.TerminateOptions) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Terminal().Terminate(requestContext(c), opts)
}

// GetTokenValueListByContext gets current user token list GetTokenValueListByContext 获取当前用户 token 列表
func GetTokenValueListByContext(c *beegocontext.Context, checkAlive ...bool) ([]string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().GetTokenValueList(requestContext(c), checkAlive...)
}

// GetTokenValueListByDeviceByContext gets current user tokens by device GetTokenValueListByDeviceByContext 按设备获取当前用户 token 列表
func GetTokenValueListByDeviceByContext(c *beegocontext.Context, device string, checkAlive ...bool) ([]string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().GetTokenValueListByDevice(requestContext(c), device, checkAlive...)
}

// GetTokenValueListByDeviceAndDeviceIDByContext gets current user tokens by device ID GetTokenValueListByDeviceAndDeviceIDByContext 按设备 ID 获取当前用户 token 列表
func GetTokenValueListByDeviceAndDeviceIDByContext(c *beegocontext.Context, device, deviceId string, checkAlive ...bool) ([]string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().GetTokenValueListByDeviceAndDeviceId(requestContext(c), device, deviceId, checkAlive...)
}

// GetOnlineTerminalCountByContext gets current user online terminal count GetOnlineTerminalCountByContext 获取当前用户在线终端数量
func GetOnlineTerminalCountByContext(c *beegocontext.Context) (int, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return 0, err
	}
	return dCtx.Terminal().GetOnlineTerminalCount(requestContext(c))
}

// GetOnlineTerminalCountByDeviceByContext gets online count by device GetOnlineTerminalCountByDeviceByContext 按设备获取在线数量
func GetOnlineTerminalCountByDeviceByContext(c *beegocontext.Context, device string) (int, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return 0, err
	}
	return dCtx.Terminal().GetOnlineTerminalCountByDevice(requestContext(c), device)
}

// GetOnlineTerminalCountByDeviceAndDeviceIDByContext gets online count by device ID GetOnlineTerminalCountByDeviceAndDeviceIDByContext 按设备 ID 获取在线数量
func GetOnlineTerminalCountByDeviceAndDeviceIDByContext(c *beegocontext.Context, device, deviceId string) (int, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return 0, err
	}
	return dCtx.Terminal().GetOnlineTerminalCountByDeviceAndDeviceId(requestContext(c), device, deviceId)
}

// GetTerminalInfoByContext gets current terminal info GetTerminalInfoByContext 获取当前终端信息
func GetTerminalInfoByContext(c *beegocontext.Context) (*manager.TerminalInfo, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().GetTerminalInfo(requestContext(c))
}

// GetTerminalListByContext gets current user terminal list GetTerminalListByContext 获取当前用户终端列表
func GetTerminalListByContext(c *beegocontext.Context, device ...string) ([]manager.TerminalInfo, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().GetTerminalList(requestContext(c), device...)
}

// GetLatestTokenValueByContext gets latest current user token GetLatestTokenValueByContext 获取当前用户最新 token
func GetLatestTokenValueByContext(c *beegocontext.Context, device ...string) (string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return "", err
	}
	return dCtx.Terminal().GetLatestTokenValue(requestContext(c), device...)
}

// SearchTokenValueByContext searches token values SearchTokenValueByContext 搜索 token 值
func SearchTokenValueByContext(c *beegocontext.Context, keyword string, start, size int) ([]string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().SearchTokenValue(requestContext(c), keyword, start, size)
}

// SearchSessionIDByContext searches session ids SearchSessionIDByContext 搜索 session ID
func SearchSessionIDByContext(c *beegocontext.Context, keyword string, start, size int) ([]string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().SearchSessionId(requestContext(c), keyword, start, size)
}

// ForEachTerminalByContext visits current user terminals ForEachTerminalByContext 遍历当前用户终端
func ForEachTerminalByContext(c *beegocontext.Context, visitor manager.TerminalVisitor) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Terminal().ForEachTerminal(requestContext(c), visitor)
}

// ForEachTerminalByDeviceByContext visits current user terminals by device ForEachTerminalByDeviceByContext 按设备遍历当前用户终端
func ForEachTerminalByDeviceByContext(c *beegocontext.Context, device string, visitor manager.TerminalVisitor) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Terminal().ForEachTerminalByDevice(requestContext(c), device, visitor)
}
