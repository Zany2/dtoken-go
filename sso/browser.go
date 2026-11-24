package sso

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Zany2/dtoken-go/contract"
)

var (
	// ErrBrowserConfig reports incomplete browser integration settings. ErrBrowserConfig 表示浏览器接入配置不完整。
	ErrBrowserConfig = errors.New("invalid sso browser client configuration")
	// ErrInvalidLoginState reports missing, expired, consumed, or unbound login state. ErrInvalidLoginState 表示登录状态缺失、过期、已消费或浏览器绑定无效。
	ErrInvalidLoginState = errors.New("invalid login state; restart login")
	// ErrInvalidReturnTo reports an unsafe application return path. ErrInvalidReturnTo 表示应用返回路径不安全。
	ErrInvalidReturnTo = errors.New("invalid application return path")
	// ErrBrowserExchange reports a failed center credential exchange or lookup. ErrBrowserExchange 表示中心凭证兑换或查询失败。
	ErrBrowserExchange = errors.New("sso browser credential exchange failed")
)

// BrowserClientOptions configures browser state while leaving local sessions application-owned. BrowserClientOptions 配置浏览器状态，本地会话仍由应用管理。
type BrowserClientOptions struct {
	// PKCE enables automatic S256 binding for Ticket and OAuth2 code login flows. PKCE 为 Ticket 与 OAuth2 授权码登录流程启用自动 S256 绑定。
	PKCE bool
	// Storage retains pending flows and is borrowed, never closed by BrowserClient. Storage 保存待完成流程，由组件借用且不会被组件关闭。
	Storage contract.Storage
	// KeyPrefix isolates deployments; empty derives a namespace from client ID and callback URL. KeyPrefix 隔离部署，空值按客户端 ID 和回调地址生成命名空间。
	KeyPrefix string
	// CallbackURL is the registered absolute HTTP(S) callback without query or fragment. CallbackURL 是已注册且不含查询串和片段的 HTTP(S) 绝对回调地址。
	CallbackURL string
	// StateCookie supplies the signing key and cookie prefix; MaxAge defaults to five minutes. StateCookie 提供签名密钥及 Cookie 名前缀，MaxAge 默认五分钟。
	StateCookie CookieOptions
	// SessionMaxAge caps local sessions; zero defaults to two hours and negative is invalid. SessionMaxAge 限制本地会话时长，零值默认两小时，负值无效。
	SessionMaxAge time.Duration
}

// BrowserLoginResult provides verified identity, safe navigation, and a bounded session deadline. BrowserLoginResult 提供已验证身份、安全返回路径及受限会话截止时间。
type BrowserLoginResult struct {
	// Identity contains center-validated identity and metadata. Identity 包含中心验证后的身份和元数据。
	Identity TicketExchangeResult
	// ReturnTo is a validated application-local path retained during login. ReturnTo 是登录期间保存且已校验的应用内路径。
	ReturnTo string
	// ExpiresAt bounds the application session, including center request latency. ExpiresAt 限制应用会话截止时间，并计入中心请求耗时。
	ExpiresAt time.Time
}

// BrowserLoginFunc establishes a local session before returning, retiring the previous session only after success. BrowserLoginFunc 返回前建立本地会话，成功后才撤销旧会话。
type BrowserLoginFunc func(http.ResponseWriter, *http.Request, *BrowserLoginResult) error

// BrowserLogoutFunc revokes the current local session and clears its cookie, tolerating repeated calls. BrowserLogoutFunc 撤销当前本地会话并清除 Cookie，应允许重复调用。
type BrowserLogoutFunc func(http.ResponseWriter, *http.Request) error

// BrowserClient composes browser login and logout on top of ClientApp; share an instance for ordinary storage. BrowserClient 基于 ClientApp 组装浏览器登录与退出，普通存储应共用同一实例。
type BrowserClient struct {
	app     *ClientApp
	options BrowserClientOptions
	origin  string
	mu      sync.Mutex
}

