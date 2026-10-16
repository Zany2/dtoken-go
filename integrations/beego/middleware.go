// @Author daixk 2026/06/06
package beego

import (
	"context"
	"net/http"

	DContext "github.com/Zany2/dtoken-go/core/context"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/manager"
	"github.com/Zany2/dtoken-go/integrations/authcheck"
	web "github.com/beego/beego/v2/server/web"
	beegocontext "github.com/beego/beego/v2/server/web/context"
)

// LogicType defines middleware logic type LogicType 定义中间件逻辑类型
type LogicType = authcheck.LogicType

const (
	// DTokenCtxKey stores request scoped DToken context DTokenCtxKey 存储请求级 DToken 上下文
	DTokenCtxKey = "DCtx"

	// LogicOr represents OR logic LogicOr 表示或逻辑
	LogicOr LogicType = authcheck.LogicOr
	// LogicAnd represents AND logic LogicAnd 表示与逻辑
	LogicAnd LogicType = authcheck.LogicAnd
)

// AuthOption defines auth option setter AuthOption 定义认证选项设置器
type AuthOption func(*AuthOptions)

// BeforeAuthHandler handles request before dtoken checks BeforeAuthHandler 在 dtoken 校验前处理请求
type BeforeAuthHandler func(ctx context.Context, c *beegocontext.Context, req *AuthHandleRequest)

// RouteAccessHandler resolves route auth, permission, and role rules RouteAccessHandler 解析路由认证、权限、角色规则
type RouteAccessHandler func(ctx context.Context, c *beegocontext.Context, req *RouteAccessRequest)

// AuthHandleRequest carries auth check metadata AuthHandleRequest 携带认证校验元数据
type AuthHandleRequest struct {
	// AuthType selects the auth type. AuthType 指定认证类型。
	AuthType string
	// CheckLogin requires login-state validation. CheckLogin 要求校验登录状态。
	CheckLogin bool
	// CheckDisable requires account-disable validation. CheckDisable 要求校验账号封禁状态。
	CheckDisable bool
	// Permissions lists required permissions. Permissions 保存所需权限列表。
	Permissions []string
	// Roles lists required roles. Roles 保存所需角色列表。
	Roles []string
	// LogicType controls permission and role matching. LogicType 控制权限和角色的匹配逻辑。
	LogicType LogicType

	next    func()
	exit    func()
	handled bool
}

// Next continues request and stops dtoken checks Next 放行请求并停止 dtoken 校验
func (req *AuthHandleRequest) Next() {
	if req.handled {
		return
	}
	req.handled = true
	if req.next != nil {
		req.next()
	}
}

// Exit stops dtoken checks after custom handling Exit 自定义处理后停止 dtoken 校验
func (req *AuthHandleRequest) Exit() {
	if req.handled {
		return
	}
	req.handled = true
	if req.exit != nil {
		req.exit()
	}
}

// IsHandled reports whether request has been handled IsHandled 判断请求是否已处理
func (req *AuthHandleRequest) IsHandled() bool {
	return req.handled
}

// RouteAccessRequest carries route access rules RouteAccessRequest 携带路由访问规则
type RouteAccessRequest struct {
	// AuthType selects the auth type. AuthType 指定认证类型。
	AuthType string
	// LogicType controls permission and role matching. LogicType 控制权限和角色的匹配逻辑。
	LogicType LogicType
	// CheckDisable requires account-disable validation. CheckDisable 要求校验账号封禁状态。
	CheckDisable bool
	// Permissions lists route-required permissions. Permissions 保存路由所需权限列表。
	Permissions []string
	// Roles lists route-required roles. Roles 保存路由所需角色列表。
	Roles []string

	skipAuth       bool
	skipPermission bool
	err            error
}

// SkipAuth skips login, permission, and role checks SkipAuth 跳过登录、权限、角色校验
func (req *RouteAccessRequest) SkipAuth() {
	req.skipAuth = true
}

