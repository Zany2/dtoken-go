package chi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	DContext "github.com/Zany2/dtoken-go/core/context"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/manager"
	"github.com/Zany2/dtoken-go/dtoken"
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
	FailFunc  func(w http.ResponseWriter, r *http.Request, err error)
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
func WithFailFunc(fn func(w http.ResponseWriter, r *http.Request, err error)) AuthOption {
	return func(o *AuthOptions) {
		o.FailFunc = fn
	}
}

// RegisterDTokenContextMiddleware registers DToken context middleware RegisterDTokenContextMiddleware 注册 DToken 上下文中间件
func RegisterDTokenContextMiddleware(opts ...AuthOption) func(http.Handler) http.Handler {
	options := defaultAuthOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mgr, err := dtoken.GetManager(options.AuthType)
			if err != nil {
				if options.FailFunc != nil {
					options.FailFunc(w, r, err)
				} else {
					writeErrorResponse(w, err)
				}
				return
			}

			chiCtx := NewChiContext(w, r).(*ChiContext)
			_ = getDTokenContext(chiCtx, mgr)
			next.ServeHTTP(w, chiCtx.r)
		})
	}
}

// AuthMiddleware checks login status AuthMiddleware 校验登录状态
func AuthMiddleware(opts ...AuthOption) func(http.Handler) http.Handler {
	options := defaultAuthOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mgr, err := dtoken.GetManager(options.AuthType)
			if err != nil {
				if options.FailFunc != nil {
					options.FailFunc(w, r, err)
				} else {
					writeErrorResponse(w, err)
				}
				return
			}

			chiCtx := NewChiContext(w, r).(*ChiContext)
			dCtx := getDTokenContext(chiCtx, mgr)
			tokenValue := dCtx.GetTokenValue()

			if !mgr.IsLogin(chiCtx.r.Context(), tokenValue) {
				if options.FailFunc != nil {
					options.FailFunc(w, chiCtx.r, derror.ErrTokenExpired)
				} else {
					writeErrorResponse(w, derror.ErrTokenExpired)
				}
				return
			}

			next.ServeHTTP(w, chiCtx.r)
		})
	}
}

// PermissionMiddleware checks permissions PermissionMiddleware 校验权限
func PermissionMiddleware(permissions []string, opts ...AuthOption) func(http.Handler) http.Handler {
	options := defaultAuthOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if len(permissions) == 0 {
				next.ServeHTTP(w, r)
				return
			}

			mgr, err := dtoken.GetManager(options.AuthType)
			if err != nil {
				if options.FailFunc != nil {
					options.FailFunc(w, r, err)
				} else {
					writeErrorResponse(w, err)
				}
				return
			}

			chiCtx := NewChiContext(w, r).(*ChiContext)
			dCtx := getDTokenContext(chiCtx, mgr)
			tokenValue := dCtx.GetTokenValue()

			var ok bool
			if options.LogicType == LogicAnd {
				ok = mgr.HasPermissionsAndByToken(chiCtx.r.Context(), tokenValue, permissions)
			} else {
				ok = mgr.HasPermissionsOrByToken(chiCtx.r.Context(), tokenValue, permissions)
			}

			if !ok {
				if options.FailFunc != nil {
					options.FailFunc(w, chiCtx.r, derror.ErrPermissionDenied)
				} else {
					writeErrorResponse(w, derror.ErrPermissionDenied)
				}
				return
			}

			next.ServeHTTP(w, chiCtx.r)
		})
	}
}

// PermissionPathMiddleware checks path permissions PermissionPathMiddleware 基于路径校验权限
func PermissionPathMiddleware(permissions []string, opts ...AuthOption) func(http.Handler) http.Handler {
	options := defaultAuthOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqPermissions := append([]string{}, permissions...)
			reqPermissions = append(reqPermissions, r.URL.Path)

			if len(reqPermissions) == 0 {
				next.ServeHTTP(w, r)
				return
			}

			mgr, err := dtoken.GetManager(options.AuthType)
			if err != nil {
				if options.FailFunc != nil {
					options.FailFunc(w, r, err)
				} else {
					writeErrorResponse(w, err)
				}
				return
			}

			chiCtx := NewChiContext(w, r).(*ChiContext)
			dCtx := getDTokenContext(chiCtx, mgr)
			tokenValue := dCtx.GetTokenValue()

			var ok bool
			if options.LogicType == LogicAnd {
				ok = mgr.HasPermissionsAndByToken(chiCtx.r.Context(), tokenValue, reqPermissions)
			} else {
				ok = mgr.HasPermissionsOrByToken(chiCtx.r.Context(), tokenValue, reqPermissions)
			}

			if !ok {
				if options.FailFunc != nil {
					options.FailFunc(w, chiCtx.r, derror.ErrPermissionDenied)
				} else {
					writeErrorResponse(w, derror.ErrPermissionDenied)
				}
				return
			}

			next.ServeHTTP(w, chiCtx.r)
		})
	}
}

