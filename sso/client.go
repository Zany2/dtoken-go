// @Author daixk 2026/05/28
package sso

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ClientConfig defines SSO client-side integration config. ClientConfig 定义 SSO 子应用接入配置。
type ClientConfig struct {
	Mode                 Mode          // Mode stores preferred SSO mode. Mode 存储首选 SSO 模式。
	ClientID             string        // ClientID stores current application id. ClientID 存储当前应用 ID。
	ClientSecret         string        // ClientSecret stores current application secret. ClientSecret 存储当前应用密钥。
	ServerURL            string        // ServerURL stores SSO server base URL. ServerURL 存储 SSO 服务端根地址。
	LoginURL             string        // LoginURL stores current client login URL. LoginURL 存储当前客户端登录地址。
	LogoutCallbackURL    string        // LogoutCallbackURL stores current client logout callback URL. LogoutCallbackURL 存储当前客户端注销回调地址。
	UseHTTPCheck         bool          // UseHTTPCheck enables remote ticket validation mode. UseHTTPCheck 启用远程 Ticket 校验模式。
	EnableSLO            bool          // EnableSLO enables single logout. EnableSLO 启用单点注销。
	RegisterCallback     bool          // RegisterCallback sends logout callback URL during login. RegisterCallback 登录时注册注销回调地址。
	CheckSign            bool          // CheckSign enables request signature checks. CheckSign 启用请求签名校验。
	SecretKey            string        // SecretKey stores request signing secret. SecretKey 存储请求签名密钥。
	LogoutCallbackMaxAge time.Duration // LogoutCallbackMaxAge stores accepted logout callback clock skew. LogoutCallbackMaxAge 存储注销回调允许时间差。
	HTTPClient           *http.Client  // HTTPClient stores optional transport client. HTTPClient 存储可选 HTTP 客户端。
	Endpoints            Endpoints     // Endpoints stores server/client protocol paths. Endpoints 存储服务端/客户端协议路径。
	Params               ParamNames    // Params stores protocol parameter names. Params 存储协议参数名。
}

// DefaultClientConfig returns default SSO client config. DefaultClientConfig 返回默认 SSO 客户端配置。
func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		Mode:                 ModeTicket,
		EnableSLO:            true,
		CheckSign:            true,
		LogoutCallbackMaxAge: DefaultLogoutCallbackMaxAge,
		Endpoints:            DefaultEndpoints(),
		Params:               DefaultParamNames(),
	}
}

// ClientApp helps a business application build SSO URLs. ClientApp 帮助业务应用构建 SSO URL。
type ClientApp struct {
	config ClientConfig // config stores client-side config. config 存储客户端配置。
}

// CredentialRequest defines a client-side SSO credential request. CredentialRequest 定义客户端侧 SSO 凭证请求。
type CredentialRequest struct {
	Mode        Mode   // Mode stores credential mode. Mode 存储凭证模式。
	Ticket      string // Ticket stores one-time ticket. Ticket 存储一次性 Ticket。
	TokenValue  string // TokenValue stores shared token. TokenValue 存储共享 Token。
	SessionID   string // SessionID stores remote session id. SessionID 存储远程会话 ID。
	Code        string // Code stores OAuth2 authorization code. Code 存储 OAuth2 授权码。
	RedirectURI string // RedirectURI stores callback URI. RedirectURI 存储回调地址。
}

// LogoutCallback stores parsed single-logout callback data. LogoutCallback 存储解析后的单点注销回调数据。
type LogoutCallback struct {
	LoginID   string     // LoginID stores subject id at the login center. LoginID 存储认证中心登录主体 ID。
	ClientID  string     // ClientID stores callback target client id. ClientID 存储回调目标客户端 ID。
	Timestamp string     // Timestamp stores callback timestamp. Timestamp 存储回调时间戳。
	Values    url.Values // Values stores raw callback form values. Values 存储原始回调表单参数。
}

