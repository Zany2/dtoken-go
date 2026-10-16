// @Author daixk 2026/06/05
package hertz

import (
	"time"

	"github.com/Zany2/dtoken-go/core/manager"
	hertzapp "github.com/cloudwego/hertz/pkg/app"
)

// SetTokenCookieByContext writes token cookie SetTokenCookieByContext 写入 Token Cookie
func SetTokenCookieByContext(ctx *hertzapp.RequestContext, token string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	dCtx.Cookie().SetToken(token)
	return nil
}

// ClearTokenCookieByContext clears token cookie ClearTokenCookieByContext 清理 Token Cookie
func ClearTokenCookieByContext(ctx *hertzapp.RequestContext) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	dCtx.Cookie().ClearToken()
	return nil
}

// LoginWithCookieByContext logs in and writes token cookie LoginWithCookieByContext 登录并写入 Token Cookie
func LoginWithCookieByContext(ctx *hertzapp.RequestContext, loginID string, deviceAndDeviceID ...string) (string, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.Cookie().Login(requestContext(ctx), loginID, deviceAndDeviceID...)
}

// LoginWithCookieTimeoutByContext logs in with timeout and writes token cookie LoginWithCookieTimeoutByContext 使用指定有效期登录并写入 Token Cookie
func LoginWithCookieTimeoutByContext(ctx *hertzapp.RequestContext, loginID string, timeout time.Duration, deviceAndDeviceID ...string) (string, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.Cookie().LoginWithTimeout(requestContext(ctx), loginID, timeout, deviceAndDeviceID...)
}

// LoginWithCookieOptionsByContext logs in with options and writes token cookie LoginWithCookieOptionsByContext 使用登录选项登录并写入 Token Cookie
func LoginWithCookieOptionsByContext(ctx *hertzapp.RequestContext, opts manager.LoginOptions) (string, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.Cookie().LoginWithOptions(requestContext(ctx), opts)
}

// LogoutWithCookieByContext logs out and clears token cookie LogoutWithCookieByContext 登出并清理 Token Cookie
func LogoutWithCookieByContext(ctx *hertzapp.RequestContext) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Cookie().Logout(requestContext(ctx))
}
