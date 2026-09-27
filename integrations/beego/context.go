// @Author daixk 2026/06/06
package beego

import (
	"bytes"
	"io"
	"net"
	"net/http"

	"github.com/Zany2/dtoken-go/core/adapter"
	beegocontext "github.com/beego/beego/v2/server/web/context"
)

// BeegoContext adapts Beego request context BeegoContext 适配 Beego 请求上下文
type BeegoContext struct {
	c *beegocontext.Context
}

// Interface assertions keep request context contracts checked at compile time 接口断言在编译期检查请求上下文契约。
var (
	_ adapter.RequestContext    = (*BeegoContext)(nil)
	_ adapter.RequestContextExt = (*BeegoContext)(nil)
)

// NewBeegoContext creates request context adapter NewBeegoContext 创建请求上下文适配器
func NewBeegoContext(c *beegocontext.Context) adapter.RequestContext {
	return &BeegoContext{c: c}
}

// Get implements adapter.RequestContext Get 实现 adapter.RequestContext 接口
func (b *BeegoContext) Get(key string) (interface{}, bool) {
	value := b.c.Input.GetData(key)
	return value, value != nil
}

// GetClientIP implements adapter.RequestContext GetClientIP 实现 adapter.RequestContext 接口
func (b *BeegoContext) GetClientIP() string {
	// Trusted proxy middleware must normalize RemoteAddr before adaptation. 可信代理中间件应在适配前规范化 RemoteAddr，避免信任伪造的转发头。
	host, _, err := net.SplitHostPort(b.c.Request.RemoteAddr)
	if err == nil {
		return host
	}
	return b.c.Request.RemoteAddr
}

// GetCookie implements adapter.RequestContext GetCookie 实现 adapter.RequestContext 接口
func (b *BeegoContext) GetCookie(key string) string {
	return b.c.Input.Cookie(key)
}

// GetHeader implements adapter.RequestContext GetHeader 实现 adapter.RequestContext 接口
func (b *BeegoContext) GetHeader(key string) string {
	return b.c.Input.Header(key)
}

// GetMethod implements adapter.RequestContext GetMethod 实现 adapter.RequestContext 接口
func (b *BeegoContext) GetMethod() string {
	return b.c.Request.Method
}

// GetPath implements adapter.RequestContext GetPath 实现 adapter.RequestContext 接口
func (b *BeegoContext) GetPath() string {
	return b.c.Request.URL.Path
}

// GetQuery implements adapter.RequestContext GetQuery 实现 adapter.RequestContext 接口
func (b *BeegoContext) GetQuery(key string) string {
	return b.c.Request.URL.Query().Get(key)
}

// Set implements adapter.RequestContext Set 实现 adapter.RequestContext 接口
func (b *BeegoContext) Set(key string, value interface{}) {
	b.c.Input.SetData(key, value)
}

