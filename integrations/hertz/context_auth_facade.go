// @Author daixk 2026/06/05
package hertz

import (
	"time"

	"github.com/Zany2/dtoken-go/core/manager"
	hertzapp "github.com/cloudwego/hertz/pkg/app"
)

// LoginByContext logs in current Hertz request LoginByContext 在当前 Hertz 请求中登录
func LoginByContext(ctx *hertzapp.RequestContext, loginID string, deviceAndDeviceID ...string) (string, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.Auth().Login(requestContext(ctx), loginID, deviceAndDeviceID...)
}

// LoginWithTimeoutByContext logs in current Hertz request with timeout LoginWithTimeoutByContext 使用指定有效期登录当前 Hertz 请求
func LoginWithTimeoutByContext(ctx *hertzapp.RequestContext, loginID string, timeout time.Duration, deviceAndDeviceID ...string) (string, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.Auth().LoginWithTimeout(requestContext(ctx), loginID, timeout, deviceAndDeviceID...)
}

// LoginWithOptionsByContext logs in current Hertz request with options LoginWithOptionsByContext 使用登录选项登录当前 Hertz 请求
func LoginWithOptionsByContext(ctx *hertzapp.RequestContext, opts manager.LoginOptions) (string, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.Auth().LoginWithOptions(requestContext(ctx), opts)
}
