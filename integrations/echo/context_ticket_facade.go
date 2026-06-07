// @Author daixk 2026/06/05
package echo

import (
	"time"

	"github.com/Zany2/dtoken-go/core/ticket"
	echo4 "github.com/labstack/echo/v4"
)

// CreateTicketByContext creates ticket CreateTicketByContext 鍒涘缓 Ticket
func CreateTicketByContext(c echo4.Context, opts ticket.CreateOptions) (*ticket.Ticket, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Ticket().Create(requestContext(c), opts)
}

// CreateTicketForCurrentLoginByContext creates ticket for current user CreateTicketForCurrentLoginByContext 为当前用户创建 Ticket
func CreateTicketForCurrentLoginByContext(c echo4.Context, opts ticket.CreateOptions) (*ticket.Ticket, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Ticket().CreateForCurrentLogin(requestContext(c), opts)
}

// CreateTicketWithTimeoutByContext creates ticket with timeout CreateTicketWithTimeoutByContext 浣跨敤鎸囧畾鏈夋晥鏈熷垱寤?Ticket
func CreateTicketWithTimeoutByContext(c echo4.Context, opts ticket.CreateOptions, timeout time.Duration) (*ticket.Ticket, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Ticket().CreateWithTimeout(requestContext(c), opts, timeout)
}

// ValidateTicketByContext validates ticket ValidateTicketByContext 鏍￠獙 Ticket
func ValidateTicketByContext(c echo4.Context, ticketValue string, opts ...ticket.ValidateOptions) (*ticket.Ticket, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Ticket().Validate(requestContext(c), ticketValue, opts...)
}

// ConsumeTicketByContext consumes ticket ConsumeTicketByContext 娑堣垂 Ticket
func ConsumeTicketByContext(c echo4.Context, ticketValue string, opts ...ticket.ValidateOptions) (*ticket.ConsumeResult, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Ticket().Consume(requestContext(c), ticketValue, opts...)
}

// RevokeTicketByContext revokes ticket RevokeTicketByContext 鎾ら攢 Ticket
func RevokeTicketByContext(c echo4.Context, ticketValue string) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Ticket().Revoke(requestContext(c), ticketValue)
}

// GetTicketStatusByContext gets ticket status GetTicketStatusByContext 鑾峰彇 Ticket 鐘舵€?
func GetTicketStatusByContext(c echo4.Context, ticketValue string) (ticket.Status, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return "", err
	}
	return dCtx.Ticket().GetStatus(requestContext(c), ticketValue)
}

// GetTicketTTLByContext gets ticket TTL GetTicketTTLByContext 鑾峰彇 Ticket 鍓╀綑鏈夋晥鏈?
func GetTicketTTLByContext(c echo4.Context, ticketValue string) (int64, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return 0, err
	}
	return dCtx.Ticket().GetTTL(requestContext(c), ticketValue)
}
