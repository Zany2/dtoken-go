// @Author daixk 2026/06/05
package context

import (
	"context"
	"time"

	"github.com/Zany2/dtoken-go/core/shortkey"
)

// Create creates a pending short key Create 创建待确认短 Key
func (c *ShortKeyContext) Create(ctx context.Context, opts shortkey.CreateOptions) (*shortkey.ShortKey, error) {
	return c.d.manager.CreateShortKey(ctx, opts)
}

// CreateWithTimeout creates a short key with timeout CreateWithTimeout 使用指定有效期创建短 Key
func (c *ShortKeyContext) CreateWithTimeout(ctx context.Context, opts shortkey.CreateOptions, timeout time.Duration) (*shortkey.ShortKey, error) {
	return c.d.manager.CreateShortKeyWithTimeout(ctx, opts, timeout)
}

// Confirm confirms a pending short key Confirm 确认待处理短 Key
func (c *ShortKeyContext) Confirm(ctx context.Context, key string, opts shortkey.ConfirmOptions) (*shortkey.ShortKey, error) {
	return c.d.manager.ConfirmShortKey(ctx, key, opts)
}

// ConfirmForCurrentLogin confirms short key for current user ConfirmForCurrentLogin 使用当前登录用户确认短 Key
func (c *ShortKeyContext) ConfirmForCurrentLogin(ctx context.Context, key string, opts shortkey.ConfirmOptions) (*shortkey.ShortKey, error) {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return nil, err
	}
	opts.LoginID = loginID
	return c.d.manager.ConfirmShortKey(ctx, key, opts)
}

// Validate validates a short key without consuming it Validate 校验短 Key 但不消费
func (c *ShortKeyContext) Validate(ctx context.Context, key string, opts ...shortkey.ValidateOptions) (*shortkey.ShortKey, error) {
	return c.d.manager.ValidateShortKey(ctx, key, opts...)
}

// Consume validates and consumes a short key Consume 校验并消费短 Key
func (c *ShortKeyContext) Consume(ctx context.Context, key string, opts ...shortkey.ValidateOptions) (*shortkey.ConsumeResult, error) {
	return c.d.manager.ConsumeShortKey(ctx, key, opts...)
}

// Revoke revokes a short key Revoke 撤销短 Key
func (c *ShortKeyContext) Revoke(ctx context.Context, key string) error {
	return c.d.manager.RevokeShortKey(ctx, key)
}

// GetStatus gets short key status GetStatus 获取短 Key 状态
func (c *ShortKeyContext) GetStatus(ctx context.Context, key string) (shortkey.Status, error) {
	return c.d.manager.GetShortKeyStatus(ctx, key)
}

// GetTTL gets short key TTL GetTTL 获取短 Key 剩余有效期
func (c *ShortKeyContext) GetTTL(ctx context.Context, key string) (int64, error) {
	return c.d.manager.GetShortKeyTTL(ctx, key)
}
