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
	// Mode stores preferred SSO mode. Mode 存储首选 SSO 模式。
	Mode Mode
	// ClientID stores current application id. ClientID 存储当前应用 ID。
	ClientID string
	// ClientSecret stores current application secret. ClientSecret 存储当前应用密钥。
	ClientSecret string
	// ServerURL stores an absolute HTTP(S) base URL without credentials, query, or fragment. ServerURL 存储不含凭据、查询串和片段的 HTTP(S) 绝对基础地址。
	ServerURL string
	// LoginURL stores current client login URL. LoginURL 存储当前客户端登录地址。
	LoginURL string
	// LogoutCallbackURL stores current client logout callback URL. LogoutCallbackURL 存储当前客户端注销回调地址。
	LogoutCallbackURL string
	// UseHTTPCheck forces HTTP exchange even when SharedStoreServer is configured. UseHTTPCheck 即使配置 SharedStoreServer 也强制通过 HTTP 兑换。
	UseHTTPCheck bool
	// SharedStoreServer borrows a local protocol service over the center's shared storage for Ticket/code exchange; nil preserves HTTP exchange. SharedStoreServer 借用连接中心共享存储的本地协议服务兑换 Ticket 或授权码，nil 保持 HTTP 兑换。
	SharedStoreServer *HTTPServer
	// EnableSLO enables single logout. EnableSLO 启用单点注销。
	EnableSLO bool
	// RegisterCallback sends the logout callback URL during authorization and credential exchange. RegisterCallback 在授权和凭证兑换时发送注销回调地址。
	RegisterCallback bool
	// CheckSign enables request signature checks. CheckSign 启用请求签名校验。
	CheckSign bool
	// ReplayProtection enables fresh signed requests and one-time logout callbacks; CheckSign is required. ReplayProtection 启用新鲜签名请求与一次性注销回调，必须同时启用 CheckSign。
	ReplayProtection *ReplayProtection
	// SecretKey stores request signing secret. SecretKey 存储请求签名密钥。
	SecretKey string
	// VerificationKey accepts one additional logout-callback signing key during rotation; outbound requests always use SecretKey. VerificationKey 在轮换期额外接受一把注销回调验签密钥；出站请求始终使用 SecretKey。
	VerificationKey string
	// LogoutCallbackMaxAge stores accepted logout callback clock skew. LogoutCallbackMaxAge 存储注销回调允许时间差。
	LogoutCallbackMaxAge time.Duration
	// HTTPClient overrides the default 10-second client; redirects require an explicit policy. HTTPClient 覆盖默认 10 秒超时客户端；重定向需要显式策略。
	HTTPClient *http.Client
	// MaxResponseBytes caps bytes read from each response body; nonpositive values use the 1 MiB default. MaxResponseBytes 限制每个响应体的读取字节数；非正值使用默认 1 MiB。
	MaxResponseBytes int64
	// RequestLimits bounds incoming logout callbacks and BrowserClient login queries. RequestLimits 限制入站注销回调及 BrowserClient 登录查询串。
	RequestLimits RequestLimits
	// Endpoints stores server/client protocol paths. Endpoints 存储服务端/客户端协议路径。
	Endpoints Endpoints
	// Params stores protocol parameter names. Params 存储协议参数名。
	Params ParamNames
}

// DefaultClientConfig returns default SSO client config. DefaultClientConfig 返回默认 SSO 客户端配置。
func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		Mode:                 ModeTicket,
		EnableSLO:            true,
		CheckSign:            true,
		LogoutCallbackMaxAge: DefaultLogoutCallbackMaxAge,
		MaxResponseBytes:     DefaultClientMaxResponseBytes,
		RequestLimits:        DefaultRequestLimits(),
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
	// CodeVerifier supplies the original PKCE proof through POST form data. CodeVerifier 通过 POST 表单提供原始 PKCE 兑换证明。
	CodeVerifier string
	// Mode stores credential mode. Mode 存储凭证模式。
	Mode Mode
	// Ticket stores one-time ticket. Ticket 存储一次性 Ticket。
	Ticket string
	// TokenValue stores shared token. TokenValue 存储共享 Token。
	TokenValue string
	// SessionID stores remote session id. SessionID 存储远程会话 ID。
	SessionID string
	// Code stores OAuth2 authorization code. Code 存储 OAuth2 授权码。
	Code string
	// RedirectURI stores callback URI. RedirectURI 存储回调地址。
	RedirectURI string
}

