// @Author daixk 2026/06/05
package fiber

import (
	"github.com/Zany2/dtoken-go/core/manager"
	gofiber "github.com/gofiber/fiber/v2"
)

// KickoutByDeviceByContext kicks out current user by device KickoutByDeviceByContext 按设备踢出当前用户
func KickoutByDeviceByContext(c *gofiber.Ctx, device string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Terminal().KickoutByDevice(requestContext(c), device)
}

// KickoutByDeviceAndDeviceIDByContext kicks out current user by device ID KickoutByDeviceAndDeviceIDByContext 按设备和设备 ID 踢出当前用户
func KickoutByDeviceAndDeviceIDByContext(c *gofiber.Ctx, deviceAndDeviceID ...string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Terminal().KickoutByDeviceAndDeviceID(requestContext(c), deviceAndDeviceID...)
}

// ReplaceByDeviceByContext replaces current user by device ReplaceByDeviceByContext 按设备顶替当前用户
func ReplaceByDeviceByContext(c *gofiber.Ctx, device string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Terminal().ReplaceByDevice(requestContext(c), device)
}

// ReplaceByDeviceAndDeviceIDByContext replaces current user by device ID ReplaceByDeviceAndDeviceIDByContext 按设备和设备 ID 顶替当前用户
func ReplaceByDeviceAndDeviceIDByContext(c *gofiber.Ctx, deviceAndDeviceID ...string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Terminal().ReplaceByDeviceAndDeviceID(requestContext(c), deviceAndDeviceID...)
}

// KickoutByLoginIDByContext kicks out all terminals of current user KickoutByLoginIDByContext 踢出当前用户全部终端
func KickoutByLoginIDByContext(c *gofiber.Ctx) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Terminal().KickoutAll(requestContext(c))
}

// ReplaceByLoginIDByContext replaces all terminals of current user ReplaceByLoginIDByContext 顶替当前用户全部终端
func ReplaceByLoginIDByContext(c *gofiber.Ctx) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Terminal().ReplaceAll(requestContext(c))
}

// TerminateByContext terminates current or specified terminal TerminateByContext 下线当前或指定终端
func TerminateByContext(c *gofiber.Ctx, opts manager.TerminateOptions) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Terminal().Terminate(requestContext(c), opts)
}

// GetTokenValueListByDeviceByContext gets current user tokens by device GetTokenValueListByDeviceByContext 按设备获取当前用户 Token 列表
func GetTokenValueListByDeviceByContext(c *gofiber.Ctx, device string, checkAlive ...bool) ([]string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().GetTokenValueListByDevice(requestContext(c), device, checkAlive...)
}

// GetTokenValueListByDeviceAndDeviceIDByContext gets current user tokens by device ID GetTokenValueListByDeviceAndDeviceIDByContext 按设备和设备 ID 获取当前用户 Token 列表
func GetTokenValueListByDeviceAndDeviceIDByContext(c *gofiber.Ctx, device, deviceID string, checkAlive ...bool) ([]string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().GetTokenValueListByDeviceAndDeviceID(requestContext(c), device, deviceID, checkAlive...)
}

// GetOnlineTerminalCountByDeviceByContext gets online count by device GetOnlineTerminalCountByDeviceByContext 按设备获取在线终端数量
func GetOnlineTerminalCountByDeviceByContext(c *gofiber.Ctx, device string) (int, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return 0, err
	}
	return dCtx.Terminal().GetOnlineTerminalCountByDevice(requestContext(c), device)
}

// GetOnlineTerminalCountByDeviceAndDeviceIDByContext gets online count by device ID GetOnlineTerminalCountByDeviceAndDeviceIDByContext 按设备和设备 ID 获取在线终端数量
func GetOnlineTerminalCountByDeviceAndDeviceIDByContext(c *gofiber.Ctx, device, deviceID string) (int, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return 0, err
	}
	return dCtx.Terminal().GetOnlineTerminalCountByDeviceAndDeviceID(requestContext(c), device, deviceID)
}

// GetTerminalInfoByContext gets current terminal info GetTerminalInfoByContext 获取当前终端信息
func GetTerminalInfoByContext(c *gofiber.Ctx) (*manager.TerminalInfo, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().GetTerminalInfo(requestContext(c))
}

// GetTerminalListByContext gets current user terminal list GetTerminalListByContext 获取当前用户终端列表
func GetTerminalListByContext(c *gofiber.Ctx, device ...string) ([]manager.TerminalInfo, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().GetTerminalList(requestContext(c), device...)
}

// GetLatestTokenValueByContext gets latest current user token GetLatestTokenValueByContext 获取当前用户最新 Token
func GetLatestTokenValueByContext(c *gofiber.Ctx, device ...string) (string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return "", err
	}
	return dCtx.Terminal().GetLatestTokenValue(requestContext(c), device...)
}

// SearchTokenValueByContext searches token values SearchTokenValueByContext 搜索 Token 值
func SearchTokenValueByContext(c *gofiber.Ctx, keyword string, start, size int) ([]string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().SearchTokenValue(requestContext(c), keyword, start, size)
}

// SearchSessionIDByContext searches session ids SearchSessionIDByContext 鎼滅。Session ID
func SearchSessionIDByContext(c *gofiber.Ctx, keyword string, start, size int) ([]string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().SearchSessionId(requestContext(c), keyword, start, size)
}

// ForEachTerminalByContext visits current user terminals ForEachTerminalByContext 遍历当前用户终端
func ForEachTerminalByContext(c *gofiber.Ctx, visitor manager.TerminalVisitor) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Terminal().ForEachTerminal(requestContext(c), visitor)
}

// ForEachTerminalByDeviceByContext visits current user terminals by device ForEachTerminalByDeviceByContext 按设备遍历当前用户终端
func ForEachTerminalByDeviceByContext(c *gofiber.Ctx, device string, visitor manager.TerminalVisitor) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Terminal().ForEachTerminalByDevice(requestContext(c), device, visitor)
}
