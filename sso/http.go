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
	"strings"
	"sync"
	"time"
)

// LoginIDResolver resolves the current SSO-center login id from an HTTP request. LoginIDResolver 从 HTTP 请求解析当前 SSO 中心登录 ID。
type LoginIDResolver func(r *http.Request) (string, bool)

// HTTPOptions defines standalone HTTP protocol behavior for SSO. HTTPOptions 定义独立 HTTP 协议层行为。
type HTTPOptions struct {
	// ServerOptions carries protocol-level server behavior. ServerOptions 包含协议层服务端行为配置。
	ServerOptions
	LoginIDResolver LoginIDResolver // LoginIDResolver resolves current center login id. LoginIDResolver 解析当前中心登录 ID。
	LoginPageURL    string          // LoginPageURL stores the fallback login page URL. LoginPageURL 存储未登录时跳转的登录页地址。
	Cookie          CookieOptions   // Cookie stores optional shared-cookie settings. Cookie 存储可选共享 Cookie 配置。
}

// DefaultHTTPOptions returns default standalone HTTP options. DefaultHTTPOptions 返回默认独立 HTTP 选项。
func DefaultHTTPOptions() HTTPOptions {
	return HTTPOptions{
		ServerOptions: DefaultServerOptions(),
		Cookie:        DefaultCookieOptions(),
	}
}

// HTTPServer exposes SSO routes by using net/http only. HTTPServer 使用标准库 net/http 暴露 SSO 路由。
type HTTPServer struct {
	server          *Server
	options         HTTPOptions
	clientSessionMu sync.Mutex // clientSessionMu serializes session-limit checks and registrations. clientSessionMu 串行化会话上限检查和注册。
}

