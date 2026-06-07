// @Author daixk 2026/06/05
package chi

import (
	"context"
	"time"

	"github.com/Zany2/dtoken-go/core/manager"
)

// LoginByCtx logs in current Chi request LoginByCtx  Chi
func LoginByCtx(ctx context.Context, loginID string, deviceAndDeviceId ...string) (string, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.Auth().Login(ctx, loginID, deviceAndDeviceId...)
}

// LoginWithTimeoutByCtx logs in current Chi request with timeout LoginWithTimeoutByCtx  Chi
func LoginWithTimeoutByCtx(ctx context.Context, loginID string, timeout time.Duration, deviceAndDeviceId ...string) (string, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.Auth().LoginWithTimeout(ctx, loginID, timeout, deviceAndDeviceId...)
}

// LoginWithOptionsByCtx logs in current Chi request with options LoginWithOptionsByCtx  Chi
func LoginWithOptionsByCtx(ctx context.Context, opts manager.LoginOptions) (string, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.Auth().LoginWithOptions(ctx, opts)
}
