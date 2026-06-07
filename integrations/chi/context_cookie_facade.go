// @Author daixk 2026/06/05
package chi

import (
	"context"
	"time"

	"github.com/Zany2/dtoken-go/core/manager"
)

// SetTokenCookieByCtx writes token cookie SetTokenCookieByCtx ?Token Cookie
func SetTokenCookieByCtx(ctx context.Context, token string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	dCtx.Cookie().SetToken(token)
	return nil
}

// ClearTokenCookieByCtx clears token cookie ClearTokenCookieByCtx ?Token Cookie
func ClearTokenCookieByCtx(ctx context.Context) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	dCtx.Cookie().ClearToken()
	return nil
}

// LoginWithCookieByCtx logs in and writes token cookie LoginWithCookieByCtx ?Token Cookie
func LoginWithCookieByCtx(ctx context.Context, loginID string, deviceAndDeviceId ...string) (string, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.Cookie().Login(ctx, loginID, deviceAndDeviceId...)
}

// LoginWithCookieTimeoutByCtx logs in with timeout and writes token cookie LoginWithCookieTimeoutByCtx  Token Cookie
func LoginWithCookieTimeoutByCtx(ctx context.Context, loginID string, timeout time.Duration, deviceAndDeviceId ...string) (string, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.Cookie().LoginWithTimeout(ctx, loginID, timeout, deviceAndDeviceId...)
}

// LoginWithCookieOptionsByCtx logs in with options and writes token cookie LoginWithCookieOptionsByCtx ?Token Cookie
func LoginWithCookieOptionsByCtx(ctx context.Context, opts manager.LoginOptions) (string, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.Cookie().LoginWithOptions(ctx, opts)
}

// LogoutWithCookieByCtx logs out and clears token cookie LogoutWithCookieByCtx ?Token Cookie
func LogoutWithCookieByCtx(ctx context.Context) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Cookie().Logout(ctx)
}
