// @Author daixk 2026/06/05
package context

import (
	"context"
	"errors"
	"time"

	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/manager"
)

// SetToken writes token cookie SetToken 写入 Token Cookie
func (c *CookieContext) SetToken(token string) {
	c.d.setTokenCookie(token)
}

// ClearToken clears token cookie ClearToken 清除 Token Cookie
func (c *CookieContext) ClearToken() {
	c.d.clearTokenCookie()
}

// Login logs in and writes token cookie Login 登录并写入 Token Cookie
func (c *CookieContext) Login(ctx context.Context, loginID string, deviceAndDeviceId ...string) (string, error) {
	token, err := c.d.manager.Login(ctx, loginID, deviceAndDeviceId...)
	if err != nil {
		return "", err
	}
	c.d.setTokenCookie(token)
	return token, nil
}

// LoginWithTimeout logs in with timeout and writes token cookie LoginWithTimeout 使用指定有效期登录并写入 Token Cookie
func (c *CookieContext) LoginWithTimeout(ctx context.Context, loginID string, timeout time.Duration, deviceAndDeviceId ...string) (string, error) {
	token, err := c.d.manager.LoginWithTimeout(ctx, loginID, timeout, deviceAndDeviceId...)
	if err != nil {
		return "", err
	}
	c.d.setTokenCookie(token)
	return token, nil
}

// LoginWithOptions logs in with options and writes token cookie LoginWithOptions 使用选项登录并写入 Token Cookie
func (c *CookieContext) LoginWithOptions(ctx context.Context, opts manager.LoginOptions) (string, error) {
	token, err := c.d.manager.LoginWithOptions(ctx, opts)
	if err != nil {
		return "", err
	}
	c.d.setTokenCookie(token)
	return token, nil
}

// Logout logs out current token and clears token cookie Logout 注销当前 Token 并清除 Cookie
func (c *CookieContext) Logout(ctx context.Context) error {
	err := c.d.Auth().Logout(ctx)
	if err == nil || errors.Is(err, derror.ErrInvalidToken) || errors.Is(err, derror.ErrNotLogin) {
		// Clear cookie on success or when token no longer exists server-side 注销成功或服务端 Token 不存在时清除 Cookie
		c.d.clearTokenCookie()
	}
	return err
}
