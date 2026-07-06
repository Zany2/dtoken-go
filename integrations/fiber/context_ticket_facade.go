// @Author daixk 2026/06/05
package fiber

import (
	"time"

	"github.com/Zany2/dtoken-go/core/ticket"
	gofiber "github.com/gofiber/fiber/v2"
)

// CreateTicketByContext creates ticket CreateTicketByContext 创建 Ticket
func CreateTicketByContext(c *gofiber.Ctx, opts ticket.CreateOptions) (*ticket.Ticket, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Ticket().Create(requestContext(c), opts)
}

// CreateTicketForCurrentLoginByContext creates ticket for current user CreateTicketForCurrentLoginByContext 为当前用户创建 Ticket
func CreateTicketForCurrentLoginByContext(c *gofiber.Ctx, opts ticket.CreateOptions) (*ticket.Ticket, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Ticket().CreateForCurrentLogin(requestContext(c), opts)
}

// CreateTicketWithTimeoutByContext creates ticket with timeout CreateTicketWithTimeoutByContext 使用指定有效期创建 Ticket
func CreateTicketWithTimeoutByContext(c *gofiber.Ctx, opts ticket.CreateOptions, timeout time.Duration) (*ticket.Ticket, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Ticket().CreateWithTimeout(requestContext(c), opts, timeout)
}

// ValidateTicketByContext validates ticket ValidateTicketByContext 校验 Ticket
func ValidateTicketByContext(c *gofiber.Ctx, ticketValue string, opts ...ticket.ValidateOptions) (*ticket.Ticket, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Ticket().Validate(requestContext(c), ticketValue, opts...)
}

// ConsumeTicketByContext consumes ticket ConsumeTicketByContext 娑堣垂 Ticket
func ConsumeTicketByContext(c *gofiber.Ctx, ticketValue string, opts ...ticket.ValidateOptions) (*ticket.ConsumeResult, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Ticket().Consume(requestContext(c), ticketValue, opts...)
}

// RevokeTicketByContext revokes ticket RevokeTicketByContext 鎾ら攢 Ticket
func RevokeTicketByContext(c *gofiber.Ctx, ticketValue string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Ticket().Revoke(requestContext(c), ticketValue)
}

// GetTicketStatusByContext gets ticket status GetTicketStatusByContext 获取 Ticket 状态
func GetTicketStatusByContext(c *gofiber.Ctx, ticketValue string) (ticket.Status, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return "", err
	}
	return dCtx.Ticket().GetStatus(requestContext(c), ticketValue)
}

// GetTicketTTLByContext gets ticket TTL GetTicketTTLByContext 获取 Ticket 剩余有效期
func GetTicketTTLByContext(c *gofiber.Ctx, ticketValue string) (int64, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return 0, err
	}
	return dCtx.Ticket().GetTTL(requestContext(c), ticketValue)
}