// NewHTTPServer creates a standalone HTTP SSO handler. NewHTTPServer 创建独立 HTTP SSO 处理器。
func NewHTTPServer(server *Server, options HTTPOptions) *HTTPServer {
	defaults := DefaultHTTPOptions()
	if options.ServerOptions.Endpoints == (Endpoints{}) {
		options.ServerOptions.Endpoints = defaults.ServerOptions.Endpoints
	}
	if options.ServerOptions.Params == (ParamNames{}) {
		options.ServerOptions.Params = defaults.ServerOptions.Params
	}
	if options.ServerOptions.Mode == "" {
		options.ServerOptions.Mode = defaults.ServerOptions.Mode
	}
	if options.ServerOptions.LogoutCallbackTimeout <= 0 {
		options.ServerOptions.LogoutCallbackTimeout = defaults.ServerOptions.LogoutCallbackTimeout
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
	if options.ServerOptions.Clients != nil && server != nil {
		_ = server.RegisterClients(options.ServerOptions.Clients)
	}
	if options.ServerOptions.AllowAnonymousClient && server != nil && !server.hasClient(ClientAnonymous) {
		_ = server.RegisterClient(&Client{
			ClientID:     ClientAnonymous,
			Name:         "Anonymous Client",
			RedirectURIs: append([]string(nil), options.ServerOptions.AllowURLs...),
			Modes:        []Mode{ModeTicket},
		})
	}
	return &HTTPServer{server: server, options: options}
}

// Register registers SSO routes into a ServeMux. Register 将 SSO 路由注册到 ServeMux。
func (h *HTTPServer) Register(mux *http.ServeMux) {
	if h == nil || mux == nil {
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
	mux := http.NewServeMux()
	h.Register(mux)
	return mux
}

// HandleAuthorize handles redirect-based SSO credential issuing. HandleAuthorize 处理基于重定向的 SSO 凭证签发。
func (h *HTTPServer) HandleAuthorize(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.server == nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse(http.StatusInternalServerError, ErrServerNotInitialized.Error()))
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse(http.StatusMethodNotAllowed, "method not allowed"))
		return
	}
	values := r.URL.Query()
	if err := h.verifySign(values); err != nil {
		status := statusFromError(err)
		writeJSON(w, status, ErrorResponse(status, err.Error()))
		return
	}
	loginID, ok := h.options.LoginIDResolver(r)
	if !ok || loginID == "" {
		h.redirectToLogin(w, r)
		return
	}

	params := h.options.ServerOptions.Params
	redirectURI := values.Get(params.Redirect)
	clientID := values.Get(params.Client)
	if clientID == "" {
		if !h.options.ServerOptions.AllowAnonymousClient {
			writeJSON(w, http.StatusBadRequest, ErrorResponse(http.StatusBadRequest, ErrClientOrClientIDEmpty.Error()))
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
		writeJSON(w, statusFromError(err), ErrorResponse(statusFromError(err), err.Error()))
		return
	}
	if !h.server.isValidRedirectURI(client, redirectURI) {
		writeJSON(w, http.StatusBadRequest, ErrorResponse(http.StatusBadRequest, ErrInvalidRedirectURI.Error()))
		return
	}
	callbackURL := values.Get(params.Callback)
	if h.options.ServerOptions.EnableSLO && callbackURL != "" && !h.server.isValidLogoutCallbackURL(client, callbackURL) {
		// Reject an untrusted logout callback before issuing any browser-delivered credential. 在签发浏览器凭证前拒绝不可信的注销回调地址。
		writeJSON(w, http.StatusBadRequest, ErrorResponse(http.StatusBadRequest, ErrInvalidCallbackURL.Error()))
		return
	}

	credentialParam := ""
	credentialValue := ""
	switch mode {
	case ModeTicket:
		ticket, issueErr := h.server.GenerateTicket(r.Context(), clientID, loginID, redirectURI, scopes, nil)
		if issueErr != nil {
			err = issueErr
			break
		}
		credentialParam = params.Ticket
		credentialValue = ticket.Ticket
	case ModeSharedToken:
		token, issueErr := h.server.GenerateSharedToken(r.Context(), clientID, loginID, scopes, nil)
		if issueErr != nil {
			err = issueErr
			break
		}
		credentialParam = params.TokenValue
		credentialValue = token.Token
	case ModeRemoteSession:
		session, issueErr := h.server.CreateRemoteSession(r.Context(), clientID, loginID, scopes, nil)
		if issueErr != nil {
			err = issueErr
			break
		}
		credentialParam = params.SessionID
		credentialValue = session.SessionID
	case ModeOAuth2:
		code, issueErr := h.server.GenerateOAuth2Code(r.Context(), clientID, loginID, redirectURI, scopes, nil)
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
		writeJSON(w, statusFromError(err), ErrorResponse(statusFromError(err), err.Error()))
		return
	}
	if h.options.ServerOptions.EnableSLO && callbackURL != "" {
		if _, err = h.registerClientSession(r.Context(), loginID, clientID, callbackURL); err != nil {
			writeJSON(w, statusFromError(err), ErrorResponse(statusFromError(err), err.Error()))
			return
		}
	}

	target, err := url.Parse(redirectURI)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse(http.StatusBadRequest, ErrInvalidRedirectURI.Error()))
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

	limit := h.options.ServerOptions.MaxRegisteredClient
	if limit > 0 {
		sessions, err := h.server.GetClientSessions(ctx, loginID)
		if err != nil {
			return nil, err
		}
		registered := false
		for i := range sessions {
			if sessions[i].ClientID == clientID {
				registered = true
				break
			}
		}
		if !registered && len(sessions) >= limit {
			return nil, ErrClientSessionLimit
		}
	}
	return h.server.RegisterClientSession(ctx, loginID, clientID, callbackURL)
}

