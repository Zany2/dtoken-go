package hertz

import (
	"context"
	"errors"
	"net/http"

	corecontext "github.com/Zany2/dtoken-go/core/context"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/manager"
	"github.com/Zany2/dtoken-go/dtoken"
	hertzapp "github.com/cloudwego/hertz/pkg/app"
)

// LogicType defines middleware logic type LogicType 定义中间件逻辑类型
type LogicType string

const (
	// DTokenCtxKey stores request scoped DToken context DTokenCtxKey 存储请求级 DToken 上下文
	DTokenCtxKey = "DCtx"

	// LogicOr represents OR logic LogicOr 表示或逻辑
	LogicOr LogicType = "OR"
	// LogicAnd represents AND logic LogicAnd 表示与逻辑
	LogicAnd LogicType = "AND"
)

// AuthOption defines auth option setter AuthOption 定义认证选项设置器
type AuthOption func(*AuthOptions)

// AuthOptions defines middleware auth options AuthOptions 定义中间件认证选项
type AuthOptions struct {
	AuthType  string
	LogicType LogicType
	FailFunc  func(c context.Context, ctx *hertzapp.RequestContext, err error)
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
func WithFailFunc(fn func(c context.Context, ctx *hertzapp.RequestContext, err error)) AuthOption {
	return func(o *AuthOptions) {
		o.FailFunc = fn
	}
}

// RegisterDTokenContextMiddleware registers DToken context middleware RegisterDTokenContextMiddleware 注册 DToken 上下文中间件
func RegisterDTokenContextMiddleware(ctx context.Context, opts ...AuthOption) hertzapp.HandlerFunc {
	options := defaultAuthOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(c context.Context, reqCtx *hertzapp.RequestContext) {
		mgr, err := dtoken.GetManager(options.AuthType)
		if err != nil {
			if options.FailFunc != nil {
				options.FailFunc(c, reqCtx, err)
			} else {
				writeErrorResponse(reqCtx, err)
			}
			reqCtx.Abort()
			return
		}

		_ = getDTokenContext(reqCtx, mgr)
		reqCtx.Next(c)
	}
}

// AuthMiddleware checks login status AuthMiddleware 校验登录状态
func AuthMiddleware(ctx context.Context, opts ...AuthOption) hertzapp.HandlerFunc {
	options := defaultAuthOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(c context.Context, reqCtx *hertzapp.RequestContext) {
		mgr, err := dtoken.GetManager(options.AuthType)
		if err != nil {
			if options.FailFunc != nil {
				options.FailFunc(c, reqCtx, err)
			} else {
				writeErrorResponse(reqCtx, err)
			}
			reqCtx.Abort()
			return
		}

		dCtx := getDTokenContext(reqCtx, mgr)
		tokenValue := dCtx.GetTokenValue()

		if !mgr.IsLogin(ctx, tokenValue) {
			if options.FailFunc != nil {
				options.FailFunc(c, reqCtx, derror.ErrTokenExpired)
			} else {
				writeErrorResponse(reqCtx, derror.ErrTokenExpired)
			}
			reqCtx.Abort()
			return
		}

		reqCtx.Next(c)
	}
}

// PermissionMiddleware checks permissions PermissionMiddleware 校验权限
func PermissionMiddleware(
	ctx context.Context,
	permissions []string,
	opts ...AuthOption,
) hertzapp.HandlerFunc {
	options := defaultAuthOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(c context.Context, reqCtx *hertzapp.RequestContext) {
		if len(permissions) == 0 {
			reqCtx.Next(c)
			return
		}

		mgr, err := dtoken.GetManager(options.AuthType)
		if err != nil {
			if options.FailFunc != nil {
				options.FailFunc(c, reqCtx, err)
			} else {
				writeErrorResponse(reqCtx, err)
			}
			reqCtx.Abort()
			return
		}

		dCtx := getDTokenContext(reqCtx, mgr)
		tokenValue := dCtx.GetTokenValue()

		var ok bool
		if options.LogicType == LogicAnd {
			ok = mgr.HasPermissionsAndByToken(ctx, tokenValue, permissions)
		} else {
			ok = mgr.HasPermissionsOrByToken(ctx, tokenValue, permissions)
		}

		if !ok {
			if options.FailFunc != nil {
				options.FailFunc(c, reqCtx, derror.ErrPermissionDenied)
			} else {
				writeErrorResponse(reqCtx, derror.ErrPermissionDenied)
			}
			reqCtx.Abort()
			return
		}

		reqCtx.Next(c)
	}
}

// RoleMiddleware checks roles RoleMiddleware 校验角色
func RoleMiddleware(
	ctx context.Context,
	roles []string,
	opts ...AuthOption,
) hertzapp.HandlerFunc {
	options := defaultAuthOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(c context.Context, reqCtx *hertzapp.RequestContext) {
		if len(roles) == 0 {
			reqCtx.Next(c)
			return
		}

		mgr, err := dtoken.GetManager(options.AuthType)
		if err != nil {
			if options.FailFunc != nil {
				options.FailFunc(c, reqCtx, err)
			} else {
				writeErrorResponse(reqCtx, err)
			}
			reqCtx.Abort()
			return
		}

		dCtx := getDTokenContext(reqCtx, mgr)
		tokenValue := dCtx.GetTokenValue()

		var ok bool
		if options.LogicType == LogicAnd {
			ok = mgr.HasRolesAndByToken(ctx, tokenValue, roles)
		} else {
			ok = mgr.HasRolesOrByToken(ctx, tokenValue, roles)
		}

		if !ok {
			if options.FailFunc != nil {
				options.FailFunc(c, reqCtx, derror.ErrRoleDenied)
			} else {
				writeErrorResponse(reqCtx, derror.ErrRoleDenied)
			}
			reqCtx.Abort()
			return
		}

		reqCtx.Next(c)
	}
}

// GetDTokenContext gets cached DToken context GetDTokenContext 获取缓存的 DToken 上下文
func GetDTokenContext(ctx *hertzapp.RequestContext) (*corecontext.DTokenContext, bool) {
	value, exists := ctx.Get(DTokenCtxKey)
	if !exists {
		return nil, false
	}

	dCtx, ok := value.(*corecontext.DTokenContext)
	return dCtx, ok
}

// getDTokenContext gets or creates dtoken context getDTokenContext 获取或创建 DToken 上下文
func getDTokenContext(ctx *hertzapp.RequestContext, mgr *manager.Manager) *corecontext.DTokenContext {
	if value, exists := ctx.Get(DTokenCtxKey); exists {
		if dCtx, ok := value.(*corecontext.DTokenContext); ok {
			return dCtx
		}
	}

	dCtx := corecontext.NewContext(NewHertzContext(ctx), mgr)
	ctx.Set(DTokenCtxKey, dCtx)
	return dCtx
}

// writeErrorResponse writes error response writeErrorResponse 写入错误响应
func writeErrorResponse(ctx *hertzapp.RequestContext, err error) {
	var dErr *derror.DTokenError
	var code int
	var message string
	var httpStatus int

	if errors.As(err, &dErr) {
		code = dErr.Code
		message = dErr.Message
		httpStatus = getHTTPStatusFromCode(code)
	} else {
		code = derror.CodeServerError
		message = err.Error()
		httpStatus = http.StatusInternalServerError
	}

	ctx.JSON(httpStatus, map[string]interface{}{
		"code":    code,
		"message": message,
		"data":    err.Error(),
	})
}

// writeSuccessResponse writes success response writeSuccessResponse 写入成功响应
func writeSuccessResponse(ctx *hertzapp.RequestContext, data interface{}) {
	ctx.JSON(http.StatusOK, map[string]interface{}{
		"code":    derror.CodeSuccess,
		"message": "success",
		"data":    data,
	})
}

// getHTTPStatusFromCode maps error code to HTTP status getHTTPStatusFromCode 映射错误码到 HTTP 状态码
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
