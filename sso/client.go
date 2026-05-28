// @Author daixk 2026/05/28
package sso

import (
	"net/url"
	"strings"
)

// ClientConfig defines SSO client-side integration config. ClientConfig 定义 SSO 子应用接入配置。
type ClientConfig struct {
	Mode              Mode       // Mode stores preferred SSO mode. Mode 存储首选 SSO 模式。
	ClientID          string     // ClientID stores current application id. ClientID 存储当前应用 ID。
	ClientSecret      string     // ClientSecret stores current application secret. ClientSecret 存储当前应用密钥。
	ServerURL         string     // ServerURL stores SSO server base URL. ServerURL 存储 SSO 服务端根地址。
	LoginURL          string     // LoginURL stores current client login URL. LoginURL 存储当前客户端登录地址。
	LogoutCallbackURL string     // LogoutCallbackURL stores current client logout callback URL. LogoutCallbackURL 存储当前客户端注销回调地址。
	UseHTTPCheck      bool       // UseHTTPCheck enables remote ticket validation mode. UseHTTPCheck 启用远程 Ticket 校验模式。
	EnableSLO         bool       // EnableSLO enables single logout. EnableSLO 启用单点注销。
	RegisterCallback  bool       // RegisterCallback sends logout callback URL during login. RegisterCallback 登录时注册注销回调地址。
	CheckSign         bool       // CheckSign enables request signature checks. CheckSign 启用请求签名校验。
	SecretKey         string     // SecretKey stores request signing secret. SecretKey 存储请求签名密钥。
	Endpoints         Endpoints  // Endpoints stores server/client protocol paths. Endpoints 存储服务端/客户端协议路径。
	Params            ParamNames // Params stores protocol parameter names. Params 存储协议参数名。
}

// DefaultClientConfig returns default SSO client config. DefaultClientConfig 返回默认 SSO 客户端配置。
func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		Mode:      ModeTicket,
		EnableSLO: true,
		CheckSign: true,
		Endpoints: DefaultEndpoints(),
		Params:    DefaultParamNames(),
	}
}

// ClientApp helps a business application build SSO URLs. ClientApp 帮助业务应用构建 SSO URL。
type ClientApp struct {
	config ClientConfig // config stores client-side config. config 存储客户端配置。
}

// NewClientApp creates an SSO client helper. NewClientApp 创建 SSO 客户端辅助对象。
func NewClientApp(cfg ClientConfig) *ClientApp {
	defaults := DefaultClientConfig()
	if cfg.Mode == "" {
		cfg.Mode = defaults.Mode
	}
	if cfg.Endpoints == (Endpoints{}) {
		cfg.Endpoints = defaults.Endpoints
	}
	if cfg.Params == (ParamNames{}) {
		cfg.Params = defaults.Params
	}
	return &ClientApp{config: cfg}
}

// Config returns client config. Config 返回客户端配置。
func (c *ClientApp) Config() ClientConfig {
	if c == nil {
		return DefaultClientConfig()
	}
	return c.config
}

// AuthURL returns SSO server authorization URL. AuthURL 返回 SSO 服务端授权地址。
func (c *ClientApp) AuthURL(redirectURI string, extra url.Values) (string, error) {
	if c == nil {
		return "", ErrServerNotInitialized
	}
	values := cloneValues(extra)
	values.Set(c.config.Params.Redirect, redirectURI)
	values.Set(c.config.Params.Mode, string(c.config.Mode))
	if c.config.ClientID != "" {
		values.Set(c.config.Params.Client, c.config.ClientID)
	}
	if c.config.RegisterCallback && c.config.LogoutCallbackURL != "" {
		values.Set(c.config.Params.Callback, c.config.LogoutCallbackURL)
	}
	return c.buildServerURL(c.config.Endpoints.Authorize, values)
}

// ExchangeTicketURL returns SSO server ticket exchange URL. ExchangeTicketURL 返回 SSO 服务端 Ticket 交换地址。
func (c *ClientApp) ExchangeTicketURL(ticket string, extra url.Values) (string, error) {
	return c.ExchangeTicketURLWithRedirect(ticket, "", extra)
}

// ExchangeTicketURLWithRedirect returns ticket exchange URL with callback URI. ExchangeTicketURLWithRedirect 返回携带回调地址的 Ticket 交换地址。
func (c *ClientApp) ExchangeTicketURLWithRedirect(ticket, redirectURI string, extra url.Values) (string, error) {
	if c == nil {
		return "", ErrServerNotInitialized
	}
	values := cloneValues(extra)
	values.Set(c.config.Params.Ticket, ticket)
	if redirectURI != "" {
		values.Set(c.config.Params.Redirect, redirectURI)
	}
	if c.config.ClientID != "" {
		values.Set(c.config.Params.Client, c.config.ClientID)
	}
	if c.config.ClientSecret != "" {
		values.Set(c.config.Params.ClientSecret, c.config.ClientSecret)
	}
	return c.buildServerURL(c.config.Endpoints.Token, values)
}

// SignoutURL returns SSO server signout URL. SignoutURL 返回 SSO 服务端单点注销地址。
func (c *ClientApp) SignoutURL(loginID string, extra url.Values) (string, error) {
	if c == nil {
		return "", ErrServerNotInitialized
	}
	values := cloneValues(extra)
	values.Set(c.config.Params.LoginID, loginID)
	if c.config.ClientID != "" {
		values.Set(c.config.Params.Client, c.config.ClientID)
	}
	return c.buildServerURL(c.config.Endpoints.Logout, values)
}

func (c *ClientApp) buildServerURL(path string, values url.Values) (string, error) {
	base, err := url.Parse(joinURL(c.config.ServerURL, path))
	if err != nil {
		return "", err
	}
	if c.config.CheckSign && c.config.SecretKey != "" {
		values = NewSignerWithParams(c.config.SecretKey, c.config.Params).AttachSign(values)
	}
	base.RawQuery = values.Encode()
	return base.String(), nil
}

func joinURL(baseURL, path string) string {
	baseURL = strings.TrimRight(baseURL, "/")
	path = "/" + strings.TrimLeft(path, "/")
	return baseURL + path
}
