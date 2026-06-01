// @Author daixk 2026/06/01
package manager

import (
	"context"
	"time"

	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/ticket"
)

// CreateTicket creates a temporary ticket. CreateTicket 创建临时 Ticket。
func (m *Manager) CreateTicket(ctx context.Context, opts ticket.CreateOptions) (*ticket.Ticket, error) {
	if m.ticketManager == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	return m.ticketManager.Create(ctx, opts)
}

// CreateTicketWithTimeout creates a temporary ticket with timeout. CreateTicketWithTimeout 使用指定有效期创建临时 Ticket。
func (m *Manager) CreateTicketWithTimeout(ctx context.Context, opts ticket.CreateOptions, timeout time.Duration) (*ticket.Ticket, error) {
	if m.ticketManager == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	return m.ticketManager.CreateWithTimeout(ctx, opts, timeout)
}

// ValidateTicket validates a ticket without consuming it. ValidateTicket 校验 Ticket 但不消费。
func (m *Manager) ValidateTicket(ctx context.Context, ticketValue string, opts ...ticket.ValidateOptions) (*ticket.Ticket, error) {
	if m.ticketManager == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	return m.ticketManager.Validate(ctx, ticketValue, opts...)
}

// ConsumeTicket validates and consumes a ticket. ConsumeTicket 校验并消费 Ticket。
func (m *Manager) ConsumeTicket(ctx context.Context, ticketValue string, opts ...ticket.ValidateOptions) (*ticket.ConsumeResult, error) {
	if m.ticketManager == nil {
		return nil, derror.ErrModuleNotEnabled
	}
	return m.ticketManager.Consume(ctx, ticketValue, opts...)
}

// RevokeTicket revokes a ticket. RevokeTicket 撤销 Ticket。
func (m *Manager) RevokeTicket(ctx context.Context, ticketValue string) error {
	if m.ticketManager == nil {
		return derror.ErrModuleNotEnabled
	}
	return m.ticketManager.Revoke(ctx, ticketValue)
}

// GetTicketStatus returns ticket lifecycle status. GetTicketStatus 返回 Ticket 生命周期状态。
func (m *Manager) GetTicketStatus(ctx context.Context, ticketValue string) (ticket.Status, error) {
	if m.ticketManager == nil {
		return ticket.StatusInvalid, derror.ErrModuleNotEnabled
	}
	return m.ticketManager.Status(ctx, ticketValue)
}

// GetTicketTTL returns ticket ttl in seconds. GetTicketTTL 获取 Ticket 剩余有效秒数。
func (m *Manager) GetTicketTTL(ctx context.Context, ticketValue string) (int64, error) {
	if m.ticketManager == nil {
		return 0, derror.ErrModuleNotEnabled
	}
	return m.ticketManager.GetTTL(ctx, ticketValue)
}
