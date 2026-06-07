// @Author daixk 2026/06/06
package beego

import (
	"time"

	"github.com/Zany2/dtoken-go/core/manager"
	beegocontext "github.com/beego/beego/v2/server/web/context"
)

// SetTokenCookieByContext writes token cookie SetTokenCookieByContext 写入 Token Cookie
func SetTokenCookieByContext(c *beegocontext.Context, token string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	dCtx.Cookie().SetToken(token)
	return nil
}

// ClearTokenCookieByContext clears token cookie ClearTokenCookieByContext 清除 Token Cookie
func ClearTokenCookieByContext(c *beegocontext.Context) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	dCtx.Cookie().ClearToken()
	return nil
}

// LoginWithCookieByContext logs in and writes token cookie LoginWithCookieByContext 登录并写入 Token Cookie
func LoginWithCookieByContext(c *beegocontext.Context, loginID string, deviceAndDeviceId ...string) (string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return "", err
	}
	return dCtx.Cookie().Login(requestContext(c), loginID, deviceAndDeviceId...)
}

// LoginWithCookieTimeoutByContext logs in with timeout and writes token cookie LoginWithCookieTimeoutByContext 使用有效期登录并写入 Token Cookie
func LoginWithCookieTimeoutByContext(c *beegocontext.Context, loginID string, timeout time.Duration, deviceAndDeviceId ...string) (string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return "", err
	}
	return dCtx.Cookie().LoginWithTimeout(requestContext(c), loginID, timeout, deviceAndDeviceId...)
}

// LoginWithCookieOptionsByContext logs in with options and writes token cookie LoginWithCookieOptionsByContext 使用选项登录并写入 Token Cookie
func LoginWithCookieOptionsByContext(c *beegocontext.Context, opts manager.LoginOptions) (string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return "", err
	}
	return dCtx.Cookie().LoginWithOptions(requestContext(c), opts)
}

// LogoutWithCookieByContext logs out and clears token cookie LogoutWithCookieByContext 登出并清除 Token Cookie
func LogoutWithCookieByContext(c *beegocontext.Context) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Cookie().Logout(requestContext(c))
}
