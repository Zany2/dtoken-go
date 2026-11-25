// @Author daixk 2026/05/29
package sso

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"
)

// LoginIDResolver resolves the current SSO-center login id from an HTTP request. LoginIDResolver 从 HTTP 请求解析当前 SSO 中心登录 ID。
type LoginIDResolver func(r *http.Request) (string, bool)

// CenterLogoutFunc revokes the resolved subject's center session and must tolerate retries. CenterLogoutFunc 撤销已解析主体的中心会话，必须允许重试。
type CenterLogoutFunc func(r *http.Request, loginID string) error

// ExchangeResultFunc enriches a consumed credential's result without changing its identity or scopes. ExchangeResultFunc 补充已消费凭证的返回信息，不得改变身份或授权范围。
type ExchangeResultFunc func(r *http.Request, clientID string, result *TicketExchangeResult) error

// AuthorizationExtraFunc resolves trusted metadata to bind to issued credentials. AuthorizationExtraFunc 解析要绑定到签发凭证的可信元数据。
type AuthorizationExtraFunc func(r *http.Request, clientID, loginID string) (map[string]any, error)

// CredentialActiveFunc checks application state for an otherwise active credential; treat metadata as read-only and return errors for lookup failures. CredentialActiveFunc 检查凭证有效时的应用状态；元数据应只读，查询故障应返回错误。
type CredentialActiveFunc func(r *http.Request, credential CredentialInfo) (bool, error)

// HTTPOptions defines standalone HTTP protocol behavior for SSO. HTTPOptions 定义独立 HTTP 协议层行为。
type HTTPOptions struct {
	// ServerOptions carries protocol-level server behavior. ServerOptions 包含协议层服务端行为配置。
	ServerOptions
	// RequestLimits bounds inbound protocol queries and form bodies. RequestLimits 限制入站协议查询串与表单体。
	RequestLimits RequestLimits
	// ReplayProtection enforces one-time signed requests and stamps logout callbacks; CheckSign is required. ReplayProtection 强制一次性签名请求并为注销回调添加元数据，必须同时启用 CheckSign。
	ReplayProtection *ReplayProtection
	// LoginIDResolver resolves current center login id. LoginIDResolver 解析当前中心登录 ID。
	LoginIDResolver LoginIDResolver
	// CenterLogout revokes the supplied account's center sessions; browser logout calls it after notifications, LogoutAccount before them. CenterLogout 撤销指定账号的中心会话；浏览器退出在通知后调用，LogoutAccount 在通知前调用。
	CenterLogout CenterLogoutFunc
	// OnLogoutResult observes the final outcome for an authenticated logout; treat it as read-only and do not write the response. OnLogoutResult 观察已认证注销的最终结果，应只读且不得写响应。
	OnLogoutResult LogoutResultFunc
	// OnHTTPError observes each rejected center HTTP response once; it must return promptly and must not expose private causes. OnHTTPError 对每次中心 HTTP 拒绝响应观察一次，必须及时返回且不得公开私有原因。
	OnHTTPError HTTPErrorFunc
	// EnrichExchangeResult adds center-session metadata after consumption and before callback registration. EnrichExchangeResult 在凭证消费后、回调登记前补充中心会话信息。
	EnrichExchangeResult ExchangeResultFunc
	// AuthorizationExtra binds application-owned center-session metadata during HTTP authorization. AuthorizationExtra 在 HTTP 授权时绑定应用管理的中心会话信息。
	AuthorizationExtra AuthorizationExtraFunc
	// AuthorizationPolicy evaluates current access at issuance, exchange, and inspection; nil preserves existing grants. AuthorizationPolicy 在签发、兑换与检查时评估当前访问权限，nil 保持已有授权行为。
	AuthorizationPolicy AuthorizationPolicyFunc
	// CheckCredentialActive checks all four modes during HTTP introspection and userinfo, after client and credential validation. CheckCredentialActive 在客户端与凭证校验后，为 HTTP introspect 和 userinfo 的四种模式检查应用状态。
	CheckCredentialActive CredentialActiveFunc
	// LoginPageURL stores the fallback login page URL. LoginPageURL 存储未登录时跳转的登录页地址。
	LoginPageURL string
	// Cookie stores optional shared-cookie settings. Cookie 存储可选共享 Cookie 配置。
	Cookie CookieOptions
}

// DefaultHTTPOptions returns default standalone HTTP options. DefaultHTTPOptions 返回默认独立 HTTP 选项。
func DefaultHTTPOptions() HTTPOptions {
	return HTTPOptions{
		ServerOptions: DefaultServerOptions(),
		RequestLimits: DefaultRequestLimits(),
		Cookie:        DefaultCookieOptions(),
	}
}

// HTTPServer exposes SSO routes by using net/http only. HTTPServer 使用标准库 net/http 暴露 SSO 路由。
type HTTPServer struct {
	server          *Server
	options         HTTPOptions
	initErr         error         // initErr preserves constructor registration failures. initErr 保存构造期间的注册错误。
	clientSessionMu sync.Mutex    // clientSessionMu serializes session-limit checks and registrations. clientSessionMu 串行化会话上限检查和注册。
	logoutSlots     chan struct{} // logoutSlots bounds deliveries across concurrent requests. logoutSlots 限制并发请求之间的通知总数。
}

