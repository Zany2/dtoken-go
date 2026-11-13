// @Author daixk 2026/06/01
package shortkey

import (
	"context"
	"time"

	capability "github.com/Zany2/dtoken-go/contract/shortkey"
	derror "github.com/Zany2/dtoken-go/errors"
	listener "github.com/Zany2/dtoken-go/event"
	"github.com/Zany2/dtoken-go/internal/utils"
)

// CreateShortKey creates a pending short key. CreateShortKey 创建待确认短 Key。
func (m *Service) CreateShortKey(ctx context.Context, opts CreateOptions) (*ShortKey, error) {
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
func (m *Service) CreateShortKeyWithTimeout(ctx context.Context, opts CreateOptions, timeout time.Duration) (*ShortKey, error) {
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
func (m *Service) ConfirmShortKey(ctx context.Context, key string, opts ConfirmOptions) (*ShortKey, error) {
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
func (m *Service) ValidateShortKey(ctx context.Context, key string, opts ...ValidateOptions) (*ShortKey, error) {
	if m.shortKeyManager == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	value, err := m.shortKeyManager.Validate(ctx, key, opts...)
	if err != nil {
		listener.Emit(m.events, listener.EventShortKeyValidate, "", "", "", key, map[string]any{
			listener.ExtraKeyAction: listener.ActionValidate,
			listener.ExtraKeyResult: false,
		})
		return value, err
	}
	m.triggerShortKeyEvent(listener.EventShortKeyValidate, value, listener.ActionValidate)
	return value, nil
}

// ConsumeShortKey validates and consumes a short key. ConsumeShortKey 校验并消费短 Key。
func (m *Service) ConsumeShortKey(ctx context.Context, key string, opts ...ValidateOptions) (*ConsumeResult, error) {
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
func (m *Service) RevokeShortKey(ctx context.Context, key string) error {
	if m.shortKeyManager == nil {
		return derror.ErrModuleNotEnabled
	}
	value, validateErr := m.shortKeyManager.Validate(ctx, key)
	status, statusErr := m.shortKeyManager.Status(ctx, key)
	err := m.shortKeyManager.Revoke(ctx, key)
	if err == nil && validateErr == nil && value != nil {
		value.Status = StatusRevoked
		m.triggerShortKeyEvent(listener.EventShortKeyRevoke, value, listener.ActionRevoke)
	} else if err == nil && statusErr == nil && status == StatusPending && key != "" {
		listener.Emit(m.events, listener.EventShortKeyRevoke, "", "", "", key, map[string]any{
			listener.ExtraKeyAction: listener.ActionRevoke,
			listener.ExtraKeyStatus: StatusRevoked,
		})
	}
	return err
}

// GetShortKeyStatus returns short key lifecycle status. GetShortKeyStatus 返回短 Key 生命周期状态。
func (m *Service) GetShortKeyStatus(ctx context.Context, key string) (Status, error) {
	if m.shortKeyManager == nil {
		return StatusInvalid, derror.ErrModuleNotEnabled
	}
	return m.shortKeyManager.Status(ctx, key)
}

// GetShortKeyTTL returns short key ttl in seconds. GetShortKeyTTL 获取短 Key 剩余有效秒数。
func (m *Service) GetShortKeyTTL(ctx context.Context, key string) (int64, error) {
	if m.shortKeyManager == nil {
		return 0, derror.ErrModuleNotEnabled
	}
	return m.shortKeyManager.GetTTL(ctx, key)
}

// triggerShortKeyEvent emits short key lifecycle events. triggerShortKeyEvent 触发短 Key 生命周期事件。
func (m *Service) triggerShortKeyEvent(event listener.Event, value *ShortKey, action string) {
	if value == nil {
		return
	}
	extra := map[string]any{
		listener.ExtraKeyAction:    action,
		listener.ExtraKeyScene:     value.Scene,
		listener.ExtraKeySourceApp: value.SourceApp,
		listener.ExtraKeyTargetApp: value.TargetApp,
		listener.ExtraKeyScopes:    value.Scopes,
		listener.ExtraKeyStatus:    value.Status,
		listener.ExtraKeyTTL:       remainingShortKeyTTLSeconds(value),
	}
	if event == listener.EventShortKeyValidate {
		extra[listener.ExtraKeyResult] = true
	}
	listener.Emit(m.events, event, value.LoginID, value.Device, value.DeviceID, value.Key, extra)
}

// remainingShortKeyTTLSeconds calculates remaining short key seconds for event data. remainingShortKeyTTLSeconds 计算事件数据中的短 Key 剩余秒数。
func remainingShortKeyTTLSeconds(value *ShortKey) int64 {
	if value == nil {
		return 0
	}

	// Prefer the same precise deadline used by short-key validation. 优先使用与短 Key 校验一致的精确截止时间。
	if !value.ExpiresAt.IsZero() {
		return utils.DurationSecondsCeil(time.Until(value.ExpiresAt))
	}

	// Preserve compatibility with records created before precise deadlines were stored. 兼容尚未存储精确截止时间的旧记录。
	if value.ExpiresIn <= 0 {
		return 0
	}
	remaining := value.CreateTime + value.ExpiresIn - time.Now().Unix()
	if remaining <= 0 {
		return 0
	}
	return remaining
}

// Service coordinates capability operations and lifecycle events. Service 协调能力操作与生命周期事件。
type Service struct {
	shortKeyManager *Manager
	events          listener.Emitter
}

// NewService creates a capability service with an optional event emitter and borrows components without transferring ownership; callers must keep them available throughout its lifetime. NewService 创建可选接入事件发布器的能力服务，仅借用组件，不转移所有权；调用方须保证组件在服务使用期间可用。
func NewService(manager *Manager, events listener.Emitter) *Service {
	return &Service{shortKeyManager: manager, events: events}
}

// The bundled service satisfies the public contract. 内置服务实现公开契约。
var _ capability.Service = (*Service)(nil)