// HandleToken handles ticket or code exchange and returns user identity JSON. HandleToken 处理 Ticket 或授权码交换并返回用户身份 JSON。
func (h *HTTPServer) HandleToken(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.server == nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse(http.StatusInternalServerError, ErrServerNotInitialized.Error()))
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse(http.StatusMethodNotAllowed, "method not allowed"))
		return
	}
	if err := r.ParseForm(); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse(http.StatusBadRequest, err.Error()))
		return
	}
	values := r.Form
	if err := h.verifySign(values); err != nil {
		status := statusFromError(err)
		writeJSON(w, status, ErrorResponse(status, err.Error()))
		return
	}
	result, err := h.exchangeCredential(r, values)
	if err != nil {
		writeJSON(w, statusFromError(err), ErrorResponse(statusFromError(err), err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, OKResponse(result))
}

// HandleIntrospect checks a credential without consuming it when possible. HandleIntrospect 尽量在不消费凭证的情况下检查凭证。
func (h *HTTPServer) HandleIntrospect(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.server == nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse(http.StatusInternalServerError, ErrServerNotInitialized.Error()))
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse(http.StatusMethodNotAllowed, "method not allowed"))
		return
	}
	if err := r.ParseForm(); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse(http.StatusBadRequest, err.Error()))
		return
	}
	values := r.Form
	if err := h.verifySign(values); err != nil {
		status := statusFromError(err)
		writeJSON(w, status, ErrorResponse(status, err.Error()))
		return
	}
	if err := h.authenticateClient(r.Context(), values); err != nil {
		writeJSON(w, statusFromError(err), ErrorResponse(statusFromError(err), err.Error()))
		return
	}
	info, err := h.introspectCredential(r, values)
	if err != nil {
		writeJSON(w, statusFromError(err), ErrorResponse(statusFromError(err), err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, OKResponse(info))
}

// HandleUserInfo returns user info for a valid credential. HandleUserInfo 返回有效凭证对应的用户信息。
func (h *HTTPServer) HandleUserInfo(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.server == nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse(http.StatusInternalServerError, ErrServerNotInitialized.Error()))
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse(http.StatusMethodNotAllowed, "method not allowed"))
		return
	}
	if err := r.ParseForm(); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse(http.StatusBadRequest, err.Error()))
		return
	}
	values := r.Form
	if err := h.verifySign(values); err != nil {
		status := statusFromError(err)
		writeJSON(w, status, ErrorResponse(status, err.Error()))
		return
	}
	if err := h.authenticateClient(r.Context(), values); err != nil {
		writeJSON(w, statusFromError(err), ErrorResponse(statusFromError(err), err.Error()))
		return
	}
	info, err := h.introspectCredential(r, values)
	if err != nil {
		writeJSON(w, statusFromError(err), ErrorResponse(statusFromError(err), err.Error()))
		return
	}
	if !info.Active {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse(http.StatusUnauthorized, "inactive credential"))
		return
	}
	writeJSON(w, http.StatusOK, OKResponse(map[string]any{
		"loginId":  info.LoginID,
		"clientId": info.ClientID,
		"scopes":   info.Scopes,
		"extra":    info.Extra,
	}))
}

