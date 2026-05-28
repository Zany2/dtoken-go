// @Author daixk 2025/12/22 15:56:00
package sso

import "time"

const (
	// DefaultTicketExpiration stores the default SSO ticket TTL. DefaultTicketExpiration 存储默认 SSO Ticket 有效期。
	DefaultTicketExpiration = 5 * time.Minute
	// TicketLength stores ticket random byte length. TicketLength 存储 Ticket 随机字节长度。
	TicketLength = 32

	// ClientKeySuffix stores SSO client key suffix. ClientKeySuffix 存储 SSO 客户端键后缀。
	ClientKeySuffix = "sso:client:"
	// TicketKeySuffix stores SSO ticket key suffix. TicketKeySuffix 存储 SSO Ticket 键后缀。
	TicketKeySuffix = "sso:ticket:"
)

// Mode defines SSO implementation mode. Mode 定义 SSO 实现模式。
type Mode string

const (
	// ModeTicket stores ticket based SSO mode. ModeTicket 存储 Ticket 模式。
	ModeTicket Mode = "ticket"
	// ModeSharedToken stores shared token SSO mode. ModeSharedToken 存储共享 Token 模式。
	ModeSharedToken Mode = "shared_token"
	// ModeRemoteSession stores remote session SSO mode. ModeRemoteSession 存储远程会话模式。
	ModeRemoteSession Mode = "remote_session"
	// ModeOAuth2 stores OAuth2 based SSO mode. ModeOAuth2 存储 OAuth2 模式。
	ModeOAuth2 Mode = "oauth2"
)