// NewBrowserClient validates integration settings without taking ownership of storage. NewBrowserClient 校验接入配置，不接管存储生命周期。
func NewBrowserClient(app *ClientApp, options BrowserClientOptions) (*BrowserClient, error) {
	if app == nil || options.Storage == nil || app.config.ClientID == "" || options.StateCookie.SecretKey == "" || options.StateCookie.MaxAge < 0 || options.SessionMaxAge < 0 {
		return nil, ErrBrowserConfig
	}
	if !distinctParamNames(app.config.Params) {
		return nil, ErrBrowserConfig
	}
	switch app.config.Mode {
	case ModeTicket, ModeOAuth2, ModeSharedToken, ModeRemoteSession:
	default:
		return nil, ErrModeUnsupported
	}
	if options.PKCE && app.config.Mode != ModeTicket && app.config.Mode != ModeOAuth2 {
		return nil, ErrModeUnsupported
	}
	callback, err := url.Parse(options.CallbackURL)
	if err != nil || !validConfigURL(options.CallbackURL) || callback.RawQuery != "" || callback.ForceQuery {
		return nil, ErrBrowserConfig
	}
	if options.StateCookie.Domain != "" {
		return nil, ErrBrowserConfig
	}
	if options.StateCookie.Name == "" {
		options.StateCookie.Name = "dtoken_sso_state"
	}
	if options.StateCookie.Path == "" {
		options.StateCookie.Path = defaultCookiePath(callback.Path)
	}
	if options.StateCookie.MaxAge == 0 {
		options.StateCookie.MaxAge = 5 * time.Minute
	}
	options.StateCookie.HTTPOnly = true
	if callback.Scheme == "https" {
		options.StateCookie.Secure = true
	}
	if options.StateCookie.SameSite == 0 {
		options.StateCookie.SameSite = http.SameSiteLaxMode
	}
	if options.StateCookie.SameSite < http.SameSiteDefaultMode || options.StateCookie.SameSite > http.SameSiteNoneMode || options.StateCookie.SameSite == http.SameSiteStrictMode || (options.StateCookie.SameSite == http.SameSiteNoneMode && !options.StateCookie.Secure) {
		return nil, ErrBrowserConfig
	}
	if (strings.HasPrefix(options.StateCookie.Name, "__Host-") && (!options.StateCookie.Secure || options.StateCookie.Path != "/")) || (strings.HasPrefix(options.StateCookie.Name, "__Secure-") && !options.StateCookie.Secure) {
		return nil, ErrBrowserConfig
	}
	probe := &http.Cookie{Name: options.StateCookie.Name + "_" + rand.Text(), Path: options.StateCookie.Path}
	if err := probe.Valid(); err != nil || !strings.HasPrefix(options.StateCookie.Path, "/") || !browserCookiePathMatches(callback.Path, options.StateCookie.Path) {
		return nil, ErrBrowserConfig
	}
	if options.SessionMaxAge == 0 {
		options.SessionMaxAge = 2 * time.Hour
	}
	if options.KeyPrefix == "" {
		digest := sha256.Sum256([]byte(url.Values{"client": {app.config.ClientID}, "callback": {options.CallbackURL}}.Encode()))
		options.KeyPrefix = "dtoken:sso:browser:" + hex.EncodeToString(digest[:]) + ":"
	}

	// Validate signing and parameter wiring before storing any browser state. 保存任何浏览器状态前验证签名与参数配置。
	if err := app.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrBrowserConfig, err)
	}
	return &BrowserClient{app: app, options: options, origin: normalizedHTTPOrigin(callback)}, nil
}

// BeginLogin stores a one-time browser-bound flow and redirects to the center; returnTo must be application-local. BeginLogin 保存一次性浏览器绑定流程并跳转中心，returnTo 必须为应用内路径。
func (b *BrowserClient) BeginLogin(w http.ResponseWriter, r *http.Request, returnTo string, extra url.Values) error {
	browserHeaders(w)
	if err := r.Context().Err(); err != nil {
		return err
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		return ErrMethodNotAllowed
	}
	if returnTo == "" {
		returnTo = "/"
	}
	if !validBrowserReturnTo(returnTo) {
		return ErrInvalidReturnTo
	}
	state, binding := rand.Text(), rand.Text()
	values := cloneValues(extra)

	// The flow owns its proof; callers cannot substitute challenge parameters. 流程自行管理证明，调用方不能替换挑战参数。
	for _, key := range []string{b.app.config.Params.CodeChallenge, b.app.config.Params.CodeChallengeMethod, b.app.config.Params.CodeVerifier} {
		if _, exists := values[key]; exists {
			return ErrInvalidPKCE
		}
	}
	values.Set(b.app.config.Params.Back, state)
	var target, verifier string
	var err error
	if b.options.PKCE {
		verifier = NewPKCEVerifier()
		target, err = b.app.AuthURLWithPKCE(b.options.CallbackURL, verifier, values)
	} else {
		target, err = b.app.AuthURL(b.options.CallbackURL, values)
	}
	if err != nil {
		return err
	}
	record := browserLoginState{Binding: browserBindingHash(binding), ReturnTo: returnTo, ExpiresAt: time.Now().Add(b.options.StateCookie.MaxAge), CodeVerifier: verifier}
	payload, err := json.Marshal(record)
	if err != nil {
		return err
	}
	if err := b.options.Storage.Set(r.Context(), b.options.KeyPrefix+state, string(payload), b.options.StateCookie.MaxAge); err != nil {
		return fmt.Errorf("%w: %w", ErrStorageUnavailable, err)
	}
	if err := r.Context().Err(); err != nil {
		return err
	}
	SetLoginIDCookie(w, b.stateCookie(state), binding)
	http.Redirect(w, r, target, http.StatusFound)
	return nil
}