// LogoutCallbackFunc handles a verified single-logout callback. LogoutCallbackFunc 处理已校验的单点注销回调。
type LogoutCallbackFunc func(r *http.Request, callback LogoutCallback) error

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
	if cfg.LogoutCallbackMaxAge <= 0 {
		cfg.LogoutCallbackMaxAge = defaults.LogoutCallbackMaxAge
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

// ExchangeTicket exchanges a ticket for login subject information. ExchangeTicket 使用 Ticket 换取登录主体信息。
func (c *ClientApp) ExchangeTicket(ctx context.Context, ticket, redirectURI string) (*TicketExchangeResult, error) {
	return c.ExchangeCredential(ctx, CredentialRequest{
		Mode:        ModeTicket,
		Ticket:      ticket,
		RedirectURI: redirectURI,
	})
}

// ExchangeOAuth2Code exchanges an OAuth2 code for login subject information. ExchangeOAuth2Code 使用 OAuth2 授权码换取登录主体信息。
func (c *ClientApp) ExchangeOAuth2Code(ctx context.Context, code, redirectURI string) (*TicketExchangeResult, error) {
	return c.ExchangeCredential(ctx, CredentialRequest{
		Mode:        ModeOAuth2,
		Code:        code,
		RedirectURI: redirectURI,
	})
}

// ExchangeCredential exchanges a supported one-time credential. ExchangeCredential 换取支持的一次性凭证。
func (c *ClientApp) ExchangeCredential(ctx context.Context, req CredentialRequest) (*TicketExchangeResult, error) {
	if c == nil {
		return nil, ErrServerNotInitialized
	}
	values := c.credentialValues(req)
	var result TicketExchangeResult
	if err := c.postForm(ctx, c.config.Endpoints.Token, values, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Introspect checks whether a credential is active. Introspect 检查凭证是否有效。
func (c *ClientApp) Introspect(ctx context.Context, req CredentialRequest) (*CredentialInfo, error) {
	if c == nil {
		return nil, ErrServerNotInitialized
	}
	values := c.credentialValues(req)
	var info CredentialInfo
	if err := c.postForm(ctx, c.config.Endpoints.Introspect, values, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// UserInfo reads user information from a valid credential. UserInfo 使用有效凭证读取用户信息。
func (c *ClientApp) UserInfo(ctx context.Context, req CredentialRequest) (*CredentialInfo, error) {
	if c == nil {
		return nil, ErrServerNotInitialized
	}
	values := c.credentialValues(req)
	var info CredentialInfo
	if err := c.postForm(ctx, c.config.Endpoints.UserInfo, values, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// Revoke revokes a credential. Revoke 撤销凭证。
func (c *ClientApp) Revoke(ctx context.Context, req CredentialRequest) error {
	if c == nil {
		return ErrServerNotInitialized
	}
	return c.postForm(ctx, c.config.Endpoints.Revoke, c.credentialValues(req), nil)
}

// VerifyLogoutCallback verifies and parses a single-logout callback request. VerifyLogoutCallback 校验并解析单点注销回调请求。
func (c *ClientApp) VerifyLogoutCallback(r *http.Request) (*LogoutCallback, error) {
	if c == nil {
		return nil, ErrServerNotInitialized
	}
	if r == nil {
		return nil, errors.New("nil request")
	}
	if r.Method != http.MethodPost {
		return nil, ErrMethodNotAllowed
	}
	if err := r.ParseForm(); err != nil {
		return nil, err
	}
	values := cloneValues(r.Form)
	if c.config.CheckSign && c.config.SecretKey != "" {
		if !NewSignerWithParams(c.config.SecretKey, c.config.Params).Verify(values) {
			return nil, ErrInvalidSign
		}
	}
	callback := &LogoutCallback{
		LoginID:   values.Get(c.config.Params.LoginID),
		ClientID:  values.Get(c.config.Params.Client),
		Timestamp: values.Get(c.config.Params.Timestamp),
		Values:    values,
	}
	if callback.LoginID == "" {
		return nil, ErrUserIDEmpty
	}
	if c.config.ClientID != "" && callback.ClientID != "" && callback.ClientID != c.config.ClientID {
		return nil, ErrClientMismatch
	}
	if err := c.verifyLogoutCallbackTime(callback.Timestamp); err != nil {
		return nil, err
	}
	return callback, nil
}

// LogoutCallbackHandler returns a standard HTTP handler for single logout. LogoutCallbackHandler 返回标准单点注销 HTTP 处理器。
func (c *ClientApp) LogoutCallbackHandler(fn LogoutCallbackFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		callback, err := c.VerifyLogoutCallback(r)
		if err != nil {
			http.Error(w, err.Error(), statusFromError(err))
			return
		}
		if fn != nil {
			if err = fn(r, *callback); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		writeJSON(w, http.StatusOK, OKResponse(map[string]string{"result": ResultOK}))
	}
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

func (c *ClientApp) verifyLogoutCallbackTime(value string) error {
	if value == "" || c.config.LogoutCallbackMaxAge <= 0 {
		return nil
	}
	timestamp, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return ErrCallbackExpired
	}
	now := time.Now()
	if timestamp.After(now.Add(c.config.LogoutCallbackMaxAge)) || timestamp.Before(now.Add(-c.config.LogoutCallbackMaxAge)) {
		return ErrCallbackExpired
	}
	return nil
}

func (c *ClientApp) credentialValues(req CredentialRequest) url.Values {
	values := url.Values{}
	mode := req.Mode
	if mode == "" {
		mode = c.config.Mode
	}
	values.Set(c.config.Params.Mode, string(mode))
	if c.config.ClientID != "" {
		values.Set(c.config.Params.Client, c.config.ClientID)
	}
	if c.config.ClientSecret != "" {
		values.Set(c.config.Params.ClientSecret, c.config.ClientSecret)
	}
	if req.Ticket != "" {
		values.Set(c.config.Params.Ticket, req.Ticket)
	}
	if req.TokenValue != "" {
		values.Set(c.config.Params.TokenValue, req.TokenValue)
	}
	if req.SessionID != "" {
		values.Set(c.config.Params.SessionID, req.SessionID)
	}
	if req.Code != "" {
		values.Set(c.config.Params.Code, req.Code)
	}
	if req.RedirectURI != "" {
		values.Set(c.config.Params.Redirect, req.RedirectURI)
	}
	return values
}

func (c *ClientApp) postForm(ctx context.Context, path string, values url.Values, out any) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if c.config.CheckSign && c.config.SecretKey != "" {
		values = NewSignerWithParams(c.config.SecretKey, c.config.Params).AttachSign(values)
	}
	target := joinURL(c.config.ServerURL, path)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, target, strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := c.config.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	payload, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("sso request failed with status %d: %s", response.StatusCode, string(payload))
	}

	var body Response
	if err = json.Unmarshal(payload, &body); err != nil {
		return err
	}
	if body.Code != 0 {
		return errors.New(body.Message)
	}
	if out == nil || body.Data == nil {
		return nil
	}
	rawData, err := json.Marshal(body.Data)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(rawData))
	decoder.UseNumber()
	return decoder.Decode(out)
}

func joinURL(baseURL, path string) string {
	baseURL = strings.TrimRight(baseURL, "/")
	path = "/" + strings.TrimLeft(path, "/")
	return baseURL + path
}
