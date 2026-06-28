// @Author daixk 2026/05/29
package sso

import (
	"context"
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
	ServerOptions
	LoginIDResolver LoginIDResolver // LoginIDResolver resolves current center login id. LoginIDResolver 解析当前中心登录 ID。
	LoginPageURL    string          // LoginPageURL stores the fallback login page URL. LoginPageURL 存储未登录时跳转的登录页地址。
	Cookie          CookieOptions   // Cookie stores optional shared-cookie settings. Cookie 存储可选共享 Cookie 配置。
}

// DefaultHTTPOptions returns default standalone HTTP options. DefaultHTTPOptions 返回默认独立 HTTP 选项。
func DefaultHTTPOptions() HTTPOptions {
	return HTTPOptions{
		ServerOptions:   DefaultServerOptions(),
		LoginIDResolver: LoginIDFromCookie(DefaultCookieOptions()),
		Cookie:          DefaultCookieOptions(),
	}
}

// HTTPServer exposes SSO routes by using net/http only. HTTPServer 使用标准库 net/http 暴露 SSO 路由。
type HTTPServer struct {
	server  *Server
	options HTTPOptions
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
	options.Cookie = normalizeCookieOptions(options.Cookie)
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

// HandleAuthorize handles redirect-based ticket issuing. HandleAuthorize 处理基于重定向的 Ticket 签发。
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
		writeJSON(w, http.StatusUnauthorized, ErrorResponse(http.StatusUnauthorized, err.Error()))
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
	ticket, err := h.server.GenerateTicket(r.Context(), clientID, loginID, redirectURI, parseScopes(values.Get(params.Scope)), nil)
	if err != nil {
		writeJSON(w, statusFromError(err), ErrorResponse(statusFromError(err), err.Error()))
		return
	}
	if h.options.ServerOptions.EnableSLO && values.Get(params.Callback) != "" {
		if _, err = h.server.RegisterClientSession(r.Context(), loginID, clientID, values.Get(params.Callback)); err != nil {
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
	query.Set(params.Ticket, ticket.Ticket)
	if state := values.Get(params.Back); state != "" {
		query.Set(params.Back, state)
	}
	target.RawQuery = query.Encode()
	http.Redirect(w, r, target.String(), http.StatusFound)
}

// HandleToken handles ticket or code exchange and returns user identity JSON. HandleToken 处理 Ticket 或授权码交换并返回用户身份 JSON。
func (h *HTTPServer) HandleToken(w http.ResponseWriter, r *http.Request) {
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
	values := r.Form
	if err := h.verifySign(values); err != nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse(http.StatusUnauthorized, err.Error()))
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
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse(http.StatusMethodNotAllowed, "method not allowed"))
		return
	}
	if err := r.ParseForm(); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse(http.StatusBadRequest, err.Error()))
		return
	}
	values := r.Form
	if err := h.verifySign(values); err != nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse(http.StatusUnauthorized, err.Error()))
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
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse(http.StatusMethodNotAllowed, "method not allowed"))
		return
	}
	if err := r.ParseForm(); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse(http.StatusBadRequest, err.Error()))
		return
	}
	values := r.Form
	if err := h.verifySign(values); err != nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse(http.StatusUnauthorized, err.Error()))
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
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse(http.StatusMethodNotAllowed, "method not allowed"))
		return
	}
	if err := r.ParseForm(); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse(http.StatusBadRequest, err.Error()))
		return
	}
	values := r.Form
	if err := h.verifySign(values); err != nil {
		writeJSON(w, http.StatusUnauthorized, ErrorResponse(http.StatusUnauthorized, err.Error()))
		return
	}
	if err := h.revokeCredential(r, values); err != nil {
		writeJSON(w, statusFromError(err), ErrorResponse(statusFromError(err), err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, OKResponse(map[string]string{"result": ResultOK}))
}

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

func (h *HTTPServer) introspectCredential(r *http.Request, values url.Values) (*CredentialInfo, error) {
	params := h.options.ServerOptions.Params
	clientID := values.Get(params.Client)
	switch mode := Mode(values.Get(params.Mode)); mode {
	case "", ModeTicket:
		ticket, err := h.server.ValidateTicket(r.Context(), values.Get(params.Ticket))
		if err != nil {
			return inactiveCredential(), nil
		}
		return TicketCredentialInfo(ticket, h.ticketTTL(r, ticket.Ticket)), nil
	case ModeSharedToken:
		token, err := h.server.ValidateSharedToken(r.Context(), values.Get(params.TokenValue), clientID)
		if err != nil {
			return inactiveCredential(), nil
		}
		return SharedTokenCredentialInfo(token, h.sharedTokenTTL(r, token.Token)), nil
	case ModeRemoteSession:
		session, err := h.server.ValidateRemoteSession(r.Context(), values.Get(params.SessionID), clientID)
		if err != nil {
			return inactiveCredential(), nil
		}
		return RemoteSessionCredentialInfo(session, h.remoteSessionTTL(r, session.SessionID)), nil
	case ModeOAuth2:
		code, err := h.server.getOAuth2Code(r.Context(), values.Get(params.Code))
		if err != nil || code.ClientID != clientID || h.server.checkOAuth2CodeAlive(code) != nil {
			return inactiveCredential(), nil
		}
		return OAuth2CodeCredentialInfo(code, h.oauth2CodeTTL(r, code.Code)), nil
	default:
		return nil, ErrModeUnsupported
	}
}

func (h *HTTPServer) revokeCredential(r *http.Request, values url.Values) error {
	params := h.options.ServerOptions.Params
	switch mode := Mode(values.Get(params.Mode)); mode {
	case "", ModeTicket:
		return h.server.RevokeTicket(r.Context(), values.Get(params.Ticket))
	case ModeSharedToken:
		return h.server.RevokeSharedToken(r.Context(), values.Get(params.TokenValue))
	case ModeRemoteSession:
		return h.server.RevokeRemoteSession(r.Context(), values.Get(params.SessionID))
	case ModeOAuth2:
		return h.server.RevokeOAuth2Code(r.Context(), values.Get(params.Code))
	default:
		return ErrModeUnsupported
	}
}

func (h *HTTPServer) ticketTTL(r *http.Request, value string) int64 {
	ttl, _ := h.server.GetTicketTTL(r.Context(), value)
	return ttl
}

func (h *HTTPServer) sharedTokenTTL(r *http.Request, value string) int64 {
	ttl, _ := h.server.GetSharedTokenTTL(r.Context(), value)
	return ttl
}

func (h *HTTPServer) remoteSessionTTL(r *http.Request, value string) int64 {
	ttl, _ := h.server.GetRemoteSessionTTL(r.Context(), value)
	return ttl
}

func (h *HTTPServer) oauth2CodeTTL(r *http.Request, value string) int64 {
	ttl, _ := h.server.GetOAuth2CodeTTL(r.Context(), value)
	return ttl
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
	loginID := r.FormValue(h.options.ServerOptions.Params.LoginID)
	if loginID == "" {
		loginID, _ = h.options.LoginIDResolver(r)
	}
	if loginID != "" && h.options.ServerOptions.EnableSLO {
		if err := h.pushLogoutCallbacks(r, loginID); err != nil {
			writeJSON(w, statusFromError(err), ErrorResponse(statusFromError(err), err.Error()))
			return
		}
	}
	ClearLoginIDCookie(w, h.options.Cookie)
	writeJSON(w, http.StatusOK, OKResponse(map[string]string{"result": ResultOK}))
}

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
	if h.options.ServerOptions.CheckSign && h.options.ServerOptions.SecretKey != "" {
		values = NewSignerWithParams(h.options.ServerOptions.SecretKey, h.options.ServerOptions.Params).AttachSign(values)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, session.LogoutCallbackURL, strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := h.options.ServerOptions.LogoutHTTPClient
	if client == nil {
		client = http.DefaultClient
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

func (h *HTTPServer) verifySign(values url.Values) error {
	if !h.options.ServerOptions.CheckSign {
		return nil
	}
	if h.options.ServerOptions.SecretKey == "" {
		return nil
	}
	if !NewSignerWithParams(h.options.ServerOptions.SecretKey, h.options.ServerOptions.Params).Verify(values) {
		return ErrInvalidSign
	}
	return nil
}

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

func writeJSON(w http.ResponseWriter, status int, response Response) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

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
		errors.Is(err, ErrCallbackExpired):
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
	Name     string        // Name stores cookie name. Name 存储 Cookie 名称。
	Domain   string        // Domain stores shared cookie domain. Domain 存储共享 Cookie 域名。
	Path     string        // Path stores cookie path. Path 存储 Cookie 路径。
	MaxAge   time.Duration // MaxAge stores cookie lifetime. MaxAge 存储 Cookie 有效期。
	Secure   bool          // Secure restricts cookie to HTTPS. Secure 限制 Cookie 仅通过 HTTPS 发送。
	HTTPOnly bool          // HTTPOnly hides cookie from scripts. HTTPOnly 禁止脚本读取 Cookie。
	SameSite http.SameSite // SameSite stores browser same-site policy. SameSite 存储浏览器同站策略。
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
		cookie, err := r.Cookie(options.Name)
		if err != nil || cookie.Value == "" {
			return "", false
		}
		return cookie.Value, true
	}
}

// SetLoginIDCookie writes shared login cookie. SetLoginIDCookie 写入共享登录 Cookie。
func SetLoginIDCookie(w http.ResponseWriter, options CookieOptions, loginID string) {
	options = normalizeCookieOptions(options)
	http.SetCookie(w, &http.Cookie{
		Name:     options.Name,
		Value:    loginID,
		Path:     defaultCookiePath(options.Path),
		Domain:   options.Domain,
		MaxAge:   int(options.MaxAge.Seconds()),
		Secure:   options.Secure,
		HttpOnly: options.HTTPOnly,
		SameSite: options.SameSite,
	})
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

func defaultCookiePath(path string) string {
	if path == "" {
		return "/"
	}
	return path
}

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