// NewHTTPServer creates a standalone HTTP SSO handler that retains client registration failures and returns HTTP 500 for them. NewHTTPServer 创建独立 HTTP SSO 处理器，保留客户端注册错误并据此返回 HTTP 500。
func NewHTTPServer(server *Server, options HTTPOptions) *HTTPServer {
	defaults := DefaultHTTPOptions()
	options.RequestLimits = normalizeRequestLimits(options.RequestLimits)
	options.ServerOptions.Endpoints = normalizeEndpoints(options.ServerOptions.Endpoints)
	options.ServerOptions.Params = normalizeParamNames(options.ServerOptions.Params)
	if options.ServerOptions.Mode == "" {
		options.ServerOptions.Mode = defaults.ServerOptions.Mode
	}
	if options.ServerOptions.LogoutCallbackTimeout <= 0 {
		options.ServerOptions.LogoutCallbackTimeout = defaults.ServerOptions.LogoutCallbackTimeout
	}
	if options.LogoutCallbackConcurrency <= 0 {
		options.LogoutCallbackConcurrency = defaults.LogoutCallbackConcurrency
	}
	if options.LogoutCallbackMaxAttempts <= 0 {
		options.LogoutCallbackMaxAttempts = defaults.LogoutCallbackMaxAttempts
	}
	if options.LogoutCallbackRetryInterval <= 0 {
		options.LogoutCallbackRetryInterval = defaults.LogoutCallbackRetryInterval
	}
	if options.ServerOptions.MaxRegisteredClient == 0 {
		options.ServerOptions.MaxRegisteredClient = defaults.ServerOptions.MaxRegisteredClient
	}
	options.Cookie = normalizeCookieOptions(options.Cookie)
	if options.Cookie.SecretKey == "" {
		options.Cookie.SecretKey = options.ServerOptions.SecretKey
	}
	if options.LoginIDResolver == nil {
		options.LoginIDResolver = LoginIDFromCookie(options.Cookie)
	}
	handler := &HTTPServer{server: server, options: options}
	if err := options.Endpoints.Validate(); err != nil {
		handler.initErr = err
		return handler
	}
	if !distinctParamNames(options.Params) {
		handler.initErr = ErrInvalidRequest
		return handler
	}
	if err := options.ReplayProtection.validateConfig(options.CheckSign, options.Params); err != nil {
		handler.initErr = err
		return handler
	}
	handler.logoutSlots = make(chan struct{}, options.LogoutCallbackConcurrency)
	if server == nil || server.storage == nil || server.serializer == nil {
		return handler
	}
	if err := server.RegisterClients(options.ServerOptions.Clients); err != nil {
		handler.initErr = fmt.Errorf("%w: %w", ErrServerNotInitialized, err)
		return handler
	}
	if options.ServerOptions.AllowAnonymousClient {
		_, err := server.GetClient(ClientAnonymous)
		if errors.Is(err, ErrClientNotFound) {
			err = server.RegisterClient(&Client{
				ClientID:     ClientAnonymous,
				Name:         "Anonymous Client",
				RedirectURIs: append([]string(nil), options.ServerOptions.AllowURLs...),
				Modes:        []Mode{ModeTicket},
			})
		}
		if err != nil {
			handler.initErr = fmt.Errorf("%w: %w", ErrServerNotInitialized, err)
		}
	}
	return handler
}

// initializationError prevents partially initialized handlers from serving requests. initializationError 阻止未完成初始化的处理器接受请求。
func (h *HTTPServer) initializationError() error {
	if h == nil || h.server == nil || h.server.storage == nil || h.server.serializer == nil {
		return ErrServerNotInitialized
	}
	return h.initErr
}

// Register registers SSO routes into a ServeMux. Register 将 SSO 路由注册到 ServeMux。
func (h *HTTPServer) Register(mux *http.ServeMux) {
	if h == nil || mux == nil || h.server == nil || h.server.storage == nil || h.server.serializer == nil {
		return
	}
	if err := h.options.Endpoints.Validate(); err != nil {
		return
	}
	endpoints := h.options.ServerOptions.Endpoints
	mux.HandleFunc(endpoints.Authorize, h.HandleAuthorize)
	mux.HandleFunc(endpoints.Token, h.HandleToken)
	mux.HandleFunc(endpoints.Introspect, h.HandleIntrospect)
	mux.HandleFunc(endpoints.UserInfo, h.HandleUserInfo)
	mux.HandleFunc(endpoints.Revoke, h.HandleRevoke)
	mux.HandleFunc(endpoints.Logout, h.HandleLogout)
}

// Handler returns a ServeMux with SSO routes registered. Handler 返回已注册 SSO 路由的 ServeMux。
func (h *HTTPServer) Handler() http.Handler {
	if err := h.initializationError(); err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h.writeHTTPError(w, r, http.StatusInternalServerError, err)
		})
	}
	mux := http.NewServeMux()
	h.Register(mux)
	return mux
}

