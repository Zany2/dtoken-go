// @Author daixk 2026/06/05
package echo

import (
	"github.com/Zany2/dtoken-go/core/manager"
	echo4 "github.com/labstack/echo/v4"
)

// LoginWithRefreshTokenByContext logs in and issues refresh token LoginWithRefreshTokenByContext 登录并签发刷新 Token
func LoginWithRefreshTokenByContext(c echo4.Context, loginID string, deviceAndDeviceId ...string) (*manager.RefreshTokenPair, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Refresh().Login(requestContext(c), loginID, deviceAndDeviceId...)
}

// LoginWithRefreshTokenOptionsByContext logs in with refresh token options LoginWithRefreshTokenOptionsByContext 使用刷新 Token 选项登录
func LoginWithRefreshTokenOptionsByContext(c echo4.Context, opts manager.RefreshTokenOptions) (*manager.RefreshTokenPair, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Refresh().LoginWithOptions(requestContext(c), opts)
}

// RefreshTokenByContext refreshes access token RefreshTokenByContext 鍒锋柊璁块棶 Token
func RefreshTokenByContext(c echo4.Context, refreshToken string) (*manager.RefreshTokenPair, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Refresh().Refresh(requestContext(c), refreshToken)
}

// RevokeRefreshTokenByContext revokes refresh token RevokeRefreshTokenByContext 鎾ら攢鍒锋柊 Token
func RevokeRefreshTokenByContext(c echo4.Context, refreshToken string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Refresh().Revoke(requestContext(c), refreshToken)
}

// GetRefreshTokenTTLByContext gets refresh token TTL GetRefreshTokenTTLByContext 鑾峰彇鍒锋柊 Token 鍓╀綑鏈夋晥鏈?
func GetRefreshTokenTTLByContext(c echo4.Context, refreshToken string) (int64, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return 0, err
	}
	return dCtx.Refresh().GetTTL(requestContext(c), refreshToken)
}
