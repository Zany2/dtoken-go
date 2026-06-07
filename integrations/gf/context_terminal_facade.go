// @Author daixk 2026/06/05
package gf

import (
	"context"

	"github.com/Zany2/dtoken-go/core/manager"
)

// KickoutByDeviceByCtx kicks out current user by device KickoutByDeviceByCtx 鎸夎澶囪涪鍑哄綋鍓嶇敤鎴?
func KickoutByDeviceByCtx(ctx context.Context, device string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().KickoutByDevice(ctx, device)
}

// KickoutByDeviceAndDeviceIDByCtx kicks out current user by device ID KickoutByDeviceAndDeviceIDByCtx 鎸夎澶囧拰璁惧 ID 韪㈠嚭褰撳墠鐢ㄦ埛
func KickoutByDeviceAndDeviceIDByCtx(ctx context.Context, deviceAndDeviceId ...string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().KickoutByDeviceAndDeviceId(ctx, deviceAndDeviceId...)
}

// ReplaceByDeviceByCtx replaces current user by device ReplaceByDeviceByCtx 鎸夎澶囬《鏇垮綋鍓嶇敤鎴?
func ReplaceByDeviceByCtx(ctx context.Context, device string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().ReplaceByDevice(ctx, device)
}

// ReplaceByDeviceAndDeviceIDByCtx replaces current user by device ID ReplaceByDeviceAndDeviceIDByCtx 鎸夎澶囧拰璁惧 ID 椤舵浛褰撳墠鐢ㄦ埛
func ReplaceByDeviceAndDeviceIDByCtx(ctx context.Context, deviceAndDeviceId ...string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().ReplaceByDeviceAndDeviceId(ctx, deviceAndDeviceId...)
}

// KickoutByLoginIDByCtx kicks out all terminals of current user KickoutByLoginIDByCtx 韪㈠嚭褰撳墠鐢ㄦ埛鍏ㄩ儴缁堢
func KickoutByLoginIDByCtx(ctx context.Context) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().KickoutAll(ctx)
}

// ReplaceByLoginIDByCtx replaces all terminals of current user ReplaceByLoginIDByCtx 椤舵浛褰撳墠鐢ㄦ埛鍏ㄩ儴缁堢
func ReplaceByLoginIDByCtx(ctx context.Context) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().ReplaceAll(ctx)
}

// TerminateByCtx terminates current or specified terminal TerminateByCtx 涓嬬嚎褰撳墠鎴栨寚瀹氱粓绔?
func TerminateByCtx(ctx context.Context, opts manager.TerminateOptions) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().Terminate(ctx, opts)
}

// GetTokenValueListByDeviceByCtx gets current user tokens by device GetTokenValueListByDeviceByCtx 鎸夎澶囪幏鍙栧綋鍓嶇敤鎴?Token 鍒楄〃
func GetTokenValueListByDeviceByCtx(ctx context.Context, device string, checkAlive ...bool) ([]string, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().GetTokenValueListByDevice(ctx, device, checkAlive...)
}

// GetTokenValueListByDeviceAndDeviceIDByCtx gets current user tokens by device ID GetTokenValueListByDeviceAndDeviceIDByCtx 鎸夎澶囧拰璁惧 ID 鑾峰彇褰撳墠鐢ㄦ埛 Token 鍒楄〃
func GetTokenValueListByDeviceAndDeviceIDByCtx(ctx context.Context, device, deviceId string, checkAlive ...bool) ([]string, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().GetTokenValueListByDeviceAndDeviceId(ctx, device, deviceId, checkAlive...)
}

// GetOnlineTerminalCountByDeviceByCtx gets online count by device GetOnlineTerminalCountByDeviceByCtx 鎸夎澶囪幏鍙栧湪绾跨粓绔暟
func GetOnlineTerminalCountByDeviceByCtx(ctx context.Context, device string) (int, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return 0, err
	}
	return dCtx.Terminal().GetOnlineTerminalCountByDevice(ctx, device)
}

// GetOnlineTerminalCountByDeviceAndDeviceIDByCtx gets online count by device ID GetOnlineTerminalCountByDeviceAndDeviceIDByCtx 鎸夎澶囧拰璁惧 ID 鑾峰彇鍦ㄧ嚎缁堢鏁?
func GetOnlineTerminalCountByDeviceAndDeviceIDByCtx(ctx context.Context, device, deviceId string) (int, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return 0, err
	}
	return dCtx.Terminal().GetOnlineTerminalCountByDeviceAndDeviceId(ctx, device, deviceId)
}

// GetTerminalInfoByCtx gets current terminal info GetTerminalInfoByCtx 鑾峰彇褰撳墠缁堢淇℃伅
func GetTerminalInfoByCtx(ctx context.Context) (*manager.TerminalInfo, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().GetTerminalInfo(ctx)
}

// GetTerminalListByCtx gets current user terminal list GetTerminalListByCtx 鑾峰彇褰撳墠鐢ㄦ埛缁堢鍒楄〃
func GetTerminalListByCtx(ctx context.Context, device ...string) ([]manager.TerminalInfo, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().GetTerminalList(ctx, device...)
}

// GetLatestTokenValueByCtx gets latest current user token GetLatestTokenValueByCtx 鑾峰彇褰撳墠鐢ㄦ埛鏈€鏂?Token
func GetLatestTokenValueByCtx(ctx context.Context, device ...string) (string, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.Terminal().GetLatestTokenValue(ctx, device...)
}

// SearchTokenValueByCtx searches token values SearchTokenValueByCtx 鎼滅储 Token 鍊?
func SearchTokenValueByCtx(ctx context.Context, keyword string, start, size int) ([]string, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().SearchTokenValue(ctx, keyword, start, size)
}

// SearchSessionIDByCtx searches session ids SearchSessionIDByCtx 鎼滅储 Session ID
func SearchSessionIDByCtx(ctx context.Context, keyword string, start, size int) ([]string, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().SearchSessionId(ctx, keyword, start, size)
}

// ForEachTerminalByCtx visits current user terminals ForEachTerminalByCtx 閬嶅巻褰撳墠鐢ㄦ埛缁堢
func ForEachTerminalByCtx(ctx context.Context, visitor manager.TerminalVisitor) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().ForEachTerminal(ctx, visitor)
}

// ForEachTerminalByDeviceByCtx visits current user terminals by device ForEachTerminalByDeviceByCtx 鎸夎澶囬亶鍘嗗綋鍓嶇敤鎴风粓绔?
func ForEachTerminalByDeviceByCtx(ctx context.Context, device string, visitor manager.TerminalVisitor) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Terminal().ForEachTerminalByDevice(ctx, device, visitor)
}
