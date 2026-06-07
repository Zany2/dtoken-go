// @Author daixk 2026/06/05
package chi

import (
	"context"

	"github.com/Zany2/dtoken-go/core/manager"
)

// KickoutByDeviceByCtx kicks out current user by device KickoutByDeviceByCtx ?
func KickoutByDeviceByCtx(ctx context.Context, device string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().KickoutByDevice(ctx, device)
}

// KickoutByDeviceAndDeviceIDByCtx delegates to DToken context KickoutByDeviceAndDeviceIDByCtx 转发到 DToken 上下文。
func KickoutByDeviceAndDeviceIDByCtx(ctx context.Context, deviceAndDeviceId ...string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().KickoutByDeviceAndDeviceId(ctx, deviceAndDeviceId...)
}

// ReplaceByDeviceByCtx replaces current user by device ReplaceByDeviceByCtx ?
func ReplaceByDeviceByCtx(ctx context.Context, device string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().ReplaceByDevice(ctx, device)
}

// ReplaceByDeviceAndDeviceIDByCtx delegates to DToken context ReplaceByDeviceAndDeviceIDByCtx 转发到 DToken 上下文。
func ReplaceByDeviceAndDeviceIDByCtx(ctx context.Context, deviceAndDeviceId ...string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().ReplaceByDeviceAndDeviceId(ctx, deviceAndDeviceId...)
}

// KickoutByLoginIDByCtx delegates to DToken context KickoutByLoginIDByCtx 转发到 DToken 上下文。
func KickoutByLoginIDByCtx(ctx context.Context) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().KickoutAll(ctx)
}

// ReplaceByLoginIDByCtx delegates to DToken context ReplaceByLoginIDByCtx 转发到 DToken 上下文。
func ReplaceByLoginIDByCtx(ctx context.Context) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().ReplaceAll(ctx)
}

// TerminateByCtx terminates current or specified terminal TerminateByCtx ?
func TerminateByCtx(ctx context.Context, opts manager.TerminateOptions) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().Terminate(ctx, opts)
}

// GetTokenValueListByDeviceByCtx delegates to DToken context GetTokenValueListByDeviceByCtx 转发到 DToken 上下文。
func GetTokenValueListByDeviceByCtx(ctx context.Context, device string, checkAlive ...bool) ([]string, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().GetTokenValueListByDevice(ctx, device, checkAlive...)
}

// GetTokenValueListByDeviceAndDeviceIDByCtx delegates to DToken context GetTokenValueListByDeviceAndDeviceIDByCtx 转发到 DToken 上下文。
func GetTokenValueListByDeviceAndDeviceIDByCtx(ctx context.Context, device, deviceId string, checkAlive ...bool) ([]string, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().GetTokenValueListByDeviceAndDeviceId(ctx, device, deviceId, checkAlive...)
}

// GetOnlineTerminalCountByDeviceByCtx delegates to DToken context GetOnlineTerminalCountByDeviceByCtx 转发到 DToken 上下文。
func GetOnlineTerminalCountByDeviceByCtx(ctx context.Context, device string) (int, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return 0, err
	}
	return dCtx.Terminal().GetOnlineTerminalCountByDevice(ctx, device)
}

// GetOnlineTerminalCountByDeviceAndDeviceIDByCtx gets online count by device ID GetOnlineTerminalCountByDeviceAndDeviceIDByCtx ?ID ?
func GetOnlineTerminalCountByDeviceAndDeviceIDByCtx(ctx context.Context, device, deviceId string) (int, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return 0, err
	}
	return dCtx.Terminal().GetOnlineTerminalCountByDeviceAndDeviceId(ctx, device, deviceId)
}

// GetTerminalInfoByCtx gets current terminal info GetTerminalInfoByCtx
func GetTerminalInfoByCtx(ctx context.Context) (*manager.TerminalInfo, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().GetTerminalInfo(ctx)
}

// GetTerminalListByCtx delegates to DToken context GetTerminalListByCtx 转发到 DToken 上下文。
func GetTerminalListByCtx(ctx context.Context, device ...string) ([]manager.TerminalInfo, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().GetTerminalList(ctx, device...)
}

// GetLatestTokenValueByCtx gets latest current user token GetLatestTokenValueByCtx ?Token
func GetLatestTokenValueByCtx(ctx context.Context, device ...string) (string, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.Terminal().GetLatestTokenValue(ctx, device...)
}

// SearchTokenValueByCtx searches token values SearchTokenValueByCtx ?Token ?
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

// ForEachTerminalByCtx visits current user terminals ForEachTerminalByCtx
func ForEachTerminalByCtx(ctx context.Context, visitor manager.TerminalVisitor) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().ForEachTerminal(ctx, visitor)
}

// ForEachTerminalByDeviceByCtx visits current user terminals by device ForEachTerminalByDeviceByCtx ?
func ForEachTerminalByDeviceByCtx(ctx context.Context, device string, visitor manager.TerminalVisitor) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().ForEachTerminalByDevice(ctx, device, visitor)
}
