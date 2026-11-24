package sso

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"slices"
	"strings"
	"unicode"
)

// ErrSSOConfig identifies invalid static SSO integration configuration. ErrSSOConfig 标识无效的 SSO 静态接入配置。
var ErrSSOConfig = errors.New("invalid SSO configuration")

// ConfigurationError identifies a configuration field without including secret values. ConfigurationError 标识配置字段，不包含秘密值。
type ConfigurationError struct {
	// Field names the configuration location. Field 指定配置位置。
	Field string
	// Reason provides a fixed diagnostic description. Reason 提供固定的诊断说明。
	Reason string
	cause  error
}

// Error returns field-oriented diagnostics without echoing configured values. Error 返回面向字段的诊断，不回显配置值。
func (e *ConfigurationError) Error() string { return "SSO config " + e.Field + ": " + e.Reason }

// Unwrap preserves configuration and domain error classifications. Unwrap 保留配置及领域错误分类。
func (e *ConfigurationError) Unwrap() []error {
	if e.cause == nil {
		return []error{ErrSSOConfig}
	}
	return []error{ErrSSOConfig, e.cause}
}

// configError creates a bounded diagnostic with an optional existing error classification. configError 创建固定诊断，可保留已有错误分类。
func configError(field, reason string, cause error) error {
	return &ConfigurationError{Field: field, Reason: reason, cause: cause}
}

// validSSOMode recognizes implemented protocol modes. validSSOMode 识别已实现的协议模式。
func validSSOMode(mode Mode) bool {
	return mode == ModeTicket || mode == ModeOAuth2 || mode == ModeSharedToken || mode == ModeRemoteSession
}

// hasConfigWhitespace rejects whitespace and control characters in protocol identifiers. hasConfigWhitespace 拒绝协议标识中的空白和控制字符。
func hasConfigWhitespace(value string) bool {
	return strings.IndexFunc(value, func(ch rune) bool { return unicode.IsSpace(ch) || unicode.IsControl(ch) }) >= 0
}

// Validate checks normalized parameter names without changing the receiver. Validate 检查补齐默认值后的参数名，不修改接收值。
func (p ParamNames) Validate() error {
	if !distinctParamNames(normalizeParamNames(p)) {
		return configError("Params", "names must be distinct and contain no whitespace or controls", ErrInvalidRequest)
	}
	return nil
}

// Validate checks the six active center endpoints as distinct literal paths, not ServeMux patterns. Validate 将六个已启用中心端点检查为互异字面路径，不接受 ServeMux 模式。
func (e Endpoints) Validate() error {
	e = normalizeEndpoints(e)
	seen := make(map[string]bool)
	for _, endpoint := range []struct{ name, value string }{
		{"Authorize", e.Authorize}, {"Token", e.Token}, {"Introspect", e.Introspect},
		{"UserInfo", e.UserInfo}, {"Revoke", e.Revoke}, {"Logout", e.Logout},
	} {
		decoded, err := url.PathUnescape(endpoint.value)
		lower := strings.ToLower(endpoint.value)
		if err != nil || !strings.HasPrefix(decoded, "/") || strings.HasSuffix(decoded, "/") || path.Clean(decoded) != decoded || strings.ContainsAny(decoded, "{}\\?#") || hasConfigWhitespace(decoded) || strings.Contains(lower, "%2f") || strings.Contains(lower, "%5c") {
			return configError("Endpoints."+endpoint.name, "must be a clean absolute literal path without a trailing slash, query, fragment, or pattern", nil)
		}
		if seen[decoded] {
			return configError("Endpoints."+endpoint.name, "duplicates another center endpoint", nil)
		}
		seen[decoded] = true
	}
	return nil
}

// validConfigURL validates browser-facing HTTP(S) addresses without echoing parse errors. validConfigURL 校验面向浏览器的 HTTP(S) 地址，不回显解析错误。
func validConfigURL(value string) bool {
	u, err := url.Parse(value)
	return err == nil && normalizedHTTPOrigin(u) != "" && !strings.Contains(value, "#") && !strings.ContainsAny(u.Path, "\\") && !hasConfigWhitespace(value) && !hasConfigWhitespace(u.Path)
}

