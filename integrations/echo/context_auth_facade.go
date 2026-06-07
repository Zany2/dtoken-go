// @Author daixk 2026/06/05
package echo

import (
	"time"

	"github.com/Zany2/dtoken-go/core/manager"
	echo4 "github.com/labstack/echo/v4"
)

// LoginByContext logs in current Echo request LoginByContext 在当前 Echo 请求中登录
func LoginByContext(c echo4.Context, loginID string, deviceAndDeviceId ...string) (string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return "", err
	}
	return dCtx.Auth().Login(requestContext(c), loginID, deviceAndDeviceId...)
}

// LoginWithTimeoutByContext logs in current Echo request with timeout LoginWithTimeoutByContext 使用指定有效期登录当前 Echo 请求
func LoginWithTimeoutByContext(c echo4.Context, loginID string, timeout time.Duration, deviceAndDeviceId ...string) (string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return "", err
	}
	return dCtx.Auth().LoginWithTimeout(requestContext(c), loginID, timeout, deviceAndDeviceId...)
}

// LoginWithOptionsByContext logs in current Echo request with options LoginWithOptionsByContext 使用登录选项登录当前 Echo 请求
func LoginWithOptionsByContext(c echo4.Context, opts manager.LoginOptions) (string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return "", err
	}
	return dCtx.Auth().LoginWithOptions(requestContext(c), opts)
}
