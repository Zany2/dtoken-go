package adapter

import (
	"net/http"
)

// CookieOptions Cookie设置选项
type CookieOptions struct {
	// Name Cookie名称
	Name string
	// Value Cookie值
	Value string
	// MaxAge 过期时间（秒），0表示删除cookie，-1表示会话cookie
	MaxAge int
	// Path 路径
	Path string
	// Domain 域名
	Domain string
	// Secure 是否只在HTTPS下生效
	Secure bool
	// HttpOnly 是否禁止JS访问
	HttpOnly bool
	// SameSite SameSite属性（Strict、Lax、None）
	SameSite string
}

// RequestContext 定义请求上下文接口，用于抽象不同Web框架的请求/响应
type RequestContext interface {
	// ============== 请求方法 ==============

	// GetHeader 获取请求头
	GetHeader(key string) string

	// GetHeaders 获取所有请求头
	GetHeaders() map[string][]string

	// GetQuery 获取查询参数
	GetQuery(key string) string

	// GetQueryAll 获取所有查询参数
	GetQueryAll() map[string][]string

	// GetPostForm 获取POST表单参数
	GetPostForm(key string) string

	// GetCookie 获取Cookie
	GetCookie(key string) string

	// GetBody 获取请求体字节数据
	GetBody() ([]byte, error)

	// GetClientIP 获取客户端IP地址
	GetClientIP() string

	// GetMethod 获取请求方法（GET、POST等）
	GetMethod() string

	// GetPath 获取请求路径
	GetPath() string

	// GetURL 获取完整请求URL
	GetURL() string

	// GetUserAgent 获取User-Agent
	GetUserAgent() string

	// IsTLS 检查请求是否通过HTTPS发起
	IsTLS() bool

	// ============== 响应方法 ==============

	// SetStatusCode 设置HTTP响应状态码
	SetStatusCode(code int)

	// SetHeader 设置响应头
	SetHeader(key, value string)

	// Write 直接写入响应体字节数据
	Write(data []byte) (int, error)

	// SetCookie 设置Cookie（兼容旧版本的方法）
	SetCookie(name, value string, maxAge int, path, domain string, secure, httpOnly bool)

	// SetCookieWithOptions 使用选项设置Cookie
	SetCookieWithOptions(options *CookieOptions)

	// ============== 上下文存储方法 ==============

	// Set 设置上下文值
	Set(key string, value any)

	// Get 获取上下文值
	Get(key string) (any, bool)

	// GetString 从上下文获取字符串值
	GetString(key string) string

	// MustGet 获取上下文值，不存在则panic
	MustGet(key string) any

	// ============== 工具方法 ==============

	// Abort 中止请求处理
	Abort()

	// IsAborted 检查请求是否已中止
	IsAborted() bool
}

// RequestContextExt 扩展接口
type RequestContextExt interface {
	RequestContext

	// ============== 高级响应方法 ==============

	// JSON 返回JSON格式响应（自动设置Content-Type为application/json）
	JSON(code int, v any) error

	// ============== 高级访问（用于框架逃逸） ==============

	// GetRawRequest 获取原始 *http.Request 对象（用于高级操作或调用框架特有功能）
	GetRawRequest() *http.Request

	// GetRawResponseWriter 获取原始 http.ResponseWriter 对象（用于流式响应等）
	GetRawResponseWriter() http.ResponseWriter
}
