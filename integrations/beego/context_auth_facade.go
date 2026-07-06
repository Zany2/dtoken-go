// @Author daixk 2026/06/06
package beego

import (
	"time"

	"github.com/Zany2/dtoken-go/core/manager"
	beegocontext "github.com/beego/beego/v2/server/web/context"
)

// LoginByContext logs in current Gin request LoginByContext 在当前 Gin 请求中登录
func LoginByContext(c *beegocontext.Context, loginID string, deviceAndDeviceID ...string) (string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return "", err
	}
	return dCtx.Auth().Login(requestContext(c), loginID, deviceAndDeviceID...)
}

// LoginWithTimeoutByContext logs in current Gin request with timeout LoginWithTimeoutByContext 使用指定有效期登录当前 Gin 请求
func LoginWithTimeoutByContext(c *beegocontext.Context, loginID string, timeout time.Duration, deviceAndDeviceID ...string) (string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return "", err
	}
	return dCtx.Auth().LoginWithTimeout(requestContext(c), loginID, timeout, deviceAndDeviceID...)
}

// LoginWithOptionsByContext logs in current Gin request with options LoginWithOptionsByContext 使用登录选项登录当前 Gin 请求
func LoginWithOptionsByContext(c *beegocontext.Context, opts manager.LoginOptions) (string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return "", err
	}
	return dCtx.Auth().LoginWithOptions(requestContext(c), opts)
}