// Validate checks a client registration without writing storage; redirect lists may be empty for direct core API use. Validate 检查客户端注册信息且不写存储，直接使用核心 API 时回调列表可为空。
func (c *Client) Validate() error {
	if c == nil || c.ClientID == "" || hasConfigWhitespace(c.ClientID) {
		return configError("Client.ClientID", "must be nonempty without whitespace or controls", ErrClientOrClientIDEmpty)
	}
	if err := validateVerificationKey(c.SecretKey, c.VerificationKey, "Client.VerificationKey"); err != nil {
		return err
	}
	for _, mode := range c.Modes {
		if !validSSOMode(mode) {
			return configError("Client.Modes", "contains an unsupported mode", ErrModeUnsupported)
		}
	}
	for _, redirect := range c.RedirectURIs {
		if !validConfigURL(redirect) {
			return configError("Client.RedirectURIs", "must contain absolute HTTP(S) URLs without userinfo or fragments", ErrInvalidRedirectURI)
		}
	}
	for _, origin := range c.AllowOrigins {
		u, err := url.Parse(origin)
		if err != nil || !validConfigURL(origin) || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.ForceQuery {
			return configError("Client.AllowOrigins", "must contain HTTP(S) origins without paths or queries", ErrInvalidCallbackURL)
		}
	}
	for _, scope := range c.Scopes {
		if scope == "" || hasConfigWhitespace(scope) || strings.Contains(scope, ",") {
			return configError("Client.Scopes", "must contain nonempty scope names without separators", ErrInvalidScope)
		}
	}
	if c.RequirePKCE && len(c.Modes) != 0 && !slices.Contains(c.Modes, ModeTicket) && !slices.Contains(c.Modes, ModeOAuth2) {
		return configError("Client.RequirePKCE", "requires at least one one-time credential mode", ErrModeUnsupported)
	}
	return nil
}

