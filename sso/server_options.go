// @Author daixk 2026/05/28
package sso

// ServerOptions defines deployable SSO server protocol options. ServerOptions 定义可部署 SSO 服务端协议选项。
type ServerOptions struct {
	Mode                 Mode              // Mode stores preferred SSO mode. Mode 存储首选 SSO 模式。
	HomeRoute            string            // HomeRoute stores default landing route. HomeRoute 存储默认主页地址。
	EnableSLO            bool              // EnableSLO enables single logout. EnableSLO 启用单点注销。
	AutoRenewTimeout     bool              // AutoRenewTimeout reserves center token renewal behavior. AutoRenewTimeout 预留中心 Token 续期行为。
	MaxRegisteredClient  int               // MaxRegisteredClient stores max client records per account. MaxRegisteredClient 存储账号可记录客户端上限。
	CheckSign            bool              // CheckSign enables request signature checks. CheckSign 启用请求签名校验。
	AllowAnonymousClient bool              // AllowAnonymousClient allows anonymous client access. AllowAnonymousClient 允许匿名客户端接入。
	AllowURLs            []string          // AllowURLs stores callback allow-list for anonymous clients. AllowURLs 存储匿名客户端回调白名单。
	SecretKey            string            // SecretKey stores default request signing secret. SecretKey 存储默认请求签名密钥。
	Endpoints            Endpoints         // Endpoints stores protocol paths. Endpoints 存储协议路径。
	Params               ParamNames        // Params stores protocol parameter names. Params 存储协议参数名。
	Clients              map[string]Client // Clients stores initial client registrations. Clients 存储初始化客户端配置。
}

// DefaultServerOptions returns default deployable server options. DefaultServerOptions 返回默认服务端协议选项。
func DefaultServerOptions() ServerOptions {
	return ServerOptions{
		Mode:                ModeTicket,
		EnableSLO:           true,
		MaxRegisteredClient: 32,
		CheckSign:           true,
		Endpoints:           DefaultEndpoints(),
		Params:              DefaultParamNames(),
		Clients:             make(map[string]Client),
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
