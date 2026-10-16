// @Author daixk 2026/06/05
package chi

import (
	"context"
	"time"

	"github.com/Zany2/dtoken-go/core/manager"
)

// SetTokenCookieByCtx writes token cookie SetTokenCookieByCtx 写入 Token Cookie
func SetTokenCookieByCtx(ctx context.Context, token string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	dCtx.Cookie().SetToken(token)
	return nil
}

// ClearTokenCookieByCtx clears token cookie ClearTokenCookieByCtx 清理 Token Cookie
func ClearTokenCookieByCtx(ctx context.Context) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	dCtx.Cookie().ClearToken()
	return nil
}

// LoginWithCookieByCtx logs in and writes token cookie LoginWithCookieByCtx 登录并写入 Token Cookie
func LoginWithCookieByCtx(ctx context.Context, loginID string, deviceAndDeviceID ...string) (string, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.Cookie().Login(ctx, loginID, deviceAndDeviceID...)
}

// LoginWithCookieTimeoutByCtx logs in with timeout and writes token cookie LoginWithCookieTimeoutByCtx 使用指定有效期登录并写入 Token Cookie
func LoginWithCookieTimeoutByCtx(ctx context.Context, loginID string, timeout time.Duration, deviceAndDeviceID ...string) (string, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.Cookie().LoginWithTimeout(ctx, loginID, timeout, deviceAndDeviceID...)
}

// LoginWithCookieOptionsByCtx logs in with options and writes token cookie LoginWithCookieOptionsByCtx 使用登录选项登录并写入 Token Cookie
func LoginWithCookieOptionsByCtx(ctx context.Context, opts manager.LoginOptions) (string, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.Cookie().LoginWithOptions(ctx, opts)
}

// LogoutWithCookieByCtx logs out and clears token cookie LogoutWithCookieByCtx 登出并清理 Token Cookie
func LogoutWithCookieByCtx(ctx context.Context) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Cookie().Logout(ctx)
}