// HandleRevoke revokes a supported SSO credential. HandleRevoke 撤销支持的 SSO 凭证。
func (h *HTTPServer) HandleRevoke(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.server == nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse(http.StatusInternalServerError, ErrServerNotInitialized.Error()))
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse(http.StatusMethodNotAllowed, "method not allowed"))
		return
	}
	if err := r.ParseForm(); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse(http.StatusBadRequest, err.Error()))
		return
	}
	values := r.Form
	if err := h.verifySign(values); err != nil {
		status := statusFromError(err)
		writeJSON(w, status, ErrorResponse(status, err.Error()))
		return
	}
	if err := h.authenticateClient(r.Context(), values); err != nil {
		writeJSON(w, statusFromError(err), ErrorResponse(statusFromError(err), err.Error()))
		return
	}
	if err := h.revokeCredential(r, values); err != nil {
		writeJSON(w, statusFromError(err), ErrorResponse(statusFromError(err), err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, OKResponse(map[string]string{"result": ResultOK}))
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
func (h *HTTPServer) exchangeCredential(r *http.Request, values url.Values) (any, error) {
	params := h.options.ServerOptions.Params
	if codeValue := values.Get(params.Code); codeValue != "" {
		code, err := h.server.ConsumeOAuth2Code(
			r.Context(),
			codeValue,
			values.Get(params.Client),
			values.Get(params.ClientSecret),
			values.Get(params.Redirect),
		)
		if err != nil {
			return nil, err
		}
		return OAuth2CodeResult(code), nil
	}
	ticket, err := h.server.ConsumeTicket(
		r.Context(),
		values.Get(params.Ticket),
		values.Get(params.Client),
		values.Get(params.ClientSecret),
		values.Get(params.Redirect),
	)
	if err != nil {
		return nil, err
	}
	return TicketResult(ticket), nil
}

// introspectCredential inspects a credential by mode. introspectCredential 按模式检查凭证。
func (h *HTTPServer) introspectCredential(r *http.Request, values url.Values) (*CredentialInfo, error) {
	params := h.options.ServerOptions.Params
	clientID := values.Get(params.Client)
	switch mode := Mode(values.Get(params.Mode)); mode {
	case "", ModeTicket:
		ticket, err := h.server.ValidateTicket(r.Context(), values.Get(params.Ticket))
		if err != nil || ticket.ClientID != clientID {
			if err == nil {
				err = ErrClientMismatch
			}
			return credentialIntrospectionFailure(err)
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

// HandleLogout clears optional shared cookie, pushes logout callbacks, and returns success. HandleLogout 清除共享 Cookie、推送注销回调并返回成功。
func (h *HTTPServer) HandleLogout(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.server == nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse(http.StatusInternalServerError, ErrServerNotInitialized.Error()))
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse(http.StatusMethodNotAllowed, "method not allowed"))
		return
	}
	if err := r.ParseForm(); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse(http.StatusBadRequest, err.Error()))
		return
	}
	if err := h.verifySign(r.Form); err != nil {
		writeJSON(w, statusFromError(err), ErrorResponse(statusFromError(err), err.Error()))
		return
	}
	loginID, ok := h.options.LoginIDResolver(r)
	if !ok || loginID == "" {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse(http.StatusUnauthorized, "not logged in"))
		return
	}
	requestedLoginID := r.FormValue(h.options.ServerOptions.Params.LoginID)
	if requestedLoginID != "" && requestedLoginID != loginID {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse(http.StatusUnauthorized, ErrClientMismatch.Error()))
		return
	}
	if h.options.ServerOptions.EnableSLO {
		if err := h.pushLogoutCallbacks(r, loginID); err != nil {
			writeJSON(w, statusFromError(err), ErrorResponse(statusFromError(err), err.Error()))
			return
		}
	}
	ClearLoginIDCookie(w, h.options.Cookie)
	writeJSON(w, http.StatusOK, OKResponse(map[string]string{"result": ResultOK}))
}

// pushLogoutCallbacks notifies registered client sessions about logout. pushLogoutCallbacks 向已注册客户端会话推送登出通知。
func (h *HTTPServer) pushLogoutCallbacks(r *http.Request, loginID string) error {
	sessions, err := h.server.GetClientSessions(r.Context(), loginID)
	if err != nil {
		return err
	}
	var wg sync.WaitGroup
	errCh := make(chan error, len(sessions))
	for _, session := range sessions {
		if session.LogoutCallbackURL == "" {
			continue
		}
		session := session
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := h.postLogoutCallback(r, session); err != nil {
				errCh <- err
			}
		}()
	}
	wg.Wait()
	close(errCh)
	if !h.options.ServerOptions.LogoutCallbackBestEffort {
		// Return the first error if any返回第一个错误（如果有）
		select {
		case err := <-errCh:
			return err
		default:
		}
	}
	return h.server.ClearClientSessions(r.Context(), loginID)
}

