// @Author daixk 2026/06/05
package fiber

import (
	"time"

	"github.com/Zany2/dtoken-go/core/manager"
	gofiber "github.com/gofiber/fiber/v2"
)

// SetTokenCookieByContext writes token cookie SetTokenCookieByContext 鍐欏叆 Token Cookie
func SetTokenCookieByContext(c *gofiber.Ctx, token string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	dCtx.Cookie().SetToken(token)
	return nil
}

// ClearTokenCookieByContext clears token cookie ClearTokenCookieByContext 娓呯悊 Token Cookie
func ClearTokenCookieByContext(c *gofiber.Ctx) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	dCtx.Cookie().ClearToken()
	return nil
}

// LoginWithCookieByContext logs in and writes token cookie LoginWithCookieByContext 登录并写入 Token Cookie
func LoginWithCookieByContext(c *gofiber.Ctx, loginID string, deviceAndDeviceId ...string) (string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return "", err
	}
	return dCtx.Cookie().Login(requestContext(c), loginID, deviceAndDeviceId...)
}

// LoginWithCookieTimeoutByContext logs in with timeout and writes token cookie LoginWithCookieTimeoutByContext 使用指定有效期登录并写入 Token Cookie
func LoginWithCookieTimeoutByContext(c *gofiber.Ctx, loginID string, timeout time.Duration, deviceAndDeviceId ...string) (string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return "", err
	}
	return dCtx.Cookie().LoginWithTimeout(requestContext(c), loginID, timeout, deviceAndDeviceId...)
}

// LoginWithCookieOptionsByContext logs in with options and writes token cookie LoginWithCookieOptionsByContext 使用登录选项登录并写入 Token Cookie
func LoginWithCookieOptionsByContext(c *gofiber.Ctx, opts manager.LoginOptions) (string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return "", err
	}
	return dCtx.Cookie().LoginWithOptions(requestContext(c), opts)
}

// LogoutWithCookieByContext logs out and clears token cookie LogoutWithCookieByContext 閫€鍑虹櫥褰曞苟娓呯悊 Token Cookie
func LogoutWithCookieByContext(c *gofiber.Ctx) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Cookie().Logout(requestContext(c))
}