// CompleteLogin consumes valid browser state before contacting the center; failures require a new login flow. CompleteLogin 在联系中心前消费有效浏览器状态，失败后必须重新发起登录。
func (b *BrowserClient) CompleteLogin(w http.ResponseWriter, r *http.Request) (*BrowserLoginResult, error) {
	browserHeaders(w)
	if err := r.Context().Err(); err != nil {
		return nil, err
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		return nil, ErrMethodNotAllowed
	}
	values, err := parseRequestQuery(r, b.app.config.RequestLimits)
	if err != nil {
		return nil, err
	}
	params := b.app.config.Params
	state := values.Get(params.Back)
	if len(values[params.Back]) != 1 || !validBrowserStateID(state) {
		return nil, ErrInvalidLoginState
	}
	req := CredentialRequest{Mode: b.app.config.Mode, RedirectURI: b.options.CallbackURL}
	credentialKey := map[Mode]string{ModeTicket: params.Ticket, ModeOAuth2: params.Code, ModeSharedToken: params.TokenValue, ModeRemoteSession: params.SessionID}[req.Mode]
	for _, key := range []string{params.Ticket, params.Code, params.TokenValue, params.SessionID} {
		if key == credentialKey {
			if len(values[key]) != 1 || values.Get(key) == "" {
				return nil, ErrInvalidRequest
			}
		} else if len(values[key]) != 0 {
			return nil, ErrInvalidRequest
		}
	}
	req.Ticket, req.Code = values.Get(params.Ticket), values.Get(params.Code)
	req.TokenValue, req.SessionID = values.Get(params.TokenValue), values.Get(params.SessionID)
	if err := validateScalarParameters(values, params); err != nil {
		return nil, err
	}
	record, err := b.consumeState(r, state)
	if err != nil {
		return nil, err
	}
	ClearLoginIDCookie(w, b.stateCookie(state))
	req.CodeVerifier = record.CodeVerifier

	started := time.Now()
	var identity *TicketExchangeResult
	if req.Mode == ModeTicket || req.Mode == ModeOAuth2 {
		identity, err = b.app.ExchangeCredential(r.Context(), req)
	} else {
		var info *CredentialInfo
		info, err = b.app.UserInfo(r.Context(), req)
		if err == nil {
			if !info.Active || info.ClientID != b.app.config.ClientID || info.Mode != req.Mode || info.ExpiresIn < -1 || info.ExpiresIn == 0 {
				err = ErrInvalidRequest
			} else {
				identity = &TicketExchangeResult{LoginID: info.LoginID, CenterID: info.LoginID, Scopes: info.Scopes, Extra: info.Extra, RemainTokenTimeout: info.ExpiresIn}
			}
		}
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrBrowserExchange, err)
	}
	if err := r.Context().Err(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrBrowserExchange, err)
	}
	expiresAt, err := browserSessionDeadline(identity, started, b.options.SessionMaxAge)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrBrowserExchange, err)
	}
	return &BrowserLoginResult{Identity: *identity, ReturnTo: record.ReturnTo, ExpiresAt: expiresAt}, nil
}

// CallbackHandler establishes an application session then redirects; hooks must not write a response body or redirect. CallbackHandler 建立应用会话后跳转，钩子不得写响应体或自行跳转。
func (b *BrowserClient) CallbackHandler(login BrowserLoginFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if login == nil {
			browserWriteError(w, ErrBrowserConfig)
			return
		}
		result, err := b.CompleteLogin(w, r)
		if err != nil {
			browserWriteError(w, err)
			return
		}
		if err := login(w, r, result); err != nil {
			browserWriteError(w, err)
			return
		}
		if err := r.Context().Err(); err != nil {
			browserWriteError(w, err)
			return
		}
		http.Redirect(w, r, result.ReturnTo, http.StatusSeeOther)
	}
}

