// @Author daixk 2026/06/05
package echo

import (
	"time"

	"github.com/Zany2/dtoken-go/core/manager"
	echo4 "github.com/labstack/echo/v4"
)

// SetTokenCookieByContext writes token cookie SetTokenCookieByContext 鍐欏叆 Token Cookie
func SetTokenCookieByContext(c echo4.Context, token string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	dCtx.Cookie().SetToken(token)
	return nil
}

// ClearTokenCookieByContext clears token cookie ClearTokenCookieByContext 娓呯悊 Token Cookie
func ClearTokenCookieByContext(c echo4.Context) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	dCtx.Cookie().ClearToken()
	return nil
}

// LoginWithCookieByContext logs in and writes token cookie LoginWithCookieByContext 登录并写入 Token Cookie
func LoginWithCookieByContext(c echo4.Context, loginID string, deviceAndDeviceId ...string) (string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return "", err
	}
	return dCtx.Cookie().Login(requestContext(c), loginID, deviceAndDeviceId...)
}

// LoginWithCookieTimeoutByContext logs in with timeout and writes token cookie LoginWithCookieTimeoutByContext 使用指定有效期登录并写入 Token Cookie
func LoginWithCookieTimeoutByContext(c echo4.Context, loginID string, timeout time.Duration, deviceAndDeviceId ...string) (string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return "", err
	}
	return dCtx.Cookie().LoginWithTimeout(requestContext(c), loginID, timeout, deviceAndDeviceId...)
}

// LoginWithCookieOptionsByContext logs in with options and writes token cookie LoginWithCookieOptionsByContext 使用登录选项登录并写入 Token Cookie
func LoginWithCookieOptionsByContext(c echo4.Context, opts manager.LoginOptions) (string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return "", err
	}
	return dCtx.Cookie().LoginWithOptions(requestContext(c), opts)
}

// LogoutWithCookieByContext logs out and clears token cookie LogoutWithCookieByContext 閫€鍑虹櫥褰曞苟娓呯悊 Token Cookie
func LogoutWithCookieByContext(c echo4.Context) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Cookie().Logout(requestContext(c))
}
