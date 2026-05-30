// @Author daixk 2026/05/29
package httpserver

import (
	"net/http"

	"github.com/Zany2/dtoken-go/sso"
)

// LoginIDResolver resolves the current SSO-center login id from an HTTP request. LoginIDResolver 从 HTTP 请求解析当前 SSO 中心登录 ID。
type LoginIDResolver = sso.LoginIDResolver

// Options defines standalone HTTP protocol behavior for SSO. Options 定义独立 HTTP 协议层行为。
type Options = sso.HTTPOptions

// Server exposes SSO routes by using net/http only. Server 使用标准库 net/http 暴露 SSO 路由。
type Server = sso.HTTPServer

// CookieOptions defines shared-cookie behavior for same-site SSO. CookieOptions 定义同站 SSO 的共享 Cookie 行为。
type CookieOptions = sso.CookieOptions

// New creates a standalone HTTP SSO handler. New 创建独立 HTTP SSO 处理器。
func New(server *sso.Server, options Options) *Server {
	return sso.NewHTTPServer(server, options)
}

// DefaultOptions returns default standalone HTTP options. DefaultOptions 返回默认独立 HTTP 选项。
func DefaultOptions() Options {
	return sso.DefaultHTTPOptions()
}

// DefaultCookieOptions returns default shared-cookie options. DefaultCookieOptions 返回默认共享 Cookie 配置。
func DefaultCookieOptions() CookieOptions {
	return sso.DefaultCookieOptions()
}

// LoginIDFromCookie creates a resolver that reads login id from shared cookie. LoginIDFromCookie 创建从共享 Cookie 读取登录 ID 的解析器。
func LoginIDFromCookie(options CookieOptions) LoginIDResolver {
	return sso.LoginIDFromCookie(options)
}

// SetLoginIDCookie writes shared login cookie. SetLoginIDCookie 写入共享登录 Cookie。
func SetLoginIDCookie(w http.ResponseWriter, options CookieOptions, loginID string) {
	sso.SetLoginIDCookie(w, options, loginID)
}

// ClearLoginIDCookie clears shared login cookie. ClearLoginIDCookie 清除共享登录 Cookie。
func ClearLoginIDCookie(w http.ResponseWriter, options CookieOptions) {
	sso.ClearLoginIDCookie(w, options)
}
