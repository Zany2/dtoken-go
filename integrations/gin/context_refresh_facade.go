// @Author daixk 2026/06/05
package gin

import (
	"github.com/Zany2/dtoken-go/core/manager"
	"github.com/gin-gonic/gin"
)

// LoginWithRefreshTokenByContext logs in and issues refresh token LoginWithRefreshTokenByContext 鐧诲綍骞剁鍙戝埛鏂?Token
func LoginWithRefreshTokenByContext(c *gin.Context, loginID string, deviceAndDeviceId ...string) (*manager.RefreshTokenPair, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Refresh().Login(requestContext(c), loginID, deviceAndDeviceId...)
}

// LoginWithRefreshTokenOptionsByContext logs in with refresh token options LoginWithRefreshTokenOptionsByContext 浣跨敤鍒锋柊 Token 閫夐」鐧诲綍
func LoginWithRefreshTokenOptionsByContext(c *gin.Context, opts manager.RefreshTokenOptions) (*manager.RefreshTokenPair, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Refresh().LoginWithOptions(requestContext(c), opts)
}

// RefreshTokenByContext refreshes access token RefreshTokenByContext 鍒锋柊璁块棶 Token
func RefreshTokenByContext(c *gin.Context, refreshToken string) (*manager.RefreshTokenPair, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Refresh().Refresh(requestContext(c), refreshToken)
}

// RevokeRefreshTokenByContext revokes refresh token RevokeRefreshTokenByContext 鎾ら攢鍒锋柊 Token
func RevokeRefreshTokenByContext(c *gin.Context, refreshToken string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Refresh().Revoke(requestContext(c), refreshToken)
}

// GetRefreshTokenTTLByContext gets refresh token TTL GetRefreshTokenTTLByContext 鑾峰彇鍒锋柊 Token 鍓╀綑鏈夋晥鏈?
func GetRefreshTokenTTLByContext(c *gin.Context, refreshToken string) (int64, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return 0, err
	}
	return dCtx.Refresh().GetTTL(requestContext(c), refreshToken)
}
