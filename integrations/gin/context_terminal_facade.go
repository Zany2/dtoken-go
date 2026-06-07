// @Author daixk 2026/06/05
package gin

import (
	"github.com/Zany2/dtoken-go/core/manager"
	"github.com/gin-gonic/gin"
)

// KickoutByDeviceByContext kicks out current user by device KickoutByDeviceByContext 鎸夎澶囪涪鍑哄綋鍓嶇敤鎴?
func KickoutByDeviceByContext(c *gin.Context, device string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Terminal().KickoutByDevice(requestContext(c), device)
}

// KickoutByDeviceAndDeviceIDByContext kicks out current user by device ID KickoutByDeviceAndDeviceIDByContext 鎸夎澶囧拰璁惧 ID 韪㈠嚭褰撳墠鐢ㄦ埛
func KickoutByDeviceAndDeviceIDByContext(c *gin.Context, deviceAndDeviceId ...string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Terminal().KickoutByDeviceAndDeviceId(requestContext(c), deviceAndDeviceId...)
}

// ReplaceByDeviceByContext replaces current user by device ReplaceByDeviceByContext 鎸夎澶囬《鏇垮綋鍓嶇敤鎴?
func ReplaceByDeviceByContext(c *gin.Context, device string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Terminal().ReplaceByDevice(requestContext(c), device)
}

// ReplaceByDeviceAndDeviceIDByContext replaces current user by device ID ReplaceByDeviceAndDeviceIDByContext 鎸夎澶囧拰璁惧 ID 椤舵浛褰撳墠鐢ㄦ埛
func ReplaceByDeviceAndDeviceIDByContext(c *gin.Context, deviceAndDeviceId ...string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Terminal().ReplaceByDeviceAndDeviceId(requestContext(c), deviceAndDeviceId...)
}

// KickoutByLoginIDByContext kicks out all terminals of current user KickoutByLoginIDByContext 韪㈠嚭褰撳墠鐢ㄦ埛鍏ㄩ儴缁堢
func KickoutByLoginIDByContext(c *gin.Context) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Terminal().KickoutAll(requestContext(c))
}

// ReplaceByLoginIDByContext replaces all terminals of current user ReplaceByLoginIDByContext 椤舵浛褰撳墠鐢ㄦ埛鍏ㄩ儴缁堢
func ReplaceByLoginIDByContext(c *gin.Context) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Terminal().ReplaceAll(requestContext(c))
}

// TerminateByContext terminates current or specified terminal TerminateByContext 涓嬬嚎褰撳墠鎴栨寚瀹氱粓绔?
func TerminateByContext(c *gin.Context, opts manager.TerminateOptions) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Terminal().Terminate(requestContext(c), opts)
}

// GetTokenValueListByDeviceByContext gets current user tokens by device GetTokenValueListByDeviceByContext 鎸夎澶囪幏鍙栧綋鍓嶇敤鎴?Token 鍒楄〃
func GetTokenValueListByDeviceByContext(c *gin.Context, device string, checkAlive ...bool) ([]string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().GetTokenValueListByDevice(requestContext(c), device, checkAlive...)
}

// GetTokenValueListByDeviceAndDeviceIDByContext gets current user tokens by device ID GetTokenValueListByDeviceAndDeviceIDByContext 鎸夎澶囧拰璁惧 ID 鑾峰彇褰撳墠鐢ㄦ埛 Token 鍒楄〃
func GetTokenValueListByDeviceAndDeviceIDByContext(c *gin.Context, device, deviceId string, checkAlive ...bool) ([]string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().GetTokenValueListByDeviceAndDeviceId(requestContext(c), device, deviceId, checkAlive...)
}

// GetOnlineTerminalCountByDeviceByContext gets online count by device GetOnlineTerminalCountByDeviceByContext 鎸夎澶囪幏鍙栧湪绾跨粓绔暟
func GetOnlineTerminalCountByDeviceByContext(c *gin.Context, device string) (int, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return 0, err
	}
	return dCtx.Terminal().GetOnlineTerminalCountByDevice(requestContext(c), device)
}

// GetOnlineTerminalCountByDeviceAndDeviceIDByContext gets online count by device ID GetOnlineTerminalCountByDeviceAndDeviceIDByContext 鎸夎澶囧拰璁惧 ID 鑾峰彇鍦ㄧ嚎缁堢鏁?
func GetOnlineTerminalCountByDeviceAndDeviceIDByContext(c *gin.Context, device, deviceId string) (int, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return 0, err
	}
	return dCtx.Terminal().GetOnlineTerminalCountByDeviceAndDeviceId(requestContext(c), device, deviceId)
}

// GetTerminalInfoByContext gets current terminal info GetTerminalInfoByContext 鑾峰彇褰撳墠缁堢淇℃伅
func GetTerminalInfoByContext(c *gin.Context) (*manager.TerminalInfo, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().GetTerminalInfo(requestContext(c))
}

// GetTerminalListByContext gets current user terminal list GetTerminalListByContext 鑾峰彇褰撳墠鐢ㄦ埛缁堢鍒楄〃
func GetTerminalListByContext(c *gin.Context, device ...string) ([]manager.TerminalInfo, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().GetTerminalList(requestContext(c), device...)
}

// GetLatestTokenValueByContext gets latest current user token GetLatestTokenValueByContext 鑾峰彇褰撳墠鐢ㄦ埛鏈€鏂?Token
func GetLatestTokenValueByContext(c *gin.Context, device ...string) (string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return "", err
	}
	return dCtx.Terminal().GetLatestTokenValue(requestContext(c), device...)
}

// SearchTokenValueByContext searches token values SearchTokenValueByContext 鎼滅储 Token 鍊?
func SearchTokenValueByContext(c *gin.Context, keyword string, start, size int) ([]string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().SearchTokenValue(requestContext(c), keyword, start, size)
}

// SearchSessionIDByContext searches session ids SearchSessionIDByContext 鎼滅储 Session ID
func SearchSessionIDByContext(c *gin.Context, keyword string, start, size int) ([]string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Terminal().SearchSessionId(requestContext(c), keyword, start, size)
}

// ForEachTerminalByContext visits current user terminals ForEachTerminalByContext 閬嶅巻褰撳墠鐢ㄦ埛缁堢
func ForEachTerminalByContext(c *gin.Context, visitor manager.TerminalVisitor) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Terminal().ForEachTerminal(requestContext(c), visitor)
}

// ForEachTerminalByDeviceByContext visits current user terminals by device ForEachTerminalByDeviceByContext 鎸夎澶囬亶鍘嗗綋鍓嶇敤鎴风粓绔?
func ForEachTerminalByDeviceByContext(c *gin.Context, device string, visitor manager.TerminalVisitor) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Terminal().ForEachTerminalByDevice(requestContext(c), device, visitor)
}
