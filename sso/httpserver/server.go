// @Author daixk 2026/05/29
package httpserver

import (
	"net/http"

	"github.com/Zany2/dtoken-go/sso"
)

// LoginIDResolver resolves the current SSO-center login id from an HTTP request. LoginIDResolver 从 HTTP 请求解析当前 SSO 中心登录 ID。
type LoginIDResolver = sso.LoginIDResolver

// CenterLogoutFunc revokes the resolved subject's center session and must tolerate retries. CenterLogoutFunc 撤销已解析主体的中心会话，必须允许重试。
type CenterLogoutFunc = sso.CenterLogoutFunc

// LogoutResultFunc observes the final authenticated logout outcome. LogoutResultFunc 观察已认证注销的最终结果。
type LogoutResultFunc = sso.LogoutResultFunc

// HTTPErrorFunc observes private center HTTP errors without controlling the response. HTTPErrorFunc 观察中心 HTTP 私有错误，不控制响应。
type HTTPErrorFunc = sso.HTTPErrorFunc

// LogoutResult describes lifecycle progress and callback outcomes. LogoutResult 描述生命周期进度与回调结果。
type LogoutResult = sso.LogoutResult

// AccountLogoutResult describes trusted account revocation progress and notification failures. AccountLogoutResult 描述可信账号撤销进度及通知失败。
type AccountLogoutResult = sso.AccountLogoutResult

// ExchangeResultFunc enriches a consumed credential's result without changing its identity or scopes. ExchangeResultFunc 补充已消费凭证的返回信息，不得改变身份或授权范围。
type ExchangeResultFunc = sso.ExchangeResultFunc

// AuthorizationExtraFunc resolves trusted metadata to bind to issued credentials. AuthorizationExtraFunc 解析要绑定到签发凭证的可信元数据。
type AuthorizationExtraFunc = sso.AuthorizationExtraFunc

// AuthorizationPolicyFunc evaluates validated application access across HTTP credential stages. AuthorizationPolicyFunc 在 HTTP 凭证各阶段评估已校验的应用访问权限。
type AuthorizationPolicyFunc = sso.AuthorizationPolicyFunc

// AuthorizationRequest contains trusted identity and the maximum grant for policy evaluation. AuthorizationRequest 包含用于策略求值的可信身份及最大授权范围。
type AuthorizationRequest = sso.AuthorizationRequest

// AuthorizationDecision permits access and optionally reduces scopes. AuthorizationDecision 允许访问并可缩减权限范围。
type AuthorizationDecision = sso.AuthorizationDecision

// CredentialActiveFunc checks application state during introspection and userinfo. CredentialActiveFunc 在 introspect 和 userinfo 时检查应用状态。
type CredentialActiveFunc = sso.CredentialActiveFunc

// Options defines standalone HTTP protocol behavior for SSO. Options 定义独立 HTTP 协议层行为。
type Options = sso.HTTPOptions

// RequestLimits bounds inbound protocol queries and form bodies. RequestLimits 限制入站协议查询串与表单体。
type RequestLimits = sso.RequestLimits

// Server exposes SSO routes by using net/http only. Server 使用标准库 net/http 暴露 SSO 路由。
type Server = sso.HTTPServer

// CookieOptions defines shared-cookie behavior for same-site SSO. CookieOptions 定义同站 SSO 的共享 Cookie 行为。
type CookieOptions = sso.CookieOptions

// New creates a standalone HTTP SSO handler. New 创建独立 HTTP SSO 处理器。
func New(server *sso.Server, options Options) *Server {
	return sso.NewHTTPServer(server, options)
}

// NewChecked validates local configuration and returns initialization errors before serving. NewChecked 校验本地配置，并在提供服务前返回初始化错误。
func NewChecked(server *sso.Server, options Options) (*Server, error) {
	return sso.NewHTTPServerChecked(server, options)
}

// DefaultOptions returns default standalone HTTP options. DefaultOptions 返回默认独立 HTTP 选项。
func DefaultOptions() Options {
	return sso.DefaultHTTPOptions()
}

// DefaultRequestLimits returns the inbound query and form limits. DefaultRequestLimits 返回入站查询串与表单限制。
func DefaultRequestLimits() RequestLimits {
	return sso.DefaultRequestLimits()
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
