package sso

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"
	"unicode"
)

var (
	// ErrRemoteAuthConfig reports invalid remote authentication configuration. ErrRemoteAuthConfig 表示远程鉴权配置无效。
	ErrRemoteAuthConfig = errors.New("invalid SSO remote authentication configuration")
	// ErrRemoteCredential reports a missing, malformed, inactive, or expired credential. ErrRemoteCredential 表示凭证缺失、格式错误、失效或过期。
	ErrRemoteCredential = errors.New("SSO credential is missing or inactive")
	// ErrRemoteScope reports insufficient scopes on an otherwise valid credential. ErrRemoteScope 表示有效凭证缺少所需授权范围。
	ErrRemoteScope = errors.New("insufficient SSO scopes")
	// ErrRemoteUnavailable reports failed or inconsistent center validation; never authorize on this error. ErrRemoteUnavailable 表示中心校验失败或结果不一致，遇此错误不得放行。
	ErrRemoteUnavailable = errors.New("SSO remote validation unavailable")
)

// RemoteCredentialResolver reads a reusable credential from the application request; use ErrRemoteCredential for missing or invalid input. RemoteCredentialResolver 从应用请求读取可复用凭证，输入缺失或无效时使用 ErrRemoteCredential。
type RemoteCredentialResolver func(r *http.Request) (string, error)

// RemoteAuthErrorFunc handles a rejected request without calling the protected handler; errors may contain private upstream details. RemoteAuthErrorFunc 处理拒绝的请求且不得调用受保护处理器，错误可能包含上游私有详情。
type RemoteAuthErrorFunc func(w http.ResponseWriter, r *http.Request, err error)

// RemoteAuthOptions configures per-request center validation without caching or local session creation. RemoteAuthOptions 配置每次请求的中心校验，不缓存或创建本地会话。
type RemoteAuthOptions struct {
	// ResolveCredential defaults to RemoteBearerCredential; custom resolvers may read application-owned session storage. ResolveCredential 默认使用 RemoteBearerCredential，自定义解析器可读取应用管理的会话存储。
	ResolveCredential RemoteCredentialResolver
	// RequiredScopes requires every listed scope using exact case-sensitive matching. RequiredScopes 要求具备全部列出的授权范围，按大小写敏感的精确值匹配。
	RequiredScopes []string
	// Timeout bounds resolution and center validation; zero defaults to three seconds and negative is invalid. Timeout 限制凭证解析和中心校验时间，零默认三秒，负值无效。
	Timeout time.Duration
	// OnError overrides the bounded default error response and must not expose raw errors. OnError 覆盖默认固定错误响应，不得直接公开原始错误。
	OnError RemoteAuthErrorFunc
}

// RemoteAuth validates reusable SSO credentials on each protected request; share it across handlers. RemoteAuth 在每个受保护请求中校验可复用 SSO 凭证，可由多个处理器共用。
type RemoteAuth struct {
	app     *ClientApp
	options RemoteAuthOptions
}

// NewRemoteAuth validates configuration without network access or taking ownership of the HTTP client. NewRemoteAuth 校验配置，不访问网络或接管 HTTP 客户端生命周期。
func NewRemoteAuth(app *ClientApp, options RemoteAuthOptions) (*RemoteAuth, error) {
	if app == nil || app.config.ClientID == "" || options.Timeout < 0 || !distinctParamNames(app.config.Params) {
		return nil, ErrRemoteAuthConfig
	}
	if app.config.Mode != ModeRemoteSession && app.config.Mode != ModeSharedToken {
		return nil, ErrModeUnsupported
	}
	if err := app.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrRemoteAuthConfig, err)
	}
	for _, scope := range options.RequiredScopes {
		if scope == "" || strings.IndexFunc(scope, func(ch rune) bool { return unicode.IsSpace(ch) || unicode.IsControl(ch) }) >= 0 {
			return nil, ErrRemoteAuthConfig
		}
	}
	options.RequiredScopes = slices.Clone(options.RequiredScopes)
	if options.ResolveCredential == nil {
		options.ResolveCredential = RemoteBearerCredential
	}
	if options.Timeout == 0 {
		options.Timeout = 3 * time.Second
	}
	return &RemoteAuth{app: app, options: options}, nil
}

// RemoteBearerCredential accepts one Authorization Bearer header and never reads URL parameters or cookies. RemoteBearerCredential 接受唯一的 Authorization Bearer 请求头，不读取 URL 参数或 Cookie。
func RemoteBearerCredential(r *http.Request) (string, error) {
	if r == nil || len(r.Header.Values("Authorization")) != 1 {
		return "", ErrRemoteCredential
	}
	value := r.Header.Get("Authorization")
	scheme, token, found := strings.Cut(value, " ")
	if !found || !strings.EqualFold(scheme, "Bearer") {
		return "", ErrRemoteCredential
	}
	token = strings.TrimLeft(token, " ")
	if token == "" {
		return "", ErrRemoteCredential
	}

	// Bearer grammar rejects combined headers, control characters, and trailing data. Bearer 语法拒绝合并请求头、控制字符和尾随数据。
	padding := false
	for _, ch := range token {
		if ch == '=' {
			padding = true
			continue
		}
		if padding || !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || strings.ContainsRune("-._~+/", ch)) {
			return "", ErrRemoteCredential
		}
	}
	if token[0] == '=' {
		return "", ErrRemoteCredential
	}
	return token, nil
}

