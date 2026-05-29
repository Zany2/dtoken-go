// @Author daixk 2026/05/28
package sso

import "strings"

const (
	// MessageTicketExchange stores ticket-exchange message type. MessageTicketExchange 存储 Ticket 交换消息类型。
	MessageTicketExchange = "ticket.exchange"
	// MessageSignout stores server-side signout message type. MessageSignout 存储服务端单点注销消息类型。
	MessageSignout = "session.signout"
	// MessageLogoutCallback stores client logout callback message type. MessageLogoutCallback 存储客户端注销回调消息类型。
	MessageLogoutCallback = "session.logout_callback"
)

const (
	// ResultOK stores the conventional success value. ResultOK 存储约定成功值。
	ResultOK = "ok"
	// ClientWildcard allows a trusted caller to target all clients. ClientWildcard 允许可信调用方匹配全部客户端。
	ClientWildcard = "*"
	// ClientAnonymous stores anonymous client id. ClientAnonymous 存储匿名客户端标识。
	ClientAnonymous = "anonymous"
)

// Endpoints stores DToken-Go SSO HTTP paths. Endpoints 存储 DToken-Go SSO HTTP 路径。
type Endpoints struct {
	Authorize      string // Authorize stores login-center authorization path. Authorize 存储登录中心授权路径。
	Token          string // Token stores ticket/code exchange path. Token 存储 Ticket/授权码交换路径。
	Introspect     string // Introspect stores credential introspection path. Introspect 存储凭证检查路径。
	UserInfo       string // UserInfo stores user info path. UserInfo 存储用户信息路径。
	Revoke         string // Revoke stores credential revoke path. Revoke 存储凭证撤销路径。
	Logout         string // Logout stores center logout path. Logout 存储中心注销路径。
	Message        string // Message stores message receiving path. Message 存储消息接收路径。
	ClientLogin    string // ClientLogin stores client login entry path. ClientLogin 存储子应用登录入口路径。
	ClientCallback string // ClientCallback stores client callback path. ClientCallback 存储子应用回调路径。
	ClientLogout   string // ClientLogout stores client logout path. ClientLogout 存储子应用注销路径。
	ClientMessage  string // ClientMessage stores client message receiving path. ClientMessage 存储子应用消息接收路径。
}

// DefaultEndpoints returns default DToken-Go SSO HTTP paths. DefaultEndpoints 返回默认 DToken-Go SSO HTTP 路径。
func DefaultEndpoints() Endpoints {
	return Endpoints{
		Authorize:      "/sso/authorize",
		Token:          "/sso/token",
		Introspect:     "/sso/introspect",
		UserInfo:       "/sso/userinfo",
		Revoke:         "/sso/revoke",
		Logout:         "/sso/logout",
		Message:        "/sso/messages",
		ClientLogin:    "/sso/login",
		ClientCallback: "/sso/callback",
		ClientLogout:   "/sso/logout",
		ClientMessage:  "/sso/messages",
	}
}

// AddPrefix returns a copy with prefix prepended to every path. AddPrefix 返回给全部路径追加前缀后的副本。
func (e Endpoints) AddPrefix(prefix string) Endpoints {
	e.Authorize = joinPath(prefix, e.Authorize)
	e.Token = joinPath(prefix, e.Token)
	e.Introspect = joinPath(prefix, e.Introspect)
	e.UserInfo = joinPath(prefix, e.UserInfo)
	e.Revoke = joinPath(prefix, e.Revoke)
	e.Logout = joinPath(prefix, e.Logout)
	e.Message = joinPath(prefix, e.Message)
	e.ClientLogin = joinPath(prefix, e.ClientLogin)
	e.ClientCallback = joinPath(prefix, e.ClientCallback)
	e.ClientLogout = joinPath(prefix, e.ClientLogout)
	e.ClientMessage = joinPath(prefix, e.ClientMessage)
	return e
}

