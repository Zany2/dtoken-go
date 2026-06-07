// @Author daixk 2026/06/05
package context

import (
	"context"
	"time"

	"github.com/Zany2/dtoken-go/core/manager"
)

// Value gets current request token value Value 获取当前请求 Token 值
func (c *AuthContext) Value() string {
	return c.d.GetTokenValue()
}

// Login logs in a subject Login 登录指定主体
func (c *AuthContext) Login(ctx context.Context, loginID string, deviceAndDeviceId ...string) (string, error) {
	return c.d.manager.Login(ctx, loginID, deviceAndDeviceId...)
}

// LoginWithTimeout logs in with timeout LoginWithTimeout 使用指定有效期登录
func (c *AuthContext) LoginWithTimeout(ctx context.Context, loginID string, timeout time.Duration, deviceAndDeviceId ...string) (string, error) {
	return c.d.manager.LoginWithTimeout(ctx, loginID, timeout, deviceAndDeviceId...)
}

// LoginWithOptions logs in with options LoginWithOptions 使用选项登录
func (c *AuthContext) LoginWithOptions(ctx context.Context, opts manager.LoginOptions) (string, error) {
	return c.d.manager.LoginWithOptions(ctx, opts)
}

// IsLogin checks current token login state IsLogin 检查当前 Token 是否已登录
func (c *AuthContext) IsLogin(ctx context.Context) bool {
	token := c.d.GetTokenValue()
	if token == "" {
		return false
	}
	return c.d.manager.IsLogin(ctx, token)
}

// CheckLogin checks current token login state CheckLogin 校验当前 Token 登录状态
func (c *AuthContext) CheckLogin(ctx context.Context) error {
	token, err := c.d.requireToken()
	if err != nil {
		return err
	}
	return c.d.manager.CheckLogin(ctx, token)
}

// GetLoginID gets current token login ID GetLoginID 获取当前 Token 对应登录 ID
func (c *AuthContext) GetLoginID(ctx context.Context) (string, error) {
	return c.d.currentLoginID(ctx)
}

// LoginByToken validates and logs in current token LoginByToken 使用当前 Token 登录
func (c *AuthContext) LoginByToken(ctx context.Context) error {
	token, err := c.d.requireToken()
	if err != nil {
		return err
	}
	return c.d.manager.LoginByToken(ctx, token)
}

// Logout logs out current token Logout 注销当前 Token
func (c *AuthContext) Logout(ctx context.Context) error {
	token, err := c.d.requireToken()
	if err != nil {
		return err
	}
	return c.d.manager.Logout(ctx, token)
}

// GetTokenInfo gets current token info GetTokenInfo 获取当前 Token 信息
func (c *AuthContext) GetTokenInfo(ctx context.Context) (*manager.TokenInfo, error) {
	token, err := c.d.requireToken()
	if err != nil {
		return nil, err
	}
	return c.d.manager.GetTokenInfo(ctx, token)
}

// IntrospectToken inspects current token IntrospectToken 内省当前 Token
func (c *AuthContext) IntrospectToken(ctx context.Context) (*manager.TokenIntrospection, error) {
	token, err := c.d.requireToken()
	if err != nil {
		return nil, err
	}
	return c.d.manager.IntrospectToken(ctx, token)
}

// GetDevice gets current token device GetDevice 获取当前 Token 设备类型
func (c *AuthContext) GetDevice(ctx context.Context) (string, error) {
	token, err := c.d.requireToken()
	if err != nil {
		return "", err
	}
	return c.d.manager.GetDevice(ctx, token)
}

// GetDeviceId gets current token device ID GetDeviceId 获取当前 Token 设备 ID
func (c *AuthContext) GetDeviceId(ctx context.Context) (string, error) {
	token, err := c.d.requireToken()
	if err != nil {
		return "", err
	}
	return c.d.manager.GetDeviceId(ctx, token)
}

// GetTokenCreateTime gets current token create time GetTokenCreateTime 获取当前 Token 创建时间
func (c *AuthContext) GetTokenCreateTime(ctx context.Context) (int64, error) {
	token, err := c.d.requireToken()
	if err != nil {
		return 0, err
	}
	return c.d.manager.GetTokenCreateTime(ctx, token)
}

// GetTokenTTL gets current token TTL GetTokenTTL 获取当前 Token 剩余有效期
func (c *AuthContext) GetTokenTTL(ctx context.Context) (int64, error) {
	token, err := c.d.requireToken()
	if err != nil {
		return 0, err
	}
	return c.d.manager.GetTokenTTL(ctx, token)
}

// RenewTimeout renews current token timeout RenewTimeout 续期当前 Token
func (c *AuthContext) RenewTimeout(ctx context.Context, timeout time.Duration) error {
	token, err := c.d.requireToken()
	if err != nil {
		return err
	}
	return c.d.manager.RenewTimeout(ctx, token, timeout)
}

// Kickout kicks out current token Kickout 踢出当前 Token
func (c *AuthContext) Kickout(ctx context.Context) error {
	token, err := c.d.requireToken()
	if err != nil {
		return err
	}
	return c.d.manager.Kickout(ctx, token)
}

// Replace replaces current token Replace 顶替当前 Token
func (c *AuthContext) Replace(ctx context.Context) error {
	token, err := c.d.requireToken()
	if err != nil {
		return err
	}
	return c.d.manager.Replace(ctx, token)
}
