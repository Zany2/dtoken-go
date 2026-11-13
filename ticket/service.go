// @Author daixk 2026/06/01
package ticket

import (
	"context"
	"time"

	capability "github.com/Zany2/dtoken-go/contract/ticket"
	derror "github.com/Zany2/dtoken-go/errors"
	listener "github.com/Zany2/dtoken-go/event"
	"github.com/Zany2/dtoken-go/internal/utils"
)

// CreateTicket creates a temporary  CreateTicket 创建临时 Ticket。
func (m *Service) CreateTicket(ctx context.Context, opts CreateOptions) (*Ticket, error) {
	if m.ticketManager == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	value, err := m.ticketManager.Create(ctx, opts)
	if err != nil {
		return nil, err
	}
	m.triggerTicketEvent(listener.EventTicketCreate, value, listener.ActionCreate)
	return value, nil
}

// CreateTicketWithTimeout creates a temporary ticket with timeout. CreateTicketWithTimeout 使用指定有效期创建临时 Ticket。
func (m *Service) CreateTicketWithTimeout(ctx context.Context, opts CreateOptions, timeout time.Duration) (*Ticket, error) {
	if m.ticketManager == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	value, err := m.ticketManager.CreateWithTimeout(ctx, opts, timeout)
	if err != nil {
		return nil, err
	}
	m.triggerTicketEvent(listener.EventTicketCreate, value, listener.ActionCreate)
	return value, nil
}

// ValidateTicket validates a ticket without consuming it. ValidateTicket 校验 Ticket 但不消费。
func (m *Service) ValidateTicket(ctx context.Context, ticketValue string, opts ...ValidateOptions) (*Ticket, error) {
	if m.ticketManager == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	value, err := m.ticketManager.Validate(ctx, ticketValue, opts...)
	if err != nil {
		listener.Emit(m.events, listener.EventTicketValidate, "", "", "", ticketValue, map[string]any{
			listener.ExtraKeyAction: listener.ActionValidate,
			listener.ExtraKeyResult: false,
		})
		return value, err
	}
	m.triggerTicketEvent(listener.EventTicketValidate, value, listener.ActionValidate)
	return value, nil
}

// ConsumeTicket validates and consumes a  ConsumeTicket 校验并消费 Ticket。
func (m *Service) ConsumeTicket(ctx context.Context, ticketValue string, opts ...ValidateOptions) (*ConsumeResult, error) {
	if m.ticketManager == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	result, err := m.ticketManager.Consume(ctx, ticketValue, opts...)
	if result != nil {
		m.triggerTicketEvent(listener.EventTicketConsume, result.Ticket, listener.ActionConsume)
	}
	return result, err
}

// RevokeTicket revokes a  RevokeTicket 撤销 Ticket。
func (m *Service) RevokeTicket(ctx context.Context, ticketValue string) error {
	if m.ticketManager == nil {
		return derror.ErrModuleNotEnabled
	}
	value, validateErr := m.ticketManager.Validate(ctx, ticketValue)
	err := m.ticketManager.Revoke(ctx, ticketValue)
	if err == nil && validateErr == nil && value != nil {
		value.Status = StatusRevoked
		m.triggerTicketEvent(listener.EventTicketRevoke, value, listener.ActionRevoke)
	}
	return err
}

// GetTicketStatus returns ticket lifecycle status. GetTicketStatus 返回 Ticket 生命周期状态。
func (m *Service) GetTicketStatus(ctx context.Context, ticketValue string) (Status, error) {
	if m.ticketManager == nil {
		return StatusInvalid, derror.ErrModuleNotEnabled
	}
	return m.ticketManager.Status(ctx, ticketValue)
}

// GetTicketTTL returns ticket ttl in seconds. GetTicketTTL 获取 Ticket 剩余有效秒数。
func (m *Service) GetTicketTTL(ctx context.Context, ticketValue string) (int64, error) {
	if m.ticketManager == nil {
		return 0, derror.ErrModuleNotEnabled
	}
	return m.ticketManager.GetTTL(ctx, ticketValue)
}

// triggerTicketEvent emits ticket lifecycle events. triggerTicketEvent 触发 Ticket 生命周期事件。
func (m *Service) triggerTicketEvent(event listener.Event, value *Ticket, action string) {
	if value == nil {
		return
	}
	extra := map[string]any{
		listener.ExtraKeyAction:    action,
		listener.ExtraKeySource:    value.Source,
		listener.ExtraKeySourceApp: value.SourceApp,
		listener.ExtraKeyTargetApp: value.TargetApp,
		listener.ExtraKeyScopes:    value.Scopes,
		listener.ExtraKeyStatus:    value.Status,
		listener.ExtraKeyTTL:       remainingTicketTTLSeconds(value),
	}
	if event == listener.EventTicketValidate {
		extra[listener.ExtraKeyResult] = true
	}
	listener.Emit(m.events, event, value.LoginID, value.Device, value.DeviceID, value.Ticket, extra)
}

// remainingTicketTTLSeconds calculates remaining ticket seconds for event data. remainingTicketTTLSeconds 计算事件数据中的 Ticket 剩余秒数。
func remainingTicketTTLSeconds(value *Ticket) int64 {
	if value == nil {
		return 0
	}

	// Prefer the same precise deadline used by ticket validation. 优先使用与 Ticket 校验一致的精确截止时间。
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
	ticketManager *Manager
	events        listener.Emitter
}

// NewService creates a capability service with an optional event emitter and borrows components without transferring ownership; callers must keep them available throughout its lifetime. NewService 创建可选接入事件发布器的能力服务，仅借用组件，不转移所有权；调用方须保证组件在服务使用期间可用。
func NewService(manager *Manager, events listener.Emitter) *Service {
	return &Service{ticketManager: manager, events: events}
}

// The bundled service satisfies the public contract. 内置服务实现公开契约。
var _ capability.Service = (*Service)(nil)
