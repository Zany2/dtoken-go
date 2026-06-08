// @Author daixk 2026/06/01
package manager

import (
	"context"
	"time"

	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/listener"
	"github.com/Zany2/dtoken-go/core/shortkey"
)

// CreateShortKey creates a pending short key. CreateShortKey 创建待确认短 Key。
func (m *Manager) CreateShortKey(ctx context.Context, opts shortkey.CreateOptions) (*shortkey.ShortKey, error) {
	if m.shortKeyManager == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	value, err := m.shortKeyManager.Create(ctx, opts)
	if err != nil {
		return nil, err
	}
	m.triggerShortKeyEvent(listener.EventShortKeyCreate, value, listener.ActionCreate)
	return value, nil
}

// CreateShortKeyWithTimeout creates a pending short key with timeout. CreateShortKeyWithTimeout 使用指定有效期创建待确认短 Key。
func (m *Manager) CreateShortKeyWithTimeout(ctx context.Context, opts shortkey.CreateOptions, timeout time.Duration) (*shortkey.ShortKey, error) {
	if m.shortKeyManager == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	value, err := m.shortKeyManager.CreateWithTimeout(ctx, opts, timeout)
	if err != nil {
		return nil, err
	}
	m.triggerShortKeyEvent(listener.EventShortKeyCreate, value, listener.ActionCreate)
	return value, nil
}

// ConfirmShortKey confirms a pending short key. ConfirmShortKey 确认待处理短 Key。
func (m *Manager) ConfirmShortKey(ctx context.Context, key string, opts shortkey.ConfirmOptions) (*shortkey.ShortKey, error) {
	if m.shortKeyManager == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	value, err := m.shortKeyManager.Confirm(ctx, key, opts)
	if err != nil {
		return nil, err
	}
	m.triggerShortKeyEvent(listener.EventShortKeyConfirm, value, listener.ActionConfirm)
	return value, nil
}

// ValidateShortKey validates a short key without consuming it. ValidateShortKey 校验短 Key 但不消费。
func (m *Manager) ValidateShortKey(ctx context.Context, key string, opts ...shortkey.ValidateOptions) (*shortkey.ShortKey, error) {
	if m.shortKeyManager == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	value, err := m.shortKeyManager.Validate(ctx, key, opts...)
	if value != nil {
		m.triggerShortKeyEvent(listener.EventShortKeyValidate, value, listener.ActionValidate)
	}
	return value, err
}

// ConsumeShortKey validates and consumes a short key. ConsumeShortKey 校验并消费短 Key。
func (m *Manager) ConsumeShortKey(ctx context.Context, key string, opts ...shortkey.ValidateOptions) (*shortkey.ConsumeResult, error) {
	if m.shortKeyManager == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	result, err := m.shortKeyManager.Consume(ctx, key, opts...)
	if result != nil {
		m.triggerShortKeyEvent(listener.EventShortKeyConsume, result.ShortKey, listener.ActionConsume)
	}
	return result, err
}

// RevokeShortKey revokes a short key. RevokeShortKey 撤销短 Key。
func (m *Manager) RevokeShortKey(ctx context.Context, key string) error {
	if m.shortKeyManager == nil {
		return derror.ErrModuleNotEnabled
	}
	value, _ := m.shortKeyManager.Validate(ctx, key)
	err := m.shortKeyManager.Revoke(ctx, key)
	if err == nil {
		if value != nil {
			m.triggerShortKeyEvent(listener.EventShortKeyRevoke, value, listener.ActionRevoke)
		} else if key != "" {
			m.triggerEvent(listener.EventShortKeyRevoke, "", "", "", key, map[string]any{
				listener.ExtraKeyAction: listener.ActionRevoke,
			})
		}
	}
	return err
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

func (m *Manager) triggerShortKeyEvent(event listener.Event, value *shortkey.ShortKey, action string) {
	if value == nil {
		return
	}
	m.triggerEvent(event, value.LoginID, value.Device, value.DeviceId, value.Key, map[string]any{
		listener.ExtraKeyAction:    action,
		listener.ExtraKeyScene:     value.Scene,
		listener.ExtraKeySourceApp: value.SourceApp,
		listener.ExtraKeyTargetApp: value.TargetApp,
		listener.ExtraKeyScopes:    value.Scopes,
		listener.ExtraKeyStatus:    value.Status,
		listener.ExtraKeyTTL:       value.ExpiresIn,
	})
}
