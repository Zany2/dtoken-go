// @Author daixk 2026/06/05
package hertz

import (
	"github.com/Zany2/dtoken-go/core/manager"
	hertzapp "github.com/cloudwego/hertz/pkg/app"
)

// LoginWithRefreshTokenByContext logs in and issues refresh token LoginWithRefreshTokenByContext ?Token
func LoginWithRefreshTokenByContext(ctx *hertzapp.RequestContext, loginID string, deviceAndDeviceId ...string) (*manager.RefreshTokenPair, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Refresh().Login(requestContext(ctx), loginID, deviceAndDeviceId...)
}

// LoginWithRefreshTokenOptionsByContext logs in with refresh token options LoginWithRefreshTokenOptionsByContext  Token
func LoginWithRefreshTokenOptionsByContext(ctx *hertzapp.RequestContext, opts manager.RefreshTokenOptions) (*manager.RefreshTokenPair, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Refresh().LoginWithOptions(requestContext(ctx), opts)
}

// RefreshTokenByContext refreshes access token RefreshTokenByContext  Token
func RefreshTokenByContext(ctx *hertzapp.RequestContext, refreshToken string) (*manager.RefreshTokenPair, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Refresh().Refresh(requestContext(ctx), refreshToken)
}

// RevokeRefreshTokenByContext revokes refresh token RevokeRefreshTokenByContext  Token
func RevokeRefreshTokenByContext(ctx *hertzapp.RequestContext, refreshToken string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Refresh().Revoke(requestContext(ctx), refreshToken)
}

// GetRefreshTokenTTLByContext gets refresh token TTL GetRefreshTokenTTLByContext  Token ?
func GetRefreshTokenTTLByContext(ctx *hertzapp.RequestContext, refreshToken string) (int64, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return 0, err
	}
	return dCtx.Refresh().GetTTL(requestContext(ctx), refreshToken)
}
