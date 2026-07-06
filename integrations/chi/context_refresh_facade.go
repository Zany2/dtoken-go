// @Author daixk 2026/06/05
package chi

import (
	"context"

	"github.com/Zany2/dtoken-go/core/manager"
)

// LoginWithRefreshTokenByCtx logs in and issues refresh token LoginWithRefreshTokenByCtx 登录并签发刷新 Token
func LoginWithRefreshTokenByCtx(ctx context.Context, loginID string, deviceAndDeviceID ...string) (*manager.RefreshTokenPair, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Refresh().Login(ctx, loginID, deviceAndDeviceID...)
}

// LoginWithRefreshTokenOptionsByCtx logs in with refresh token options LoginWithRefreshTokenOptionsByCtx 使用刷新 Token 选项登录
func LoginWithRefreshTokenOptionsByCtx(ctx context.Context, opts manager.RefreshTokenOptions) (*manager.RefreshTokenPair, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Refresh().LoginWithOptions(ctx, opts)
}

// RefreshTokenByCtx refreshes access token RefreshTokenByCtx 刷新访问 Token
func RefreshTokenByCtx(ctx context.Context, refreshToken string) (*manager.RefreshTokenPair, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Refresh().Refresh(ctx, refreshToken)
}

// RevokeRefreshTokenByCtx revokes refresh token RevokeRefreshTokenByCtx 撤销刷新 Token
func RevokeRefreshTokenByCtx(ctx context.Context, refreshToken string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Refresh().Revoke(ctx, refreshToken)
}

// GetRefreshTokenTTLByCtx gets refresh token TTL GetRefreshTokenTTLByCtx 获取刷新 Token 剩余有效期
func GetRefreshTokenTTLByCtx(ctx context.Context, refreshToken string) (int64, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return 0, err
	}
	return dCtx.Refresh().GetTTL(ctx, refreshToken)
}
