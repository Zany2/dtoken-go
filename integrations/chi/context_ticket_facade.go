// @Author daixk 2026/06/05
package chi

import (
	"context"
	"time"

	"github.com/Zany2/dtoken-go/core/ticket"
)

// CreateTicketByCtx creates ticket CreateTicketByCtx 创建 Ticket
func CreateTicketByCtx(ctx context.Context, opts ticket.CreateOptions) (*ticket.Ticket, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Ticket().Create(ctx, opts)
}

// CreateTicketForCurrentLoginByCtx creates ticket for current user CreateTicketForCurrentLoginByCtx 为当前用户创建 Ticket
func CreateTicketForCurrentLoginByCtx(ctx context.Context, opts ticket.CreateOptions) (*ticket.Ticket, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Ticket().CreateForCurrentLogin(ctx, opts)
}

// CreateTicketWithTimeoutByCtx creates ticket with timeout CreateTicketWithTimeoutByCtx 使用指定有效期创建 Ticket
func CreateTicketWithTimeoutByCtx(ctx context.Context, opts ticket.CreateOptions, timeout time.Duration) (*ticket.Ticket, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Ticket().CreateWithTimeout(ctx, opts, timeout)
}

// ValidateTicketByCtx validates ticket ValidateTicketByCtx 校验 Ticket
func ValidateTicketByCtx(ctx context.Context, ticketValue string, opts ...ticket.ValidateOptions) (*ticket.Ticket, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Ticket().Validate(ctx, ticketValue, opts...)
}

// ConsumeTicketByCtx consumes ticket ConsumeTicketByCtx 消费 Ticket
func ConsumeTicketByCtx(ctx context.Context, ticketValue string, opts ...ticket.ValidateOptions) (*ticket.ConsumeResult, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Ticket().Consume(ctx, ticketValue, opts...)
}

// RevokeTicketByCtx revokes ticket RevokeTicketByCtx 撤销 Ticket
func RevokeTicketByCtx(ctx context.Context, ticketValue string) error {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return err
	}
	return dCtx.Ticket().Revoke(ctx, ticketValue)
}

// GetTicketStatusByCtx gets ticket status GetTicketStatusByCtx 获取 Ticket 状态
func GetTicketStatusByCtx(ctx context.Context, ticketValue string) (ticket.Status, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.Ticket().GetStatus(ctx, ticketValue)
}

// GetTicketTTLByCtx gets ticket TTL GetTicketTTLByCtx 获取 Ticket 剩余有效期
func GetTicketTTLByCtx(ctx context.Context, ticketValue string) (int64, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return 0, err
	}
	return dCtx.Ticket().GetTTL(ctx, ticketValue)
}
