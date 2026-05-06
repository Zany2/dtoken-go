package echo

import (
	"context"
	"errors"
	"net/http"

	corecontext "github.com/Zany2/dtoken-go/core/context"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/manager"
	"github.com/Zany2/dtoken-go/dtoken"
	echo4 "github.com/labstack/echo/v4"
)

// LogicType defines permission and role check logic LogicType 定义权限与角色校验逻辑
type LogicType string

const (
	// DTokenCtxKey stores request scoped DToken context DTokenCtxKey 存储请求级 DToken 上下文
	DTokenCtxKey = "DCtx"

	// LogicOr uses OR logic for checks LogicOr 使用 OR 逻辑校验
	LogicOr LogicType = "OR"
	// LogicAnd uses AND logic for checks LogicAnd 使用 AND 逻辑校验
	LogicAnd LogicType = "AND"
)

// AuthOption defines auth option setter AuthOption 定义认证选项设置器
type AuthOption func(*AuthOptions)

// AuthOptions carries middleware auth options AuthOptions 保存中间件认证选项
type AuthOptions struct {
	AuthType  string
	LogicType LogicType
	FailFunc  func(c echo4.Context, err error) error
}

// defaultAuthOptions returns default middleware options defaultAuthOptions 返回默认中间件选项
func defaultAuthOptions() *AuthOptions {
	return &AuthOptions{LogicType: LogicAnd}
}

// WithAuthType sets middleware auth type WithAuthType 设置中间件认证类型
func WithAuthType(authType string) AuthOption {
	return func(o *AuthOptions) {
		o.AuthType = authType
	}
}

// WithLogicType sets middleware logic type WithLogicType 设置中间件逻辑类型
func WithLogicType(logicType LogicType) AuthOption {
	return func(o *AuthOptions) {
		o.LogicType = logicType
	}
}

// WithFailFunc sets auth failure callback WithFailFunc 设置认证失败回调
func WithFailFunc(fn func(c echo4.Context, err error) error) AuthOption {
	return func(o *AuthOptions) {
		o.FailFunc = fn
	}
}

// RegisterDTokenContextMiddleware initializes DToken context per request RegisterDTokenContextMiddleware 初始化每个请求的 DToken 上下文
func RegisterDTokenContextMiddleware(ctx context.Context, opts ...AuthOption) echo4.MiddlewareFunc {
	options := defaultAuthOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(next echo4.HandlerFunc) echo4.HandlerFunc {
		return func(c echo4.Context) error {
			mgr, err := dtoken.GetManager(options.AuthType)
			if err != nil {
				if options.FailFunc != nil {
					return options.FailFunc(c, err)
				}
				return writeErrorResponse(c, err)
			}

			_ = getDTokenContext(c, mgr)
			return next(c)
		}
	}
}

// AuthMiddleware checks whether current request is authenticated AuthMiddleware 校验当前请求是否已认证
func AuthMiddleware(ctx context.Context, opts ...AuthOption) echo4.MiddlewareFunc {
	options := defaultAuthOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(next echo4.HandlerFunc) echo4.HandlerFunc {
		return func(c echo4.Context) error {
			mgr, err := dtoken.GetManager(options.AuthType)
			if err != nil {
				if options.FailFunc != nil {
					return options.FailFunc(c, err)
				}
				return writeErrorResponse(c, err)
			}

			dCtx := getDTokenContext(c, mgr)
			tokenValue := dCtx.GetTokenValue()
			if !mgr.IsLogin(ctx, tokenValue) {
				if options.FailFunc != nil {
					return options.FailFunc(c, derror.ErrTokenExpired)
				}
				return writeErrorResponse(c, derror.ErrTokenExpired)
			}

			return next(c)
		}
	}
}

