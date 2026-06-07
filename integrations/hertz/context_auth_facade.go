// @Author daixk 2026/06/05
package hertz

import (
	"time"

	"github.com/Zany2/dtoken-go/core/manager"
	hertzapp "github.com/cloudwego/hertz/pkg/app"
)

// LoginByContext logs in current Hertz request LoginByContext  Hertz
func LoginByContext(ctx *hertzapp.RequestContext, loginID string, deviceAndDeviceId ...string) (string, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.Auth().Login(requestContext(ctx), loginID, deviceAndDeviceId...)
}

// LoginWithTimeoutByContext logs in current Hertz request with timeout LoginWithTimeoutByContext  Hertz
func LoginWithTimeoutByContext(ctx *hertzapp.RequestContext, loginID string, timeout time.Duration, deviceAndDeviceId ...string) (string, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.Auth().LoginWithTimeout(requestContext(ctx), loginID, timeout, deviceAndDeviceId...)
}

// LoginWithOptionsByContext logs in current Hertz request with options LoginWithOptionsByContext  Hertz
func LoginWithOptionsByContext(ctx *hertzapp.RequestContext, opts manager.LoginOptions) (string, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.Auth().LoginWithOptions(requestContext(ctx), opts)
}