// postLogoutCallback posts one client logout callback. postLogoutCallback 发送一次客户端登出回调。
func (h *HTTPServer) postLogoutCallback(r *http.Request, session ClientSession) error {
	ctx := r.Context()
	if h.options.ServerOptions.LogoutCallbackTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, h.options.ServerOptions.LogoutCallbackTimeout)
		defer cancel()
	}
	values := url.Values{}
	values.Set(h.options.ServerOptions.Params.LoginID, session.LoginID)
	values.Set(h.options.ServerOptions.Params.Client, session.ClientID)
	values.Set(h.options.ServerOptions.Params.Timestamp, time.Now().Format(time.RFC3339))
	if h.options.ServerOptions.CheckSign {
		if h.options.ServerOptions.SecretKey == "" {
			return ErrSignSecretRequired
		}
		values = NewSignerWithParams(h.options.ServerOptions.SecretKey, h.options.ServerOptions.Params).AttachSign(values)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, session.LogoutCallbackURL, strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := h.options.ServerOptions.LogoutHTTPClient
	if client == nil {
		client = &http.Client{
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	} else if client.CheckRedirect == nil {
		// Clone caller transport settings while preventing implicit redirect-based SSRF. 复制调用方传输配置，同时阻止隐式重定向造成 SSRF。
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
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("logout callback failed with status %d", response.StatusCode)
	}
	return nil
}

// verifySign validates request signatures when enabled. verifySign 在启用时校验请求签名。
func (h *HTTPServer) verifySign(values url.Values) error {
	if !h.options.ServerOptions.CheckSign {
		return nil
	}
	if h.options.ServerOptions.SecretKey == "" {
		return ErrSignSecretRequired
	}
	if !NewSignerWithParams(h.options.ServerOptions.SecretKey, h.options.ServerOptions.Params).Verify(values) {
		return ErrInvalidSign
	}
	return nil
}

// redirectToLogin redirects authorization requests to the login page. redirectToLogin 将授权请求重定向到登录页。
func (h *HTTPServer) redirectToLogin(w http.ResponseWriter, r *http.Request) {
	if h.options.LoginPageURL == "" {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse(http.StatusUnauthorized, "not logged in"))
		return
	}
	target, err := url.Parse(h.options.LoginPageURL)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse(http.StatusUnauthorized, "not logged in"))
		return
	}
	query := target.Query()
	// Only pass relative path to prevent open redirect只传递相对路径，防止开放重定向
	backPath := r.URL.Path
	if r.URL.RawQuery != "" {
		backPath += "?" + r.URL.RawQuery
	}
	query.Set(h.options.ServerOptions.Params.Back, backPath)
	target.RawQuery = query.Encode()
	http.Redirect(w, r, target.String(), http.StatusFound)
}

// writeJSON writes a protocol JSON response. writeJSON 写入协议 JSON 响应。
func writeJSON(w http.ResponseWriter, status int, response Response) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
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

