// @Author daixk 2025/12/22 15:56:00
package sso

import "time"

const (
	// DefaultTicketExpiration stores the default SSO ticket TTL. DefaultTicketExpiration 存储默认 SSO Ticket 有效期。
	DefaultTicketExpiration = 5 * time.Minute
	// DefaultSharedTokenExpiration stores the default SSO shared token TTL. DefaultSharedTokenExpiration 存储默认 SSO 共享 Token 有效期。
	DefaultSharedTokenExpiration = 2 * time.Hour
	// DefaultRemoteSessionExpiration stores the default SSO remote session TTL. DefaultRemoteSessionExpiration 存储默认 SSO 远程会话有效期。
	DefaultRemoteSessionExpiration = 2 * time.Hour
	// DefaultOAuth2CodeExpiration stores the default SSO OAuth2 code TTL. DefaultOAuth2CodeExpiration 存储默认 SSO OAuth2 授权码有效期。
	DefaultOAuth2CodeExpiration = 10 * time.Minute

	// TicketLength stores ticket random byte length before hex encoding. TicketLength 存储十六进制编码前的 Ticket 随机字节长度。
	TicketLength = 32
	// SharedTokenLength stores shared token random byte length before hex encoding. SharedTokenLength 存储十六进制编码前的共享 Token 随机字节长度。
	SharedTokenLength = 32
	// RemoteSessionLength stores remote session random byte length before hex encoding. RemoteSessionLength 存储十六进制编码前的远程会话随机字节长度。
	RemoteSessionLength = 32
	// OAuth2CodeLength stores OAuth2 code random byte length before hex encoding. OAuth2CodeLength 存储十六进制编码前的 OAuth2 授权码随机字节长度。
	OAuth2CodeLength = 32

	// ClientKeySuffix stores SSO client key suffix. ClientKeySuffix 存储 SSO 客户端键后缀。
	ClientKeySuffix = "sso:client:"
	// TicketKeySuffix stores SSO ticket key suffix. TicketKeySuffix 存储 SSO Ticket 键后缀。
	TicketKeySuffix = "sso:ticket:"
	// SharedTokenKeySuffix stores SSO shared token key suffix. SharedTokenKeySuffix 存储 SSO 共享 Token 键后缀。
	SharedTokenKeySuffix = "sso:shared-token:"
	// RemoteSessionKeySuffix stores SSO remote session key suffix. RemoteSessionKeySuffix 存储 SSO 远程会话键后缀。
	RemoteSessionKeySuffix = "sso:remote-session:"
	// OAuth2CodeKeySuffix stores SSO OAuth2 code key suffix. OAuth2CodeKeySuffix 存储 SSO OAuth2 授权码键后缀。
	OAuth2CodeKeySuffix = "sso:oauth2:code:"
	// ClientSessionKeySuffix stores SSO client session key suffix. ClientSessionKeySuffix 存储 SSO 客户端会话键后缀。
	ClientSessionKeySuffix = "sso:client-session:"
)

// Mode defines SSO implementation mode for current and future flows. Mode 定义当前和未来 SSO 流程的实现模式。
type Mode string

const (
	// ModeTicket stores ticket based SSO mode. ModeTicket 存储基于一次性 Ticket 的 SSO 模式。
	ModeTicket Mode = "ticket"
	// ModeSharedToken stores shared token SSO mode. ModeSharedToken 存储共享 Token 模式。
	ModeSharedToken Mode = "shared_token"
	// ModeRemoteSession stores remote session SSO mode. ModeRemoteSession 存储远程会话模式。
	ModeRemoteSession Mode = "remote_session"
	// ModeOAuth2 stores OAuth2 based SSO mode. ModeOAuth2 存储 OAuth2 模式。
	ModeOAuth2 Mode = "oauth2"
)
