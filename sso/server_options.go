// @Author daixk 2026/05/28
package sso

import (
	"net/http"
	"time"
)

// ServerOptions defines deployable SSO server protocol options. ServerOptions 定义可部署 SSO 服务端协议选项。
type ServerOptions struct {
	Mode                     Mode              // Mode stores preferred SSO mode. Mode 存储首选 SSO 模式。
	HomeRoute                string            // HomeRoute stores default landing route. HomeRoute 存储默认主页地址。
	EnableSLO                bool              // EnableSLO enables single logout. EnableSLO 启用单点注销。
	AutoRenewTimeout         bool              // AutoRenewTimeout reserves center token renewal behavior. AutoRenewTimeout 预留中心 Token 续期行为。
	MaxRegisteredClient      int               // MaxRegisteredClient stores max client records per account. MaxRegisteredClient 存储账号可记录客户端上限。
	LogoutCallbackBestEffort bool              // LogoutCallbackBestEffort continues logout when a client callback fails. LogoutCallbackBestEffort 在客户端回调失败时仍继续注销。
	LogoutCallbackTimeout    time.Duration     // LogoutCallbackTimeout stores timeout for each logout callback. LogoutCallbackTimeout 存储单个注销回调超时时间。
	LogoutHTTPClient         *http.Client      // LogoutHTTPClient stores optional logout callback HTTP client. LogoutHTTPClient 存储可选注销回调 HTTP 客户端。
	CheckSign                bool              // CheckSign enables request signature checks. CheckSign 启用请求签名校验。
	AllowAnonymousClient     bool              // AllowAnonymousClient allows anonymous client access. AllowAnonymousClient 允许匿名客户端接入。
	AllowURLs                []string          // AllowURLs stores callback allow-list for anonymous clients. AllowURLs 存储匿名客户端回调白名单。
	SecretKey                string            // SecretKey stores default request signing secret. SecretKey 存储默认请求签名密钥。
	Endpoints                Endpoints         // Endpoints stores protocol paths. Endpoints 存储协议路径。
	Params                   ParamNames        // Params stores protocol parameter names. Params 存储协议参数名。
	Clients                  map[string]Client // Clients stores initial client registrations. Clients 存储初始化客户端配置。
}

// DefaultServerOptions returns default deployable server options. DefaultServerOptions 返回默认服务端协议选项。
func DefaultServerOptions() ServerOptions {
	return ServerOptions{
		Mode:                  ModeTicket,
		EnableSLO:             true,
		MaxRegisteredClient:   32,
		LogoutCallbackTimeout: 3 * time.Second,
		CheckSign:             true,
		Endpoints:             DefaultEndpoints(),
		Params:                DefaultParamNames(),
		Clients:               make(map[string]Client),
	}
}

// RegisterClients registers initial clients into server storage. RegisterClients 将初始客户端注册到服务端存储。
func (s *Server) RegisterClients(clients map[string]Client) error {
	for id, client := range clients {
		if client.ClientID == "" {
			client.ClientID = id
		}
		if err := s.RegisterClient(&client); err != nil {
			return err
		}
	}
	return nil
}

// TicketResult converts a consumed ticket into a client-facing result. TicketResult 将已消费 Ticket 转为客户端结果。
func TicketResult(ticket *Ticket) *TicketExchangeResult {
	if ticket == nil {
		return nil
	}
	return &TicketExchangeResult{
		LoginID:  ticket.LoginID,
		CenterID: ticket.LoginID,
		Scopes:   append([]string(nil), ticket.Scopes...),
		Extra:    cloneMap(ticket.Extra),
	}
}

// OAuth2CodeResult converts a consumed OAuth2 code into a client-facing result. OAuth2CodeResult 将已消费授权码转为客户端结果。
func OAuth2CodeResult(code *OAuth2Code) *TicketExchangeResult {
	if code == nil {
		return nil
	}
	return &TicketExchangeResult{
		LoginID:  code.LoginID,
		CenterID: code.LoginID,
		Scopes:   append([]string(nil), code.Scopes...),
		Extra:    cloneMap(code.Extra),
	}
}

// TicketCredentialInfo converts a ticket into normalized credential info. TicketCredentialInfo 将 Ticket 转为标准化凭证信息。
func TicketCredentialInfo(ticket *Ticket, ttl int64) *CredentialInfo {
	if ticket == nil {
		return inactiveCredential()
	}
	return &CredentialInfo{
		Active:    true,
		Mode:      ModeTicket,
		LoginID:   ticket.LoginID,
		ClientID:  ticket.ClientID,
		Scopes:    append([]string(nil), ticket.Scopes...),
		ExpiresIn: ttl,
		Extra:     cloneMap(ticket.Extra),
	}
}

// SharedTokenCredentialInfo converts a shared token into normalized credential info. SharedTokenCredentialInfo 将共享 Token 转为标准化凭证信息。
func SharedTokenCredentialInfo(token *SharedToken, ttl int64) *CredentialInfo {
	if token == nil {
		return inactiveCredential()
	}
	return &CredentialInfo{
		Active:    true,
		Mode:      ModeSharedToken,
		LoginID:   token.LoginID,
		ClientID:  token.ClientID,
		Scopes:    append([]string(nil), token.Scopes...),
		ExpiresIn: ttl,
		Extra:     cloneMap(token.Extra),
	}
}

// RemoteSessionCredentialInfo converts a remote session into normalized credential info. RemoteSessionCredentialInfo 将远程会话转为标准化凭证信息。
func RemoteSessionCredentialInfo(session *RemoteSession, ttl int64) *CredentialInfo {
	if session == nil {
		return inactiveCredential()
	}
	return &CredentialInfo{
		Active:    true,
		Mode:      ModeRemoteSession,
		LoginID:   session.LoginID,
		ClientID:  session.ClientID,
		Scopes:    append([]string(nil), session.Scopes...),
		ExpiresIn: ttl,
		Extra:     cloneMap(session.Extra),
	}
}

// OAuth2CodeCredentialInfo converts an OAuth2 code into normalized credential info. OAuth2CodeCredentialInfo 将授权码转为标准化凭证信息。
func OAuth2CodeCredentialInfo(code *OAuth2Code, ttl int64) *CredentialInfo {
	if code == nil {
		return inactiveCredential()
	}
	return &CredentialInfo{
		Active:    true,
		Mode:      ModeOAuth2,
		LoginID:   code.LoginID,
		ClientID:  code.ClientID,
		Scopes:    append([]string(nil), code.Scopes...),
		ExpiresIn: ttl,
		Extra:     cloneMap(code.Extra),
	}
}

func inactiveCredential() *CredentialInfo {
	return &CredentialInfo{Active: false}
}

func cloneMap(values map[string]any) map[string]any {
	if len(values) == 0 {
		return nil
	}
	copied := make(map[string]any, len(values))
	for key, value := range values {
		copied[key] = value
	}
	return copied
}
