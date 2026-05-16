// @Author daixk 2025/12/22 15:56:00
package kratos

import (
	"context"
	"net/http"

	corecontext "github.com/Zany2/dtoken-go/core/context"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/manager"
	"github.com/Zany2/dtoken-go/integrations/internal/authcheck"
	kerrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
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

// FailFunc defines auth failure callback FailFunc 定义认证失败回调
type FailFunc func(ctx context.Context, err error) error

// AuthOption defines auth option setter AuthOption 定义认证选项设置器
type AuthOption func(*AuthOptions)

// AuthOptions defines middleware auth options AuthOptions 定义中间件认证选项
type AuthOptions struct {
	AuthType  string
	LogicType LogicType
	FailFunc  FailFunc
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
func WithFailFunc(fn FailFunc) AuthOption {
	return func(o *AuthOptions) {
		o.FailFunc = fn
	}
}

// RegisterDTokenContextMiddleware registers DToken context middleware RegisterDTokenContextMiddleware 注册 DToken 上下文中间件
func RegisterDTokenContextMiddleware(opts ...AuthOption) middleware.Middleware {
	options := defaultAuthOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			mgr, err := authcheck.GetManager(options.AuthType)
			if err != nil {
				return nil, dispatchFail(ctx, options.FailFunc, err)
			}

			_, ctx = getDTokenContext(ctx, mgr)
			return next(ctx, req)
		}
	}
}

// AuthMiddleware checks login status AuthMiddleware 校验登录状态
func AuthMiddleware(opts ...AuthOption) middleware.Middleware {
	options := defaultAuthOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			mgr, err := authcheck.GetManager(options.AuthType)
			if err != nil {
				return nil, dispatchFail(ctx, options.FailFunc, err)
			}

			dCtx, ctx := getDTokenContext(ctx, mgr)
			_, err = authcheck.Check(ctx, mgr, authcheck.Request{
				TokenValue: dCtx.GetTokenValue(),
				CheckLogin: true,
				LoginError: derror.ErrNotLogin,
			})
			if err != nil {
				return nil, dispatchFail(ctx, options.FailFunc, err)
			}

			return next(ctx, req)
		}
	}
}

// PermissionMiddleware checks permissions PermissionMiddleware 校验权限
func PermissionMiddleware(permissions []string, opts ...AuthOption) middleware.Middleware {
	options := defaultAuthOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			if len(permissions) == 0 {
				return next(ctx, req)
			}

			mgr, err := authcheck.GetManager(options.AuthType)
			if err != nil {
				return nil, dispatchFail(ctx, options.FailFunc, err)
			}

			dCtx, ctx := getDTokenContext(ctx, mgr)
			tokenValue := dCtx.GetTokenValue()

			_, err = authcheck.Check(ctx, mgr, authcheck.Request{
				TokenValue:  tokenValue,
				Permissions: permissions,
				LogicType:   options.LogicType,
			})
			if err != nil {
				return nil, dispatchFail(ctx, options.FailFunc, err)
			}

			return next(ctx, req)
		}
	}
}

// PermissionPathMiddleware checks path permissions PermissionPathMiddleware 基于路径校验权限
func PermissionPathMiddleware(permissions []string, opts ...AuthOption) middleware.Middleware {
	options := defaultAuthOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			reqPermissions := append([]string{}, permissions...)
			if path := NewKratosContext(ctx).GetPath(); path != "" {
				reqPermissions = append(reqPermissions, path)
			}

			if len(reqPermissions) == 0 {
				return next(ctx, req)
			}

			mgr, err := authcheck.GetManager(options.AuthType)
			if err != nil {
				return nil, dispatchFail(ctx, options.FailFunc, err)
			}

			dCtx, ctx := getDTokenContext(ctx, mgr)
			tokenValue := dCtx.GetTokenValue()

			_, err = authcheck.Check(ctx, mgr, authcheck.Request{
				TokenValue:  tokenValue,
				Permissions: reqPermissions,
				LogicType:   options.LogicType,
			})
			if err != nil {
				return nil, dispatchFail(ctx, options.FailFunc, err)
			}

			return next(ctx, req)
		}
	}
}

