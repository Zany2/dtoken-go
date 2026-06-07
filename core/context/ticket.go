// @Author daixk 2026/06/05
package context

import (
	"context"
	"time"

	"github.com/Zany2/dtoken-go/core/ticket"
)

// Create creates a temporary ticket Create 创建临时 Ticket
func (c *TicketContext) Create(ctx context.Context, opts ticket.CreateOptions) (*ticket.Ticket, error) {
	return c.d.manager.CreateTicket(ctx, opts)
}

// CreateForCurrentLogin creates a ticket for current user CreateForCurrentLogin 为当前登录用户创建 Ticket
func (c *TicketContext) CreateForCurrentLogin(ctx context.Context, opts ticket.CreateOptions) (*ticket.Ticket, error) {
	loginID, err := c.d.currentLoginID(ctx)
	if err != nil {
		return nil, err
	}
	opts.LoginID = loginID
	return c.d.manager.CreateTicket(ctx, opts)
}

// CreateWithTimeout creates a ticket with timeout CreateWithTimeout 使用指定有效期创建 Ticket
func (c *TicketContext) CreateWithTimeout(ctx context.Context, opts ticket.CreateOptions, timeout time.Duration) (*ticket.Ticket, error) {
	return c.d.manager.CreateTicketWithTimeout(ctx, opts, timeout)
}

// Validate validates a ticket without consuming it Validate 校验 Ticket 但不消费
func (c *TicketContext) Validate(ctx context.Context, ticketValue string, opts ...ticket.ValidateOptions) (*ticket.Ticket, error) {
	return c.d.manager.ValidateTicket(ctx, ticketValue, opts...)
}

// Consume validates and consumes a ticket Consume 校验并消费 Ticket
func (c *TicketContext) Consume(ctx context.Context, ticketValue string, opts ...ticket.ValidateOptions) (*ticket.ConsumeResult, error) {
	return c.d.manager.ConsumeTicket(ctx, ticketValue, opts...)
}

// Revoke revokes a ticket Revoke 撤销 Ticket
func (c *TicketContext) Revoke(ctx context.Context, ticketValue string) error {
	return c.d.manager.RevokeTicket(ctx, ticketValue)
}

// GetStatus gets ticket status GetStatus 获取 Ticket 状态
func (c *TicketContext) GetStatus(ctx context.Context, ticketValue string) (ticket.Status, error) {
	return c.d.manager.GetTicketStatus(ctx, ticketValue)
}

// GetTTL gets ticket TTL GetTTL 获取 Ticket 剩余有效期
func (c *TicketContext) GetTTL(ctx context.Context, ticketValue string) (int64, error) {
	return c.d.manager.GetTicketTTL(ctx, ticketValue)
}