// HandleAuthorize handles redirect-based SSO credential issuing. HandleAuthorize 处理基于重定向的 SSO 凭证签发。
func (h *HTTPServer) HandleAuthorize(w http.ResponseWriter, r *http.Request) {
	// Apply browser protections before every redirect or error response. 所有重定向及错误响应都应先应用浏览器保护。
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Referrer-Policy", "no-referrer")

	if err := h.initializationError(); err != nil {
		h.writeHTTPError(w, r, http.StatusInternalServerError, err)
		return
	}
	if r.Method != http.MethodGet {
		h.writeHTTPError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	values, parseErr := parseRequestQuery(r, h.options.RequestLimits)
	if parseErr != nil {
		status := statusFromError(parseErr)
		h.writeHTTPError(w, r, status, parseErr)
		return
	}
	if err := h.verifyRequest(r.Context(), values, ActionAuthorize, false); err != nil {
		status := statusFromError(err)
		h.writeHTTPError(w, r, status, err)
		return
	}
	loginID, ok := h.options.LoginIDResolver(r)
	if err := r.Context().Err(); err != nil {
		h.writeHTTPError(w, r, statusFromError(err), err)
		return
	}
	if !ok || loginID == "" {
		h.redirectToLogin(w, r)
		return
	}

	params := h.options.ServerOptions.Params
	redirectURI := values.Get(params.Redirect)
	clientID := values.Get(params.Client)
	if clientID == "" {
		if !h.options.ServerOptions.AllowAnonymousClient {
			h.writeHTTPError(w, r, http.StatusBadRequest, ErrClientOrClientIDEmpty)
			return
		}
		clientID = ClientAnonymous
	}
	mode := Mode(values.Get(params.Mode))
	if mode == "" {
		mode = h.options.ServerOptions.Mode
	}
	scopes := parseScopes(values.Get(params.Scope))

	// Validate the redirect for every credential mode before issuing a browser-delivered value. 为所有凭证模式校验回调地址，避免浏览器投递凭证到未登记地址。
	client, err := h.server.getClient(r.Context(), clientID)
	if err != nil {
		h.writeHTTPError(w, r, statusFromError(err), err)
		return
	}
	if !h.server.isValidRedirectURI(client, redirectURI) {
		h.writeHTTPError(w, r, http.StatusBadRequest, ErrInvalidRedirectURI)
		return
	}
	callbackURL := values.Get(params.Callback)
	if h.options.ServerOptions.EnableSLO && callbackURL != "" && !h.server.isValidLogoutCallbackURL(client, callbackURL) {
		// Reject an untrusted logout callback before issuing any browser-delivered credential. 在签发浏览器凭证前拒绝不可信的注销回调地址。
		h.writeHTTPError(w, r, http.StatusBadRequest, ErrInvalidCallbackURL)
		return
	}

	if (mode != ModeTicket && mode != ModeSharedToken && mode != ModeRemoteSession && mode != ModeOAuth2) || !h.server.isModeAllowed(client, mode) {
		h.writeHTTPError(w, r, http.StatusBadRequest, ErrModeUnsupported)
		return
	}
	if !h.server.isValidScopes(client, scopes) {
		h.writeHTTPError(w, r, http.StatusBadRequest, ErrInvalidScope)
		return
	}
	issueOptions := OneTimeCredentialOptions{CodeChallenge: values.Get(params.CodeChallenge), CodeChallengeMethod: values.Get(params.CodeChallengeMethod)}
	if mode != ModeTicket && mode != ModeOAuth2 {
		if issueOptions.CodeChallenge != "" || issueOptions.CodeChallengeMethod != "" {
			h.writeHTTPError(w, r, http.StatusBadRequest, ErrModeUnsupported)
			return
		}
	} else if err := validatePKCEChallenge(issueOptions.CodeChallenge, issueOptions.CodeChallengeMethod, client.RequirePKCE); err != nil {
		h.writeHTTPError(w, r, http.StatusBadRequest, err)
		return
	}

	// Consume only after login and request validation, before application hooks and credential issuance. 仅在登录及请求校验后占用随机值，且早于应用钩子和凭证签发。
	if err := h.options.ReplayProtection.claim(r.Context(), values, params, ActionAuthorize); err != nil {
		status := statusFromError(err)
		h.writeHTTPError(w, r, status, err)
		return
	}
	scopes, err = h.evaluateAuthorization(r.Context(), AuthorizationRequest{
		Phase: AuthorizationIssue, ClientID: clientID, LoginID: loginID, Mode: mode, Scopes: scopes,
	})
	if err != nil {
		h.writeHTTPError(w, r, statusFromError(err), err)
		return
	}
	if err := r.Context().Err(); err != nil {
		h.writeHTTPError(w, r, statusFromError(err), err)
		return
	}
	var extra map[string]any
	if h.options.AuthorizationExtra != nil {
		extra, err = h.options.AuthorizationExtra(r, clientID, loginID)
		if err != nil {
			h.writeHTTPError(w, r, statusFromError(err), err)
			return
		}
	}
	if err := r.Context().Err(); err != nil {
		h.writeHTTPError(w, r, statusFromError(err), err)
		return
	}

	credentialParam := ""
	credentialValue := ""
	switch mode {
	case ModeTicket:
		ticket, issueErr := h.server.GenerateTicketWithOptions(r.Context(), clientID, loginID, redirectURI, scopes, extra, issueOptions)
		if issueErr != nil {
			err = issueErr
			break
		}
		credentialParam = params.Ticket
		credentialValue = ticket.Ticket
	case ModeSharedToken:
		token, issueErr := h.server.GenerateSharedToken(r.Context(), clientID, loginID, scopes, extra)
		if issueErr != nil {
			err = issueErr
			break
		}
		credentialParam = params.TokenValue
		credentialValue = token.Token
	case ModeRemoteSession:
		session, issueErr := h.server.CreateRemoteSession(r.Context(), clientID, loginID, scopes, extra)
		if issueErr != nil {
			err = issueErr
			break
		}
		credentialParam = params.SessionID
		credentialValue = session.SessionID
	case ModeOAuth2:
		code, issueErr := h.server.GenerateOAuth2CodeWithOptions(r.Context(), clientID, loginID, redirectURI, scopes, extra, issueOptions)
		if issueErr != nil {
			err = issueErr
			break
		}
		credentialParam = params.Code
		credentialValue = code.Code
	default:
		err = ErrModeUnsupported
	}
	if err != nil {
		h.writeHTTPError(w, r, statusFromError(err), err)
		return
	}
	if err := r.Context().Err(); err != nil {
		h.writeHTTPError(w, r, statusFromError(err), err)
		return
	}
	if h.options.ServerOptions.EnableSLO && callbackURL != "" {
		if _, err = h.registerClientSession(r.Context(), loginID, clientID, callbackURL); err != nil {
			h.writeHTTPError(w, r, statusFromError(err), err)
			return
		}
	}
	if err := r.Context().Err(); err != nil {
		h.writeHTTPError(w, r, statusFromError(err), err)
		return
	}

	target, err := url.Parse(redirectURI)
	if err != nil {
		h.writeHTTPError(w, r, http.StatusBadRequest, ErrInvalidRedirectURI)
		return
	}
	query := target.Query()
	query.Set(credentialParam, credentialValue)
	if state := values.Get(params.Back); state != "" {
		query.Set(params.Back, state)
	}
	target.RawQuery = query.Encode()
	http.Redirect(w, r, target.String(), http.StatusFound)
}

// registerClientSession enforces the HTTP server's per-account client-session limit. registerClientSession 执行 HTTP 服务端的账号客户端会话上限。
func (h *HTTPServer) registerClientSession(ctx context.Context, loginID, clientID, callbackURL string) (*ClientSession, error) {
	h.clientSessionMu.Lock()
	defer h.clientSessionMu.Unlock()

	if err := h.checkClientSessionLimit(ctx, loginID, clientID); err != nil {
		return nil, err
	}
	return h.server.RegisterClientSession(ctx, loginID, clientID, callbackURL)
}

// checkClientSessionLimit checks capacity while the caller holds clientSessionMu. checkClientSessionLimit 在调用方持有 clientSessionMu 时检查容量。
func (h *HTTPServer) checkClientSessionLimit(ctx context.Context, loginID, clientID string) error {
	limit := h.options.ServerOptions.MaxRegisteredClient
	if limit > 0 {
		sessions, err := h.server.GetClientSessions(ctx, loginID)
		if err != nil {
			return err
		}
		registered := false
		for i := range sessions {
			if sessions[i].ClientID == clientID {
				registered = true
				break
			}
		}
		if !registered && len(sessions) >= limit {
			return ErrClientSessionLimit
		}
	}
	return nil
}

// HandleToken handles ticket or code exchange and returns user identity JSON. HandleToken 处理 Ticket 或授权码交换并返回用户身份 JSON。
func (h *HTTPServer) HandleToken(w http.ResponseWriter, r *http.Request) {
	if err := h.initializationError(); err != nil {
		h.writeHTTPError(w, r, http.StatusInternalServerError, err)
		return
	}
	result, err := h.exchangeRequest(r)
	if err != nil {
		h.writeHTTPError(w, r, statusFromError(err), err)
		return
	}
	h.writeHTTPResponse(w, r, http.StatusOK, OKResponse(result))
}

// HandleIntrospect checks a credential without consuming it when possible. HandleIntrospect 尽量在不消费凭证的情况下检查凭证。
func (h *HTTPServer) HandleIntrospect(w http.ResponseWriter, r *http.Request) {
	if err := h.initializationError(); err != nil {
		h.writeHTTPError(w, r, http.StatusInternalServerError, err)
		return
	}
	if r.Method != http.MethodPost {
		h.writeHTTPError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if err := parseRequestForm(r, h.options.RequestLimits); err != nil {
		status := statusFromError(err)
		h.writeHTTPError(w, r, status, err)
		return
	}
	// Keep exchange proofs out of URLs, including POST query strings. 兑换证明不得出现在 URL 中，包括 POST 查询串。
	if _, exists := r.URL.Query()[h.options.Params.CodeVerifier]; exists {
		h.writeHTTPError(w, r, http.StatusBadRequest, ErrInvalidPKCE)
		return
	}
	values := r.Form
	if err := h.verifyRequest(r.Context(), values, ActionIntrospect, true); err != nil {
		status := statusFromError(err)
		h.writeHTTPError(w, r, status, err)
		return
	}
	if err := h.authenticateClient(r.Context(), values); err != nil {
		h.writeHTTPError(w, r, statusFromError(err), err)
		return
	}
	info, err := h.introspectCredential(r, values)
	if err != nil {
		h.writeHTTPError(w, r, statusFromError(err), err)
		return
	}
	h.writeHTTPResponse(w, r, http.StatusOK, OKResponse(info))
}

// HandleUserInfo returns user info for a valid credential. HandleUserInfo 返回有效凭证对应的用户信息。
func (h *HTTPServer) HandleUserInfo(w http.ResponseWriter, r *http.Request) {
	if err := h.initializationError(); err != nil {
		h.writeHTTPError(w, r, http.StatusInternalServerError, err)
		return
	}
	if r.Method != http.MethodPost {
		h.writeHTTPError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if err := parseRequestForm(r, h.options.RequestLimits); err != nil {
		status := statusFromError(err)
		h.writeHTTPError(w, r, status, err)
		return
	}
	// Keep exchange proofs out of URLs, including POST query strings. 兑换证明不得出现在 URL 中，包括 POST 查询串。
	if _, exists := r.URL.Query()[h.options.Params.CodeVerifier]; exists {
		h.writeHTTPError(w, r, http.StatusBadRequest, ErrInvalidPKCE)
		return
	}
	values := r.Form
	if err := h.verifyRequest(r.Context(), values, ActionUserInfo, true); err != nil {
		status := statusFromError(err)
		h.writeHTTPError(w, r, status, err)
		return
	}
	if err := h.authenticateClient(r.Context(), values); err != nil {
		h.writeHTTPError(w, r, statusFromError(err), err)
		return
	}
	info, err := h.introspectCredential(r, values)
	if err != nil {
		h.writeHTTPError(w, r, statusFromError(err), err)
		return
	}
	if !info.Active {
		h.writeHTTPError(w, r, http.StatusUnauthorized, errors.New("inactive credential"))
		return
	}
	h.writeHTTPResponse(w, r, http.StatusOK, OKResponse(map[string]any{
		"active":    info.Active,
		"mode":      info.Mode,
		"loginId":   info.LoginID,
		"clientId":  info.ClientID,
		"scopes":    info.Scopes,
		"expiresIn": info.ExpiresIn,
		"extra":     info.Extra,
	}))
}

// HandleRevoke revokes a supported SSO credential. HandleRevoke 撤销支持的 SSO 凭证。
func (h *HTTPServer) HandleRevoke(w http.ResponseWriter, r *http.Request) {
	if err := h.initializationError(); err != nil {
		h.writeHTTPError(w, r, http.StatusInternalServerError, err)
		return
	}
	if r.Method != http.MethodPost {
		h.writeHTTPError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if err := parseRequestForm(r, h.options.RequestLimits); err != nil {
		status := statusFromError(err)
		h.writeHTTPError(w, r, status, err)
		return
	}
	values := r.Form
	if err := h.verifyRequest(r.Context(), values, ActionRevoke, true); err != nil {
		status := statusFromError(err)
		h.writeHTTPError(w, r, status, err)
		return
	}
	if err := h.authenticateClient(r.Context(), values); err != nil {
		h.writeHTTPError(w, r, statusFromError(err), err)
		return
	}
	if err := h.revokeCredential(r, values); err != nil {
		h.writeHTTPError(w, r, statusFromError(err), err)
		return
	}
	h.writeHTTPResponse(w, r, http.StatusOK, OKResponse(map[string]string{"result": ResultOK}))
}

// authenticateClient validates the registered client and its optional secret. authenticateClient 校验已注册客户端及其可选密钥。
func (h *HTTPServer) authenticateClient(ctx context.Context, values url.Values) error {
	params := h.options.ServerOptions.Params
	clientID := values.Get(params.Client)
	if clientID == "" {
		return ErrClientOrClientIDEmpty
	}
	client, err := h.server.getClient(ctx, clientID)
	if err != nil {
		return err
	}
	if !clientSecretMatches(client.ClientSecret, values.Get(params.ClientSecret)) {
		return ErrInvalidClientCredentials
	}
	return nil
}

// exchangeCredential dispatches credential exchange by mode. exchangeCredential 按模式分发凭证交换。
func (h *HTTPServer) exchangeCredential(r *http.Request, values url.Values) (*TicketExchangeResult, error) {
	params := h.options.ServerOptions.Params
	mode := Mode(values.Get(params.Mode))
	codeValue := values.Get(params.Code)
	ticketValue := values.Get(params.Ticket)

	// Resolve a single mode before consuming either credential. 消费凭证前先确定唯一模式。
	if codeValue != "" && ticketValue != "" {
		return nil, ErrInvalidRequest
	}
	if mode == "" {
		mode = ModeTicket
		if codeValue != "" {
			mode = ModeOAuth2
		}
	}
	if mode != ModeTicket && mode != ModeOAuth2 {
		return nil, ErrModeUnsupported
	}
	if mode == ModeTicket && codeValue != "" || mode == ModeOAuth2 && ticketValue != "" {
		return nil, ErrInvalidRequest
	}

	clientID := values.Get(params.Client)
	callbackURL := values.Get(params.Callback)
	registerCallback := h.options.ServerOptions.EnableSLO && callbackURL != ""
	var authorization AuthorizationRequest
	var grantedScopes []string
	if registerCallback || h.options.AuthorizationPolicy != nil {
		var err error
		authorization, err = h.exchangeAuthorization(r, values, mode)
		if err != nil {
			return nil, err
		}
	}
	if registerCallback {
		// Check callback ownership and capacity before consuming a one-time credential. 消费一次性凭证前检查回调归属和登记容量。
		h.clientSessionMu.Lock()
		defer h.clientSessionMu.Unlock()
		client, err := h.server.getClient(r.Context(), clientID)
		if err != nil {
			return nil, err
		}
		if !h.server.isValidLogoutCallbackURL(client, callbackURL) {
			return nil, ErrInvalidCallbackURL
		}
		if err := h.checkClientSessionLimit(r.Context(), authorization.LoginID, clientID); err != nil {
			return nil, err
		}
	}
	if h.options.AuthorizationPolicy != nil {
		var err error
		grantedScopes, err = h.evaluateAuthorization(r.Context(), authorization)
		if err != nil {
			return nil, err
		}
	}

	var result *TicketExchangeResult
	if mode == ModeOAuth2 {
		code, err := h.server.ConsumeOAuth2CodeWithVerifier(
			r.Context(),
			codeValue,
			values.Get(params.Client),
			values.Get(params.ClientSecret),
			values.Get(params.Redirect),
			values.Get(params.CodeVerifier),
		)
		if err != nil {
			return nil, err
		}
		result = OAuth2CodeResult(code)
	} else {
		ticket, err := h.server.ConsumeTicketWithVerifier(
			r.Context(),
			ticketValue,
			values.Get(params.Client),
			values.Get(params.ClientSecret),
			values.Get(params.Redirect),
			values.Get(params.CodeVerifier),
		)
		if err != nil {
			return nil, err
		}
		result = TicketResult(ticket)
	}

	// Recheck the consumed payload against the policy input before applying a reduced grant. 应用缩减授权前，核对实际消费载荷与策略输入一致。
	if h.options.AuthorizationPolicy != nil {
		if result.LoginID != authorization.LoginID || !slices.Equal(result.Scopes, authorization.Scopes) {
			return nil, ErrAuthorizationDecision
		}
		result.Scopes = grantedScopes
	}

	// Allow application-owned session metadata without changing the credential's authorization. 允许应用补充自己管理的会话信息，但不能改变凭证授权。
	if err := r.Context().Err(); err != nil {
		return nil, err
	}
	if h.options.EnrichExchangeResult != nil {
		loginID, centerID := result.LoginID, result.CenterID
		scopes := append([]string(nil), result.Scopes...)
		if err := h.options.EnrichExchangeResult(r, clientID, result); err != nil {
			return nil, err
		}
		if result.LoginID != loginID || result.CenterID != centerID || !slices.Equal(result.Scopes, scopes) {
			return nil, errors.New("exchange result hook changed credential identity or scopes")
		}
		if result.RemainTokenTimeout < -1 || result.RemainSessionTimeout < -1 {
			return nil, errors.New("exchange result hook returned an invalid session timeout")
		}
	}
	if err := r.Context().Err(); err != nil {
		return nil, err
	}

	// Bind only the identity returned by successful consumption. 仅登记成功消费凭证后返回的身份。
	if registerCallback {
		if _, err := h.server.RegisterClientSession(r.Context(), result.LoginID, clientID, callbackURL); err != nil {
			return nil, err
		}
	}
	if err := r.Context().Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// introspectCredential inspects a credential by mode. introspectCredential 按模式检查凭证。
func (h *HTTPServer) introspectCredential(r *http.Request, values url.Values) (*CredentialInfo, error) {
	info, err := h.loadCredentialInfo(r, values)
	if contextErr := r.Context().Err(); contextErr != nil {
		return nil, contextErr
	}
	if err != nil || !info.Active {
		return info, err
	}

	// A hook may reject a valid credential, but cannot reactivate an invalid one. 钩子可以拒绝有效凭证，但不能重新激活无效凭证。
	if h.options.CheckCredentialActive != nil {
		active, err := h.options.CheckCredentialActive(r, *info)
		if contextErr := r.Context().Err(); contextErr != nil {
			return nil, contextErr
		}
		if err != nil {
			return nil, err
		}
		if !active {
			return inactiveCredential(), nil
		}
	}
	scopes, err := h.evaluateAuthorization(r.Context(), AuthorizationRequest{
		Phase: AuthorizationInspect, ClientID: info.ClientID, LoginID: info.LoginID, Mode: info.Mode, Scopes: info.Scopes,
	})
	if errors.Is(err, ErrAuthorizationPolicyUnavailable) {
		return nil, err
	}
	if errors.Is(err, ErrAuthorizationDenied) {
		return inactiveCredential(), nil
	}
	if err != nil {
		return nil, err
	}
	info.Scopes = scopes
	return info, nil
}

// loadCredentialInfo validates the stored credential and client binding before returning metadata. loadCredentialInfo 校验存储凭证与客户端绑定后返回元数据。
func (h *HTTPServer) loadCredentialInfo(r *http.Request, values url.Values) (*CredentialInfo, error) {
	params := h.options.ServerOptions.Params
	clientID := values.Get(params.Client)
	mode := Mode(values.Get(params.Mode))
	if values.Get(params.CodeVerifier) != "" && (mode == ModeSharedToken || mode == ModeRemoteSession) {
		return nil, ErrModeUnsupported
	}
	switch mode {
	case "", ModeTicket:
		if err := h.server.checkClientMode(r.Context(), clientID, ModeTicket); err != nil {
			return nil, err
		}
		ticket, err := h.server.ValidateTicket(r.Context(), values.Get(params.Ticket))
		if err != nil || ticket.ClientID != clientID {
			if err == nil {
				err = ErrClientMismatch
			}
			return credentialIntrospectionFailure(err)
		}
		client, err := h.server.getClient(r.Context(), clientID)
		if err != nil {
			return nil, err
		}
		if err := verifyPKCE(ticket.CodeChallenge, ticket.CodeChallengeMethod, values.Get(params.CodeVerifier), client.RequirePKCE); err != nil {
			return nil, err
		}
		ttl, err := h.server.GetTicketTTL(r.Context(), ticket.Ticket)
		if err != nil {
			return nil, err
		}
		return TicketCredentialInfo(ticket, ttl), nil
	case ModeSharedToken:
		token, err := h.server.ValidateSharedToken(r.Context(), values.Get(params.TokenValue), clientID)
		if err != nil {
			return credentialIntrospectionFailure(err)
		}
		ttl, err := h.server.GetSharedTokenTTL(r.Context(), token.Token)
		if err != nil {
			return nil, err
		}
		return SharedTokenCredentialInfo(token, ttl), nil
	case ModeRemoteSession:
		session, err := h.server.ValidateRemoteSession(r.Context(), values.Get(params.SessionID), clientID)
		if err != nil {
			return credentialIntrospectionFailure(err)
		}
		ttl, err := h.server.GetRemoteSessionTTL(r.Context(), session.SessionID)
		if err != nil {
			return nil, err
		}
		return RemoteSessionCredentialInfo(session, ttl), nil
	case ModeOAuth2:
		if err := h.server.checkClientMode(r.Context(), clientID, ModeOAuth2); err != nil {
			return nil, err
		}
		code, err := h.server.getOAuth2Code(r.Context(), values.Get(params.Code))
		if err != nil {
			return credentialIntrospectionFailure(err)
		}
		if code.ClientID != clientID {
			return credentialIntrospectionFailure(ErrClientMismatch)
		}
		if err = h.server.checkOAuth2CodeAlive(code); err != nil {
			return credentialIntrospectionFailure(err)
		}
		client, err := h.server.getClient(r.Context(), clientID)
		if err != nil {
			return nil, err
		}
		if err := verifyPKCE(code.CodeChallenge, code.CodeChallengeMethod, values.Get(params.CodeVerifier), client.RequirePKCE); err != nil {
			return nil, err
		}
		ttl, err := h.server.GetOAuth2CodeTTL(r.Context(), code.Code)
		if err != nil {
			return nil, err
		}
		return OAuth2CodeCredentialInfo(code, ttl), nil
	default:
		return nil, ErrModeUnsupported
	}
}

// credentialIntrospectionFailure converts expected invalid credentials to an inactive result while preserving infrastructure failures. credentialIntrospectionFailure 将预期的无效凭证转为非活动结果，同时保留基础设施错误。
func credentialIntrospectionFailure(err error) (*CredentialInfo, error) {
	switch {
	case errors.Is(err, ErrClientMismatch),
		errors.Is(err, ErrInvalidTicket),
		errors.Is(err, ErrTicketUsed),
		errors.Is(err, ErrTicketExpired),
		errors.Is(err, ErrInvalidSharedToken),
		errors.Is(err, ErrSharedTokenExpired),
		errors.Is(err, ErrInvalidRemoteSession),
		errors.Is(err, ErrRemoteSessionExpired),
		errors.Is(err, ErrInvalidOAuth2Code),
		errors.Is(err, ErrOAuth2CodeUsed),
		errors.Is(err, ErrOAuth2CodeExpired):
		return inactiveCredential(), nil
	default:
		return nil, err
	}
}

// revokeCredential revokes a credential by mode. revokeCredential 按模式撤销凭证。
func (h *HTTPServer) revokeCredential(r *http.Request, values url.Values) error {
	params := h.options.ServerOptions.Params
	clientID := values.Get(params.Client)
	switch mode := Mode(values.Get(params.Mode)); mode {
	case "", ModeTicket:
		ticket, err := h.server.getTicket(r.Context(), values.Get(params.Ticket))
		if err != nil {
			if errors.Is(err, ErrInvalidTicket) {
				return nil
			}
			return err
		}
		if ticket.ClientID != clientID {
			return ErrClientMismatch
		}
		return h.server.RevokeTicket(r.Context(), values.Get(params.Ticket))
	case ModeSharedToken:
		token, err := h.server.getSharedToken(r.Context(), values.Get(params.TokenValue))
		if err != nil {
			if errors.Is(err, ErrInvalidSharedToken) {
				return nil
			}
			return err
		}
		if token.ClientID != clientID {
			return ErrClientMismatch
		}
		return h.server.RevokeSharedToken(r.Context(), values.Get(params.TokenValue))
	case ModeRemoteSession:
		session, err := h.server.getRemoteSession(r.Context(), values.Get(params.SessionID))
		if err != nil {
			if errors.Is(err, ErrInvalidRemoteSession) {
				return nil
			}
			return err
		}
		if session.ClientID != clientID {
			return ErrClientMismatch
		}
		return h.server.RevokeRemoteSession(r.Context(), values.Get(params.SessionID))
	case ModeOAuth2:
		code, err := h.server.getOAuth2Code(r.Context(), values.Get(params.Code))
		if err != nil {
			if errors.Is(err, ErrInvalidOAuth2Code) {
				return nil
			}
			return err
		}
		if code.ClientID != clientID {
			return ErrClientMismatch
		}
		return h.server.RevokeOAuth2Code(r.Context(), values.Get(params.Code))
	default:
		return ErrModeUnsupported
	}
}

// HandleLogout notifies clients, revokes the center session, and clears logout state. HandleLogout 通知客户端、撤销中心会话并清理注销状态。
func (h *HTTPServer) HandleLogout(w http.ResponseWriter, r *http.Request) {
	if err := h.initializationError(); err != nil {
		h.writeHTTPError(w, r, http.StatusInternalServerError, err)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		h.writeHTTPError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	if err := parseRequestForm(r, h.options.RequestLimits); err != nil {
		status := statusFromError(err)
		h.writeHTTPError(w, r, status, err)
		return
	}
	if err := h.verifyRequest(r.Context(), r.Form, ActionLogout, true); err != nil {
		h.writeHTTPError(w, r, statusFromError(err), err)
		return
	}
	loginID, ok := h.options.LoginIDResolver(r)
	if !ok || loginID == "" {
		h.writeHTTPError(w, r, http.StatusUnauthorized, errors.New("not logged in"))
		return
	}
	requestedLoginID := r.FormValue(h.options.ServerOptions.Params.LoginID)
	if requestedLoginID != "" && requestedLoginID != loginID {
		h.writeHTTPError(w, r, http.StatusUnauthorized, ErrClientMismatch)
		return
	}
	result := LogoutResult{LoginID: loginID, Stage: "notifications"}
	if h.options.ServerOptions.EnableSLO {
		report, err := h.NotifyLogoutClients(r, loginID)
		result.Notifications = report
		if err != nil && (report == nil || !h.options.LogoutCallbackBestEffort || r.Context().Err() != nil) {
			result.Cause = err
			h.writeLogoutResult(w, r, result)
			return
		}
	}
	if err := r.Context().Err(); err != nil {
		result.Cause = err
		h.writeLogoutResult(w, r, result)
		return
	}

	// Retain callback records and the cookie until center logout succeeds so failures can be retried. 中心注销成功前保留回调记录和 Cookie，以便失败后重试。
	result.Stage = "center_logout"
	if h.options.CenterLogout != nil {
		if err := h.options.CenterLogout(r, loginID); err != nil {
			result.Cause = err
			h.writeLogoutResult(w, r, result)
			return
		}
	}
	if err := r.Context().Err(); err != nil {
		result.Cause = err
		h.writeLogoutResult(w, r, result)
		return
	}
	result.Stage = "credentials"
	if err := h.server.RevokeLoginCredentials(r.Context(), loginID); err != nil {
		result.Cause = err
		h.writeLogoutResult(w, r, result)
		return
	}
	if err := r.Context().Err(); err != nil {
		result.Cause = err
		h.writeLogoutResult(w, r, result)
		return
	}
	result.Stage = "cleanup"
	if h.options.ServerOptions.EnableSLO {
		if err := h.server.ClearClientSessions(r.Context(), loginID); err != nil {
			result.Cause = err
			h.writeLogoutResult(w, r, result)
			return
		}
	}
	if err := r.Context().Err(); err != nil {
		result.Cause = err
		h.writeLogoutResult(w, r, result)
		return
	}
	ClearLoginIDCookie(w, h.options.Cookie)
	result.Result, result.Stage = ResultOK, "complete"
	h.writeLogoutResult(w, r, result)
}

// verifyRequest authenticates before checking freshness or recording a nonce. verifyRequest 在检查时效或记录随机值前完成认证。
func (h *HTTPServer) verifyRequest(ctx context.Context, values url.Values, action string, consume bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validatePKCEParameters(values, h.options.Params, action == ActionAuthorize); err != nil {
		return err
	}
	if err := validateScalarParameters(values, h.options.Params); err != nil {
		return err
	}
	if err := validateCredentialParameters(values, h.options.Params, action); err != nil {
		return err
	}
	if err := h.verifySign(ctx, values); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	protection := h.options.ReplayProtection
	if protection == nil {
		return nil
	}
	if err := protection.validateConfig(h.options.CheckSign, h.options.Params); err != nil {
		return err
	}
	if consume {
		return protection.claim(ctx, values, h.options.Params, action)
	}
	return protection.validate(values, h.options.Params, action, time.Now())
}

// verifySign validates request signatures when enabled. verifySign 在启用时校验请求签名。
func (h *HTTPServer) verifySign(ctx context.Context, values url.Values) error {
	if !h.options.ServerOptions.CheckSign {
		return nil
	}
	params := h.options.ServerOptions.Params
	if len(values[params.Client]) > 1 {
		return ErrInvalidRequest
	}
	clientID := values.Get(params.Client)
	if clientID == "" && h.options.ServerOptions.AllowAnonymousClient {
		clientID = ClientAnonymous
	}

	// Resolve the current registration on each request; lookup failures must not fall back to another key. 每次请求读取当前注册配置；查询失败时不能降级为其他密钥。
	var client *Client
	if clientID != "" {
		var err error
		client, err = h.server.getClient(ctx, clientID)
		if err != nil {
			return err
		}
	}
	secret, verification, err := h.signingKeys(client)
	if err != nil {
		return err
	}
	if !verifyProtocolSignature(values, params, secret, verification) {
		return ErrInvalidSign
	}
	return nil
}

// signingSecret selects a client's explicit key before the shared fallback. signingSecret 优先选择客户端显式密钥，再选择共享默认密钥。
func (h *HTTPServer) signingSecret(client *Client) (string, error) {
	secret, _, err := h.signingKeys(client)
	return secret, err
}

// redirectToLogin redirects authorization requests to the login page. redirectToLogin 将授权请求重定向到登录页。
func (h *HTTPServer) redirectToLogin(w http.ResponseWriter, r *http.Request) {
	if h.options.LoginPageURL == "" {
		h.writeHTTPError(w, r, http.StatusUnauthorized, errors.New("not logged in"))
		return
	}
	target, err := url.Parse(h.options.LoginPageURL)
	if err != nil {
		h.writeHTTPError(w, r, http.StatusUnauthorized, errors.New("not logged in"))
		return
	}
	query := target.Query()

	// Pass only a relative path to prevent open redirects. 仅传递相对路径，防止开放重定向。
	backPath := r.URL.Path
	if r.URL.RawQuery != "" {
		backPath += "?" + r.URL.RawQuery
	}
	query.Set(h.options.ServerOptions.Params.Back, backPath)
	target.RawQuery = query.Encode()
	http.Redirect(w, r, target.String(), http.StatusFound)
}

// writeJSON encodes the entire response before committing HTTP headers. writeJSON 在提交 HTTP 响应头前编码完整响应。
func writeJSON(w http.ResponseWriter, status int, response Response) {
	status, payload, _ := prepareJSON(status, response)
	writeJSONBytes(w, status, payload)
}

// prepareJSON replaces an unencodable response with a fixed error without exposing partial data. prepareJSON 用固定错误替换无法编码的响应，不暴露部分数据。
func prepareJSON(status int, response Response) (int, []byte, error) {
	payload, err := json.Marshal(response)
	if err != nil {
		return http.StatusInternalServerError, []byte("{\"code\":500,\"message\":\"SSO response encoding failed\"}\n"), fmt.Errorf("%w: %w", ErrSerializeFailed, err)
	}
	return status, append(payload, '\n'), nil
}

// writeJSONBytes writes one prepared response; transport failure cannot change a committed status or undo business effects. writeJSONBytes 写入已准备的响应，传输故障不能改变已提交状态或撤销业务副作用。
func writeJSONBytes(w http.ResponseWriter, status int, payload []byte) {
	// Keep identity, credential status, and logout responses out of caches. 避免缓存身份、凭证状态及注销响应。
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_, _ = w.Write(payload)
}

// parseScopes parses a space-delimited scope string. parseScopes 解析空格分隔的 Scope 字符串。
func parseScopes(scope string) []string {
	if strings.TrimSpace(scope) == "" {
		return nil
	}
	items := strings.FieldsFunc(scope, func(r rune) bool {
		return r == ',' || r == ' '
	})
	scopes := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			scopes = append(scopes, item)
		}
	}
	return scopes
}

// CookieOptions defines shared-cookie behavior for same-site SSO. CookieOptions 定义同站 SSO 的共享 Cookie 行为。
type CookieOptions struct {
	// Name stores cookie name. Name 存储 Cookie 名称。
	Name string
	// Domain stores shared cookie domain. Domain 存储共享 Cookie 域名。
	Domain string
	// Path stores cookie path. Path 存储 Cookie 路径。
	Path string
	// MaxAge stores cookie lifetime. MaxAge 存储 Cookie 有效期。
	MaxAge time.Duration
	// Secure restricts cookie to HTTPS. Secure 限制 Cookie 仅通过 HTTPS 发送。
	Secure bool
	// HTTPOnly hides cookie from scripts. HTTPOnly 禁止脚本读取 Cookie。
	HTTPOnly bool
	// SameSite stores browser same-site policy. SameSite 存储浏览器同站策略。
	SameSite http.SameSite
	// SecretKey signs the cookie payload; empty disables cookie-based identity resolution. SecretKey 对 Cookie 载荷签名；为空时禁用基于 Cookie 的身份解析。
	SecretKey string
	// VerificationKey optionally accepts cookies signed by one companion key during rotation; new cookies use SecretKey only. VerificationKey 在轮换期可选接受一把过渡密钥签发的 Cookie，新 Cookie 仅使用 SecretKey。
	VerificationKey string
}

// DefaultCookieOptions returns default shared-cookie options. DefaultCookieOptions 返回默认共享 Cookie 配置。
func DefaultCookieOptions() CookieOptions {
	return CookieOptions{
		Name:     "dtoken_sso",
		Path:     "/",
		MaxAge:   2 * time.Hour,
		HTTPOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

// LoginIDFromCookie creates a resolver that reads login id from shared cookie. LoginIDFromCookie 创建从共享 Cookie 读取登录 ID 的解析器。
func LoginIDFromCookie(options CookieOptions) LoginIDResolver {
	options = normalizeCookieOptions(options)
	return func(r *http.Request) (string, bool) {
		if r == nil || options.SecretKey == "" {
			return "", false
		}
		cookie, err := r.Cookie(options.Name)
		if err != nil || cookie.Value == "" {
			return "", false
		}
		return decodeLoginIDCookie(cookie.Value, options.SecretKey, options.VerificationKey)
	}
}

// SetLoginIDCookie writes shared login cookie. SetLoginIDCookie 写入共享登录 Cookie。
func SetLoginIDCookie(w http.ResponseWriter, options CookieOptions, loginID string) {
	options = normalizeCookieOptions(options)
	expiresAt := time.Now().Add(options.MaxAge)
	value := encodeLoginIDCookie(loginID, options.SecretKey, expiresAt)
	if value == "" {
		ClearLoginIDCookie(w, options)
		return
	}

	// Round up short lifetimes and keep MaxAge representable on 32-bit systems. 向上取整短有效期，并确保 MaxAge 在 32 位系统中可表示。
	maxAge := int(min(durationSeconds(options.MaxAge), int64(^uint(0)>>1)))
	http.SetCookie(w, &http.Cookie{
		Name:     options.Name,
		Value:    value,
		Path:     defaultCookiePath(options.Path),
		Domain:   options.Domain,
		MaxAge:   maxAge,
		Expires:  expiresAt,
		Secure:   options.Secure,
		HttpOnly: options.HTTPOnly,
		SameSite: options.SameSite,
	})
}

// loginCookieClaims binds the identity to a server-verified deadline. loginCookieClaims 将身份绑定到服务端校验的截止时间。
type loginCookieClaims struct {
	// LoginID stores the authenticated subject. LoginID 存储已认证主体。
	LoginID string `json:"loginId"`
	// ExpiresAt stores the signed deadline. ExpiresAt 存储签名截止时间。
	ExpiresAt time.Time `json:"expiresAt"`
}

// encodeLoginIDCookie signs the versioned identity and expiry payload. encodeLoginIDCookie 对带版本的身份与过期时间载荷签名。
func encodeLoginIDCookie(loginID, secret string, expiresAt time.Time) string {
	if loginID == "" || secret == "" || expiresAt.IsZero() {
		return ""
	}
	raw, err := json.Marshal(loginCookieClaims{LoginID: loginID, ExpiresAt: expiresAt})
	if err != nil {
		return ""
	}
	payload := base64.RawURLEncoding.EncodeToString(raw)
	values := url.Values{"version": {"v1"}, "value": {payload}}
	signed := NewSigner(secret).AttachSign(values)
	return "v1." + payload + "." + signed.Get(DefaultParamNames().Sign)
}

// decodeLoginIDCookie verifies and decodes a signed login cookie. decodeLoginIDCookie 校验并解码已签名的登录 Cookie。
func decodeLoginIDCookie(value, secret, verification string) (string, bool) {
	parts := strings.Split(value, ".")
	if len(parts) != 3 || parts[0] != "v1" || parts[1] == "" || parts[2] == "" || secret == "" {
		return "", false
	}
	values := url.Values{
		"version":                {parts[0]},
		"value":                  {parts[1]},
		DefaultParamNames().Sign: {parts[2]},
	}
	if !verifyProtocolSignature(values, DefaultParamNames(), secret, verification) {
		return "", false
	}
	decoded, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", false
	}
	var claims loginCookieClaims
	if err = json.Unmarshal(decoded, &claims); err != nil || claims.LoginID == "" || !time.Now().Before(claims.ExpiresAt) {
		return "", false
	}
	return claims.LoginID, true
}

// ClearLoginIDCookie clears shared login cookie. ClearLoginIDCookie 清除共享登录 Cookie。
func ClearLoginIDCookie(w http.ResponseWriter, options CookieOptions) {
	options = normalizeCookieOptions(options)
	http.SetCookie(w, &http.Cookie{
		Name:     options.Name,
		Value:    "",
		Path:     defaultCookiePath(options.Path),
		Domain:   options.Domain,
		MaxAge:   -1,
		Expires:  time.Unix(1, 0),
		Secure:   options.Secure,
		HttpOnly: options.HTTPOnly,
		SameSite: options.SameSite,
	})
}

// defaultCookiePath normalizes an empty cookie path. defaultCookiePath 规范化空 Cookie 路径。
func defaultCookiePath(path string) string {
	if path == "" {
		return "/"
	}
	return path
}

// normalizeCookieOptions fills missing cookie option defaults. normalizeCookieOptions 补齐缺失的 Cookie 默认选项。
func normalizeCookieOptions(options CookieOptions) CookieOptions {
	defaults := DefaultCookieOptions()
	if options.Name == "" {
		options.Name = defaults.Name
	}
	if options.Path == "" {
		options.Path = defaults.Path
	}
	if options.MaxAge <= 0 {
		options.MaxAge = defaults.MaxAge
	}
	if options.SameSite == 0 {
		options.SameSite = defaults.SameSite
	}
	return options
}
