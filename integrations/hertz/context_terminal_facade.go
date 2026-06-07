// @Author daixk 2026/06/05
package hertz

import (
	"github.com/Zany2/dtoken-go/core/manager"
	hertzapp "github.com/cloudwego/hertz/pkg/app"
)

// KickoutByDeviceByContext kicks out current user by device KickoutByDeviceByContext ?
func KickoutByDeviceByContext(ctx *hertzapp.RequestContext, device string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().KickoutByDevice(requestContext(ctx), device)
}

// KickoutByDeviceAndDeviceIDByContext delegates to DToken context KickoutByDeviceAndDeviceIDByContext 转发到 DToken 上下文。
func KickoutByDeviceAndDeviceIDByContext(ctx *hertzapp.RequestContext, deviceAndDeviceId ...string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().KickoutByDeviceAndDeviceId(requestContext(ctx), deviceAndDeviceId...)
}

// ReplaceByDeviceByContext replaces current user by device ReplaceByDeviceByContext ?
func ReplaceByDeviceByContext(ctx *hertzapp.RequestContext, device string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().ReplaceByDevice(requestContext(ctx), device)
}

// ReplaceByDeviceAndDeviceIDByContext delegates to DToken context ReplaceByDeviceAndDeviceIDByContext 转发到 DToken 上下文。
func ReplaceByDeviceAndDeviceIDByContext(ctx *hertzapp.RequestContext, deviceAndDeviceId ...string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().ReplaceByDeviceAndDeviceId(requestContext(ctx), deviceAndDeviceId...)
}

// KickoutByLoginIDByContext delegates to DToken context KickoutByLoginIDByContext 转发到 DToken 上下文。
func KickoutByLoginIDByContext(ctx *hertzapp.RequestContext) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().KickoutAll(requestContext(ctx))
}

// ReplaceByLoginIDByContext delegates to DToken context ReplaceByLoginIDByContext 转发到 DToken 上下文。
func ReplaceByLoginIDByContext(ctx *hertzapp.RequestContext) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().ReplaceAll(requestContext(ctx))
}

// TerminateByContext terminates current or specified terminal TerminateByContext ?
func TerminateByContext(ctx *hertzapp.RequestContext, opts manager.TerminateOptions) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().Terminate(requestContext(ctx), opts)
}

// GetTokenValueListByDeviceByContext delegates to DToken context GetTokenValueListByDeviceByContext 转发到 DToken 上下文。
func GetTokenValueListByDeviceByContext(ctx *hertzapp.RequestContext, device string, checkAlive ...bool) ([]string, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().GetTokenValueListByDevice(requestContext(ctx), device, checkAlive...)
}

// GetTokenValueListByDeviceAndDeviceIDByContext delegates to DToken context GetTokenValueListByDeviceAndDeviceIDByContext 转发到 DToken 上下文。
func GetTokenValueListByDeviceAndDeviceIDByContext(ctx *hertzapp.RequestContext, device, deviceId string, checkAlive ...bool) ([]string, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().GetTokenValueListByDeviceAndDeviceId(requestContext(ctx), device, deviceId, checkAlive...)
}

// GetOnlineTerminalCountByDeviceByContext delegates to DToken context GetOnlineTerminalCountByDeviceByContext 转发到 DToken 上下文。
func GetOnlineTerminalCountByDeviceByContext(ctx *hertzapp.RequestContext, device string) (int, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return 0, err
	}
	return dCtx.Terminal().GetOnlineTerminalCountByDevice(requestContext(ctx), device)
}

// GetOnlineTerminalCountByDeviceAndDeviceIDByContext gets online count by device ID GetOnlineTerminalCountByDeviceAndDeviceIDByContext ?ID ?
func GetOnlineTerminalCountByDeviceAndDeviceIDByContext(ctx *hertzapp.RequestContext, device, deviceId string) (int, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return 0, err
	}
	return dCtx.Terminal().GetOnlineTerminalCountByDeviceAndDeviceId(requestContext(ctx), device, deviceId)
}

// GetTerminalInfoByContext gets current terminal info GetTerminalInfoByContext
func GetTerminalInfoByContext(ctx *hertzapp.RequestContext) (*manager.TerminalInfo, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().GetTerminalInfo(requestContext(ctx))
}

// GetTerminalListByContext delegates to DToken context GetTerminalListByContext 转发到 DToken 上下文。
func GetTerminalListByContext(ctx *hertzapp.RequestContext, device ...string) ([]manager.TerminalInfo, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().GetTerminalList(requestContext(ctx), device...)
}

// GetLatestTokenValueByContext gets latest current user token GetLatestTokenValueByContext ?Token
func GetLatestTokenValueByContext(ctx *hertzapp.RequestContext, device ...string) (string, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.Terminal().GetLatestTokenValue(requestContext(ctx), device...)
}

// SearchTokenValueByContext searches token values SearchTokenValueByContext ?Token ?
func SearchTokenValueByContext(ctx *hertzapp.RequestContext, keyword string, start, size int) ([]string, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().SearchTokenValue(requestContext(ctx), keyword, start, size)
}

// SearchSessionIDByContext searches session ids SearchSessionIDByContext ?Session ID
func SearchSessionIDByContext(ctx *hertzapp.RequestContext, keyword string, start, size int) ([]string, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().SearchSessionId(requestContext(ctx), keyword, start, size)
}

// ForEachTerminalByContext visits current user terminals ForEachTerminalByContext
func ForEachTerminalByContext(ctx *hertzapp.RequestContext, visitor manager.TerminalVisitor) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().ForEachTerminal(requestContext(ctx), visitor)
}

// ForEachTerminalByDeviceByContext visits current user terminals by device ForEachTerminalByDeviceByContext ?
func ForEachTerminalByDeviceByContext(ctx *hertzapp.RequestContext, device string, visitor manager.TerminalVisitor) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().ForEachTerminalByDevice(requestContext(ctx), device, visitor)
}