// ReplacePrefix returns a copy with the /sso prefix replaced. ReplacePrefix 返回替换 /sso 前缀后的副本。
func (e Endpoints) ReplacePrefix(prefix string) Endpoints {
	e.Authorize = replacePathPrefix(e.Authorize, prefix)
	e.Token = replacePathPrefix(e.Token, prefix)
	e.Introspect = replacePathPrefix(e.Introspect, prefix)
	e.UserInfo = replacePathPrefix(e.UserInfo, prefix)
	e.Revoke = replacePathPrefix(e.Revoke, prefix)
	e.Logout = replacePathPrefix(e.Logout, prefix)
	e.Message = replacePathPrefix(e.Message, prefix)
	e.ClientLogin = replacePathPrefix(e.ClientLogin, prefix)
	e.ClientCallback = replacePathPrefix(e.ClientCallback, prefix)
	e.ClientLogout = replacePathPrefix(e.ClientLogout, prefix)
	e.ClientMessage = replacePathPrefix(e.ClientMessage, prefix)
	return e
}

// ParamNames stores conventional SSO parameter names. ParamNames 存储 SSO 约定参数名。
type ParamNames struct {
	Redirect             string // Redirect stores callback parameter name. Redirect 存储回调地址参数名。
	Ticket               string // Ticket stores ticket parameter name. Ticket 存储 Ticket 参数名。
	Code                 string // Code stores OAuth2 authorization code parameter name. Code 存储 OAuth2 授权码参数名。
	SessionID            string // SessionID stores remote session id parameter name. SessionID 存储远程会话 ID 参数名。
	Back                 string // Back stores back-url parameter name. Back 存储返回地址参数名。
	Mode                 string // Mode stores mode parameter name. Mode 存储模式参数名。
	Scope                string // Scope stores requested scope parameter name. Scope 存储请求授权范围参数名。
	LoginID              string // LoginID stores login id parameter name. LoginID 存储登录 ID 参数名。
	Client               string // Client stores client id parameter name. Client 存储客户端 ID 参数名。
	TokenName            string // TokenName stores token name parameter name. TokenName 存储 Token 名称参数名。
	TokenValue           string // TokenValue stores token value parameter name. TokenValue 存储 Token 值参数名。
	DeviceID             string // DeviceID stores device id parameter name. DeviceID 存储设备 ID 参数名。
	ClientSecret         string // ClientSecret stores client secret parameter name. ClientSecret 存储客户端密钥参数名。
	Callback             string // Callback stores logout callback parameter name. Callback 存储注销回调参数名。
	AutoLogout           string // AutoLogout stores auto logout marker parameter name. AutoLogout 存储自动注销标记参数名。
	Name                 string // Name stores username parameter name. Name 存储用户名参数名。
	Password             string // Password stores password parameter name. Password 存储密码参数名。
	Timestamp            string // Timestamp stores timestamp parameter name. Timestamp 存储时间戳参数名。
	Nonce                string // Nonce stores nonce parameter name. Nonce 存储随机值参数名。
	Sign                 string // Sign stores sign parameter name. Sign 存储签名参数名。
	RemainSessionTimeout string // RemainSessionTimeout stores session ttl parameter name. RemainSessionTimeout 存储会话剩余有效期参数名。
	RemainTokenTimeout   string // RemainTokenTimeout stores token ttl parameter name. RemainTokenTimeout 存储 Token 剩余有效期参数名。
	SingleDeviceIDLogout string // SingleDeviceIDLogout stores single device logout marker. SingleDeviceIDLogout 存储单设备注销标记。
}

// DefaultParamNames returns default SSO parameter names. DefaultParamNames 返回默认 SSO 参数名。
func DefaultParamNames() ParamNames {
	return ParamNames{
		Redirect:             "redirect",
		Ticket:               "ticket",
		Code:                 "code",
		SessionID:            "sessionId",
		Back:                 "back",
		Mode:                 "mode",
		Scope:                "scope",
		LoginID:              "loginId",
		Client:               "client",
		TokenName:            "tokenName",
		TokenValue:           "tokenValue",
		DeviceID:             "deviceId",
		ClientSecret:         "clientSecret",
		Callback:             "callback",
		AutoLogout:           "autoLogout",
		Name:                 "name",
		Password:             "pwd",
		Timestamp:            "timestamp",
		Nonce:                "nonce",
		Sign:                 "sign",
		RemainSessionTimeout: "remainSessionTimeout",
		RemainTokenTimeout:   "remainTokenTimeout",
		SingleDeviceIDLogout: "singleDeviceIdLogout",
	}
}