// LogoutCallback stores parsed single-logout callback data. LogoutCallback 存储解析后的单点注销回调数据。
type LogoutCallback struct {
	// LoginID stores subject id at the login center. LoginID 存储认证中心登录主体 ID。
	LoginID string
	// ClientID stores callback target client id. ClientID 存储回调目标客户端 ID。
	ClientID string
	// Timestamp stores callback timestamp. Timestamp 存储回调时间戳。
	Timestamp string
	// Values stores raw callback form values. Values 存储原始回调表单参数。
	Values url.Values
}

// LogoutCallbackFunc handles a verified single-logout callback. LogoutCallbackFunc 处理已校验的单点注销回调。
type LogoutCallbackFunc func(r *http.Request, callback LogoutCallback) error

// NewClientApp creates an SSO client helper. NewClientApp 创建 SSO 客户端辅助对象。
func NewClientApp(cfg ClientConfig) *ClientApp {
	defaults := DefaultClientConfig()
	cfg.RequestLimits = normalizeRequestLimits(cfg.RequestLimits)
	if cfg.Mode == "" {
		cfg.Mode = defaults.Mode
	}
	cfg.Endpoints = normalizeEndpoints(cfg.Endpoints)
	cfg.Params = normalizeParamNames(cfg.Params)
	if cfg.LogoutCallbackMaxAge <= 0 {
		cfg.LogoutCallbackMaxAge = defaults.LogoutCallbackMaxAge
	}
	if cfg.MaxResponseBytes <= 0 {
		cfg.MaxResponseBytes = defaults.MaxResponseBytes
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
	if err := validatePKCEParameters(values, c.config.Params, true); err != nil {
		return "", err
	}
	challenge, method := values.Get(c.config.Params.CodeChallenge), values.Get(c.config.Params.CodeChallengeMethod)
	if err := validatePKCEChallenge(challenge, method, false); err != nil {
		return "", err
	}
	if challenge != "" && c.config.Mode != ModeTicket && c.config.Mode != ModeOAuth2 {
		return "", ErrModeUnsupported
	}
	values.Set(c.config.Params.Redirect, redirectURI)
	values.Set(c.config.Params.Mode, string(c.config.Mode))
	if c.config.ClientID != "" {
		values.Set(c.config.Params.Client, c.config.ClientID)
	}
	if c.config.RegisterCallback && c.config.LogoutCallbackURL != "" {
		values.Set(c.config.Params.Callback, c.config.LogoutCallbackURL)
	}
	return c.buildServerURL(c.config.Endpoints.Authorize, ActionAuthorize, values)
}

// AuthURLWithPKCE sends only the verifier's S256 challenge; retain the verifier privately for exchange. AuthURLWithPKCE 仅发送 verifier 的 S256 挑战，调用方应私下保存 verifier 用于兑换。
func (c *ClientApp) AuthURLWithPKCE(redirectURI, verifier string, extra url.Values) (string, error) {
	if c == nil {
		return "", ErrServerNotInitialized
	}
	challenge, err := PKCEChallenge(verifier)
	if err != nil {
		return "", err
	}
	values := cloneValues(extra)
	values.Set(c.config.Params.CodeChallenge, challenge)
	values.Set(c.config.Params.CodeChallengeMethod, PKCEMethodS256)
	return c.AuthURL(redirectURI, values)
}

// ExchangeTicketURL returns SSO server ticket exchange URL. ExchangeTicketURL 返回 SSO 服务端 Ticket 交换地址。
func (c *ClientApp) ExchangeTicketURL(ticket string, extra url.Values) (string, error) {
	return c.ExchangeTicketURLWithRedirect(ticket, "", extra)
}

// ExchangeTicketURLWithRedirect returns a public-client ticket URL without embedding client secrets. ExchangeTicketURLWithRedirect 返回不嵌入客户端密钥的公开客户端 Ticket 地址。
//
// Deprecated: use ExchangeTicket so confidential credentials are sent with POST. Deprecated：请使用 ExchangeTicket，通过 POST 发送机密凭证。
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
	return c.buildServerURL(c.config.Endpoints.Token, ActionToken, values)
}

