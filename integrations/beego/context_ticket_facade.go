// @Author daixk 2026/06/06
package beego

import (
	"time"

	"github.com/Zany2/dtoken-go/core/ticket"
	beegocontext "github.com/beego/beego/v2/server/web/context"
)

// CreateTicketByContext creates ticket CreateTicketByContext 创建 Ticket
func CreateTicketByContext(c *beegocontext.Context, opts ticket.CreateOptions) (*ticket.Ticket, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Ticket().Create(requestContext(c), opts)
}

// CreateTicketForCurrentLoginByContext creates ticket for current user CreateTicketForCurrentLoginByContext 为当前登录用户创建 Ticket
func CreateTicketForCurrentLoginByContext(c *beegocontext.Context, opts ticket.CreateOptions) (*ticket.Ticket, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Ticket().CreateForCurrentLogin(requestContext(c), opts)
}

// CreateTicketWithTimeoutByContext creates ticket with timeout CreateTicketWithTimeoutByContext 使用指定有效期创建 Ticket
func CreateTicketWithTimeoutByContext(c *beegocontext.Context, opts ticket.CreateOptions, timeout time.Duration) (*ticket.Ticket, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Ticket().CreateWithTimeout(requestContext(c), opts, timeout)
}

// ValidateTicketByContext validates ticket ValidateTicketByContext 校验 Ticket
func ValidateTicketByContext(c *beegocontext.Context, ticketValue string, opts ...ticket.ValidateOptions) (*ticket.Ticket, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Ticket().Validate(requestContext(c), ticketValue, opts...)
}

// ConsumeTicketByContext consumes ticket ConsumeTicketByContext 消费 Ticket
func ConsumeTicketByContext(c *beegocontext.Context, ticketValue string, opts ...ticket.ValidateOptions) (*ticket.ConsumeResult, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Ticket().Consume(requestContext(c), ticketValue, opts...)
}

// RevokeTicketByContext revokes ticket RevokeTicketByContext 撤销 Ticket
func RevokeTicketByContext(c *beegocontext.Context, ticketValue string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Ticket().Revoke(requestContext(c), ticketValue)
}

// GetTicketStatusByContext gets ticket status GetTicketStatusByContext 获取 Ticket 状态
func GetTicketStatusByContext(c *beegocontext.Context, ticketValue string) (ticket.Status, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return "", err
	}
	return dCtx.Ticket().GetStatus(requestContext(c), ticketValue)
}

// GetTicketTTLByContext gets ticket TTL GetTicketTTLByContext 获取 Ticket 剩余有效期
func GetTicketTTLByContext(c *beegocontext.Context, ticketValue string) (int64, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return 0, err
	}
	return dCtx.Ticket().GetTTL(requestContext(c), ticketValue)
}
