// @Author daixk 2026/06/05
package gf

import (
	"context"
	"time"

	"github.com/Zany2/dtoken-go/core/manager"
)

// LoginByCtx logs in current GF request LoginByCtx 在当前 GF 请求中登录
func LoginByCtx(ctx context.Context, loginID string, deviceAndDeviceId ...string) (string, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.Auth().Login(ctx, loginID, deviceAndDeviceId...)
}

// LoginWithTimeoutByCtx logs in current GF request with timeout LoginWithTimeoutByCtx 使用指定有效期登录当前 GF 请求
func LoginWithTimeoutByCtx(ctx context.Context, loginID string, timeout time.Duration, deviceAndDeviceId ...string) (string, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.Auth().LoginWithTimeout(ctx, loginID, timeout, deviceAndDeviceId...)
}

// LoginWithOptionsByCtx logs in current GF request with options LoginWithOptionsByCtx 使用登录选项登录当前 GF 请求
func LoginWithOptionsByCtx(ctx context.Context, opts manager.LoginOptions) (string, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.Auth().LoginWithOptions(ctx, opts)
}

// GetLoginIDByCtx gets login ID by context GetLoginIDByCtx 从上下文获取登录 ID
func GetLoginIDByCtx(ctx context.Context, authType ...string) (string, error) {
	dCtx, err := requireDTokenContextByCtx(ctx, authType...)
	if err != nil {
		return "", err
	}
	return dCtx.Auth().GetLoginID(ctx)
}

// GetTokenInfoByCtx gets token info by context GetTokenInfoByCtx 从上下文获取 Token 信息
func GetTokenInfoByCtx(ctx context.Context, authType ...string) (*manager.TokenInfo, error) {
	dCtx, err := requireDTokenContextByCtx(ctx, authType...)
	if err != nil {
		return nil, err
	}
	return dCtx.Auth().GetTokenInfo(ctx)
}

// IntrospectTokenByCtx inspects current token without renewal side effects IntrospectTokenByCtx 无续期副作用地检查当前 token 状态
func IntrospectTokenByCtx(ctx context.Context, authType ...string) (*manager.TokenIntrospection, error) {
	dCtx, err := requireDTokenContextByCtx(ctx, authType...)
	if err != nil {
		return nil, err
	}
	return dCtx.Auth().IntrospectToken(ctx)
}
