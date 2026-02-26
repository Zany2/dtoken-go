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

// LogicType permission/role logic type
// 权限/角色判断逻辑
type LogicType string

const (
	DTokenCtxKey = "DCtx"

	LogicOr  LogicType = "OR"  // Logical OR 任一满足
	LogicAnd LogicType = "AND" // Logical AND 全部满足
)

type AuthOption func(*AuthOptions)

type AuthOptions struct {
	AuthType  string
	LogicType LogicType
	FailFunc  func(c *gin.Context, err error)
}

func defaultAuthOptions() *AuthOptions {
	return &AuthOptions{LogicType: LogicAnd}
}

// WithAuthType sets auth type
// 设置认证类型
func WithAuthType(authType string) AuthOption {
	return func(o *AuthOptions) {
		o.AuthType = authType
	}
}

// WithLogicType sets LogicType option
// 设置逻辑类型
func WithLogicType(logicType LogicType) AuthOption {
	return func(o *AuthOptions) {
		o.LogicType = logicType
	}
}

// WithFailFunc sets auth failure callback
// 设置认证失败回调
func WithFailFunc(fn func(c *gin.Context, err error)) AuthOption {
	return func(o *AuthOptions) {
		o.FailFunc = fn
	}
}

// ============ Middlewares ============
// ============ 中间件 ============

// RegisterDTokenContextMiddleware initializes DToken context for each request
// 初始化每次请求的 DToken 上下文的中间件
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

// AuthMiddleware authentication middleware
// 认证中间件
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

		// Check if user is logged in
		// 检查用户是否已登录
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

// PermissionMiddleware permission check middleware
// 权限校验中间件
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

// RoleMiddleware role check middleware
// 角色校验中间件
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

// GetDTokenContext gets DToken context from Gin context
// 获取 DToken 上下文
func GetDTokenContext(c *gin.Context) (*DContext.DTokenContext, bool) {
	v, exists := c.Get(DTokenCtxKey)
	if !exists {
		return nil, false
	}

	ctx, ok := v.(*DContext.DTokenContext)
	return ctx, ok
}

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

// ============ Error Handling Helpers ============
// ============ 错误处理辅助函数 ============

// writeErrorResponse writes a standardized error response
// 写入标准化的错误响应
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

// writeSuccessResponse writes a standardized success response
// 写入标准化的成功响应
func writeSuccessResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"code":    derror.CodeSuccess,
		"message": "success",
		"data":    data,
	})
}

// getHTTPStatusFromCode converts DToken error code to HTTP status
// 将DToken错误码转换为HTTP状态码
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