// TicketExchangeResult stores a client-facing ticket exchange result. TicketExchangeResult 存储面向客户端的 Ticket 交换结果。
type TicketExchangeResult struct {
	LoginID              string         `json:"loginId"`                        // LoginID stores the subject id. LoginID 存储登录主体 ID。
	TokenValue           string         `json:"tokenValue,omitempty"`           // TokenValue stores optional center token. TokenValue 存储可选中心 Token。
	DeviceID             string         `json:"deviceId,omitempty"`             // DeviceID stores login device id. DeviceID 存储登录设备 ID。
	RemainTokenTimeout   int64          `json:"remainTokenTimeout,omitempty"`   // RemainTokenTimeout stores token ttl seconds. RemainTokenTimeout 存储 Token 剩余秒数。
	RemainSessionTimeout int64          `json:"remainSessionTimeout,omitempty"` // RemainSessionTimeout stores session ttl seconds. RemainSessionTimeout 存储会话剩余秒数。
	CenterID             string         `json:"centerId,omitempty"`             // CenterID stores center-side subject id. CenterID 存储认证中心主体 ID。
	Scopes               []string       `json:"scopes,omitempty"`               // Scopes stores granted scopes. Scopes 存储授权范围。
	Extra                map[string]any `json:"extra,omitempty"`                // Extra stores extension data. Extra 存储扩展数据。
}

// CredentialInfo stores a normalized SSO credential inspection result. CredentialInfo 存储标准化 SSO 凭证检查结果。
type CredentialInfo struct {
	Active    bool           `json:"active"`              // Active stores whether credential is valid. Active 存储凭证是否有效。
	Mode      Mode           `json:"mode,omitempty"`      // Mode stores SSO mode. Mode 存储 SSO 模式。
	LoginID   string         `json:"loginId,omitempty"`   // LoginID stores subject id. LoginID 存储登录主体 ID。
	ClientID  string         `json:"clientId,omitempty"`  // ClientID stores client id. ClientID 存储客户端 ID。
	Scopes    []string       `json:"scopes,omitempty"`    // Scopes stores granted scopes. Scopes 存储授权范围。
	ExpiresIn int64          `json:"expiresIn,omitempty"` // ExpiresIn stores ttl seconds. ExpiresIn 存储有效秒数。
	Extra     map[string]any `json:"extra,omitempty"`     // Extra stores extension data. Extra 存储扩展数据。
}

// Response is the conventional SSO transport response. Response 是 SSO 传输层约定响应。
type Response struct {
	Code    int            `json:"code"`            // Code stores business status code. Code 存储业务状态码。
	Message string         `json:"message"`         // Message stores status message. Message 存储状态消息。
	Data    any            `json:"data,omitempty"`  // Data stores response payload. Data 存储响应数据。
	Extra   map[string]any `json:"extra,omitempty"` // Extra stores extension data. Extra 存储扩展数据。
}

// OKResponse creates a success response. OKResponse 创建成功响应。
func OKResponse(data any) Response {
	return Response{Code: 0, Message: ResultOK, Data: data}
}

// ErrorResponse creates an error response. ErrorResponse 创建错误响应。
func ErrorResponse(code int, message string) Response {
	return Response{Code: code, Message: message}
}

func joinPath(prefix, path string) string {
	prefix = strings.TrimRight(prefix, "/")
	path = "/" + strings.TrimLeft(path, "/")
	if prefix == "" {
		return path
	}
	return prefix + path
}

func replacePathPrefix(path, prefix string) string {
	path = "/" + strings.TrimLeft(path, "/")
	prefix = "/" + strings.Trim(prefix, "/")
	if prefix == "/" {
		prefix = ""
	}
	if path == "/sso" {
		return prefix
	}
	if strings.HasPrefix(path, "/sso/") {
		return prefix + strings.TrimPrefix(path, "/sso")
	}
	return path
}
