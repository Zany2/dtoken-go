// @Author daixk 2025/12/22 15:56:00
package fiber

import (
	"context"
	"errors"
	"net/http"

	corecontext "github.com/Zany2/dtoken-go/core/context"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/manager"
	"github.com/Zany2/dtoken-go/integrations/internal/authcheck"
	gofiber "github.com/gofiber/fiber/v2"
)

// LogicType defines permission and role check logic LogicType 定义权限与角色校验的逻辑类型。
type LogicType = authcheck.LogicType

const (
	// DTokenCtxKey stores request scoped DToken context DTokenCtxKey 存储请求级 DToken 上下文
	DTokenCtxKey = "DCtx"

	LogicOr  LogicType = authcheck.LogicOr
	LogicAnd LogicType = authcheck.LogicAnd
)

// AuthOption defines auth option setter AuthOption 定义认证选项设置器
type AuthOption func(*AuthOptions)

// AuthOptions carries middleware auth options AuthOptions 保存中间件认证选项。
type AuthOptions struct {
	AuthType  string
	LogicType LogicType
	FailFunc  func(c *gofiber.Ctx, err error)
}

// defaultAuthOptions returns default middleware options defaultAuthOptions 返回默认中间件选项。
func defaultAuthOptions() *AuthOptions {
	return &AuthOptions{LogicType: LogicAnd}
}

// WithAuthType sets the auth type used by middleware WithAuthType 设置中间件使用的认证类型。
func WithAuthType(authType string) AuthOption {
	return func(o *AuthOptions) {
		o.AuthType = authType
	}
}

// WithLogicType sets the logic mode for permission and role checks WithLogicType 设置权限与角色校验的逻辑模式。
func WithLogicType(logicType LogicType) AuthOption {
	return func(o *AuthOptions) {
		o.LogicType = logicType
	}
}

// WithFailFunc sets a custom auth failure callback WithFailFunc 设置自定义认证失败回调。
func WithFailFunc(fn func(c *gofiber.Ctx, err error)) AuthOption {
	return func(o *AuthOptions) {
		o.FailFunc = fn
	}
}

// RegisterDTokenContextMiddleware initializes DToken context for each request RegisterDTokenContextMiddleware 为每个请求初始化 DToken 上下文。
func RegisterDTokenContextMiddleware(ctx context.Context, opts ...AuthOption) gofiber.Handler {
	options := defaultAuthOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(c *gofiber.Ctx) error {
		mgr, err := authcheck.GetManager(options.AuthType)
		if err != nil {
			if options.FailFunc != nil {
				options.FailFunc(c, err)
				return nil
			}
			return writeErrorResponse(c, err)
		}

		_ = getDTokenContext(c, mgr)
		return c.Next()
	}
}

// AuthMiddleware checks whether the current request is authenticated AuthMiddleware 检查当前请求是否已认证。
func AuthMiddleware(ctx context.Context, opts ...AuthOption) gofiber.Handler {
	options := defaultAuthOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(c *gofiber.Ctx) error {
		mgr, err := authcheck.GetManager(options.AuthType)
		if err != nil {
			if options.FailFunc != nil {
				options.FailFunc(c, err)
				return nil
			}
			return writeErrorResponse(c, err)
		}

		dCtx := getDTokenContext(c, mgr)
		tokenValue := dCtx.GetTokenValue()

		_, err = authcheck.Check(ctx, mgr, authcheck.Request{
			TokenValue: tokenValue,
			CheckLogin: true,
			LoginError: derror.ErrTokenExpired,
		})
		if err != nil {
			if options.FailFunc != nil {
				options.FailFunc(c, err)
				return nil
			}
			return writeErrorResponse(c, err)
		}

		return c.Next()
	}
}

// PermissionMiddleware checks whether the current token has required permissions PermissionMiddleware 检查当前 token 是否具备所需权限
func PermissionMiddleware(
	ctx context.Context,
	permissions []string,
	opts ...AuthOption,
) gofiber.Handler {
	options := defaultAuthOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(c *gofiber.Ctx) error {
		if len(permissions) == 0 {
			return c.Next()
		}

		mgr, err := authcheck.GetManager(options.AuthType)
		if err != nil {
			if options.FailFunc != nil {
				options.FailFunc(c, err)
				return nil
			}
			return writeErrorResponse(c, err)
		}

		dCtx := getDTokenContext(c, mgr)
		tokenValue := dCtx.GetTokenValue()

		_, err = authcheck.Check(ctx, mgr, authcheck.Request{
			TokenValue:  tokenValue,
			Permissions: permissions,
			LogicType:   options.LogicType,
		})
		if err != nil {
			if options.FailFunc != nil {
				options.FailFunc(c, err)
				return nil
			}
			return writeErrorResponse(c, err)
		}

		return c.Next()
	}
}

// RoleMiddleware checks whether the current token has required roles RoleMiddleware 检查当前 token 是否具备所需角色
func RoleMiddleware(
	ctx context.Context,
	roles []string,
	opts ...AuthOption,
) gofiber.Handler {
	options := defaultAuthOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(c *gofiber.Ctx) error {
		if len(roles) == 0 {
			return c.Next()
		}

		mgr, err := authcheck.GetManager(options.AuthType)
		if err != nil {
			if options.FailFunc != nil {
				options.FailFunc(c, err)
				return nil
			}
			return writeErrorResponse(c, err)
		}

		dCtx := getDTokenContext(c, mgr)
		tokenValue := dCtx.GetTokenValue()

		_, err = authcheck.Check(ctx, mgr, authcheck.Request{
			TokenValue: tokenValue,
			Roles:      roles,
			LogicType:  options.LogicType,
		})
		if err != nil {
			if options.FailFunc != nil {
				options.FailFunc(c, err)
				return nil
			}
			return writeErrorResponse(c, err)
		}

		return c.Next()
	}
}

// GetDTokenContext gets cached DToken context from Fiber request GetDTokenContext 从 Fiber 请求中获取缓存的 DToken 上下文。
func GetDTokenContext(c *gofiber.Ctx) (*corecontext.DTokenContext, bool) {
	value := c.Locals(DTokenCtxKey)
	if value == nil {
		return nil, false
	}

	dCtx, ok := value.(*corecontext.DTokenContext)
	return dCtx, ok
}

// getDTokenContext gets or creates dtoken context getDTokenContext 获取或创建 DToken 上下文
func getDTokenContext(c *gofiber.Ctx, mgr *manager.Manager) *corecontext.DTokenContext {
	if value := c.Locals(DTokenCtxKey); value != nil {
		if dCtx, ok := value.(*corecontext.DTokenContext); ok {
			return dCtx
		}
	}

	dCtx := corecontext.NewContext(NewFiberContext(c), mgr)
	c.Locals(DTokenCtxKey, dCtx)
	return dCtx
}

// writeErrorResponse writes a standard error response writeErrorResponse 写入标准错误响应。
func writeErrorResponse(c *gofiber.Ctx, err error) error {
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

	return c.Status(httpStatus).JSON(gofiber.Map{
		"code":    code,
		"message": message,
		"data":    err.Error(),
	})
}

// writeSuccessResponse writes a standard success response writeSuccessResponse 写入标准成功响应。
func writeSuccessResponse(c *gofiber.Ctx, data interface{}) error {
	return c.Status(http.StatusOK).JSON(gofiber.Map{
		"code":    derror.CodeSuccess,
		"message": "success",
		"data":    data,
	})
}

// getHTTPStatusFromCode maps DToken error code to HTTP status getHTTPStatusFromCode 将 DToken 错误码映射为 HTTP 状态码。
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
