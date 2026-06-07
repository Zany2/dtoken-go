// @Author daixk 2026/06/05
package gin

import (
	"time"

	"github.com/Zany2/dtoken-go/core/manager"
	"github.com/gin-gonic/gin"
)

// LoginByContext logs in current Gin request LoginByContext 鍦ㄥ綋鍓?Gin 璇锋眰涓櫥褰?
func LoginByContext(c *gin.Context, loginID string, deviceAndDeviceId ...string) (string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return "", err
	}
	return dCtx.Auth().Login(requestContext(c), loginID, deviceAndDeviceId...)
}

// LoginWithTimeoutByContext logs in current Gin request with timeout LoginWithTimeoutByContext 浣跨敤鎸囧畾鏈夋晥鏈熺櫥褰曞綋鍓?Gin 璇锋眰
func LoginWithTimeoutByContext(c *gin.Context, loginID string, timeout time.Duration, deviceAndDeviceId ...string) (string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return "", err
	}
	return dCtx.Auth().LoginWithTimeout(requestContext(c), loginID, timeout, deviceAndDeviceId...)
}

// LoginWithOptionsByContext logs in current Gin request with options LoginWithOptionsByContext 浣跨敤鐧诲綍閫夐」鐧诲綍褰撳墠 Gin 璇锋眰
func LoginWithOptionsByContext(c *gin.Context, opts manager.LoginOptions) (string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return "", err
	}
	return dCtx.Auth().LoginWithOptions(requestContext(c), opts)
}