// RoleMiddleware checks roles RoleMiddleware 校验角色
func RoleMiddleware(roles []string, opts ...AuthOption) middleware.Middleware {
	options := defaultAuthOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			if len(roles) == 0 {
				return next(ctx, req)
			}

			mgr, err := authcheck.GetManager(options.AuthType)
			if err != nil {
				return nil, dispatchFail(ctx, options.FailFunc, err)
			}

			dCtx, ctx := getDTokenContext(ctx, mgr)
			tokenValue := dCtx.GetTokenValue()

			_, err = authcheck.Check(ctx, mgr, authcheck.Request{
				TokenValue: tokenValue,
				Roles:      roles,
				LogicType:  options.LogicType,
			})
			if err != nil {
				return nil, dispatchFail(ctx, options.FailFunc, err)
			}

			return next(ctx, req)
		}
	}
}

// GetDTokenContext gets cached DToken context GetDTokenContext 获取缓存的 DToken 上下文
func GetDTokenContext(ctx context.Context) (*corecontext.DTokenContext, bool) {
	value := ctx.Value(DTokenCtxKey)
	if value == nil {
		return nil, false
	}

	dCtx, ok := value.(*corecontext.DTokenContext)
	return dCtx, ok
}

// GetDTokenContextByCtx gets cached DToken context by ctx GetDTokenContextByCtx 从上下文获取 DToken 上下文
func GetDTokenContextByCtx(ctx context.Context) (*corecontext.DTokenContext, bool) {
	return GetDTokenContext(ctx)
}

// GetLoginIDByCtx gets login ID by context GetLoginIDByCtx 从上下文获取登录 ID
func GetLoginIDByCtx(ctx context.Context, authType ...string) (string, error) {
	mgr, err := authcheck.GetManager(firstAuthType(authType...))
	if err != nil {
		return "", err
	}

	dCtx, ctx := getDTokenContext(ctx, mgr)
	return mgr.GetLoginID(ctx, dCtx.GetTokenValue())
}

// GetTokenInfoByCtx gets token info by context GetTokenInfoByCtx 从上下文获取 Token 信息
func GetTokenInfoByCtx(ctx context.Context, authType ...string) (*manager.TokenInfo, error) {
	mgr, err := authcheck.GetManager(firstAuthType(authType...))
	if err != nil {
		return nil, err
	}

	dCtx, ctx := getDTokenContext(ctx, mgr)
	return mgr.GetTokenInfo(ctx, dCtx.GetTokenValue())
}

// getDTokenContext gets or creates dtoken context getDTokenContext 获取或创建 DToken 上下文
func getDTokenContext(ctx context.Context, mgr *manager.Manager) (*corecontext.DTokenContext, context.Context) {
	if dCtx, ok := GetDTokenContext(ctx); ok {
		return dCtx, ctx
	}

	kratosCtx := NewKratosContext(ctx).(*KratosContext)
	dCtx := corecontext.NewContext(kratosCtx, mgr)
	ctx = context.WithValue(ctx, DTokenCtxKey, dCtx)
	kratosCtx.ctx = ctx

	return dCtx, ctx
}

// dispatchFail dispatches auth failure dispatchFail 分发认证失败处理
func dispatchFail(ctx context.Context, failFunc FailFunc, err error) error {
	if failFunc != nil {
		return failFunc(ctx, err)
	}
	return writeErrorResponse(err)
}

// writeErrorResponse converts error to kratos error writeErrorResponse 转换为 Kratos 错误
func writeErrorResponse(err error) error {
	code, message := getErrorCodeAndMessage(err)
	return kerrors.New(getHTTPStatusFromCode(code), getReasonFromCode(code), message).WithCause(err)
}

// getErrorCodeAndMessage gets error code and message getErrorCodeAndMessage 获取错误码和错误消息
func getErrorCodeAndMessage(err error) (int, string) {
	return authcheck.GetErrorCodeAndMessage(err)
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
	default:
		return http.StatusInternalServerError
	}
}

// firstAuthType returns the first optional auth type firstAuthType 返回第一个可选认证类型
func firstAuthType(authType ...string) string {
	if len(authType) == 0 {
		return ""
	}
	return authType[0]
}

// getReasonFromCode maps error code to reason getReasonFromCode 映射错误码到错误原因
func getReasonFromCode(code int) string {
	switch code {
	case derror.CodeNotLogin, derror.CodeTokenInvalid, derror.CodeTokenExpired, derror.CodeActiveTimeout, derror.CodeKickedOut:
		return "UNAUTHORIZED"
	case derror.CodePermissionDenied:
		return "PERMISSION_DENIED"
	case derror.CodeAccountDisabled:
		return "ACCOUNT_DISABLED"
	case derror.CodeBadRequest:
		return "BAD_REQUEST"
	case derror.CodeNotFound:
		return "NOT_FOUND"
	default:
		return "INTERNAL_SERVER_ERROR"
	}
}
