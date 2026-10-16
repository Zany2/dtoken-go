// @Author daixk 2026/06/05
package hertz

import (
	"github.com/Zany2/dtoken-go/core/manager"
	hertzapp "github.com/cloudwego/hertz/pkg/app"
)

// LoginWithRefreshTokenByContext logs in and issues refresh token LoginWithRefreshTokenByContext 登录并签发刷新 Token
func LoginWithRefreshTokenByContext(ctx *hertzapp.RequestContext, loginID string, deviceAndDeviceID ...string) (*manager.RefreshTokenPair, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Refresh().Login(requestContext(ctx), loginID, deviceAndDeviceID...)
}

// LoginWithRefreshTokenOptionsByContext logs in with refresh token options LoginWithRefreshTokenOptionsByContext 使用刷新 Token 选项登录
func LoginWithRefreshTokenOptionsByContext(ctx *hertzapp.RequestContext, opts manager.RefreshTokenOptions) (*manager.RefreshTokenPair, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Refresh().LoginWithOptions(requestContext(ctx), opts)
}

// RefreshTokenByContext refreshes access token RefreshTokenByContext 刷新访问 Token
func RefreshTokenByContext(ctx *hertzapp.RequestContext, refreshToken string) (*manager.RefreshTokenPair, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Refresh().Refresh(requestContext(ctx), refreshToken)
}

// RevokeRefreshTokenByContext revokes refresh token RevokeRefreshTokenByContext 撤销刷新 Token
func RevokeRefreshTokenByContext(ctx *hertzapp.RequestContext, refreshToken string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Refresh().Revoke(requestContext(ctx), refreshToken)
}

// GetRefreshTokenTTLByContext gets refresh token TTL GetRefreshTokenTTLByContext 获取刷新 Token 剩余有效期
func GetRefreshTokenTTLByContext(ctx *hertzapp.RequestContext, refreshToken string) (int64, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return 0, err
	}
	return dCtx.Refresh().GetTTL(requestContext(ctx), refreshToken)
}