// RoleMiddleware checks roles RoleMiddleware 校验角色
func RoleMiddleware(roles []string, opts ...AuthOption) func(http.Handler) http.Handler {
	options := defaultAuthOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if len(roles) == 0 {
				next.ServeHTTP(w, r)
				return
			}

			mgr, err := dtoken.GetManager(options.AuthType)
			if err != nil {
				if options.FailFunc != nil {
					options.FailFunc(w, r, err)
				} else {
					writeErrorResponse(w, err)
				}
				return
			}

			chiCtx := NewChiContext(w, r).(*ChiContext)
			dCtx := getDTokenContext(chiCtx, mgr)
			tokenValue := dCtx.GetTokenValue()

			var ok bool
			if options.LogicType == LogicAnd {
				ok = mgr.HasRolesAndByToken(chiCtx.r.Context(), tokenValue, roles)
			} else {
				ok = mgr.HasRolesOrByToken(chiCtx.r.Context(), tokenValue, roles)
			}

			if !ok {
				if options.FailFunc != nil {
					options.FailFunc(w, chiCtx.r, derror.ErrRoleDenied)
				} else {
					writeErrorResponse(w, derror.ErrRoleDenied)
				}
				return
			}

			next.ServeHTTP(w, chiCtx.r)
		})
	}
}

// GetDTokenContext gets cached DToken context GetDTokenContext 获取缓存的 DToken 上下文
func GetDTokenContext(r *http.Request) (*DContext.DTokenContext, bool) {
	v := r.Context().Value(DTokenCtxKey)
	if v == nil {
		return nil, false
	}

	dCtx, ok := v.(*DContext.DTokenContext)
	return dCtx, ok
}

// GetDTokenContextByCtx gets DToken context by context GetDTokenContextByCtx 从上下文获取 DToken 上下文
func GetDTokenContextByCtx(ctx context.Context) (*DContext.DTokenContext, bool) {
	v := ctx.Value(DTokenCtxKey)
	if v == nil {
		return nil, false
	}

	dCtx, ok := v.(*DContext.DTokenContext)
	return dCtx, ok
}

// GetLoginIDByCtx gets login ID by context GetLoginIDByCtx 从上下文获取登录 ID
func GetLoginIDByCtx(ctx context.Context) (string, error) {
	dCtx, ok := GetDTokenContextByCtx(ctx)
	if !ok {
		return "", derror.ErrNotLogin
	}
	return dCtx.GetLoginID(ctx)
}

// GetTokenInfoByCtx gets token info by context GetTokenInfoByCtx 从上下文获取 Token 信息
func GetTokenInfoByCtx(ctx context.Context) (*manager.TokenInfo, error) {
	dCtx, ok := GetDTokenContextByCtx(ctx)
	if !ok {
		return nil, derror.ErrNotLogin
	}
	return dCtx.GetTokenInfo(ctx)
}

// getDTokenContext gets or creates dtoken context getDTokenContext 获取或创建 DToken 上下文
func getDTokenContext(chiCtx *ChiContext, mgr *manager.Manager) *DContext.DTokenContext {
	if v := chiCtx.r.Context().Value(DTokenCtxKey); v != nil {
		if dCtx, ok := v.(*DContext.DTokenContext); ok {
			return dCtx
		}
	}

	dCtx := DContext.NewContext(chiCtx, mgr)
	chiCtx.Set(DTokenCtxKey, dCtx)
	return dCtx
}

// writeErrorResponse writes error response writeErrorResponse 写入错误响应
func writeErrorResponse(w http.ResponseWriter, err error) {
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    code,
		"message": message,
		"data":    err.Error(),
	})
}

// writeSuccessResponse writes success response writeSuccessResponse 写入成功响应
func writeSuccessResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
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