// LogoutHandler requires same-origin POST, revokes local state, then redirects locally or to center logout. LogoutHandler 要求同源 POST，撤销本地状态后跳转本地页面或中心注销入口。
func (b *BrowserClient) LogoutHandler(logout BrowserLogoutFunc, global bool, returnTo string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		browserHeaders(w)
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			browserWriteError(w, ErrMethodNotAllowed)
			return
		}
		if len(r.Header.Values("Origin")) != 1 || browserRequestOrigin(r.Header.Get("Origin")) != b.origin || r.Header.Get("Sec-Fetch-Site") == "cross-site" {
			http.Error(w, "invalid logout origin", http.StatusForbidden)
			return
		}
		if logout == nil {
			browserWriteError(w, ErrBrowserConfig)
			return
		}
		target := returnTo
		if target == "" {
			target = "/"
		}
		if !validBrowserReturnTo(target) {
			browserWriteError(w, ErrInvalidReturnTo)
			return
		}
		if global {
			var err error
			target, err = b.app.SignoutURL("", nil)
			if err != nil {
				browserWriteError(w, err)
				return
			}
		}
		if err := r.Context().Err(); err != nil {
			browserWriteError(w, err)
			return
		}
		if err := logout(w, r); err != nil {
			browserWriteError(w, err)
			return
		}
		if err := r.Context().Err(); err != nil {
			browserWriteError(w, err)
			return
		}
		http.Redirect(w, r, target, http.StatusSeeOther)
	}
}

// browserSessionDeadline limits local login without overflowing remote seconds. browserSessionDeadline 限制本地登录期限且避免远端秒数溢出。
func browserSessionDeadline(identity *TicketExchangeResult, started time.Time, maxAge time.Duration) (time.Time, error) {
	if identity == nil || identity.LoginID == "" {
		return time.Time{}, ErrUserIDEmpty
	}
	for _, seconds := range []int64{identity.RemainTokenTimeout, identity.RemainSessionTimeout} {
		if seconds < -1 {
			return time.Time{}, ErrInvalidRequest
		}
		if seconds > 0 && seconds <= int64(maxAge/time.Second) {
			maxAge = min(maxAge, time.Duration(seconds)*time.Second)
		}
	}
	deadline := started.Add(maxAge)
	if !time.Now().Before(deadline) {
		return time.Time{}, ErrRequestExpired
	}
	return deadline, nil
}

// validBrowserReturnTo rejects external, ambiguous, and control-containing redirect targets. validBrowserReturnTo 拒绝外部、歧义及含控制字符的跳转目标。
func validBrowserReturnTo(value string) bool {
	u, err := url.Parse(value)
	if err != nil || u.IsAbs() || u.Host != "" || u.Opaque != "" || !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") || strings.HasPrefix(u.Path, "//") {
		return false
	}
	decoded, err := url.PathUnescape(value)
	if err != nil {
		return false
	}
	for _, ch := range decoded {
		if ch == '\\' || ch < 0x20 || ch == 0x7f {
			return false
		}
	}
	return true
}

// browserCookiePathMatches follows browser cookie path matching. browserCookiePathMatches 遵循浏览器 Cookie 路径匹配规则。
func browserCookiePathMatches(path, cookiePath string) bool {
	path = defaultCookiePath(path)
	return path == cookiePath || (strings.HasPrefix(path, cookiePath) && (strings.HasSuffix(cookiePath, "/") || strings.HasPrefix(strings.TrimPrefix(path, cookiePath), "/")))
}

// browserHeaders prevents caching and callback credential referrer leakage. browserHeaders 防止缓存及回调凭证通过来源地址泄露。
func browserHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Referrer-Policy", "no-referrer")
}

// browserWriteError returns bounded public errors without exposing upstream bodies or credentials. browserWriteError 返回固定公开错误，不泄露上游响应体或凭证。
func browserWriteError(w http.ResponseWriter, err error) {
	browserHeaders(w)
	status, message := http.StatusInternalServerError, "sso browser operation failed"
	switch {
	case errors.Is(err, ErrRequestTooLarge):
		status, message = http.StatusRequestEntityTooLarge, ErrRequestTooLarge.Error()
	case errors.Is(err, ErrBrowserExchange):
		status, message = http.StatusBadGateway, "sso exchange failed; restart login"
	case errors.Is(err, ErrInvalidLoginState):
		status, message = http.StatusBadRequest, ErrInvalidLoginState.Error()
	case errors.Is(err, ErrInvalidReturnTo), errors.Is(err, ErrInvalidRequest), errors.Is(err, ErrInvalidPKCE):
		status, message = http.StatusBadRequest, "invalid browser request"
	case errors.Is(err, ErrMethodNotAllowed):
		status, message = http.StatusMethodNotAllowed, "method not allowed"
	}
	http.Error(w, message, status)
}
