// @Author daixk 2025/12/22 15:56:00
package gin

import (
	"net/http"
	"net/url"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/gin-gonic/gin"
)

// GinContext adapts request context 适配 Gin 请求上下文
type GinContext struct {
	c *gin.Context
}

// Interface assertion keeps request context contract checked at compile time 接口断言在编译期检查请求上下文契约
var _ adapter.RequestContext = (*GinContext)(nil)

// NewGinContext creates request context adapter 创建请求上下文适配器
func NewGinContext(c *gin.Context) adapter.RequestContext {
	return &GinContext{
		c: c,
	}
}

// Get implements adapter.RequestContext 实现 adapter.RequestContext 接口
func (g *GinContext) Get(key string) (interface{}, bool) {
	return g.c.Get(key)
}

// GetClientIP implements adapter.RequestContext 实现 adapter.RequestContext 接口
func (g *GinContext) GetClientIP() string {
	return g.c.ClientIP()
}

// GetCookie implements adapter.RequestContext 实现 adapter.RequestContext 接口
func (g *GinContext) GetCookie(key string) string {
	cookie, _ := g.c.Cookie(key)
	return cookie
}

// GetHeader implements adapter.RequestContext 实现 adapter.RequestContext 接口
func (g *GinContext) GetHeader(key string) string {
	return g.c.GetHeader(key)
}

// GetMethod implements adapter.RequestContext 实现 adapter.RequestContext 接口
func (g *GinContext) GetMethod() string {
	return g.c.Request.Method
}

// GetPath implements adapter.RequestContext 实现 adapter.RequestContext 接口
func (g *GinContext) GetPath() string {
	return g.c.Request.URL.Path
}

// GetQuery implements adapter.RequestContext 实现 adapter.RequestContext 接口
func (g *GinContext) GetQuery(key string) string {
	// Gin's query cache can outlive a replaced request. Gin 的查询缓存可能在请求替换后仍保留旧值，因此直接读取当前 URL。
	return g.c.Request.URL.Query().Get(key)
}

// Set implements adapter.RequestContext 实现 adapter.RequestContext 接口
func (g *GinContext) Set(key string, value interface{}) {
	g.c.Set(key, value)
}

// SetCookie implements adapter.RequestContext 实现 adapter.RequestContext 接口
func (g *GinContext) SetCookie(name string, value string, maxAge int, path string, domain string, secure bool, httpOnly bool) {
	g.SetCookieWithOptions(&adapter.CookieOptions{
		Name: name, Value: value, MaxAge: maxAge, Path: path, Domain: domain, Secure: secure, HttpOnly: httpOnly,
	})
}

// SetHeader implements adapter.RequestContext 实现 adapter.RequestContext 接口
func (g *GinContext) SetHeader(key string, value string) {
	g.c.Header(key, value)
}

// GetHeaders implements adapter.RequestContext 实现 adapter.RequestContext 接口
func (g *GinContext) GetHeaders() map[string][]string {
	return g.c.Request.Header
}

// GetQueryAll implements adapter.RequestContext 实现 adapter.RequestContext 接口
func (g *GinContext) GetQueryAll() map[string][]string {
	return g.c.Request.URL.Query()
}

// GetPostForm implements adapter.RequestContext 实现 adapter.RequestContext 接口
func (g *GinContext) GetPostForm(key string) string {
	// Parse the current request with Gin's multipart limit, bypassing its context-level form cache. 使用 Gin 的 multipart 内存限制解析当前请求，绕过上下文级旧表单缓存。
	if g.c.Request.PostForm == nil {
		_, _ = g.c.MultipartForm()
	}
	return g.c.Request.PostForm.Get(key)
}

// GetBody implements adapter.RequestContext 实现 adapter.RequestContext 接口
func (g *GinContext) GetBody() ([]byte, error) {
	return g.c.GetRawData()
}

// GetURL implements adapter.RequestContext 实现 adapter.RequestContext 接口
func (g *GinContext) GetURL() string {
	return g.c.Request.URL.String()
}

// GetUserAgent implements adapter.RequestContext 实现 adapter.RequestContext 接口
func (g *GinContext) GetUserAgent() string {
	return g.c.GetHeader("User-Agent")
}

// SetCookieWithOptions implements adapter.RequestContext 实现 adapter.RequestContext 接口
func (g *GinContext) SetCookieWithOptions(options *adapter.CookieOptions) {
	sameSite := http.SameSiteLaxMode
	switch options.SameSite {
	case "Strict":
		sameSite = http.SameSiteStrictMode
	case "None":
		sameSite = http.SameSiteNoneMode
	}
	path := options.Path
	if path == "" {
		path = "/"
	}

	// Apply SameSite to this cookie without changing later application cookies. 仅为当前 Cookie 应用 SameSite，不改变业务后续 Cookie 的设置。
	http.SetCookie(g.c.Writer, &http.Cookie{
		Name: options.Name, Value: url.QueryEscape(options.Value), MaxAge: options.MaxAge,
		Path: path, Domain: options.Domain, Secure: options.Secure, HttpOnly: options.HttpOnly, SameSite: sameSite,
	})
}

// GetString implements adapter.RequestContext 实现 adapter.RequestContext 接口
func (g *GinContext) GetString(key string) string {
	v := g.c.GetString(key)
	return v
}

// MustGet implements adapter.RequestContext 实现 adapter.RequestContext 接口
func (g *GinContext) MustGet(key string) any {
	v, exists := g.c.Get(key)
	if !exists {
		panic("key not found: " + key)
	}
	return v
}

// Abort implements adapter.RequestContext 实现 adapter.RequestContext 接口
func (g *GinContext) Abort() {
	g.c.Abort()
}

// IsAborted implements adapter.RequestContext 实现 adapter.RequestContext 接口
func (g *GinContext) IsAborted() bool {
	return g.c.IsAborted()
}

// IsTLS implements adapter.RequestContext 实现 adapter.RequestContext 接口
func (g *GinContext) IsTLS() bool {
	return g.c.Request.TLS != nil
}

// SetStatusCode implements adapter.RequestContext 实现 adapter.RequestContext 接口
func (g *GinContext) SetStatusCode(code int) {
	g.c.Status(code)
}

// Write implements adapter.RequestContext 实现 adapter.RequestContext 接口
func (g *GinContext) Write(data []byte) (int, error) {
	return g.c.Writer.Write(data)
}
