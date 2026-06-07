// @Author daixk 2026/06/06
package beego

import (
	"github.com/Zany2/dtoken-go/core/manager"
	beegocontext "github.com/beego/beego/v2/server/web/context"
)

// LoginWithRefreshTokenByContext logs in and issues refresh token LoginWithRefreshTokenByContext 登录并签发刷新 Token
func LoginWithRefreshTokenByContext(c *beegocontext.Context, loginID string, deviceAndDeviceId ...string) (*manager.RefreshTokenPair, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Refresh().Login(requestContext(c), loginID, deviceAndDeviceId...)
}

// LoginWithRefreshTokenOptionsByContext logs in with refresh token options LoginWithRefreshTokenOptionsByContext 使用刷新 Token 选项登录
func LoginWithRefreshTokenOptionsByContext(c *beegocontext.Context, opts manager.RefreshTokenOptions) (*manager.RefreshTokenPair, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Refresh().LoginWithOptions(requestContext(c), opts)
}

// RefreshTokenByContext refreshes access token RefreshTokenByContext 刷新访问 Token
func RefreshTokenByContext(c *beegocontext.Context, refreshToken string) (*manager.RefreshTokenPair, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Refresh().Refresh(requestContext(c), refreshToken)
}

// RevokeRefreshTokenByContext revokes refresh token RevokeRefreshTokenByContext 撤销刷新 Token
func RevokeRefreshTokenByContext(c *beegocontext.Context, refreshToken string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Refresh().Revoke(requestContext(c), refreshToken)
}

// GetRefreshTokenTTLByContext gets refresh token TTL GetRefreshTokenTTLByContext 获取刷新 Token 剩余有效期
func GetRefreshTokenTTLByContext(c *beegocontext.Context, refreshToken string) (int64, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return 0, err
	}
	return dCtx.Refresh().GetTTL(requestContext(c), refreshToken)
}
