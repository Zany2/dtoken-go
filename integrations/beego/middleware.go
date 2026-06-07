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
	AuthType     string
	CheckLogin   bool
	CheckDisable bool
	Permissions  []string
	Roles        []string
	LogicType    LogicType

	next    func()
	exit    func()
	handled bool
}

// Next continues request and stops dtoken checks Next 放行请求并停止 dtoken 校验
func (req *AuthHandleRequest) Next() {
	req.handled = true
	if req.next != nil {
		req.next()
	}
}

// Exit stops dtoken checks after custom handling Exit 自定义处理后停止 dtoken 校验
func (req *AuthHandleRequest) Exit() {
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
	AuthType     string
	LogicType    LogicType
	CheckDisable bool
	Permissions  []string
	Roles        []string

	skipAuth       bool
	skipPermission bool
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
	AuthType           string
	LogicType          LogicType
	FailFunc           func(c *beegocontext.Context, err error)
	BeforeAuthHandler  BeforeAuthHandler
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
		mgr, err := authcheck.GetManager(options.AuthType)
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
		authReq := newAuthHandleRequest(options, nil, func() {
			markAborted(c)
		})
		authReq.CheckLogin = true
		if runBeforeAuthHandler(ctx, c, options, authReq) {
			return
		}

		mgr, err := authcheck.GetManager(options.AuthType)
		if err != nil {
			dispatchFail(c, options, err)
			return
		}

		dCtx := getDContext(c, mgr)
		tokenValue := dCtx.GetTokenValue()

		_, err = authcheck.Check(ctx, mgr, authcheck.Request{
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
		accessReq := newRouteAccessRequest(options)
		if options.RouteAccessHandler != nil {
			options.RouteAccessHandler(ctx, c, accessReq)
		}

		if accessReq.skipAuth {
			return
		}

		mgr, err := authcheck.GetManager(accessReq.AuthType)
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

		_, err = authcheck.Check(ctx, mgr, req)
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
		authReq := newAuthHandleRequest(options, nil, func() {
			markAborted(c)
		})
		authReq.Permissions = append([]string{}, permissions...)
		if runBeforeAuthHandler(ctx, c, options, authReq) {
			return
		}

		if len(permissions) == 0 {
			return
		}

		mgr, err := authcheck.GetManager(options.AuthType)
		if err != nil {
			dispatchFail(c, options, err)
			return
		}

		dCtx := getDContext(c, mgr)
		tokenValue := dCtx.GetTokenValue()

		_, err = authcheck.Check(ctx, mgr, authcheck.Request{
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
		reqPermissions := append([]string{}, permissions...)
		reqPermissions = append(reqPermissions, c.Request.URL.Path)

		authReq := newAuthHandleRequest(options, nil, func() {
			markAborted(c)
		})
		authReq.Permissions = append([]string{}, reqPermissions...)
		if runBeforeAuthHandler(ctx, c, options, authReq) {
			return
		}

		if len(reqPermissions) == 0 {
			return
		}

		mgr, err := authcheck.GetManager(options.AuthType)
		if err != nil {
			dispatchFail(c, options, err)
			return
		}

		dCtx := getDContext(c, mgr)
		tokenValue := dCtx.GetTokenValue()

		_, err = authcheck.Check(ctx, mgr, authcheck.Request{
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
		authReq := newAuthHandleRequest(options, nil, func() {
			markAborted(c)
		})
		authReq.Roles = append([]string{}, roles...)
		if runBeforeAuthHandler(ctx, c, options, authReq) {
			return
		}

		if len(roles) == 0 {
			return
		}

		mgr, err := authcheck.GetManager(options.AuthType)
		if err != nil {
			dispatchFail(c, options, err)
			return
		}

		dCtx := getDContext(c, mgr)
		tokenValue := dCtx.GetTokenValue()

		_, err = authcheck.Check(ctx, mgr, authcheck.Request{
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
	return req.IsHandled()
}

// dispatchFail writes or dispatches auth failure dispatchFail 写入或分发认证失败响应
func dispatchFail(c *beegocontext.Context, options *AuthOptions, err error) {
	if options.FailFunc != nil {
		options.FailFunc(c, err)
		return
	}
	writeErrorResponse(c, err)
}

// GetDTokenContext gets cached DToken context GetDTokenContext 获取缓存的 DToken 上下文
func GetDTokenContext(c *beegocontext.Context) (*DContext.DTokenContext, bool) {
	v := c.Input.GetData(DTokenCtxKey)
	ctx, ok := v.(*DContext.DTokenContext)
	return ctx, ok
}

// getDContext gets or creates DToken context getDContext 获取或创建 DToken 上下文
func getDContext(c *beegocontext.Context, mgr *manager.Manager) *DContext.DTokenContext {
	if v := c.Input.GetData(DTokenCtxKey); v != nil {
		if dCtx, ok := v.(*DContext.DTokenContext); ok {
			if dCtx.GetManager() == mgr {
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