// SkipPermission skips permission and role checks after login SkipPermission 登录后跳过权限和角色校验
func (req *RouteAccessRequest) SkipPermission() {
	req.skipPermission = true
	req.Permissions = nil
	req.Roles = nil
}

// RequirePermissions appends required permissions RequirePermissions 追加当前路由所需权限
func (req *RouteAccessRequest) RequirePermissions(permissions ...string) {
	req.skipPermission = false
	req.Permissions = append(req.Permissions, permissions...)
}

// RequireRoles appends required roles RequireRoles 追加当前路由所需角色
func (req *RouteAccessRequest) RequireRoles(roles ...string) {
	req.skipPermission = false
	req.Roles = append(req.Roles, roles...)
}

// SetLogicType sets permission and role logic type SetLogicType 设置权限和角色逻辑类型
func (req *RouteAccessRequest) SetLogicType(logicType LogicType) {
	req.LogicType = logicType
}

// AuthOptions defines middleware auth options AuthOptions 定义中间件认证选项
type AuthOptions struct {
	// AuthType selects the auth type. AuthType 指定认证类型。
	AuthType string
	// Manager overrides auth type, request cache and global selection. Manager 优先于认证类型、请求缓存及全局选择。
	Manager *manager.Manager
	// LogicType controls permission and role matching. LogicType 控制权限和角色的匹配逻辑。
	LogicType LogicType
	// FailFunc handles authentication failures. FailFunc 处理认证失败。
	FailFunc func(c *beegocontext.Context, err error)
	// BeforeAuthHandler runs before authentication checks. BeforeAuthHandler 在认证校验前执行。
	BeforeAuthHandler BeforeAuthHandler
	// RouteAccessHandler resolves route-specific access rules. RouteAccessHandler 解析路由级访问规则。
	RouteAccessHandler RouteAccessHandler
}

// defaultAuthOptions returns default auth options defaultAuthOptions 返回默认认证选项
func defaultAuthOptions() *AuthOptions {
	return &AuthOptions{LogicType: LogicAnd}
}

// WithAuthType sets auth type WithAuthType 设置认证类型
func WithAuthType(authType string) AuthOption {
	return func(o *AuthOptions) {
		o.AuthType = authType
	}
}

// WithManager sets the manager used by middleware. WithManager 设置中间件使用的 Manager。
func WithManager(mgr *manager.Manager) AuthOption {
	return func(o *AuthOptions) {
		if mgr != nil {
			o.Manager = mgr
		}
	}
}

// WithLogicType sets logic type WithLogicType 设置逻辑类型
func WithLogicType(logicType LogicType) AuthOption {
	return func(o *AuthOptions) {
		o.LogicType = logicType
	}
}

// WithFailFunc sets auth failure callback WithFailFunc 设置认证失败回调
func WithFailFunc(fn func(c *beegocontext.Context, err error)) AuthOption {
	return func(o *AuthOptions) {
		o.FailFunc = fn
	}
}

// WithBeforeAuthHandler sets pre auth handler WithBeforeAuthHandler 设置认证前置处理器
func WithBeforeAuthHandler(fn BeforeAuthHandler) AuthOption {
	return func(o *AuthOptions) {
		o.BeforeAuthHandler = fn
	}
}

// WithRouteAccessHandler sets route access handler WithRouteAccessHandler 设置路由访问处理器
func WithRouteAccessHandler(fn RouteAccessHandler) AuthOption {
	return func(o *AuthOptions) {
		o.RouteAccessHandler = fn
	}
}

// RegisterDTokenContextMiddleware registers DToken context middleware RegisterDTokenContextMiddleware 注册 DToken 上下文过滤器
func RegisterDTokenContextMiddleware(ctx context.Context, opts ...AuthOption) web.FilterFunc {
	options := defaultAuthOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(c *beegocontext.Context) {
		if isRequestAborted(c) {
			return
		}

		mgr, err := resolveRequestManager(c, options.Manager, options.AuthType)
		if err != nil {
			dispatchFail(c, options, err)
			return
		}

		_ = getDContext(c, mgr)
	}
}

