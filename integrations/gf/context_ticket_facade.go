// @Author daixk 2026/06/05
package gf

import (
	"context"
	"time"

	"github.com/Zany2/dtoken-go/core/ticket"
)

// CreateTicketByCtx creates ticket CreateTicketByCtx 鍒涘缓 Ticket
func CreateTicketByCtx(ctx context.Context, opts ticket.CreateOptions) (*ticket.Ticket, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Ticket().Create(ctx, opts)
}

// CreateTicketForCurrentLoginByCtx creates ticket for current user CreateTicketForCurrentLoginByCtx 涓哄綋鍓嶇敤鎴峰垱寤?Ticket
func CreateTicketForCurrentLoginByCtx(ctx context.Context, opts ticket.CreateOptions) (*ticket.Ticket, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Ticket().CreateForCurrentLogin(ctx, opts)
}

// CreateTicketWithTimeoutByCtx creates ticket with timeout CreateTicketWithTimeoutByCtx 浣跨敤鎸囧畾鏈夋晥鏈熷垱寤?Ticket
func CreateTicketWithTimeoutByCtx(ctx context.Context, opts ticket.CreateOptions, timeout time.Duration) (*ticket.Ticket, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Ticket().CreateWithTimeout(ctx, opts, timeout)
}

// ValidateTicketByCtx validates ticket ValidateTicketByCtx 鏍￠獙 Ticket
func ValidateTicketByCtx(ctx context.Context, ticketValue string, opts ...ticket.ValidateOptions) (*ticket.Ticket, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Ticket().Validate(ctx, ticketValue, opts...)
}

// ConsumeTicketByCtx consumes ticket ConsumeTicketByCtx 娑堣垂 Ticket
func ConsumeTicketByCtx(ctx context.Context, ticketValue string, opts ...ticket.ValidateOptions) (*ticket.ConsumeResult, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Ticket().Consume(ctx, ticketValue, opts...)
}

// RevokeTicketByCtx revokes ticket RevokeTicketByCtx 鎾ら攢 Ticket
func RevokeTicketByCtx(ctx context.Context, ticketValue string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Ticket().Revoke(ctx, ticketValue)
}

// GetTicketStatusByCtx gets ticket status GetTicketStatusByCtx 鑾峰彇 Ticket 鐘舵€?
func GetTicketStatusByCtx(ctx context.Context, ticketValue string) (ticket.Status, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.Ticket().GetStatus(ctx, ticketValue)
}

// GetTicketTTLByCtx gets ticket TTL GetTicketTTLByCtx 鑾峰彇 Ticket 鍓╀綑鏈夋晥鏈?
func GetTicketTTLByCtx(ctx context.Context, ticketValue string) (int64, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return 0, err
	}
	return dCtx.Ticket().GetTTL(ctx, ticketValue)
}
