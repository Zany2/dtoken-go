package gin

import (
	"context"
	"errors"
	"net/http"

	DContext "github.com/Zany2/dtoken-go/core/context"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/manager"
	"github.com/Zany2/dtoken-go/dtoken"
	"github.com/gin-gonic/gin"
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
	FailFunc  func(c *gin.Context, err error)
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
func WithFailFunc(fn func(c *gin.Context, err error)) AuthOption {
	return func(o *AuthOptions) {
		o.FailFunc = fn
	}
}

// -------------------------------------------------- Middleware - 中间件 --------------------------------------------------
// RegisterDTokenContextMiddleware registers DToken context middleware RegisterDTokenContextMiddleware 注册 DToken 上下文中间件
func RegisterDTokenContextMiddleware(ctx context.Context, opts ...AuthOption) gin.HandlerFunc {
	options := defaultAuthOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(c *gin.Context) {
		mgr, err := dtoken.GetManager(options.AuthType)
		if err != nil {
			if options.FailFunc != nil {
				options.FailFunc(c, err)
			} else {
				writeErrorResponse(c, err)
			}
			return
		}

		_ = getDContext(c, mgr)
	}
}

// AuthMiddleware checks login status AuthMiddleware 校验登录状态
func AuthMiddleware(ctx context.Context, opts ...AuthOption) gin.HandlerFunc {
	options := defaultAuthOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(c *gin.Context) {
		mgr, err := dtoken.GetManager(options.AuthType)
		if err != nil {
			if options.FailFunc != nil {
				options.FailFunc(c, err)
			} else {
				writeErrorResponse(c, err)
			}
			c.Abort()
			return
		}

		dCtx := getDContext(c, mgr)
		tokenValue := dCtx.GetTokenValue()

		if !dtoken.IsLogin(ctx, tokenValue) {
			if options.FailFunc != nil {
				options.FailFunc(c, derror.ErrTokenExpired)
			} else {
				writeErrorResponse(c, derror.ErrTokenExpired)
			}
			c.Abort()
			return
		}

		c.Next()
	}
}

// PermissionMiddleware checks permissions PermissionMiddleware 校验权限
func PermissionMiddleware(
	ctx context.Context,
	permissions []string,
	opts ...AuthOption,
) gin.HandlerFunc {

	options := defaultAuthOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(c *gin.Context) {
		if len(permissions) == 0 {
			c.Next()
			return
		}

		mgr, err := dtoken.GetManager(options.AuthType)
		if err != nil {
			if options.FailFunc != nil {
				options.FailFunc(c, err)
			} else {
				writeErrorResponse(c, err)
			}
			c.Abort()
			return
		}

		dCtx := getDContext(c, mgr)
		tokenValue := dCtx.GetTokenValue()

		var ok bool
		if options.LogicType == LogicAnd {
			ok = mgr.HasPermissionsAndByToken(ctx, tokenValue, permissions)
		} else {
			ok = mgr.HasPermissionsOrByToken(ctx, tokenValue, permissions)
		}

		if !ok {
			if options.FailFunc != nil {
				options.FailFunc(c, derror.ErrPermissionDenied)
			} else {
				writeErrorResponse(c, derror.ErrPermissionDenied)
			}
			c.Abort()
			return
		}

		c.Next()
	}
}

// RoleMiddleware checks roles RoleMiddleware 校验角色
func RoleMiddleware(
	ctx context.Context,
	roles []string,
	opts ...AuthOption,
) gin.HandlerFunc {

	options := defaultAuthOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(c *gin.Context) {
		if len(roles) == 0 {
			c.Next()
			return
		}

		mgr, err := dtoken.GetManager(options.AuthType)
		if err != nil {
			if options.FailFunc != nil {
				options.FailFunc(c, err)
			} else {
				writeErrorResponse(c, err)
			}
			c.Abort()
			return
		}

		dCtx := getDContext(c, mgr)
		tokenValue := dCtx.GetTokenValue()

		var ok bool
		if options.LogicType == LogicAnd {
			ok = mgr.HasRolesAndByToken(ctx, tokenValue, roles)
		} else {
			ok = mgr.HasRolesOrByToken(ctx, tokenValue, roles)
		}

		if !ok {
			if options.FailFunc != nil {
				options.FailFunc(c, derror.ErrRoleDenied)
			} else {
				writeErrorResponse(c, derror.ErrRoleDenied)
			}
			c.Abort()
			return
		}

		c.Next()
	}
}

// -------------------------------------------------- Context Helpers - 上下文辅助函数 --------------------------------------------------
// GetDTokenContext gets cached DToken context GetDTokenContext 获取缓存的 DToken 上下文
func GetDTokenContext(c *gin.Context) (*DContext.DTokenContext, bool) {
	v, exists := c.Get(DTokenCtxKey)
	if !exists {
		return nil, false
	}

	ctx, ok := v.(*DContext.DTokenContext)
	return ctx, ok
}

// getDContext gets or creates DToken context getDContext 获取或创建 DToken 上下文
func getDContext(c *gin.Context, mgr *manager.Manager) *DContext.DTokenContext {
	if v, exists := c.Get(DTokenCtxKey); exists {
		if dCtx, ok := v.(*DContext.DTokenContext); ok {
			return dCtx
		}
	}

	dCtx := DContext.NewContext(NewGinContext(c), mgr)
	c.Set(DTokenCtxKey, dCtx)

	return dCtx
}

// -------------------------------------------------- Response Helpers - 响应辅助函数 --------------------------------------------------
// writeErrorResponse writes error response writeErrorResponse 写入错误响应
func writeErrorResponse(c *gin.Context, err error) {
	var dtErr *derror.DTokenError
	var code int
	var message string
	var httpStatus int

	if errors.As(err, &dtErr) {
		code = dtErr.Code
		message = dtErr.Message
		httpStatus = getHTTPStatusFromCode(code)
	} else {
		code = derror.CodeServerError
		message = err.Error()
		httpStatus = http.StatusInternalServerError
	}

	c.JSON(httpStatus, gin.H{
		"code":    code,
		"message": message,
		"data":    err.Error(),
	})
}

// writeSuccessResponse writes success response writeSuccessResponse 写入成功响应
func writeSuccessResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
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
