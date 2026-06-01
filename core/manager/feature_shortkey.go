// @Author daixk 2026/06/01
package manager

import (
	"context"
	"time"

	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/shortkey"
)

// CreateShortKey creates a pending short key. CreateShortKey 创建待确认短 Key。
func (m *Manager) CreateShortKey(ctx context.Context, opts shortkey.CreateOptions) (*shortkey.ShortKey, error) {
	if m.shortKeyManager == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	return m.shortKeyManager.Create(ctx, opts)
}

// CreateShortKeyWithTimeout creates a pending short key with timeout. CreateShortKeyWithTimeout 使用指定有效期创建待确认短 Key。
func (m *Manager) CreateShortKeyWithTimeout(ctx context.Context, opts shortkey.CreateOptions, timeout time.Duration) (*shortkey.ShortKey, error) {
	if m.shortKeyManager == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	return m.shortKeyManager.CreateWithTimeout(ctx, opts, timeout)
}

// ConfirmShortKey confirms a pending short key. ConfirmShortKey 确认待处理短 Key。
func (m *Manager) ConfirmShortKey(ctx context.Context, key string, opts shortkey.ConfirmOptions) (*shortkey.ShortKey, error) {
	if m.shortKeyManager == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	return m.shortKeyManager.Confirm(ctx, key, opts)
}

// ValidateShortKey validates a short key without consuming it. ValidateShortKey 校验短 Key 但不消费。
func (m *Manager) ValidateShortKey(ctx context.Context, key string, opts ...shortkey.ValidateOptions) (*shortkey.ShortKey, error) {
	if m.shortKeyManager == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	return m.shortKeyManager.Validate(ctx, key, opts...)
}

// ConsumeShortKey validates and consumes a short key. ConsumeShortKey 校验并消费短 Key。
func (m *Manager) ConsumeShortKey(ctx context.Context, key string, opts ...shortkey.ValidateOptions) (*shortkey.ConsumeResult, error) {
	if m.shortKeyManager == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	return m.shortKeyManager.Consume(ctx, key, opts...)
}

// RevokeShortKey revokes a short key. RevokeShortKey 撤销短 Key。
func (m *Manager) RevokeShortKey(ctx context.Context, key string) error {
	if m.shortKeyManager == nil {
		return derror.ErrModuleNotEnabled
	}
	return m.shortKeyManager.Revoke(ctx, key)
}

// GetShortKeyStatus returns short key lifecycle status. GetShortKeyStatus 返回短 Key 生命周期状态。
func (m *Manager) GetShortKeyStatus(ctx context.Context, key string) (shortkey.Status, error) {
	if m.shortKeyManager == nil {
		return shortkey.StatusInvalid, derror.ErrModuleNotEnabled
	}
	return m.shortKeyManager.Status(ctx, key)
}

// GetShortKeyTTL returns short key ttl in seconds. GetShortKeyTTL 获取短 Key 剩余有效秒数。
func (m *Manager) GetShortKeyTTL(ctx context.Context, key string) (int64, error) {
	if m.shortKeyManager == nil {
		return 0, derror.ErrModuleNotEnabled
	}
	return m.shortKeyManager.GetTTL(ctx, key)
}