// AuthMiddleware checks login status AuthMiddleware 校验登录状态
func AuthMiddleware(ctx context.Context, opts ...AuthOption) web.FilterFunc {
	options := defaultAuthOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(c *beegocontext.Context) {
		if isRequestAborted(c) {
			return
		}

		authReq := newAuthHandleRequest(options, nil, func() {
			markAborted(c)
		})
		authReq.CheckLogin = true
		if runBeforeAuthHandler(requestContext(c), c, options, authReq) {
			return
		}

		mgr, err := resolveRequestManager(c, options.Manager, options.AuthType)
		if err != nil {
			dispatchFail(c, options, err)
			return
		}

		dCtx := getDContext(c, mgr)
		tokenValue := dCtx.GetTokenValue()

		_, err = authcheck.Check(requestContext(c), mgr, authcheck.Request{
			TokenValue: tokenValue,
			CheckLogin: true,
			LoginError: derror.ErrTokenExpired,
		})
		if err != nil {
			dispatchFail(c, options, err)
		}
	}
}

// AccessMiddleware resolves route rules and checks login, permissions, and roles AccessMiddleware 解析路由规则并校验登录、权限、角色
func AccessMiddleware(ctx context.Context, opts ...AuthOption) web.FilterFunc {
	options := defaultAuthOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(c *beegocontext.Context) {
		if isRequestAborted(c) {
			return
		}

		accessReq := newRouteAccessRequest(options)
		if options.RouteAccessHandler != nil {
			options.RouteAccessHandler(requestContext(c), c, accessReq)
		}

		if isRequestAborted(c) {
			return
		}
		if accessReq.err != nil {
			dispatchFail(c, options, accessReq.err)
			return
		}

		if accessReq.skipAuth {
			return
		}

		mgr, err := resolveRequestManager(c, options.Manager, accessReq.AuthType)
		if err != nil {
			dispatchFail(c, options, err)
			return
		}

		dCtx := getDContext(c, mgr)
		tokenValue := dCtx.GetTokenValue()

		req := authcheck.Request{
			TokenValue:   tokenValue,
			CheckLogin:   true,
			CheckDisable: accessReq.CheckDisable,
			LoginError:   derror.ErrTokenExpired,
		}

		if !accessReq.skipPermission {
			req.Permissions = append([]string{}, accessReq.Permissions...)
			req.Roles = append([]string{}, accessReq.Roles...)
			req.LogicType = accessReq.LogicType
		}

		_, err = authcheck.Check(requestContext(c), mgr, req)
		if err != nil {
			dispatchFail(c, options, err)
		}
	}
}

// PermissionMiddleware checks permissions PermissionMiddleware 校验权限
func PermissionMiddleware(ctx context.Context, permissions []string, opts ...AuthOption) web.FilterFunc {
	options := defaultAuthOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(c *beegocontext.Context) {
		if isRequestAborted(c) {
			return
		}

		authReq := newAuthHandleRequest(options, nil, func() {
			markAborted(c)
		})
		authReq.Permissions = append([]string{}, permissions...)
		if runBeforeAuthHandler(requestContext(c), c, options, authReq) {
			return
		}

		if len(permissions) == 0 {
			return
		}

		mgr, err := resolveRequestManager(c, options.Manager, options.AuthType)
		if err != nil {
			dispatchFail(c, options, err)
			return
		}

		dCtx := getDContext(c, mgr)
		tokenValue := dCtx.GetTokenValue()

		_, err = authcheck.Check(requestContext(c), mgr, authcheck.Request{
			TokenValue:  tokenValue,
			Permissions: permissions,
			LogicType:   options.LogicType,
		})
		if err != nil {
			dispatchFail(c, options, err)
		}
	}
}

