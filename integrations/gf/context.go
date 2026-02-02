package gf

import (
	"net/http"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/gogf/gf/v2/net/ghttp"
)

// GFContext is the GoFrame context adapter that implements adapter.RequestContext.
// GFContext 是 GoFrame 上下文适配器，实现了 adapter.RequestContext 接口。
type GFContext struct {
	c       *ghttp.Request
	aborted bool
}

// NewGFContext creates a new GF context adapter.
// NewGFContext 创建一个新的 GF 上下文适配器。
func NewGFContext(c *ghttp.Request) adapter.RequestContext {
	return &GFContext{
		c: c,
	}
}

// Get implements adapter.RequestContext.
// Get 实现 adapter.RequestContext 接口。
func (g *GFContext) Get(key string) (interface{}, bool) {
	v := g.c.Get(key)
	return v, v.IsNil()
}

// GetClientIP implements adapter.RequestContext.
// GetClientIP 实现 adapter.RequestContext 接口。
func (g *GFContext) GetClientIP() string {
	return g.c.GetClientIp()
}

// GetCookie implements adapter.RequestContext.
// GetCookie 实现 adapter.RequestContext 接口。
func (g *GFContext) GetCookie(key string) string {
	return g.c.Cookie.Get(key).String()
}

// GetHeader implements adapter.RequestContext.
// GetHeader 实现 adapter.RequestContext 接口。
func (g *GFContext) GetHeader(key string) string {
	return g.c.Header.Get(key)
}

// GetMethod implements adapter.RequestContext.
// GetMethod 实现 adapter.RequestContext 接口。
func (g *GFContext) GetMethod() string {
	return g.c.Method
}

// GetPath implements adapter.RequestContext.
// GetPath 实现 adapter.RequestContext 接口。
func (g *GFContext) GetPath() string {
	return g.c.Request.URL.Path
}

// GetQuery implements adapter.RequestContext.
// GetQuery 实现 adapter.RequestContext 接口。
func (g *GFContext) GetQuery(key string) string {
	return g.c.Request.URL.Query().Get(key)
}

// Set implements adapter.RequestContext.
// Set 实现 adapter.RequestContext 接口。
func (g *GFContext) Set(key string, value interface{}) {
	g.c.SetCtxVar(key, value)
}

// SetCookie implements adapter.RequestContext.
// SetCookie 实现 adapter.RequestContext 接口。
func (g *GFContext) SetCookie(name string, value string, maxAge int, path string, domain string, secure bool, httpOnly bool) {
	g.c.Cookie.SetHttpCookie(&http.Cookie{
		Name:     name,
		Value:    value,
		MaxAge:   maxAge,
		Path:     path,
		Domain:   domain,
		Secure:   secure,
		HttpOnly: httpOnly,
	})
}

// SetHeader implements adapter.RequestContext.
// SetHeader 实现 adapter.RequestContext 接口。
func (g *GFContext) SetHeader(key string, value string) {
	g.c.Header.Set(key, value)
}

// ============================================================================
// Additional Required Methods - 额外必需的方法
// ============================================================================

// GetHeaders implements adapter.RequestContext.
// GetHeaders 实现 adapter.RequestContext 接口。
func (g *GFContext) GetHeaders() map[string][]string {
	return g.c.Header
}

// GetQueryAll implements adapter.RequestContext.
// GetQueryAll 实现 adapter.RequestContext 接口。
func (g *GFContext) GetQueryAll() map[string][]string {
	return g.c.Request.URL.Query()
}

// GetPostForm implements adapter.RequestContext.
// GetPostForm 实现 adapter.RequestContext 接口。
func (g *GFContext) GetPostForm(key string) string {
	return g.c.GetForm(key).String()
}

// GetBody implements adapter.RequestContext.
// GetBody 实现 adapter.RequestContext 接口。
func (g *GFContext) GetBody() ([]byte, error) {
	body := g.c.GetBody()
	return body, nil
}

// GetURL implements adapter.RequestContext.
// GetURL 实现 adapter.RequestContext 接口。
func (g *GFContext) GetURL() string {
	return g.c.Request.URL.String()
}

// GetUserAgent implements adapter.RequestContext.
// GetUserAgent 实现 adapter.RequestContext 接口。
func (g *GFContext) GetUserAgent() string {
	return g.c.Header.Get("User-Agent")
}

// SetCookieWithOptions implements adapter.RequestContext.
// SetCookieWithOptions 实现 adapter.RequestContext 接口。
func (g *GFContext) SetCookieWithOptions(options *adapter.CookieOptions) {
	cookie := &http.Cookie{
		Name:     options.Name,
		Value:    options.Value,
		MaxAge:   options.MaxAge,
		Path:     options.Path,
		Domain:   options.Domain,
		Secure:   options.Secure,
		HttpOnly: options.HttpOnly,
		SameSite: http.SameSite(0), // Default to SameSiteNone
	}

	// Set SameSite attribute
	// 设置 SameSite 属性
	switch options.SameSite {
	case "Strict":
		cookie.SameSite = http.SameSiteStrictMode
	case "Lax":
		cookie.SameSite = http.SameSiteLaxMode
	case "None":
		cookie.SameSite = http.SameSiteNoneMode
	}

	g.c.Cookie.SetHttpCookie(cookie)
}

// GetString implements adapter.RequestContext.
// GetString 实现 adapter.RequestContext 接口。
func (g *GFContext) GetString(key string) string {
	v := g.c.Get(key)
	return v.String()
}

// MustGet implements adapter.RequestContext.
// MustGet 实现 adapter.RequestContext 接口。
func (g *GFContext) MustGet(key string) any {
	v := g.c.Get(key)
	if v.IsNil() {
		panic("key not found: " + key)
	}
	return v
}

// Abort implements adapter.RequestContext.
// Abort 实现 adapter.RequestContext 接口。
func (g *GFContext) Abort() {
	g.aborted = true
}

// IsAborted implements adapter.RequestContext.
// IsAborted 实现 adapter.RequestContext 接口。
func (g *GFContext) IsAborted() bool {
	return g.aborted
}

// IsTLS implements adapter.RequestContext.
// IsTLS 实现 adapter.RequestContext 接口。
func (g *GFContext) IsTLS() bool {
	return g.c.TLS != nil
}

// SetStatusCode implements adapter.RequestContext.
// SetStatusCode 实现 adapter.RequestContext 接口。
func (g *GFContext) SetStatusCode(code int) {
	g.c.Response.WriteStatus(code)
}

// Write implements adapter.RequestContext.
// Write 实现 adapter.RequestContext 接口。
func (g *GFContext) Write(data []byte) (int, error) {
	g.c.Response.Write(data)
	return len(data), nil
}