// SignoutURL returns an SSO server signout URL. The loginID argument is a legacy hint; the server must resolve and verify the trusted logout subject. SignoutURL 返回 SSO 服务端单点注销地址。loginID 参数仅为兼容保留的提示值，服务端必须解析并校验可信注销主体。
func (c *ClientApp) SignoutURL(loginID string, extra url.Values) (string, error) {
	if c == nil {
		return "", ErrServerNotInitialized
	}
	values := cloneValues(extra)
	values.Set(c.config.Params.LoginID, loginID)
	if c.config.ClientID != "" {
		values.Set(c.config.Params.Client, c.config.ClientID)
	}
	return c.buildServerURL(c.config.Endpoints.Logout, ActionLogout, values)
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
	if c.config.RegisterCallback && c.config.LogoutCallbackURL != "" {
		values.Set(c.config.Params.Callback, c.config.LogoutCallbackURL)
	}
	var result TicketExchangeResult
	if c.config.SharedStoreServer != nil && !c.config.UseHTTPCheck {
		shared, err := c.exchangeSharedStore(ctx, values)
		if err != nil {
			return nil, err
		}
		result = *shared
	} else {
		if err := c.postForm(ctx, c.config.Endpoints.Token, ActionToken, values, &result); err != nil {
			return nil, err
		}
	}
	if result.LoginID == "" {
		return nil, errors.New("sso exchange response is missing loginId")
	}
	if result.RemainTokenTimeout < -1 || result.RemainSessionTimeout < -1 {
		return nil, errors.New("sso exchange response has an invalid session timeout")
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
	if err := c.postForm(ctx, c.config.Endpoints.Introspect, ActionIntrospect, values, &info); err != nil {
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
	if err := c.postForm(ctx, c.config.Endpoints.UserInfo, ActionUserInfo, values, &info); err != nil {
		return nil, err
	}
	if info.LoginID == "" {
		return nil, errors.New("sso userinfo response is missing loginId")
	}
	return &info, nil
}

// Revoke revokes a credential. Revoke 撤销凭证。
func (c *ClientApp) Revoke(ctx context.Context, req CredentialRequest) error {
	if c == nil {
		return ErrServerNotInitialized
	}
	return c.postForm(ctx, c.config.Endpoints.Revoke, ActionRevoke, c.credentialValues(req), nil)
}

// VerifyLogoutCallback verifies and parses a single-logout callback request. VerifyLogoutCallback 校验并解析单点注销回调请求。
func (c *ClientApp) VerifyLogoutCallback(r *http.Request) (*LogoutCallback, error) {
	if c == nil {
		return nil, ErrServerNotInitialized
	}
	if err := c.config.ReplayProtection.validateConfig(c.config.CheckSign, c.config.Params); err != nil {
		return nil, err
	}
	if c.config.ReplayProtection != nil && c.config.ClientID == "" {
		return nil, ErrClientOrClientIDEmpty
	}
	if r == nil {
		return nil, errors.New("nil request")
	}
	if err := r.Context().Err(); err != nil {
		return nil, err
	}
	if r.Method != http.MethodPost {
		return nil, ErrMethodNotAllowed
	}
	if err := parseRequestForm(r, c.config.RequestLimits); err != nil {
		return nil, err
	}
	values := cloneValues(r.Form)
	if !distinctParamNames(c.config.Params) {
		return nil, ErrInvalidRequest
	}
	if err := validateScalarParameters(values, c.config.Params); err != nil {
		return nil, err
	}
	if c.config.CheckSign {
		if c.config.SecretKey == "" {
			return nil, ErrSignSecretRequired
		}
		if !verifyProtocolSignature(values, c.config.Params, c.config.SecretKey, c.config.VerificationKey) {
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
	if c.config.ClientID != "" {
		if callback.ClientID == "" || callback.ClientID != c.config.ClientID {
			return nil, ErrClientMismatch
		}
	}
	if c.config.ReplayProtection != nil {
		if err := c.config.ReplayProtection.claim(r.Context(), values, c.config.Params, ActionLogoutCallback); err != nil {
			return nil, err
		}
	} else {
		if err := c.verifyLogoutCallbackTime(callback.Timestamp); err != nil {
			return nil, err
		}
	}
	if err := r.Context().Err(); err != nil {
		return nil, err
	}
	return callback, nil
}

// LogoutCallbackHandler returns a standard HTTP handler for single logout. LogoutCallbackHandler 返回标准单点注销 HTTP 处理器。
func (c *ClientApp) LogoutCallbackHandler(fn LogoutCallbackFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Pragma", "no-cache")

		callback, err := c.VerifyLogoutCallback(r)
		if err != nil {
			// Keep diagnostics inside the direct API, including wrapped storage errors. 将诊断信息保留在直接调用 API 内，包括包装的存储错误。
			status := statusFromError(err)
			http.Error(w, http.StatusText(status), status)
			return
		}
		if fn != nil {
			if err = fn(r, *callback); err != nil {
				http.Error(w, "logout callback failed", http.StatusInternalServerError)
				return
			}
		}
		if err := r.Context().Err(); err != nil {
			http.Error(w, "logout callback canceled", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, OKResponse(map[string]string{"result": ResultOK}))
	}
}

// buildServerURL builds and optionally signs an SSO server URL. buildServerURL 构建并按需签名 SSO 服务端 URL。
func (c *ClientApp) buildServerURL(path, action string, values url.Values) (string, error) {
	if _, exists := values[c.config.Params.CodeVerifier]; exists {
		return "", ErrInvalidPKCE
	}
	base, err := joinURL(c.config.ServerURL, path)
	if err != nil {
		return "", err
	}
	values, err = c.signRequest(values, action)
	if err != nil {
		return "", err
	}
	base.RawQuery = values.Encode()
	return base.String(), nil
}

// signRequest stamps fresh metadata before signing each protocol operation. signRequest 在签名每个协议操作前写入新鲜元数据。
func (c *ClientApp) signRequest(values url.Values, action string) (url.Values, error) {
	if !distinctParamNames(c.config.Params) {
		return nil, ErrInvalidRequest
	}
	if err := c.config.ReplayProtection.validateConfig(c.config.CheckSign, c.config.Params); err != nil {
		return nil, err
	}
	if c.config.ReplayProtection != nil {
		if c.config.ClientID == "" {
			return nil, ErrClientOrClientIDEmpty
		}
		values = c.config.ReplayProtection.stamp(values, c.config.Params, action)
	}
	if err := validateScalarParameters(values, c.config.Params); err != nil {
		return nil, err
	}
	if err := validateCredentialParameters(values, c.config.Params, action); err != nil {
		return nil, err
	}
	if c.config.CheckSign {
		if c.config.SecretKey == "" {
			return nil, ErrSignSecretRequired
		}
		values = NewSignerWithParams(c.config.SecretKey, c.config.Params).AttachSign(values)
	}
	return values, nil
}

// verifyLogoutCallbackTime validates logout callback freshness. verifyLogoutCallbackTime 校验登出回调时效性。
func (c *ClientApp) verifyLogoutCallbackTime(value string) error {
	if c.config.LogoutCallbackMaxAge <= 0 {
		return nil
	}
	if value == "" {
		return ErrCallbackExpired
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

// credentialValues converts a credential request to form values. credentialValues 将凭证请求转换为表单参数。
func (c *ClientApp) credentialValues(req CredentialRequest) url.Values {
	values := url.Values{}
	if req.CodeVerifier != "" {
		values.Set(c.config.Params.CodeVerifier, req.CodeVerifier)
	}
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

// postForm posts a form and decodes the protocol response. postForm 提交表单并解码协议响应。
func (c *ClientApp) postForm(ctx context.Context, path, action string, values url.Values, out any) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	values, err := c.signRequest(values, action)
	if err != nil {
		return err
	}
	target, err := joinURL(c.config.ServerURL, path)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, target.String(), strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := c.config.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	// Do not implicitly forward credential forms or mutate the caller's client. 不隐式转发凭证表单，也不修改调用方客户端。
	if client.CheckRedirect == nil {
		clientCopy := *client
		clientCopy.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		}
		client = &clientCopy
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	// Bound actual body bytes, including bodies decoded by the transport; Content-Length is not authoritative. 限制实际响应体字节，包括传输层解压后的内容，不依赖 Content-Length。
	limit := c.config.MaxResponseBytes
	payload, err := io.ReadAll(io.LimitReader(response.Body, limit))
	if err != nil {
		return err
	}
	if int64(len(payload)) == limit {
		// Probe one extra byte without adding to the configured limit, avoiding integer overflow. 单独探测一个额外字节，不对配置上限做加法，避免整数溢出。
		var extra [1]byte
		if _, err := io.ReadFull(response.Body, extra[:]); err == nil {
			return fmt.Errorf("sso response exceeds %d bytes", limit)
		} else if err != io.EOF {
			return err
		}
	}

	// Custom transports and readers may ignore cancellation; never accept their late success. 自定义传输和读取器可能忽略取消，不接受其延迟返回的成功结果。
	if err := ctx.Err(); err != nil {
		return err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("sso request failed with status %d: %s", response.StatusCode, string(payload))
	}

	// Decode data only once so integer identities and metadata retain precision. 数据只解码一次，保留整数标识及元数据精度。
	var body struct {
		Code    *int            `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err = json.Unmarshal(payload, &body); err != nil {
		return err
	}
	if body.Code == nil {
		return errors.New("sso response is missing code")
	}
	if *body.Code != 0 {
		return fmt.Errorf("sso response failed with code %d: %s", *body.Code, body.Message)
	}
	if out == nil {
		return nil
	}
	if len(body.Data) == 0 || bytes.Equal(bytes.TrimSpace(body.Data), []byte("null")) {
		return errors.New("sso response is missing data")
	}
	decoder := json.NewDecoder(bytes.NewReader(body.Data))
	decoder.UseNumber()
	return decoder.Decode(out)
}

// joinURL validates and joins a base URL with a protocol path, retaining escaped prefixes. joinURL 校验并连接基础地址和协议路径，保留转义的路径前缀。
func joinURL(baseURL, path string) (*url.URL, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	if (base.Scheme != "http" && base.Scheme != "https") || base.Hostname() == "" || base.User != nil || base.RawQuery != "" || base.ForceQuery || strings.Contains(baseURL, "#") {
		return nil, errors.New("sso server URL must be absolute HTTP(S) without credentials, query, or fragment")
	}
	endpoint, err := url.Parse(path)
	if err != nil {
		return nil, err
	}
	if endpoint.IsAbs() || endpoint.Host != "" || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.ForceQuery || strings.Contains(path, "#") {
		return nil, errors.New("sso endpoint must be a path without query or fragment")
	}

	// Avoid resolving a leading slash against the host and losing the gateway prefix. 避免首斜杠按根路径解析而丢失网关前缀。
	rawPath := strings.TrimRight(base.EscapedPath(), "/") + "/" + strings.TrimLeft(endpoint.EscapedPath(), "/")
	base.Path, err = url.PathUnescape(rawPath)
	if err != nil {
		return nil, err
	}
	base.RawPath = rawPath
	return base, nil
}
