package gf

import (
	"context"
	"errors"
	"net/http"

	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/manager"

	DContext "github.com/Zany2/dtoken-go/core/context"
	"github.com/Zany2/dtoken-go/dtoken"
	"github.com/Zany2/dtoken-go/integrations/internal/authcheck"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
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

// AuthOptions defines middleware auth options AuthOptions 定义中间件认证选项
type AuthOptions struct {
	AuthType  string
	LogicType LogicType
	FailFunc  func(r *ghttp.Request, err error)
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
func WithFailFunc(fn func(r *ghttp.Request, err error)) AuthOption {
	return func(o *AuthOptions) {
		o.FailFunc = fn
	}
}

// RegisterDTokenContextMiddleware registers DToken context middleware RegisterDTokenContextMiddleware 注册 DToken 上下文中间件
func RegisterDTokenContextMiddleware(ctx context.Context, opts ...AuthOption) ghttp.HandlerFunc {
	options := defaultAuthOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(r *ghttp.Request) {
		mgr, err := authcheck.GetManager(options.AuthType)
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

// AuthMiddleware checks login status AuthMiddleware 校验登录状态
func AuthMiddleware(ctx context.Context, opts ...AuthOption) ghttp.HandlerFunc {
	options := defaultAuthOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(r *ghttp.Request) {
		mgr, err := authcheck.GetManager(options.AuthType)
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

		_, err = authcheck.Check(ctx, mgr, authcheck.Request{
			TokenValue: tokenValue,
			CheckLogin: true,
			LoginError: derror.ErrTokenExpired,
		})
		if err != nil {
			if options.FailFunc != nil {
				options.FailFunc(r, err)
			} else {
				writeErrorResponse(r, err)
			}
			return
		}

		r.Middleware.Next()
	}
}

// PermissionMiddleware checks permissions PermissionMiddleware 校验权限
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
		if len(permissions) == 0 {
			r.Middleware.Next()
			return
		}

		mgr, err := authcheck.GetManager(options.AuthType)
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

		_, err = authcheck.Check(ctx, mgr, authcheck.Request{
			TokenValue:  tokenValue,
			Permissions: permissions,
			LogicType:   options.LogicType,
		})
		if err != nil {
			if options.FailFunc != nil {
				options.FailFunc(r, err)
			} else {
				writeErrorResponse(r, err)
			}
			return
		}

		r.Middleware.Next()
	}
}

// PermissionPathMiddleware checks path permissions PermissionPathMiddleware 基于路径校验权限
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
		reqPermissions := append([]string{}, permissions...)
		reqPermissions = append(reqPermissions, r.URL.Path)

		if len(reqPermissions) == 0 {
			r.Middleware.Next()
			return
		}

		mgr, err := authcheck.GetManager(options.AuthType)
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

		_, err = authcheck.Check(ctx, mgr, authcheck.Request{
			TokenValue:  tokenValue,
			Permissions: reqPermissions,
			LogicType:   options.LogicType,
		})
		if err != nil {
			if options.FailFunc != nil {
				options.FailFunc(r, err)
			} else {
				writeErrorResponse(r, err)
			}
			return
		}

		r.Middleware.Next()
	}
}

// RoleMiddleware checks roles RoleMiddleware 校验角色
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
		if len(roles) == 0 {
			r.Middleware.Next()
			return
		}

		mgr, err := authcheck.GetManager(options.AuthType)
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

		_, err = authcheck.Check(ctx, mgr, authcheck.Request{
			TokenValue: tokenValue,
			Roles:      roles,
			LogicType:  options.LogicType,
		})
		if err != nil {
			if options.FailFunc != nil {
				options.FailFunc(r, err)
			} else {
				writeErrorResponse(r, err)
			}
			return
		}

		r.Middleware.Next()
	}
}

// GetDTokenContext gets cached DToken context GetDTokenContext 获取缓存的 DToken 上下文
func GetDTokenContext(r *ghttp.Request) (*DContext.DTokenContext, bool) {
	v := r.GetCtxVar(DTokenCtxKey)
	if v == nil {
		return nil, false
	}

	ctx, ok := v.Val().(*DContext.DTokenContext)
	return ctx, ok
}

// GetDTokenContextByCtx gets DToken context by context GetDTokenContextByCtx 从上下文获取 DToken 上下文
func GetDTokenContextByCtx(ctx context.Context) (*DContext.DTokenContext, bool) {
	request := g.RequestFromCtx(ctx)
	ctxVar := request.GetCtxVar(DTokenCtxKey)
	if ctxVar == nil {
		return nil, false
	}

	tokenContext, ok := ctxVar.Val().(*DContext.DTokenContext)
	return tokenContext, ok
}

// GetLoginIDByCtx gets login ID by context GetLoginIDByCtx 从上下文获取登录 ID
func GetLoginIDByCtx(ctx context.Context, authType ...string) (string, error) {
	mgr, err := dtoken.GetManager(authType...)
	if err != nil {
		return "", err
	}

	tokenValue := getDContext(g.RequestFromCtx(ctx), mgr).GetTokenValue()
	return mgr.GetLoginID(ctx, tokenValue)
}

// GetTokenInfoByCtx gets token info by context GetTokenInfoByCtx 从上下文获取 Token 信息
func GetTokenInfoByCtx(ctx context.Context, authType ...string) (*manager.TokenInfo, error) {
	mgr, err := dtoken.GetManager(authType...)
	if err != nil {
		return nil, err
	}

	return mgr.GetTokenInfo(ctx, getDContext(g.RequestFromCtx(ctx), mgr).GetTokenValue())
}

// getDContext gets or creates DToken context getDContext 获取或创建 DToken 上下文
func getDContext(r *ghttp.Request, mgr *manager.Manager) *DContext.DTokenContext {
	if v := r.GetCtxVar(DTokenCtxKey); v != nil {
		if dCtx, ok := v.Val().(*DContext.DTokenContext); ok {
			return dCtx
		}
	}

	dCtx := DContext.NewContext(NewGFContext(r), mgr)
	r.SetCtxVar(DTokenCtxKey, dCtx)

	return dCtx
}

// writeErrorResponse writes error response writeErrorResponse 写入错误响应
func writeErrorResponse(r *ghttp.Request, err error) {
	var saErr *derror.DTokenError
	var code int
	var message string
	var httpStatus int

	if errors.As(err, &saErr) {
		code = saErr.Code
		message = saErr.Message
		httpStatus = getHTTPStatusFromCode(code)
	} else {
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

// writeSuccessResponse writes success response writeSuccessResponse 写入成功响应
func writeSuccessResponse(r *ghttp.Request, data interface{}) {
	r.Response.WriteStatusExit(http.StatusOK, g.Map{
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