// PermissionMiddleware checks whether current token has required permissions PermissionMiddleware 校验当前 token 是否具备所需权限
func PermissionMiddleware(ctx context.Context, permissions []string, opts ...AuthOption) echo4.MiddlewareFunc {
	options := defaultAuthOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(next echo4.HandlerFunc) echo4.HandlerFunc {
		return func(c echo4.Context) error {
			if len(permissions) == 0 {
				return next(c)
			}

			mgr, err := dtoken.GetManager(options.AuthType)
			if err != nil {
				if options.FailFunc != nil {
					return options.FailFunc(c, err)
				}
				return writeErrorResponse(c, err)
			}

			dCtx := getDTokenContext(c, mgr)
			tokenValue := dCtx.GetTokenValue()

			var ok bool
			if options.LogicType == LogicAnd {
				ok = mgr.HasPermissionsAndByToken(ctx, tokenValue, permissions)
			} else {
				ok = mgr.HasPermissionsOrByToken(ctx, tokenValue, permissions)
			}

			if !ok {
				if options.FailFunc != nil {
					return options.FailFunc(c, derror.ErrPermissionDenied)
				}
				return writeErrorResponse(c, derror.ErrPermissionDenied)
			}

			return next(c)
		}
	}
}

// RoleMiddleware checks whether current token has required roles RoleMiddleware 校验当前 token 是否具备所需角色
func RoleMiddleware(ctx context.Context, roles []string, opts ...AuthOption) echo4.MiddlewareFunc {
	options := defaultAuthOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(next echo4.HandlerFunc) echo4.HandlerFunc {
		return func(c echo4.Context) error {
			if len(roles) == 0 {
				return next(c)
			}

			mgr, err := dtoken.GetManager(options.AuthType)
			if err != nil {
				if options.FailFunc != nil {
					return options.FailFunc(c, err)
				}
				return writeErrorResponse(c, err)
			}

			dCtx := getDTokenContext(c, mgr)
			tokenValue := dCtx.GetTokenValue()

			var ok bool
			if options.LogicType == LogicAnd {
				ok = mgr.HasRolesAndByToken(ctx, tokenValue, roles)
			} else {
				ok = mgr.HasRolesOrByToken(ctx, tokenValue, roles)
			}

			if !ok {
				if options.FailFunc != nil {
					return options.FailFunc(c, derror.ErrRoleDenied)
				}
				return writeErrorResponse(c, derror.ErrRoleDenied)
			}

			return next(c)
		}
	}
}

// GetDTokenContext gets cached DToken context from Echo request GetDTokenContext 从 Echo 请求中获取缓存的 DToken 上下文
func GetDTokenContext(c echo4.Context) (*corecontext.DTokenContext, bool) {
	value := c.Get(DTokenCtxKey)
	if value == nil {
		return nil, false
	}

	dCtx, ok := value.(*corecontext.DTokenContext)
	return dCtx, ok
}

// getDTokenContext gets or creates dtoken context getDTokenContext 获取或创建 DToken 上下文
func getDTokenContext(c echo4.Context, mgr *manager.Manager) *corecontext.DTokenContext {
	if value := c.Get(DTokenCtxKey); value != nil {
		if dCtx, ok := value.(*corecontext.DTokenContext); ok {
			return dCtx
		}
	}

	dCtx := corecontext.NewContext(NewEchoContext(c), mgr)
	c.Set(DTokenCtxKey, dCtx)
	return dCtx
}

// writeErrorResponse writes standard error response writeErrorResponse 写入标准错误响应
func writeErrorResponse(c echo4.Context, err error) error {
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

	return c.JSON(httpStatus, echo4.Map{
		"code":    code,
		"message": message,
		"data":    err.Error(),
	})
}

// writeSuccessResponse writes standard success response writeSuccessResponse 写入标准成功响应
func writeSuccessResponse(c echo4.Context, data interface{}) error {
	return c.JSON(http.StatusOK, echo4.Map{
		"code":    derror.CodeSuccess,
		"message": "success",
		"data":    data,
	})
}

// getHTTPStatusFromCode maps DToken code to HTTP status getHTTPStatusFromCode 映射 DToken 错误码到 HTTP 状态码
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
