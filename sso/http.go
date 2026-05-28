// @Author daixk 2026/05/29
package sso

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
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
	options.Cookie = normalizeCookieOptions(options.Cookie)
	if options.LoginIDResolver == nil {
		options.LoginIDResolver = LoginIDFromCookie(options.Cookie)
	}
	if options.ServerOptions.Clients != nil && server != nil {
		_ = server.RegisterClients(options.ServerOptions.Clients)
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
		clientID = ClientAnonymous
	}
	ticket, err := h.server.GenerateTicket(r.Context(), clientID, loginID, redirectURI, parseScopes(values.Get(params.Scope)), nil)
	if err != nil {
		writeJSON(w, statusFromError(err), ErrorResponse(statusFromError(err), err.Error()))
		return
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

// HandleToken handles ticket exchange and returns user identity JSON. HandleToken 处理 Ticket 交换并返回用户身份 JSON。
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
	params := h.options.ServerOptions.Params
	ticket, err := h.server.ConsumeTicket(
		r.Context(),
		values.Get(params.Ticket),
		values.Get(params.Client),
		values.Get(params.ClientSecret),
		values.Get(params.Redirect),
	)
	if err != nil {
		writeJSON(w, statusFromError(err), ErrorResponse(statusFromError(err), err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, OKResponse(TicketResult(ticket)))
}

// HandleLogout clears optional shared cookie and returns success. HandleLogout 清除可选共享 Cookie 并返回成功。
func (h *HTTPServer) HandleLogout(w http.ResponseWriter, r *http.Request) {
	if h == nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse(http.StatusInternalServerError, ErrServerNotInitialized.Error()))
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse(http.StatusMethodNotAllowed, "method not allowed"))
		return
	}
	ClearLoginIDCookie(w, h.options.Cookie)
	writeJSON(w, http.StatusOK, OKResponse(map[string]string{"result": ResultOK}))
}

func (h *HTTPServer) verifySign(values url.Values) error {
	if !h.options.ServerOptions.CheckSign {
		return nil
	}
	if h.options.ServerOptions.SecretKey == "" {
		return nil
	}
	if !NewSignerWithParams(h.options.ServerOptions.SecretKey, h.options.ServerOptions.Params).Verify(values) {
		return errors.New("invalid sign")
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
	query.Set(h.options.ServerOptions.Params.Back, r.URL.String())
	target.RawQuery = query.Encode()
	http.Redirect(w, r, target.String(), http.StatusFound)
}

func writeJSON(w http.ResponseWriter, status int, response Response) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
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
		errors.Is(err, ErrModeUnsupported),
		errors.Is(err, ErrStorageCapabilityUnsupported):
		return http.StatusBadRequest
	case errors.Is(err, ErrInvalidClientCredentials):
		return http.StatusUnauthorized
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
