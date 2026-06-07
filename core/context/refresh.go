// @Author daixk 2026/06/05
package context

import (
	"context"

	"github.com/Zany2/dtoken-go/core/manager"
)

// Login logs in and returns access and refresh tokens Login 登录并返回访问令牌和刷新令牌
func (c *RefreshContext) Login(ctx context.Context, loginID string, deviceAndDeviceId ...string) (*manager.RefreshTokenPair, error) {
	return c.d.manager.LoginWithRefreshToken(ctx, loginID, deviceAndDeviceId...)
}

// LoginWithOptions logs in with refresh token options LoginWithOptions 使用刷新令牌选项登录
func (c *RefreshContext) LoginWithOptions(ctx context.Context, opts manager.RefreshTokenOptions) (*manager.RefreshTokenPair, error) {
	return c.d.manager.LoginWithRefreshTokenOptions(ctx, opts)
}

// Refresh rotates refresh token Refresh 刷新并轮换令牌
func (c *RefreshContext) Refresh(ctx context.Context, refreshToken string) (*manager.RefreshTokenPair, error) {
	return c.d.manager.RefreshToken(ctx, refreshToken)
}

// Revoke revokes refresh token Revoke 撤销刷新令牌
func (c *RefreshContext) Revoke(ctx context.Context, refreshToken string) error {
	return c.d.manager.RevokeRefreshToken(ctx, refreshToken)
}

// GetTTL gets refresh token TTL GetTTL 获取刷新令牌剩余有效期
func (c *RefreshContext) GetTTL(ctx context.Context, refreshToken string) (int64, error) {
	return c.d.manager.GetRefreshTokenTTL(ctx, refreshToken)
}