// PermissionPathMiddleware checks path permissions PermissionPathMiddleware 基于路径校验权限
func PermissionPathMiddleware(ctx context.Context, permissions []string, opts ...AuthOption) web.FilterFunc {
	options := defaultAuthOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(c *beegocontext.Context) {
		if isRequestAborted(c) {
			return
		}

		reqPermissions := append([]string{}, permissions...)
		reqPermissions = append(reqPermissions, c.Request.URL.Path)

		authReq := newAuthHandleRequest(options, nil, func() {
			markAborted(c)
		})
		authReq.Permissions = append([]string{}, reqPermissions...)
		if runBeforeAuthHandler(requestContext(c), c, options, authReq) {
			return
		}

		if len(reqPermissions) == 0 {
			return
		}

		mgr, err := resolveRequestManager(c, options.Manager, options.AuthType)
		if err != nil {
			dispatchFail(c, options, err)
			return
		}

		dCtx := getDContext(c, mgr)
		tokenValue := dCtx.GetTokenValue()

		_, err = authcheck.Check(requestContext(c), mgr, authcheck.Request{
			TokenValue:  tokenValue,
			Permissions: reqPermissions,
			LogicType:   options.LogicType,
		})
		if err != nil {
			dispatchFail(c, options, err)
		}
	}
}

// RoleMiddleware checks roles RoleMiddleware 校验角色
func RoleMiddleware(ctx context.Context, roles []string, opts ...AuthOption) web.FilterFunc {
	options := defaultAuthOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(c *beegocontext.Context) {
		if isRequestAborted(c) {
			return
		}

		authReq := newAuthHandleRequest(options, nil, func() {
			markAborted(c)
		})
		authReq.Roles = append([]string{}, roles...)
		if runBeforeAuthHandler(requestContext(c), c, options, authReq) {
			return
		}

		if len(roles) == 0 {
			return
		}

		mgr, err := resolveRequestManager(c, options.Manager, options.AuthType)
		if err != nil {
			dispatchFail(c, options, err)
			return
		}

		dCtx := getDContext(c, mgr)
		tokenValue := dCtx.GetTokenValue()

		_, err = authcheck.Check(requestContext(c), mgr, authcheck.Request{
			TokenValue: tokenValue,
			Roles:      roles,
			LogicType:  options.LogicType,
		})
		if err != nil {
			dispatchFail(c, options, err)
		}
	}
}

// newAuthHandleRequest creates auth handle request newAuthHandleRequest 创建认证处理请求
func newAuthHandleRequest(options *AuthOptions, next func(), exit func()) *AuthHandleRequest {
	return &AuthHandleRequest{
		AuthType:  options.AuthType,
		LogicType: options.LogicType,
		next:      next,
		exit:      exit,
	}
}

// newRouteAccessRequest creates route access request newRouteAccessRequest 创建路由访问请求
func newRouteAccessRequest(options *AuthOptions) *RouteAccessRequest {
	return &RouteAccessRequest{
		AuthType:  options.AuthType,
		LogicType: options.LogicType,
	}
}

// runBeforeAuthHandler executes pre auth handler runBeforeAuthHandler 执行认证前置处理器
func runBeforeAuthHandler(ctx context.Context, c *beegocontext.Context, options *AuthOptions, req *AuthHandleRequest) bool {
	if options.BeforeAuthHandler == nil {
		return false
	}

	options.BeforeAuthHandler(ctx, c, req)
	return req.IsHandled() || isRequestAborted(c)
}

// dispatchFail writes or dispatches auth failure dispatchFail 写入或分发认证失败响应
func dispatchFail(c *beegocontext.Context, options *AuthOptions, err error) {
	// Started stops return-on-output filters without committing a response status. Started 可停止遇输出即返回的过滤器，且不会提前提交响应状态码。
	markAborted(c)

	if options.FailFunc != nil {
		options.FailFunc(c, err)
		return
	}
	writeErrorResponse(c, err)
}