// Authenticate validates current center state and all required scopes without consuming or renewing the credential. Authenticate 校验当前中心状态及全部所需范围，不消费或续期凭证。
func (a *RemoteAuth) Authenticate(r *http.Request) (*CredentialInfo, error) {
	if a == nil || a.app == nil || a.options.ResolveCredential == nil || r == nil {
		return nil, ErrRemoteAuthConfig
	}
	ctx, cancel := context.WithTimeout(r.Context(), a.options.Timeout)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrRemoteUnavailable, err)
	}
	credential, err := a.options.ResolveCredential(r.WithContext(ctx))
	if ctx.Err() != nil {
		return nil, fmt.Errorf("%w: %w", ErrRemoteUnavailable, ctx.Err())
	}
	if err != nil {
		if errors.Is(err, ErrRemoteCredential) {
			return nil, ErrRemoteCredential
		}
		return nil, fmt.Errorf("%w: %w", ErrRemoteUnavailable, err)
	}
	if credential == "" {
		return nil, ErrRemoteCredential
	}

	request := CredentialRequest{Mode: a.app.config.Mode}
	if request.Mode == ModeRemoteSession {
		request.SessionID = credential
	} else {
		request.TokenValue = credential
	}
	started := time.Now()
	info, err := a.app.Introspect(ctx, request)
	if ctx.Err() != nil {
		return nil, fmt.Errorf("%w: %w", ErrRemoteUnavailable, ctx.Err())
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrRemoteUnavailable, err)
	}
	if info == nil || !info.Active {
		return nil, ErrRemoteCredential
	}
	if info.ClientID != a.app.config.ClientID || info.Mode != request.Mode || info.LoginID == "" || info.ExpiresIn < -1 {
		return nil, ErrRemoteUnavailable
	}

	// Charge request latency conservatively and compare seconds without overflowing a duration. 保守计入请求耗时，按秒比较以避免 duration 溢出。
	if info.ExpiresIn == 0 || (info.ExpiresIn > 0 && int64(time.Since(started)/time.Second) >= info.ExpiresIn) {
		return nil, ErrRemoteCredential
	}
	for _, required := range a.options.RequiredScopes {
		if !slices.Contains(info.Scopes, required) {
			return nil, ErrRemoteScope
		}
	}
	return info, nil
}

// remoteIdentityKey isolates validated SSO identity from other authentication contexts. remoteIdentityKey 将已校验 SSO 身份与其他认证上下文隔离。
type remoteIdentityKey struct{}

// RemoteIdentityFromContext reads the validated request identity; treat the returned identity and metadata as read-only. RemoteIdentityFromContext 读取已校验的请求身份，返回的身份及元数据应只读。
func RemoteIdentityFromContext(ctx context.Context) (*CredentialInfo, bool) {
	if ctx == nil {
		return nil, false
	}
	info, ok := ctx.Value(remoteIdentityKey{}).(*CredentialInfo)
	return info, ok && info != nil
}

// Middleware validates each request before injecting identity; the validation timeout does not shorten the business request context. Middleware 每次校验成功后才注入身份，鉴权超时不缩短业务请求上下文。
func (a *RemoteAuth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		browserHeaders(w)
		if next == nil {
			a.writeError(w, r, ErrRemoteAuthConfig)
			return
		}
		info, err := a.Authenticate(r)
		if err != nil {
			a.writeError(w, r, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), remoteIdentityKey{}, info)))
	})
}

// writeError keeps raw transport bodies and credentials out of the default public response. writeError 避免默认公开响应泄露传输响应体与凭证。
func (a *RemoteAuth) writeError(w http.ResponseWriter, r *http.Request, err error) {
	if a != nil && a.options.OnError != nil {
		a.options.OnError(w, r, err)
		return
	}
	status, message := http.StatusServiceUnavailable, "SSO validation unavailable"
	switch {
	case errors.Is(err, ErrRemoteUnavailable):
		// Upstream causes must not override the outer failure classification. 上游原因不能覆盖外层失败分类。
	case errors.Is(err, ErrRemoteAuthConfig):
		status, message = http.StatusInternalServerError, "SSO authentication is not configured"
	case errors.Is(err, ErrRemoteCredential):
		status, message = http.StatusUnauthorized, "SSO authentication required"
	case errors.Is(err, ErrRemoteScope):
		status, message = http.StatusForbidden, "insufficient SSO scopes"
	}
	writeJSON(w, status, ErrorResponse(status, message))
}
