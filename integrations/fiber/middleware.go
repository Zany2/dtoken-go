// @Author daixk 2025/12/22 15:56:00
package fiber

import (
	"context"
	"net/http"

	corecontext "github.com/Zany2/dtoken-go/core/context"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/manager"
	"github.com/Zany2/dtoken-go/integrations/authcheck"
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

// BeforeAuthHandler handles request before dtoken checks BeforeAuthHandler 在 dtoken 校验前处理请求
type BeforeAuthHandler func(ctx context.Context, c *gofiber.Ctx, req *AuthHandleRequest)

// AuthHandleRequest carries auth check metadata AuthHandleRequest 携带认证校验元数据
type AuthHandleRequest struct {
	AuthType     string
	CheckLogin   bool
	CheckDisable bool
	Permissions  []string
	Roles        []string
	LogicType    LogicType

	next    func() error
	result  error
	handled bool
}

// Next continues request and stops dtoken checks Next 放行请求并停止 dtoken 校验
func (req *AuthHandleRequest) Next() {
	req.handled = true
	if req.next != nil {
		req.result = req.next()
	}
}

// Exit stops dtoken checks after custom handling Exit 自定义处理后停止 dtoken 校验
func (req *AuthHandleRequest) Exit() {
	req.handled = true
}

// IsHandled reports whether request has been handled IsHandled 判断请求是否已处理
func (req *AuthHandleRequest) IsHandled() bool {
	return req.handled
}

// AuthOptions carries middleware auth options AuthOptions 保存中间件认证选项。
type AuthOptions struct {
	AuthType          string
	LogicType         LogicType
	FailFunc          func(c *gofiber.Ctx, err error)
	BeforeAuthHandler BeforeAuthHandler
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

// WithBeforeAuthHandler sets pre auth handler WithBeforeAuthHandler 设置认证前置处理器
func WithBeforeAuthHandler(fn BeforeAuthHandler) AuthOption {
	return func(o *AuthOptions) {
		o.BeforeAuthHandler = fn
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
		authReq := newAuthHandleRequest(options, func() error {
			return c.Next()
		})
		authReq.CheckLogin = true
		if runBeforeAuthHandler(ctx, c, options, authReq) {
			return authReq.result
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
		authReq := newAuthHandleRequest(options, func() error {
			return c.Next()
		})
		authReq.Permissions = append([]string{}, permissions...)
		if runBeforeAuthHandler(ctx, c, options, authReq) {
			return authReq.result
		}

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
		authReq := newAuthHandleRequest(options, func() error {
			return c.Next()
		})
		authReq.Roles = append([]string{}, roles...)
		if runBeforeAuthHandler(ctx, c, options, authReq) {
			return authReq.result
		}

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

// newAuthHandleRequest creates auth handle request newAuthHandleRequest 创建认证处理请求
func newAuthHandleRequest(options *AuthOptions, next func() error) *AuthHandleRequest {
	return &AuthHandleRequest{
		AuthType:  options.AuthType,
		LogicType: options.LogicType,
		next:      next,
	}
}

// runBeforeAuthHandler executes pre auth handler runBeforeAuthHandler 执行认证前置处理器
func runBeforeAuthHandler(ctx context.Context, c *gofiber.Ctx, options *AuthOptions, req *AuthHandleRequest) bool {
	if options.BeforeAuthHandler == nil {
		return false
	}

	options.BeforeAuthHandler(ctx, c, req)
	return req.IsHandled()
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
	code, message := authcheck.GetErrorCodeAndMessage(err)
	httpStatus := getHTTPStatusFromCode(code)

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
