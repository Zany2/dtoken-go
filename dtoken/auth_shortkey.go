// @Author daixk 2026/06/01
package dtoken

import (
	"context"

	"github.com/Zany2/dtoken-go/core/shortkey"
)

// CreateShortKey creates a pending short key. CreateShortKey 创建待确认短 Key。
func (a *Auth) CreateShortKey(ctx context.Context) (*shortkey.ShortKey, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.CreateShortKey(ctx, shortkey.CreateOptions{})
}

// CreateShortKeyWithOptions creates a short key with options. CreateShortKeyWithOptions 使用选项创建短 Key。
func (a *Auth) CreateShortKeyWithOptions(ctx context.Context, opts shortkey.CreateOptions) (*shortkey.ShortKey, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.CreateShortKey(ctx, opts)
}

// ConfirmShortKey confirms a pending short key. ConfirmShortKey 确认待处理短 Key。
func (a *Auth) ConfirmShortKey(ctx context.Context, key string, loginID string) (*shortkey.ShortKey, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.ConfirmShortKey(ctx, key, shortkey.ConfirmOptions{LoginID: loginID})
}

// ConfirmShortKeyWithOptions confirms a pending short key with options. ConfirmShortKeyWithOptions 使用选项确认短 Key。
func (a *Auth) ConfirmShortKeyWithOptions(ctx context.Context, key string, opts shortkey.ConfirmOptions) (*shortkey.ShortKey, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.ConfirmShortKey(ctx, key, opts)
}

// ValidateShortKey validates a short key without consuming it. ValidateShortKey 校验短 Key 但不消费。
func (a *Auth) ValidateShortKey(ctx context.Context, key string) (*shortkey.ShortKey, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.ValidateShortKey(ctx, key)
}

// ValidateShortKeyWithOptions validates a short key with constraints. ValidateShortKeyWithOptions 使用约束校验短 Key。
func (a *Auth) ValidateShortKeyWithOptions(ctx context.Context, key string, opts shortkey.ValidateOptions) (*shortkey.ShortKey, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.ValidateShortKey(ctx, key, opts)
}

// ConsumeShortKey validates and consumes a short key. ConsumeShortKey 校验并消费短 Key。
func (a *Auth) ConsumeShortKey(ctx context.Context, key string) (*shortkey.ConsumeResult, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.ConsumeShortKey(ctx, key)
}

// ConsumeShortKeyWithOptions validates and consumes a short key with constraints. ConsumeShortKeyWithOptions 使用约束校验并消费短 Key。
func (a *Auth) ConsumeShortKeyWithOptions(ctx context.Context, key string, opts shortkey.ValidateOptions) (*shortkey.ConsumeResult, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.ConsumeShortKey(ctx, key, opts)
}

// RevokeShortKey revokes a short key. RevokeShortKey 撤销短 Key。
func (a *Auth) RevokeShortKey(ctx context.Context, key string) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.RevokeShortKey(ctx, key)
}

// GetShortKeyStatus returns short key lifecycle status. GetShortKeyStatus 返回短 Key 生命周期状态。
func (a *Auth) GetShortKeyStatus(ctx context.Context, key string) (shortkey.Status, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return shortkey.StatusInvalid, err
	}
	return mgr.GetShortKeyStatus(ctx, key)
}

// GetShortKeyTTL returns short key ttl in seconds. GetShortKeyTTL 获取短 Key 剩余有效秒数。
func (a *Auth) GetShortKeyTTL(ctx context.Context, key string) (int64, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return 0, err
	}
	return mgr.GetShortKeyTTL(ctx, key)
}