// Validate checks HTTP configuration before registration, preserving existing nonpositive default rules. Validate 在注册前检查 HTTP 配置，保留已有非正值默认规则。
func (o HTTPOptions) Validate() error {
	if err := validateVerificationKey(o.SecretKey, o.VerificationKey, "VerificationKey"); err != nil {
		return err
	}
	if o.VerificationKey != "" && !o.CheckSign {
		return configError("VerificationKey", "requires CheckSign", nil)
	}
	if err := o.Endpoints.Validate(); err != nil {
		return err
	}
	if err := o.Params.Validate(); err != nil {
		return err
	}
	if o.Mode != "" && !validSSOMode(o.Mode) {
		return configError("Mode", "unsupported server mode", ErrModeUnsupported)
	}
	if err := o.ReplayProtection.validateConfig(o.CheckSign, normalizeParamNames(o.Params)); err != nil {
		return configError("ReplayProtection", "requires valid storage, signing, and parameters", err)
	}
	if o.LoginPageURL != "" && !validConfigURL(o.LoginPageURL) && !validBrowserReturnTo(o.LoginPageURL) {
		return configError("LoginPageURL", "must be a safe local path or absolute HTTP(S) URL", nil)
	}
	cookie := normalizeCookieOptions(o.Cookie)
	if cookie.SecretKey == "" {
		cookie.SecretKey = o.SecretKey
	}
	if err := validateVerificationKey(cookie.SecretKey, cookie.VerificationKey, "Cookie.VerificationKey"); err != nil {
		return err
	}
	if o.LoginIDResolver == nil && cookie.SecretKey == "" {
		return configError("LoginIDResolver", "provide a resolver or a signed center cookie key", ErrSignSecretRequired)
	}
	probe := &http.Cookie{Name: cookie.Name, Path: cookie.Path, Domain: cookie.Domain}
	if probe.Valid() != nil || !strings.HasPrefix(cookie.Path, "/") || cookie.SameSite < 0 || cookie.SameSite > http.SameSiteNoneMode || (cookie.SameSite == http.SameSiteNoneMode && !cookie.Secure) ||
		(strings.HasPrefix(cookie.Name, "__Host-") && (!cookie.Secure || cookie.Domain != "" || cookie.Path != "/")) || (strings.HasPrefix(cookie.Name, "__Secure-") && !cookie.Secure) {
		return configError("Cookie", "invalid cookie attributes or secure prefix combination", nil)
	}

	// Sort keys for stable first-error reporting without depending on map iteration order. 对键排序，避免首个错误受映射遍历顺序影响。
	keys := make([]string, 0, len(o.Clients))
	for key := range o.Clients {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	ids := make(map[string]bool)
	anonymousSigned := o.SecretKey != ""
	for _, key := range keys {
		client := o.Clients[key]
		if client.ClientID == "" {
			client.ClientID = key
		}
		if err := client.Validate(); err != nil {
			return err
		}
		if client.VerificationKey != "" && !o.CheckSign {
			return configError("Clients.VerificationKey", "requires CheckSign", nil)
		}
		if ids[client.ClientID] {
			return configError("Clients", "duplicate effective client IDs", nil)
		}
		ids[client.ClientID] = true
		if client.ClientID == ClientAnonymous && client.SecretKey != "" {
			anonymousSigned = true
		}
		if o.CheckSign && client.SecretKey == "" && o.SecretKey == "" {
			return configError("Clients.SecretKey", "known signed clients require a client key or fallback key", ErrSignSecretRequired)
		}
	}
	if o.AllowAnonymousClient {
		for _, target := range o.AllowURLs {
			if !validConfigURL(target) {
				return configError("AllowURLs", "must contain absolute HTTP(S) callback URLs", ErrInvalidRedirectURI)
			}
		}
		if o.CheckSign && !anonymousSigned {
			return configError("SecretKey", "anonymous signing requires a fallback or explicit anonymous client key", ErrSignSecretRequired)
		}
	}
	return nil
}

// Validate checks client helper configuration locally without sending requests. Validate 在本地检查客户端辅助配置，不发送请求。
func (c ClientConfig) Validate() error {
	c = NewClientApp(c).config
	if err := validateVerificationKey(c.SecretKey, c.VerificationKey, "VerificationKey"); err != nil {
		return err
	}
	if c.VerificationKey != "" && !c.CheckSign {
		return configError("VerificationKey", "requires CheckSign", nil)
	}
	if c.ClientID == "" || hasConfigWhitespace(c.ClientID) {
		return configError("ClientID", "must be nonempty without whitespace or controls", ErrClientOrClientIDEmpty)
	}
	if !validSSOMode(c.Mode) {
		return configError("Mode", "unsupported client mode", ErrModeUnsupported)
	}
	if err := c.Params.Validate(); err != nil {
		return err
	}
	if err := c.Endpoints.Validate(); err != nil {
		return err
	}
	if _, err := joinURL(c.ServerURL, c.Endpoints.Authorize); err != nil || !validConfigURL(c.ServerURL) {
		return configError("ServerURL", "must be an absolute HTTP(S) base URL without userinfo, query, or fragment", nil)
	}
	if c.CheckSign && c.SecretKey == "" {
		return configError("SecretKey", "required when CheckSign is enabled", ErrSignSecretRequired)
	}
	if err := c.ReplayProtection.validateConfig(c.CheckSign, c.Params); err != nil {
		return configError("ReplayProtection", "requires valid storage and signing", err)
	}
	if c.RegisterCallback && !validConfigURL(c.LogoutCallbackURL) {
		return configError("LogoutCallbackURL", "callback registration requires an absolute HTTP(S) URL", ErrInvalidCallbackURL)
	}
	return nil
}

// NewClientAppChecked validates configuration before creating a client helper. NewClientAppChecked 在创建客户端辅助对象前校验配置。
func NewClientAppChecked(config ClientConfig) (*ClientApp, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return NewClientApp(config), nil
}

// NewHTTPServerChecked validates all local settings before registering clients; storage failures can still leave partial registrations. NewHTTPServerChecked 注册客户端前校验所有本地配置，存储故障仍可能留下部分注册。
func NewHTTPServerChecked(server *Server, options HTTPOptions) (*HTTPServer, error) {
	if server == nil || server.storage == nil || server.serializer == nil {
		return nil, ErrServerNotInitialized
	}
	if err := options.Validate(); err != nil {
		return nil, err
	}
	h := NewHTTPServer(server, options)
	if err := h.InitializationError(); err != nil {
		return nil, err
	}
	return h, nil
}

// InitializationError exposes retained constructor failures without probing storage or remote services. InitializationError 公开保留的构造错误，不探测存储或远端服务。
func (h *HTTPServer) InitializationError() error { return h.initializationError() }

// RegisterClientChecked validates registration metadata before using the existing storage lifecycle. RegisterClientChecked 使用已有存储生命周期前校验注册元数据。
func (s *Server) RegisterClientChecked(client *Client) error {
	if s == nil || s.storage == nil || s.serializer == nil {
		return ErrServerNotInitialized
	}
	if err := client.Validate(); err != nil {
		return err
	}
	return s.RegisterClient(client)
}

// Validate exposes normalized client helper configuration checks without network access. Validate 公开规范化客户端辅助配置检查，不访问网络。
func (c *ClientApp) Validate() error {
	if c == nil {
		return fmt.Errorf("%w: %w", ErrSSOConfig, ErrServerNotInitialized)
	}
	return c.config.Validate()
}
