// @Author daixk 2026/06/01
package dtoken

import (
	"context"

	"github.com/Zany2/dtoken-go/core/ticket"
)

// CreateTicket creates a temporary ticket. CreateTicket 创建临时 Ticket。
func CreateTicket(ctx context.Context, loginID string, authType ...string) (*ticket.Ticket, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.CreateTicket(ctx, ticket.CreateOptions{LoginID: loginID})
}

// CreateTicketWithOptions creates a temporary ticket with options. CreateTicketWithOptions 使用选项创建临时 Ticket。
func CreateTicketWithOptions(ctx context.Context, opts ticket.CreateOptions, authType ...string) (*ticket.Ticket, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.CreateTicket(ctx, opts)
}

// ValidateTicket validates a ticket without consuming it. ValidateTicket 校验 Ticket 但不消费。
func ValidateTicket(ctx context.Context, ticketValue string, authType ...string) (*ticket.Ticket, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.ValidateTicket(ctx, ticketValue)
}

// ValidateTicketWithOptions validates a ticket with constraints. ValidateTicketWithOptions 使用约束校验 Ticket。
func ValidateTicketWithOptions(ctx context.Context, ticketValue string, opts ticket.ValidateOptions, authType ...string) (*ticket.Ticket, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.ValidateTicket(ctx, ticketValue, opts)
}

// ConsumeTicket validates and consumes a ticket. ConsumeTicket 校验并消费 Ticket。
func ConsumeTicket(ctx context.Context, ticketValue string, authType ...string) (*ticket.ConsumeResult, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.ConsumeTicket(ctx, ticketValue)
}

// ConsumeTicketWithOptions validates and consumes a ticket with constraints. ConsumeTicketWithOptions 使用约束校验并消费 Ticket。
func ConsumeTicketWithOptions(ctx context.Context, ticketValue string, opts ticket.ValidateOptions, authType ...string) (*ticket.ConsumeResult, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.ConsumeTicket(ctx, ticketValue, opts)
}

// RevokeTicket revokes a ticket. RevokeTicket 撤销 Ticket。
func RevokeTicket(ctx context.Context, ticketValue string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.RevokeTicket(ctx, ticketValue)
}

// GetTicketStatus returns ticket lifecycle status. GetTicketStatus 返回 Ticket 生命周期状态。
func GetTicketStatus(ctx context.Context, ticketValue string, authType ...string) (ticket.Status, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return ticket.StatusInvalid, err
	}
	return mgr.GetTicketStatus(ctx, ticketValue)
}

// GetTicketTTL returns ticket ttl in seconds. GetTicketTTL 获取 Ticket 剩余有效秒数。
func GetTicketTTL(ctx context.Context, ticketValue string, authType ...string) (int64, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return 0, err
	}
	return mgr.GetTicketTTL(ctx, ticketValue)
}
