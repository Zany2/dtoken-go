// @Author daixk 2026/06/05
package hertz

import (
	"time"

	"github.com/Zany2/dtoken-go/core/manager"
	hertzapp "github.com/cloudwego/hertz/pkg/app"
)

// SetTokenCookieByContext writes token cookie SetTokenCookieByContext ?Token Cookie
func SetTokenCookieByContext(ctx *hertzapp.RequestContext, token string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	dCtx.Cookie().SetToken(token)
	return nil
}

// ClearTokenCookieByContext clears token cookie ClearTokenCookieByContext ?Token Cookie
func ClearTokenCookieByContext(ctx *hertzapp.RequestContext) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	dCtx.Cookie().ClearToken()
	return nil
}

// LoginWithCookieByContext logs in and writes token cookie LoginWithCookieByContext ?Token Cookie
func LoginWithCookieByContext(ctx *hertzapp.RequestContext, loginID string, deviceAndDeviceId ...string) (string, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.Cookie().Login(requestContext(ctx), loginID, deviceAndDeviceId...)
}

// LoginWithCookieTimeoutByContext logs in with timeout and writes token cookie LoginWithCookieTimeoutByContext  Token Cookie
func LoginWithCookieTimeoutByContext(ctx *hertzapp.RequestContext, loginID string, timeout time.Duration, deviceAndDeviceId ...string) (string, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.Cookie().LoginWithTimeout(requestContext(ctx), loginID, timeout, deviceAndDeviceId...)
}

// LoginWithCookieOptionsByContext logs in with options and writes token cookie LoginWithCookieOptionsByContext ?Token Cookie
func LoginWithCookieOptionsByContext(ctx *hertzapp.RequestContext, opts manager.LoginOptions) (string, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.Cookie().LoginWithOptions(requestContext(ctx), opts)
}

// LogoutWithCookieByContext logs out and clears token cookie LogoutWithCookieByContext ?Token Cookie
func LogoutWithCookieByContext(ctx *hertzapp.RequestContext) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Cookie().Logout(requestContext(ctx))
}
