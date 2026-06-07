// @Author daixk 2026/06/05
package context

import (
	"context"

	"github.com/Zany2/dtoken-go/core/manager"
)

// Get gets current account session Get 获取当前账号 Session
func (c *SessionContext) Get(ctx context.Context) (*manager.Session, error) {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return nil, err
	}
	return c.d.manager.GetSession(ctx, loginID)
}

// GetByToken gets current token session GetByToken 根据当前 Token 获取 Session
func (c *SessionContext) GetByToken(ctx context.Context) (*manager.Session, error) {
	token, err := c.d.requireToken()
	if err != nil {
		return nil, err
	}
	return c.d.manager.GetSessionByToken(ctx, token)
}

// SetValue sets one session data value SetValue 设置一项 Session 扩展数据
func (c *SessionContext) SetValue(ctx context.Context, key string, value any) error {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return err
	}
	return c.d.manager.SetSessionValue(ctx, loginID, key, value)
}

// GetValue gets one session data value GetValue 获取一项 Session 扩展数据
func (c *SessionContext) GetValue(ctx context.Context, key string) (any, bool, error) {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return nil, false, err
	}
	return c.d.manager.GetSessionValue(ctx, loginID, key)
}

// DeleteValue deletes one session data value DeleteValue 删除一项 Session 扩展数据
func (c *SessionContext) DeleteValue(ctx context.Context, key string) error {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return err
	}
	return c.d.manager.DeleteSessionValue(ctx, loginID, key)
}
