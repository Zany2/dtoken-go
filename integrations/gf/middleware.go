package gf

import (
	"context"
	"errors"
	"net/http"

	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/manager"

	DContext "github.com/Zany2/dtoken-go/core/context"
	"github.com/Zany2/dtoken-go/dtoken"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

// LogicType defines the logic type for permission/role checking.
// LogicType 定义权限/角色判断的逻辑类型。
type LogicType string

const (
	DTokenCtxKey = "DCtx"

	// LogicOr represents logical OR operation.
	// LogicOr 表示逻辑或操作。
	LogicOr LogicType = "OR"
	// LogicAnd represents logical AND operation.
	// LogicAnd 表示逻辑与操作。
	LogicAnd LogicType = "AND"
)

type AuthOption func(*AuthOptions)

type AuthOptions struct {
	AuthType  string
	LogicType LogicType
	FailFunc  func(r *ghttp.Request, err error)
}

func defaultAuthOptions() *AuthOptions {
	// Default to AND logic
	// 默认使用 AND 逻辑
	return &AuthOptions{LogicType: LogicAnd}
}

// WithAuthType sets the authentication type.
// WithAuthType 设置认证类型。
func WithAuthType(authType string) AuthOption {
	return func(o *AuthOptions) {
		o.AuthType = authType
	}
}

// WithLogicType sets the logic type for permission/role checking.
// WithLogicType 设置权限/角色判断的逻辑类型。
func WithLogicType(logicType LogicType) AuthOption {
	return func(o *AuthOptions) {
		o.LogicType = logicType
	}
}

// WithFailFunc sets the authentication failure callback function.
// WithFailFunc 设置认证失败时的回调函数。
func WithFailFunc(fn func(r *ghttp.Request, err error)) AuthOption {
	return func(o *AuthOptions) {
		o.FailFunc = fn
	}
}

// ============================================================================
// Middlewares - 中间件
// ============================================================================

// RegisterDTokenContextMiddleware initializes DToken context for each request.
// RegisterDTokenContextMiddleware 为每个请求初始化 DToken 上下文。
func RegisterDTokenContextMiddleware(ctx context.Context, opts ...AuthOption) ghttp.HandlerFunc {
	options := defaultAuthOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(r *ghttp.Request) {
		mgr, err := dtoken.GetManager(options.AuthType)
		if err != nil {
			if options.FailFunc != nil {
				options.FailFunc(r, err)
			} else {
				writeErrorResponse(r, err)
			}
			return
		}

		_ = getDContext(r, mgr)
		r.Middleware.Next()
	}
}

// AuthMiddleware provides authentication middleware functionality.
// AuthMiddleware 提供认证中间件功能。
func AuthMiddleware(ctx context.Context, opts ...AuthOption) ghttp.HandlerFunc {
	options := defaultAuthOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(r *ghttp.Request) {
		mgr, err := dtoken.GetManager(options.AuthType)
		if err != nil {
			if options.FailFunc != nil {
				options.FailFunc(r, err)
			} else {
				writeErrorResponse(r, err)
			}
			return
		}

		dCtx := getDContext(r, mgr)
		tokenValue := dCtx.GetTokenValue()

		// Check if user is logged in
		// 检查用户是否已登录
		if !dtoken.IsLogin(ctx, tokenValue) {
			if options.FailFunc != nil {
				options.FailFunc(r, derror.ErrTokenExpired)
			} else {
				writeErrorResponse(r, derror.ErrTokenExpired)
			}
			return
		}

		r.Middleware.Next()
	}
}

// PermissionMiddleware provides permission checking middleware functionality.
// PermissionMiddleware 提供权限校验中间件功能。
func PermissionMiddleware(
	ctx context.Context,
	permissions []string,
	opts ...AuthOption,
) ghttp.HandlerFunc {

	options := defaultAuthOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(r *ghttp.Request) {
		// No permission required, pass through directly
		// 无需权限时直接放行
		if len(permissions) == 0 {
			r.Middleware.Next()
			return
		}

		// Get Manager instance
		// 获取 Manager 实例
		mgr, err := dtoken.GetManager(options.AuthType)
		if err != nil {
			if options.FailFunc != nil {
				options.FailFunc(r, err)
			} else {
				writeErrorResponse(r, err)
			}
			return
		}

		dCtx := getDContext(r, mgr)
		tokenValue := dCtx.GetTokenValue()

		// Check permissions
		// 检查权限
		var ok bool
		if options.LogicType == LogicAnd {
			ok = mgr.HasPermissionsAndByToken(ctx, tokenValue, permissions)
		} else {
			ok = mgr.HasPermissionsOrByToken(ctx, tokenValue, permissions)
		}

		if !ok {
			if options.FailFunc != nil {
				options.FailFunc(r, derror.ErrPermissionDenied)
			} else {
				writeErrorResponse(r, derror.ErrPermissionDenied)
			}
			return
		}

		r.Middleware.Next()
	}
}

// PermissionPathMiddleware provides path-based permission checking middleware functionality.
// PermissionPathMiddleware 提供基于路径的权限校验中间件功能。
func PermissionPathMiddleware(
	ctx context.Context,
	permissions []string,
	opts ...AuthOption,
) ghttp.HandlerFunc {

	options := defaultAuthOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(r *ghttp.Request) {
		// Create a per-request copy of permissions and append current path
		// 为每个请求创建权限副本并追加当前路径
		reqPermissions := append([]string{}, permissions...)
		reqPermissions = append(reqPermissions, r.URL.Path)

		if len(reqPermissions) == 0 {
			r.Middleware.Next()
			return
		}

		// Get Manager instance
		// 获取 Manager 实例
		mgr, err := dtoken.GetManager(options.AuthType)
		if err != nil {
			if options.FailFunc != nil {
				options.FailFunc(r, err)
			} else {
				writeErrorResponse(r, err)
			}
			return
		}

		dCtx := getDContext(r, mgr)
		tokenValue := dCtx.GetTokenValue()

		// Check permissions
		// 检查权限
		var ok bool
		if options.LogicType == LogicAnd {
			ok = mgr.HasPermissionsAndByToken(ctx, tokenValue, reqPermissions)
		} else {
			ok = mgr.HasPermissionsOrByToken(ctx, tokenValue, reqPermissions)
		}

		if !ok {
			if options.FailFunc != nil {
				options.FailFunc(r, derror.ErrPermissionDenied)
			} else {
				writeErrorResponse(r, derror.ErrPermissionDenied)
			}
			return
		}

		r.Middleware.Next()
	}
}

// RoleMiddleware provides role checking middleware functionality.
// RoleMiddleware 提供角色校验中间件功能。
func RoleMiddleware(
	ctx context.Context,
	roles []string,
	opts ...AuthOption,
) ghttp.HandlerFunc {

	options := defaultAuthOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(r *ghttp.Request) {
		// No role required, pass through directly
		// 无需角色时直接放行
		if len(roles) == 0 {
			r.Middleware.Next()
			return
		}

		// Get Manager instance
		// 获取 Manager 实例
		mgr, err := dtoken.GetManager(options.AuthType)
		if err != nil {
			if options.FailFunc != nil {
				options.FailFunc(r, err)
			} else {
				writeErrorResponse(r, err)
			}
			return
		}

		dCtx := getDContext(r, mgr)
		tokenValue := dCtx.GetTokenValue()

		// Check roles
		// 检查角色
		var ok bool
		if options.LogicType == LogicAnd {
			ok = mgr.HasRolesAndByToken(ctx, tokenValue, roles)
		} else {
			ok = mgr.HasRolesOrByToken(ctx, tokenValue, roles)
		}

		if !ok {
			if options.FailFunc != nil {
				options.FailFunc(r, derror.ErrRoleDenied)
			} else {
				writeErrorResponse(r, derror.ErrRoleDenied)
			}
			return
		}

		r.Middleware.Next()
	}
}

// GetDTokenContext gets DToken context from GoFrame request.
// GetDTokenContext 从 GoFrame 请求中获取 DToken 上下文。
func GetDTokenContext(r *ghttp.Request) (*DContext.DTokenContext, bool) {
	v := r.GetCtxVar(DTokenCtxKey)
	if v == nil {
		return nil, false
	}

	ctx, ok := v.Val().(*DContext.DTokenContext)
	return ctx, ok
}

// GetDTokenContextByCtx gets DToken context from GoFrame context.
// GetDTokenContextByCtx 从 GoFrame 上下文中获取 DToken 上下文。
func GetDTokenContextByCtx(ctx context.Context) (*DContext.DTokenContext, bool) {
	request := g.RequestFromCtx(ctx)
	ctxVar := request.GetCtxVar(DTokenCtxKey)
	if ctxVar == nil {
		return nil, false
	}

	tokenContext, ok := ctxVar.Val().(*DContext.DTokenContext)
	return tokenContext, ok
}

// GetLoginIDByCtx gets the login ID from the context.
// GetLoginIDByCtx 从上下文中获取登录 ID。
func GetLoginIDByCtx(ctx context.Context, authType ...string) (string, error) {
	mgr, err := dtoken.GetManager(authType...)
	if err != nil {
		return "", err
	}

	tokenValue := getDContext(g.RequestFromCtx(ctx), mgr).GetTokenValue()
	return mgr.GetLoginID(ctx, tokenValue)
}

// GetTokenInfoByCtx gets the token information from the context.
// GetTokenInfoByCtx 从上下文中获取 Token 信息。
func GetTokenInfoByCtx(ctx context.Context, authType ...string) (*manager.TokenInfo, error) {
	mgr, err := dtoken.GetManager(authType...)
	if err != nil {
		return nil, err
	}

	return mgr.GetTokenInfo(ctx, getDContext(g.RequestFromCtx(ctx), mgr).GetTokenValue())
}

// getDContext returns or creates the DToken context for the request.
// getDContext 获取或创建当前请求的 DToken 上下文。
func getDContext(r *ghttp.Request, mgr *manager.Manager) *DContext.DTokenContext {
	// Try to get from context
	// 尝试从上下文中获取
	if v := r.GetCtxVar(DTokenCtxKey); v != nil {
		// gvar.Var -> interface{} -> *DTokenContext
		if dCtx, ok := v.Val().(*DContext.DTokenContext); ok {
			return dCtx
		}
	}

	// Create new context and cache it
	// 创建新的上下文并缓存
	dCtx := DContext.NewContext(NewGFContext(r), mgr)
	r.SetCtxVar(DTokenCtxKey, dCtx)

	return dCtx
}

// ============================================================================
// Error Handling Helpers - 错误处理辅助函数
// ============================================================================

// writeErrorResponse writes a standardized error response.
// writeErrorResponse 写入标准化的错误响应。
func writeErrorResponse(r *ghttp.Request, err error) {
	var saErr *derror.DTokenError
	var code int
	var message string
	var httpStatus int

	// Check if it's a DTokenError
	// 检查是否为 DTokenError
	if errors.As(err, &saErr) {
		code = saErr.Code
		message = saErr.Message
		httpStatus = getHTTPStatusFromCode(code)
	} else {
		// Handle standard errors
		// 处理标准错误
		code = derror.CodeServerError
		message = err.Error()
		httpStatus = http.StatusInternalServerError
	}

	r.Response.WriteStatusExit(httpStatus, g.Map{
		"code":    code,
		"message": message,
		"data":    err.Error(),
	})
}

// writeSuccessResponse writes a standardized success response.
// writeSuccessResponse 写入标准化的成功响应。
func writeSuccessResponse(r *ghttp.Request, data interface{}) {
	r.Response.WriteStatusExit(http.StatusOK, g.Map{
		"code":    derror.CodeSuccess,
		"message": "success",
		"data":    data,
	})
}

// getHTTPStatusFromCode converts DToken error code to HTTP status code.
// getHTTPStatusFromCode 将 DToken 错误码转换为 HTTP 状态码。
func getHTTPStatusFromCode(code int) int {
	switch code {
	case derror.CodeNotLogin:
		return http.StatusUnauthorized
	case derror.CodePermissionDenied:
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