// isRequestAborted recognizes native output and request-adapter aborts. isRequestAborted 识别原生响应及请求适配器的中止状态。
func isRequestAborted(c *beegocontext.Context) bool {
	if c != nil && c.ResponseWriter != nil && c.ResponseWriter.Started {
		return true
	}
	dCtx, ok := GetDTokenContext(c)
	return ok && dCtx.GetRequestContext() != nil && dCtx.GetRequestContext().IsAborted()
}

// resolveRequestManager honors explicit selection before inheriting the request manager. resolveRequestManager 优先使用显式配置，否则继承请求 Manager。
func resolveRequestManager(c *beegocontext.Context, explicit *manager.Manager, authType string) (*manager.Manager, error) {
	if explicit != nil {
		return authcheck.ResolveManager(explicit, authType)
	}
	cached, _ := GetDTokenContext(c)
	if authType == "" && cached != nil && cached.GetManager() == nil {
		return nil, derror.ErrManagerNotFound
	}
	return authcheck.ResolveManagerFromContext(authType, cached)
}

// GetDTokenContext gets cached DToken context GetDTokenContext 获取缓存的 DToken 上下文
func GetDTokenContext(c *beegocontext.Context) (*DContext.DTokenContext, bool) {
	if c == nil || c.Input == nil {
		return nil, false
	}

	v := c.Input.GetData(DTokenCtxKey)
	ctx, ok := v.(*DContext.DTokenContext)
	if !ok || ctx == nil {
		return nil, false
	}
	return ctx, ok
}

// getDContext gets or creates DToken context getDContext 获取或创建 DToken 上下文
func getDContext(c *beegocontext.Context, mgr *manager.Manager) *DContext.DTokenContext {
	if v := c.Input.GetData(DTokenCtxKey); v != nil {
		if dCtx, ok := v.(*DContext.DTokenContext); ok {
			if dCtx != nil && dCtx.GetManager() == mgr {
				return dCtx
			}
		}
	}

	dCtx := DContext.NewContext(NewBeegoContext(c), mgr)
	c.Input.SetData(DTokenCtxKey, dCtx)

	return dCtx
}

// writeErrorResponse writes error response writeErrorResponse 写入错误响应
func writeErrorResponse(c *beegocontext.Context, err error) {
	code, message := authcheck.GetErrorCodeAndMessage(err)
	httpStatus := getHTTPStatusFromCode(code)

	writeJSON(c, httpStatus, map[string]any{
		"code":    code,
		"message": message,
		"data":    err.Error(),
	})
}

// writeSuccessResponse writes success response writeSuccessResponse 写入成功响应
func writeSuccessResponse(c *beegocontext.Context, data interface{}) {
	writeJSON(c, http.StatusOK, map[string]any{
		"code":    derror.CodeSuccess,
		"message": "success",
		"data":    data,
	})
}

// markAborted marks current request as handled markAborted 标记当前请求已处理
func markAborted(c *beegocontext.Context) {
	if c != nil && c.ResponseWriter != nil && !c.ResponseWriter.Started {
		c.ResponseWriter.Started = true
	}
}

// getHTTPStatusFromCode maps error code to HTTP status getHTTPStatusFromCode 映射错误码到 HTTP 状态码
func getHTTPStatusFromCode(code int) int {
	switch code {
	case derror.CodeNotLogin, derror.CodeTokenInvalid, derror.CodeTokenExpired, derror.CodeActiveTimeout, derror.CodeKickedOut:
		return http.StatusUnauthorized
	case derror.CodePermissionDenied, derror.CodeAccountDisabled:
		return http.StatusForbidden
	case derror.CodeBadRequest:
		return http.StatusBadRequest
	case derror.CodeNotFound:
		return http.StatusNotFound
	case derror.CodeServerError:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}
