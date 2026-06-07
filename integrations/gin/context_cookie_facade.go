// @Author daixk 2026/06/05
package gin

import (
	"time"

	"github.com/Zany2/dtoken-go/core/manager"
	"github.com/gin-gonic/gin"
)

// SetTokenCookieByContext writes token cookie SetTokenCookieByContext 鍐欏叆 Token Cookie
func SetTokenCookieByContext(c *gin.Context, token string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	dCtx.Cookie().SetToken(token)
	return nil
}

// ClearTokenCookieByContext clears token cookie ClearTokenCookieByContext 娓呯悊 Token Cookie
func ClearTokenCookieByContext(c *gin.Context) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	dCtx.Cookie().ClearToken()
	return nil
}

// LoginWithCookieByContext logs in and writes token cookie LoginWithCookieByContext 鐧诲綍骞跺啓鍏?Token Cookie
func LoginWithCookieByContext(c *gin.Context, loginID string, deviceAndDeviceId ...string) (string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return "", err
	}
	return dCtx.Cookie().Login(requestContext(c), loginID, deviceAndDeviceId...)
}

// LoginWithCookieTimeoutByContext logs in with timeout and writes token cookie LoginWithCookieTimeoutByContext 浣跨敤鎸囧畾鏈夋晥鏈熺櫥褰曞苟鍐欏叆 Token Cookie
func LoginWithCookieTimeoutByContext(c *gin.Context, loginID string, timeout time.Duration, deviceAndDeviceId ...string) (string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return "", err
	}
	return dCtx.Cookie().LoginWithTimeout(requestContext(c), loginID, timeout, deviceAndDeviceId...)
}

// LoginWithCookieOptionsByContext logs in with options and writes token cookie LoginWithCookieOptionsByContext 浣跨敤鐧诲綍閫夐」鐧诲綍骞跺啓鍏?Token Cookie
func LoginWithCookieOptionsByContext(c *gin.Context, opts manager.LoginOptions) (string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return "", err
	}
	return dCtx.Cookie().LoginWithOptions(requestContext(c), opts)
}

// LogoutWithCookieByContext logs out and clears token cookie LogoutWithCookieByContext 閫€鍑虹櫥褰曞苟娓呯悊 Token Cookie
func LogoutWithCookieByContext(c *gin.Context) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Cookie().Logout(requestContext(c))
}