// SetCookie implements adapter.RequestContext SetCookie 实现 adapter.RequestContext 接口
func (b *BeegoContext) SetCookie(name string, value string, maxAge int, path string, domain string, secure bool, httpOnly bool) {
	b.setHTTPCookie(&http.Cookie{
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

// SetHeader implements adapter.RequestContext SetHeader 实现 adapter.RequestContext 接口
func (b *BeegoContext) SetHeader(key string, value string) {
	b.c.Output.Header(key, value)
}

// GetHeaders implements adapter.RequestContext GetHeaders 实现 adapter.RequestContext 接口
func (b *BeegoContext) GetHeaders() map[string][]string {
	return b.c.Request.Header
}

// GetQueryAll implements adapter.RequestContext GetQueryAll 实现 adapter.RequestContext 接口
func (b *BeegoContext) GetQueryAll() map[string][]string {
	return b.c.Request.URL.Query()
}

// GetPostForm implements adapter.RequestContext GetPostForm 实现 adapter.RequestContext 接口
func (b *BeegoContext) GetPostForm(key string) string {
	return b.c.Request.PostFormValue(key)
}

// GetBody implements adapter.RequestContext GetBody 实现 adapter.RequestContext 接口
func (b *BeegoContext) GetBody() ([]byte, error) {
	if len(b.c.Input.RequestBody) > 0 {
		return append([]byte{}, b.c.Input.RequestBody...), nil
	}
	if b.c.Request == nil || b.c.Request.Body == nil {
		return nil, nil
	}
	body, err := io.ReadAll(b.c.Request.Body)
	if err != nil {
		return nil, err
	}
	b.c.Request.Body = io.NopCloser(bytes.NewReader(body))
	return body, nil
}

// GetURL implements adapter.RequestContext GetURL 实现 adapter.RequestContext 接口
func (b *BeegoContext) GetURL() string {
	return b.c.Request.URL.String()
}

// GetUserAgent implements adapter.RequestContext GetUserAgent 实现 adapter.RequestContext 接口
func (b *BeegoContext) GetUserAgent() string {
	return b.c.Request.UserAgent()
}

// SetCookieWithOptions implements adapter.RequestContext SetCookieWithOptions 实现 adapter.RequestContext 接口
func (b *BeegoContext) SetCookieWithOptions(options *adapter.CookieOptions) {
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
	b.setHTTPCookie(cookie)
}

// GetString implements adapter.RequestContext GetString 实现 adapter.RequestContext 接口
func (b *BeegoContext) GetString(key string) string {
	value, ok := b.c.Input.GetData(key).(string)
	if !ok {
		return ""
	}
	return value
}

// MustGet implements adapter.RequestContext MustGet 实现 adapter.RequestContext 接口
func (b *BeegoContext) MustGet(key string) any {
	value := b.c.Input.GetData(key)
	if value == nil {
		panic("key not found: " + key)
	}
	return value
}

// Abort implements adapter.RequestContext Abort 实现 adapter.RequestContext 接口
func (b *BeegoContext) Abort() {
	markAborted(b.c)
}

// IsAborted implements adapter.RequestContext IsAborted 实现 adapter.RequestContext 接口
func (b *BeegoContext) IsAborted() bool {
	return b.c != nil && b.c.ResponseWriter != nil && b.c.ResponseWriter.Started
}

// IsTLS implements adapter.RequestContext IsTLS 实现 adapter.RequestContext 接口
func (b *BeegoContext) IsTLS() bool {
	return b.c.Request.TLS != nil
}

// SetStatusCode implements adapter.RequestContext SetStatusCode 实现 adapter.RequestContext 接口
func (b *BeegoContext) SetStatusCode(code int) {
	b.c.Output.SetStatus(code)
}

// Write implements adapter.RequestContext Write 实现 adapter.RequestContext 接口
func (b *BeegoContext) Write(data []byte) (int, error) {
	// Flush Beego's pending status before bypassing Output.Body. 绕过 Output.Body 直接写入时，先提交 Beego 暂存的状态码。
	if b.c.Output.Status != 0 {
		b.c.ResponseWriter.WriteHeader(b.c.Output.Status)
		b.c.Output.Status = 0
	}
	return b.c.ResponseWriter.Write(data)
}

// JSON writes JSON response JSON 写入 JSON 响应
func (b *BeegoContext) JSON(code int, v any) error {
	b.c.Output.SetStatus(code)
	return b.c.Output.JSON(v, false, false)
}

// GetRawRequest gets raw http request GetRawRequest 获取原始 HTTP 请求
func (b *BeegoContext) GetRawRequest() *http.Request {
	return b.c.Request
}

// GetRawResponseWriter gets raw response writer GetRawResponseWriter 获取原始响应写入器
func (b *BeegoContext) GetRawResponseWriter() http.ResponseWriter {
	return b.c.ResponseWriter
}

// setHTTPCookie writes a cookie through net/http setHTTPCookie 通过 net/http 写入 Cookie
func (b *BeegoContext) setHTTPCookie(cookie *http.Cookie) {
	http.SetCookie(b.c.ResponseWriter, cookie)
}

// writeJSON writes JSON without relying on controller helpers writeJSON 不依赖控制器辅助方法写入 JSON
func writeJSON(ctx *beegocontext.Context, code int, value any) {
	ctx.Output.SetStatus(code)
	_ = ctx.Output.JSON(value, false, false)
}
