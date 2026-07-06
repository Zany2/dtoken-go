// @Author daixk 2026/06/05
package chi

import (
	"context"

	"github.com/Zany2/dtoken-go/core/manager"
)

// KickoutByDeviceByCtx kicks out current user by device KickoutByDeviceByCtx 按设备踢出当前用户
func KickoutByDeviceByCtx(ctx context.Context, device string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().KickoutByDevice(ctx, device)
}

// KickoutByDeviceAndDeviceIDByCtx kicks out current user by device ID KickoutByDeviceAndDeviceIDByCtx 按设备和设备 ID 踢出当前用户
func KickoutByDeviceAndDeviceIDByCtx(ctx context.Context, deviceAndDeviceID ...string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().KickoutByDeviceAndDeviceID(ctx, deviceAndDeviceID...)
}

// ReplaceByDeviceByCtx replaces current user by device ReplaceByDeviceByCtx 按设备顶替当前用户
func ReplaceByDeviceByCtx(ctx context.Context, device string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().ReplaceByDevice(ctx, device)
}

// ReplaceByDeviceAndDeviceIDByCtx replaces current user by device ID ReplaceByDeviceAndDeviceIDByCtx 按设备和设备 ID 顶替当前用户
func ReplaceByDeviceAndDeviceIDByCtx(ctx context.Context, deviceAndDeviceID ...string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().ReplaceByDeviceAndDeviceID(ctx, deviceAndDeviceID...)
}

// KickoutByLoginIDByCtx kicks out all terminals of current user KickoutByLoginIDByCtx 踢出当前用户全部终端
func KickoutByLoginIDByCtx(ctx context.Context) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().KickoutAll(ctx)
}

// ReplaceByLoginIDByCtx replaces all terminals of current user ReplaceByLoginIDByCtx 顶替当前用户全部终端
func ReplaceByLoginIDByCtx(ctx context.Context) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().ReplaceAll(ctx)
}

// TerminateByCtx terminates current or specified terminal TerminateByCtx 下线当前或指定终端
func TerminateByCtx(ctx context.Context, opts manager.TerminateOptions) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().Terminate(ctx, opts)
}

// GetTokenValueListByDeviceByCtx gets current user tokens by device GetTokenValueListByDeviceByCtx 按设备获取当前用户 Token 列表
func GetTokenValueListByDeviceByCtx(ctx context.Context, device string, checkAlive ...bool) ([]string, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().GetTokenValueListByDevice(ctx, device, checkAlive...)
}

// GetTokenValueListByDeviceAndDeviceIDByCtx gets current user tokens by device ID GetTokenValueListByDeviceAndDeviceIDByCtx 按设备和设备 ID 获取当前用户 Token 列表
func GetTokenValueListByDeviceAndDeviceIDByCtx(ctx context.Context, device, deviceID string, checkAlive ...bool) ([]string, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().GetTokenValueListByDeviceAndDeviceID(ctx, device, deviceID, checkAlive...)
}

// GetOnlineTerminalCountByDeviceByCtx gets online count by device GetOnlineTerminalCountByDeviceByCtx 按设备获取在线终端数量
func GetOnlineTerminalCountByDeviceByCtx(ctx context.Context, device string) (int, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return 0, err
	}
	return dCtx.Terminal().GetOnlineTerminalCountByDevice(ctx, device)
}

// GetOnlineTerminalCountByDeviceAndDeviceIDByCtx gets online count by device ID GetOnlineTerminalCountByDeviceAndDeviceIDByCtx 按设备和设备 ID 获取在线终端数量
func GetOnlineTerminalCountByDeviceAndDeviceIDByCtx(ctx context.Context, device, deviceID string) (int, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return 0, err
	}
	return dCtx.Terminal().GetOnlineTerminalCountByDeviceAndDeviceID(ctx, device, deviceID)
}

// GetTerminalInfoByCtx gets current terminal info GetTerminalInfoByCtx 获取当前终端信息
func GetTerminalInfoByCtx(ctx context.Context) (*manager.TerminalInfo, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().GetTerminalInfo(ctx)
}

// GetTerminalListByCtx gets current user terminal list GetTerminalListByCtx 获取当前用户终端列表
func GetTerminalListByCtx(ctx context.Context, device ...string) ([]manager.TerminalInfo, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().GetTerminalList(ctx, device...)
}

// GetLatestTokenValueByCtx gets latest current user token GetLatestTokenValueByCtx 获取当前用户最新 Token
func GetLatestTokenValueByCtx(ctx context.Context, device ...string) (string, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.Terminal().GetLatestTokenValue(ctx, device...)
}

// SearchTokenValueByCtx searches token values SearchTokenValueByCtx 搜索 Token 值
func SearchTokenValueByCtx(ctx context.Context, keyword string, start, size int) ([]string, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().SearchTokenValue(ctx, keyword, start, size)
}

// SearchSessionIDByCtx searches session ids SearchSessionIDByCtx ?Session ID
func SearchSessionIDByCtx(ctx context.Context, keyword string, start, size int) ([]string, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().SearchSessionId(ctx, keyword, start, size)
}

// ForEachTerminalByCtx visits current user terminals ForEachTerminalByCtx 遍历当前用户终端
func ForEachTerminalByCtx(ctx context.Context, visitor manager.TerminalVisitor) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().ForEachTerminal(ctx, visitor)
}

// ForEachTerminalByDeviceByCtx visits current user terminals by device ForEachTerminalByDeviceByCtx 按设备遍历当前用户终端
func ForEachTerminalByDeviceByCtx(ctx context.Context, device string, visitor manager.TerminalVisitor) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().ForEachTerminalByDevice(ctx, device, visitor)
}
