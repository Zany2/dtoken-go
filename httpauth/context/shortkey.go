// @Author daixk 2026/06/05
package context

import (
	"context"
	"time"

	"github.com/Zany2/dtoken-go/contract/shortkey"
	derror "github.com/Zany2/dtoken-go/errors"
)

// Create creates a pending short key Create 创建待确认短 Key
func (c *ShortKeyContext) Create(ctx context.Context, opts shortkey.CreateOptions) (*shortkey.ShortKey, error) {
	service := c.d.shortKeyService()
	if service == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	return service.CreateShortKey(ctx, opts)
}

// CreateWithTimeout creates a short key with timeout CreateWithTimeout 使用指定有效期创建短 Key
func (c *ShortKeyContext) CreateWithTimeout(ctx context.Context, opts shortkey.CreateOptions, timeout time.Duration) (*shortkey.ShortKey, error) {
	service := c.d.shortKeyService()
	if service == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	return service.CreateShortKeyWithTimeout(ctx, opts, timeout)
}

// Confirm confirms a pending short key Confirm 确认待处理短 Key
func (c *ShortKeyContext) Confirm(ctx context.Context, key string, opts shortkey.ConfirmOptions) (*shortkey.ShortKey, error) {
	service := c.d.shortKeyService()
	if service == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	return service.ConfirmShortKey(ctx, key, opts)
}

// ConfirmForCurrentLogin confirms short key for current user ConfirmForCurrentLogin 使用当前登录用户确认短 Key
func (c *ShortKeyContext) ConfirmForCurrentLogin(ctx context.Context, key string, opts shortkey.ConfirmOptions) (*shortkey.ShortKey, error) {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return nil, err
	}
	opts.LoginID = loginID
	service := c.d.shortKeyService()
	if service == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	return service.ConfirmShortKey(ctx, key, opts)
}

// Validate validates a short key without consuming it Validate 校验短 Key 但不消费
func (c *ShortKeyContext) Validate(ctx context.Context, key string, opts ...shortkey.ValidateOptions) (*shortkey.ShortKey, error) {
	service := c.d.shortKeyService()
	if service == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	return service.ValidateShortKey(ctx, key, opts...)
}

// Consume validates and consumes a short key Consume 校验并消费短 Key
func (c *ShortKeyContext) Consume(ctx context.Context, key string, opts ...shortkey.ValidateOptions) (*shortkey.ConsumeResult, error) {
	service := c.d.shortKeyService()
	if service == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	return service.ConsumeShortKey(ctx, key, opts...)
}

// Revoke revokes a short key Revoke 撤销短 Key
func (c *ShortKeyContext) Revoke(ctx context.Context, key string) error {
	service := c.d.shortKeyService()
	if service == nil {
		return derror.ErrModuleNotEnabled
	}
	return service.RevokeShortKey(ctx, key)
}

// GetStatus gets short key status GetStatus 获取短 Key 状态
func (c *ShortKeyContext) GetStatus(ctx context.Context, key string) (shortkey.Status, error) {
	service := c.d.shortKeyService()
	if service == nil {
		return shortkey.StatusInvalid, derror.ErrModuleNotEnabled
	}
	return service.GetShortKeyStatus(ctx, key)
}

// GetTTL gets short key TTL GetTTL 获取短 Key 剩余有效期
func (c *ShortKeyContext) GetTTL(ctx context.Context, key string) (int64, error) {
	service := c.d.shortKeyService()
	if service == nil {
		return 0, derror.ErrModuleNotEnabled
	}
	return service.GetShortKeyTTL(ctx, key)
}
