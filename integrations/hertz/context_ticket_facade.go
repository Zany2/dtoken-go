// @Author daixk 2026/06/05
package hertz

import (
	"time"

	"github.com/Zany2/dtoken-go/core/ticket"
	hertzapp "github.com/cloudwego/hertz/pkg/app"
)

// CreateTicketByContext creates ticket CreateTicketByContext 创建 Ticket
func CreateTicketByContext(ctx *hertzapp.RequestContext, opts ticket.CreateOptions) (*ticket.Ticket, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Ticket().Create(requestContext(ctx), opts)
}

// CreateTicketForCurrentLoginByContext creates ticket for current user CreateTicketForCurrentLoginByContext 为当前用户创建 Ticket
func CreateTicketForCurrentLoginByContext(ctx *hertzapp.RequestContext, opts ticket.CreateOptions) (*ticket.Ticket, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Ticket().CreateForCurrentLogin(requestContext(ctx), opts)
}

// CreateTicketWithTimeoutByContext creates ticket with timeout CreateTicketWithTimeoutByContext 使用指定有效期创建 Ticket
func CreateTicketWithTimeoutByContext(ctx *hertzapp.RequestContext, opts ticket.CreateOptions, timeout time.Duration) (*ticket.Ticket, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Ticket().CreateWithTimeout(requestContext(ctx), opts, timeout)
}

// ValidateTicketByContext validates ticket ValidateTicketByContext 校验 Ticket
func ValidateTicketByContext(ctx *hertzapp.RequestContext, ticketValue string, opts ...ticket.ValidateOptions) (*ticket.Ticket, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Ticket().Validate(requestContext(ctx), ticketValue, opts...)
}

// ConsumeTicketByContext consumes ticket ConsumeTicketByContext 消费 Ticket
func ConsumeTicketByContext(ctx *hertzapp.RequestContext, ticketValue string, opts ...ticket.ValidateOptions) (*ticket.ConsumeResult, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Ticket().Consume(requestContext(ctx), ticketValue, opts...)
}

// RevokeTicketByContext revokes ticket RevokeTicketByContext 撤销 Ticket
func RevokeTicketByContext(ctx *hertzapp.RequestContext, ticketValue string) error {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return err
	}
	return dCtx.Ticket().Revoke(requestContext(ctx), ticketValue)
}

// GetTicketStatusByContext gets ticket status GetTicketStatusByContext 获取 Ticket 状态
func GetTicketStatusByContext(ctx *hertzapp.RequestContext, ticketValue string) (ticket.Status, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.Ticket().GetStatus(requestContext(ctx), ticketValue)
}

// GetTicketTTLByContext gets ticket TTL GetTicketTTLByContext 获取 Ticket 剩余有效期
func GetTicketTTLByContext(ctx *hertzapp.RequestContext, ticketValue string) (int64, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return 0, err
	}
	return dCtx.Ticket().GetTTL(requestContext(ctx), ticketValue)
}
