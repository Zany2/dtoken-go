// @Author daixk 2026/06/01
package ticket

import (
	"time"

	capability "github.com/Zany2/dtoken-go/contract/ticket"
)

const (
	// DefaultTicketTTL stores default ticket ttl. DefaultTicketTTL 存储默认 Ticket 有效期。
	DefaultTicketTTL = 5 * time.Minute
	// TicketLength stores ticket random byte length before hex encoding. TicketLength 存储 Ticket 十六进制编码前的随机字节长度。
	TicketLength = 32
	// TicketKeySuffix stores ticket key suffix. TicketKeySuffix 存储 Ticket 存储键后缀。
	TicketKeySuffix = "ticket:"
)

// Status aliases the public capability state. Status 引用公开能力状态。
type Status = capability.Status

const (
	// StatusValid identifies the capability state. StatusValid 标识能力状态。
	StatusValid = capability.StatusValid
	// StatusConsumed identifies the capability state. StatusConsumed 标识能力状态。
	StatusConsumed = capability.StatusConsumed
	// StatusRevoked identifies the capability state. StatusRevoked 标识能力状态。
	StatusRevoked = capability.StatusRevoked
	// StatusExpired identifies the capability state. StatusExpired 标识能力状态。
	StatusExpired = capability.StatusExpired
	// StatusInvalid identifies the capability state. StatusInvalid 标识能力状态。
	StatusInvalid = capability.StatusInvalid
)
