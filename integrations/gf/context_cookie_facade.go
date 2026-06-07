// @Author daixk 2026/06/05
package gf

import (
	"context"
	"time"

	"github.com/Zany2/dtoken-go/core/manager"
)

// SetTokenCookieByCtx writes token cookie SetTokenCookieByCtx 鍐欏叆 Token Cookie
func SetTokenCookieByCtx(ctx context.Context, token string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	dCtx.Cookie().SetToken(token)
	return nil
}

// ClearTokenCookieByCtx clears token cookie ClearTokenCookieByCtx 娓呯悊 Token Cookie
func ClearTokenCookieByCtx(ctx context.Context) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	dCtx.Cookie().ClearToken()
	return nil
}

// LoginWithCookieByCtx logs in and writes token cookie LoginWithCookieByCtx 鐧诲綍骞跺啓鍏?Token Cookie
func LoginWithCookieByCtx(ctx context.Context, loginID string, deviceAndDeviceId ...string) (string, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.Cookie().Login(ctx, loginID, deviceAndDeviceId...)
}

// LoginWithCookieTimeoutByCtx logs in with timeout and writes token cookie LoginWithCookieTimeoutByCtx 浣跨敤鎸囧畾鏈夋晥鏈熺櫥褰曞苟鍐欏叆 Token Cookie
func LoginWithCookieTimeoutByCtx(ctx context.Context, loginID string, timeout time.Duration, deviceAndDeviceId ...string) (string, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.Cookie().LoginWithTimeout(ctx, loginID, timeout, deviceAndDeviceId...)
}

// LoginWithCookieOptionsByCtx logs in with options and writes token cookie LoginWithCookieOptionsByCtx 浣跨敤鐧诲綍閫夐」鐧诲綍骞跺啓鍏?Token Cookie
func LoginWithCookieOptionsByCtx(ctx context.Context, opts manager.LoginOptions) (string, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.Cookie().LoginWithOptions(ctx, opts)
}

// LogoutWithCookieByCtx logs out and clears token cookie LogoutWithCookieByCtx 閫€鍑虹櫥褰曞苟娓呯悊 Token Cookie
func LogoutWithCookieByCtx(ctx context.Context) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Cookie().Logout(ctx)
}
