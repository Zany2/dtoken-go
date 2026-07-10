// @Author daixk 2026/06/05
package hertz

import (
	"github.com/Zany2/dtoken-go/core/manager"
	hertzapp "github.com/cloudwego/hertz/pkg/app"
)

// KickoutByDeviceByContext kicks out current user by device KickoutByDeviceByContext 按设备踢出当前用户
func KickoutByDeviceByContext(ctx *hertzapp.RequestContext, device string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().KickoutByDevice(requestContext(ctx), device)
}

// KickoutByDeviceAndDeviceIDByContext kicks out current user by device ID KickoutByDeviceAndDeviceIDByContext 按设备和设备 ID 踢出当前用户
func KickoutByDeviceAndDeviceIDByContext(ctx *hertzapp.RequestContext, deviceAndDeviceID ...string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().KickoutByDeviceAndDeviceID(requestContext(ctx), deviceAndDeviceID...)
}

// ReplaceByDeviceByContext replaces current user by device ReplaceByDeviceByContext 按设备顶替当前用户
func ReplaceByDeviceByContext(ctx *hertzapp.RequestContext, device string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().ReplaceByDevice(requestContext(ctx), device)
}

// ReplaceByDeviceAndDeviceIDByContext replaces current user by device ID ReplaceByDeviceAndDeviceIDByContext 按设备和设备 ID 顶替当前用户
func ReplaceByDeviceAndDeviceIDByContext(ctx *hertzapp.RequestContext, deviceAndDeviceID ...string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().ReplaceByDeviceAndDeviceID(requestContext(ctx), deviceAndDeviceID...)
}

// KickoutByLoginIDByContext kicks out all terminals of current user KickoutByLoginIDByContext 踢出当前用户全部终端
func KickoutByLoginIDByContext(ctx *hertzapp.RequestContext) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().KickoutAll(requestContext(ctx))
}

// ReplaceByLoginIDByContext replaces all terminals of current user ReplaceByLoginIDByContext 顶替当前用户全部终端
func ReplaceByLoginIDByContext(ctx *hertzapp.RequestContext) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().ReplaceAll(requestContext(ctx))
}

// TerminateByContext terminates current or specified terminal TerminateByContext 下线当前或指定终端
func TerminateByContext(ctx *hertzapp.RequestContext, opts manager.TerminateOptions) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().Terminate(requestContext(ctx), opts)
}

// GetTokenValueListByDeviceByContext gets current user tokens by device GetTokenValueListByDeviceByContext 按设备获取当前用户 Token 列表
func GetTokenValueListByDeviceByContext(ctx *hertzapp.RequestContext, device string, checkAlive ...bool) ([]string, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().GetTokenValueListByDevice(requestContext(ctx), device, checkAlive...)
}

// GetTokenValueListByDeviceAndDeviceIDByContext gets current user tokens by device ID GetTokenValueListByDeviceAndDeviceIDByContext 按设备和设备 ID 获取当前用户 Token 列表
func GetTokenValueListByDeviceAndDeviceIDByContext(ctx *hertzapp.RequestContext, device, deviceID string, checkAlive ...bool) ([]string, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().GetTokenValueListByDeviceAndDeviceID(requestContext(ctx), device, deviceID, checkAlive...)
}

// GetOnlineTerminalCountByDeviceByContext gets online count by device GetOnlineTerminalCountByDeviceByContext 按设备获取在线终端数量
func GetOnlineTerminalCountByDeviceByContext(ctx *hertzapp.RequestContext, device string) (int, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return 0, err
	}
	return dCtx.Terminal().GetOnlineTerminalCountByDevice(requestContext(ctx), device)
}

// GetOnlineTerminalCountByDeviceAndDeviceIDByContext gets online count by device ID GetOnlineTerminalCountByDeviceAndDeviceIDByContext 按设备和设备 ID 获取在线终端数量
func GetOnlineTerminalCountByDeviceAndDeviceIDByContext(ctx *hertzapp.RequestContext, device, deviceID string) (int, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return 0, err
	}
	return dCtx.Terminal().GetOnlineTerminalCountByDeviceAndDeviceID(requestContext(ctx), device, deviceID)
}

// GetTerminalInfoByContext gets current terminal info GetTerminalInfoByContext 获取当前终端信息
func GetTerminalInfoByContext(ctx *hertzapp.RequestContext) (*manager.TerminalInfo, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().GetTerminalInfo(requestContext(ctx))
}

// GetTerminalListByContext gets current user terminal list GetTerminalListByContext 获取当前用户终端列表
func GetTerminalListByContext(ctx *hertzapp.RequestContext, device ...string) ([]manager.TerminalInfo, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().GetTerminalList(requestContext(ctx), device...)
}

// GetLatestTokenValueByContext gets latest current user token GetLatestTokenValueByContext 获取当前用户最新 Token
func GetLatestTokenValueByContext(ctx *hertzapp.RequestContext, device ...string) (string, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.Terminal().GetLatestTokenValue(requestContext(ctx), device...)
}

// SearchTokenValueByContext searches token values SearchTokenValueByContext 搜索 Token 值
func SearchTokenValueByContext(ctx *hertzapp.RequestContext, keyword string, start, size int) ([]string, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().SearchTokenValue(requestContext(ctx), keyword, start, size)
}

// SearchSessionIDByContext searches session ids SearchSessionIDByContext 搜索 Session ID
func SearchSessionIDByContext(ctx *hertzapp.RequestContext, keyword string, start, size int) ([]string, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().SearchSessionId(requestContext(ctx), keyword, start, size)
}

// ForEachTerminalByContext visits current user terminals ForEachTerminalByContext 遍历当前用户终端
func ForEachTerminalByContext(ctx *hertzapp.RequestContext, visitor manager.TerminalVisitor) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().ForEachTerminal(requestContext(ctx), visitor)
}

// ForEachTerminalByDeviceByContext visits current user terminals by device ForEachTerminalByDeviceByContext 按设备遍历当前用户终端
func ForEachTerminalByDeviceByContext(ctx *hertzapp.RequestContext, device string, visitor manager.TerminalVisitor) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().ForEachTerminalByDevice(requestContext(ctx), device, visitor)
}