// statusFromError maps protocol errors to HTTP status codes. statusFromError 将协议错误映射为 HTTP 状态码。
func statusFromError(err error) int {
	switch {
	case errors.Is(err, ErrClientOrClientIDEmpty),
		errors.Is(err, ErrInvalidRedirectURI),
		errors.Is(err, ErrRedirectURIMismatch),
		errors.Is(err, ErrInvalidScope),
		errors.Is(err, ErrUserIDEmpty),
		errors.Is(err, ErrClientMismatch),
		errors.Is(err, ErrInvalidTicket),
		errors.Is(err, ErrTicketUsed),
		errors.Is(err, ErrTicketExpired),
		errors.Is(err, ErrInvalidSharedToken),
		errors.Is(err, ErrSharedTokenExpired),
		errors.Is(err, ErrInvalidRemoteSession),
		errors.Is(err, ErrRemoteSessionExpired),
		errors.Is(err, ErrInvalidOAuth2Code),
		errors.Is(err, ErrOAuth2CodeUsed),
		errors.Is(err, ErrOAuth2CodeExpired),
		errors.Is(err, ErrModeUnsupported),
		errors.Is(err, ErrStorageCapabilityUnsupported),
		errors.Is(err, ErrInvalidCallbackURL),
		errors.Is(err, ErrCallbackExpired),
		errors.Is(err, ErrClientSessionLimit):
		return http.StatusBadRequest
	case errors.Is(err, ErrInvalidClientCredentials),
		errors.Is(err, ErrInvalidSign):
		return http.StatusUnauthorized
	case errors.Is(err, ErrMethodNotAllowed):
		return http.StatusMethodNotAllowed
	case errors.Is(err, ErrClientNotFound):
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

// CookieOptions defines shared-cookie behavior for same-site SSO. CookieOptions 定义同站 SSO 的共享 Cookie 行为。
type CookieOptions struct {
	Name      string        // Name stores cookie name. Name 存储 Cookie 名称。
	Domain    string        // Domain stores shared cookie domain. Domain 存储共享 Cookie 域名。
	Path      string        // Path stores cookie path. Path 存储 Cookie 路径。
	MaxAge    time.Duration // MaxAge stores cookie lifetime. MaxAge 存储 Cookie 有效期。
	Secure    bool          // Secure restricts cookie to HTTPS. Secure 限制 Cookie 仅通过 HTTPS 发送。
	HTTPOnly  bool          // HTTPOnly hides cookie from scripts. HTTPOnly 禁止脚本读取 Cookie。
	SameSite  http.SameSite // SameSite stores browser same-site policy. SameSite 存储浏览器同站策略。
	SecretKey string        // SecretKey signs the cookie payload; empty disables cookie-based identity resolution. SecretKey 对 Cookie 载荷签名；为空时禁用基于 Cookie 的身份解析。
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
		return decodeLoginIDCookie(cookie.Value, options.SecretKey)
	}
}

// SetLoginIDCookie writes shared login cookie. SetLoginIDCookie 写入共享登录 Cookie。
func SetLoginIDCookie(w http.ResponseWriter, options CookieOptions, loginID string) {
	options = normalizeCookieOptions(options)
	value := encodeLoginIDCookie(loginID, options.SecretKey)
	if value == "" {
		ClearLoginIDCookie(w, options)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     options.Name,
		Value:    value,
		Path:     defaultCookiePath(options.Path),
		Domain:   options.Domain,
		MaxAge:   int(options.MaxAge.Seconds()),
		Secure:   options.Secure,
		HttpOnly: options.HTTPOnly,
		SameSite: options.SameSite,
	})
}

// encodeLoginIDCookie signs and encodes a login id for cookie storage. encodeLoginIDCookie 对登录 ID 签名并编码为 Cookie 值。
func encodeLoginIDCookie(loginID, secret string) string {
	if loginID == "" || secret == "" {
		return ""
	}
	payload := base64.RawURLEncoding.EncodeToString([]byte(loginID))
	values := url.Values{"value": {payload}}
	signed := NewSigner(secret).AttachSign(values)
	return payload + "." + signed.Get(DefaultParamNames().Sign)
}

// decodeLoginIDCookie verifies and decodes a signed login cookie. decodeLoginIDCookie 校验并解码已签名的登录 Cookie。
func decodeLoginIDCookie(value, secret string) (string, bool) {
	parts := strings.SplitN(value, ".", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" || secret == "" {
		return "", false
	}
	values := url.Values{
		"value":                  {parts[0]},
		DefaultParamNames().Sign: {parts[1]},
	}
	if !NewSigner(secret).Verify(values) {
		return "", false
	}
	decoded, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil || len(decoded) == 0 {
		return "", false
	}
	return string(decoded), true
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
