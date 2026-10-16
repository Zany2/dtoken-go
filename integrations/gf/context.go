// @Author daixk 2025/12/22 15:56:00
package gf

import (
	"net/http"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/net/ghttp"
)

// GFContext adapts request context 适配 GoFrame 请求上下文
type GFContext struct {
	c       *ghttp.Request
	aborted bool
	bodyErr error
}

// Interface assertion keeps request context contract checked at compile time 接口断言在编译期检查请求上下文契约
var _ adapter.RequestContext = (*GFContext)(nil)

// NewGFContext creates request context adapter 创建请求上下文适配器
func NewGFContext(c *ghttp.Request) adapter.RequestContext {
	return &GFContext{
		c: c,
	}
}

// Get implements adapter.RequestContext 实现 adapter.RequestContext 接口
func (g *GFContext) Get(key string) (interface{}, bool) {
	v := g.c.GetCtxVar(key)
	return v.Val(), !v.IsNil()
}

// GetClientIP implements adapter.RequestContext 实现 adapter.RequestContext 接口
func (g *GFContext) GetClientIP() string {
	// Forwarded headers are not verified by GoFrame. GoFrame 不验证转发头的来源，仅采用连接地址。
	return g.c.GetRemoteIp()
}

// GetCookie implements adapter.RequestContext 实现 adapter.RequestContext 接口
func (g *GFContext) GetCookie(key string) string {
	v := g.c.Cookie.Get(key)
	if v == nil {
		return ""
	}
	return v.String()
}

// GetHeader implements adapter.RequestContext 实现 adapter.RequestContext 接口
func (g *GFContext) GetHeader(key string) string {
	return g.c.Header.Get(key)
}

// GetMethod implements adapter.RequestContext 实现 adapter.RequestContext 接口
func (g *GFContext) GetMethod() string {
	return g.c.Method
}

// GetPath implements adapter.RequestContext 实现 adapter.RequestContext 接口
func (g *GFContext) GetPath() string {
	return g.c.Request.URL.Path
}

// GetQuery implements adapter.RequestContext 实现 adapter.RequestContext 接口
func (g *GFContext) GetQuery(key string) string {
	return g.c.Request.URL.Query().Get(key)
}

// Set implements adapter.RequestContext 实现 adapter.RequestContext 接口
func (g *GFContext) Set(key string, value interface{}) {
	g.c.SetCtxVar(key, value)
}

// SetCookie implements adapter.RequestContext 实现 adapter.RequestContext 接口
func (g *GFContext) SetCookie(name string, value string, maxAge int, path string, domain string, secure bool, httpOnly bool) {
	g.c.Cookie.SetHttpCookie(&http.Cookie{
		Name:     name,
		Value:    value,
		MaxAge:   maxAge,
		Path:     path,
		Domain:   domain,
		Secure:   secure,
		HttpOnly: httpOnly,
		SameSite: http.SameSiteLaxMode,
	})
}

// SetHeader implements adapter.RequestContext 实现 adapter.RequestContext 接口
func (g *GFContext) SetHeader(key string, value string) {
	g.c.Response.Header().Set(key, value)
}

// GetHeaders implements adapter.RequestContext 实现 adapter.RequestContext 接口
func (g *GFContext) GetHeaders() map[string][]string {
	return g.c.Header
}

// GetQueryAll implements adapter.RequestContext 实现 adapter.RequestContext 接口
func (g *GFContext) GetQueryAll() map[string][]string {
	return g.c.Request.URL.Query()
}

// GetPostForm implements adapter.RequestContext 实现 adapter.RequestContext 接口
func (g *GFContext) GetPostForm(key string) string {
	return g.c.GetForm(key).String()
}

// GetBody implements adapter.RequestContext 实现 adapter.RequestContext 接口
func (g *GFContext) GetBody() (body []byte, err error) {
	if g.bodyErr != nil {
		return nil, g.bodyErr
	}

	// Translate GoFrame's read failure without exposing its cached partial body. 转换 GoFrame 的读取异常，避免后续返回缓存的残缺请求体。
	defer func() {
		if recovered := recover(); recovered != nil {
			if readErr, ok := recovered.(*gerror.Error); ok {
				g.bodyErr = readErr
				body, err = nil, readErr
				return
			}
			panic(recovered)
		}
	}()
	return g.c.GetBody(), nil
}

// GetURL implements adapter.RequestContext 实现 adapter.RequestContext 接口
func (g *GFContext) GetURL() string {
	return g.c.Request.URL.String()
}

// GetUserAgent implements adapter.RequestContext 实现 adapter.RequestContext 接口
func (g *GFContext) GetUserAgent() string {
	return g.c.Header.Get("User-Agent")
}

// SetCookieWithOptions implements adapter.RequestContext 实现 adapter.RequestContext 接口
func (g *GFContext) SetCookieWithOptions(options *adapter.CookieOptions) {
	cookie := &http.Cookie{
		Name:     options.Name,
		Value:    options.Value,
		MaxAge:   options.MaxAge,
		Path:     options.Path,
		Domain:   options.Domain,
		Secure:   options.Secure,
		HttpOnly: options.HttpOnly,
		SameSite: http.SameSiteLaxMode,
	}

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

// GetString implements adapter.RequestContext 实现 adapter.RequestContext 接口
func (g *GFContext) GetString(key string) string {
	v := g.c.GetCtxVar(key)
	return v.String()
}

// MustGet implements adapter.RequestContext 实现 adapter.RequestContext 接口
func (g *GFContext) MustGet(key string) any {
	v := g.c.GetCtxVar(key)
	if v.IsNil() {
		panic("key not found: " + key)
	}
	return v.Val()
}

// Abort implements adapter.RequestContext 实现 adapter.RequestContext 接口
func (g *GFContext) Abort() {
	g.aborted = true
}

// IsAborted implements adapter.RequestContext 实现 adapter.RequestContext 接口
func (g *GFContext) IsAborted() bool {
	return g.aborted || g.c.IsExited()
}

// IsTLS implements adapter.RequestContext 实现 adapter.RequestContext 接口
func (g *GFContext) IsTLS() bool {
	return g.c.TLS != nil
}

// SetStatusCode implements adapter.RequestContext 实现 adapter.RequestContext 接口
func (g *GFContext) SetStatusCode(code int) {
	g.c.Response.WriteHeader(code)
}

// Write implements adapter.RequestContext 实现 adapter.RequestContext 接口
func (g *GFContext) Write(data []byte) (int, error) {
	g.c.Response.Write(data)
	return len(data), nil
}
