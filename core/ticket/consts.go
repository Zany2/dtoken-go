// @Author daixk 2026/06/01
package ticket

import "time"

const (
	// DefaultTicketTTL stores default ticket ttl. DefaultTicketTTL 存储默认 Ticket 有效期。
	DefaultTicketTTL = 5 * time.Minute
	// TicketLength stores ticket random byte length before hex encoding. TicketLength 存储 Ticket 十六进制编码前的随机字节长度。
	TicketLength = 32
	// TicketKeySuffix stores ticket key suffix. TicketKeySuffix 存储 Ticket 存储键后缀。
	TicketKeySuffix = "ticket:"
)

// Status defines ticket lifecycle state. Status 定义 Ticket 生命周期状态。
type Status string

const (
	// StatusValid indicates the ticket can still be used. StatusValid 表示 Ticket 当前可用。
	StatusValid Status = "valid"
	// StatusConsumed indicates the ticket has been consumed. StatusConsumed 表示 Ticket 已消费。
	StatusConsumed Status = "consumed"
	// StatusRevoked indicates the ticket has been revoked. StatusRevoked 表示 Ticket 已撤销。
	StatusRevoked Status = "revoked"
	// StatusExpired indicates the ticket has expired. StatusExpired 表示 Ticket 已过期。
	StatusExpired Status = "expired"
	// StatusInvalid indicates the ticket is missing or malformed. StatusInvalid 表示 Ticket 无效或不存在。
	StatusInvalid Status = "invalid"
)
