// @Author daixk 2025/12/22 15:56:00
package hertz

import (
	"context"
	derror "github.com/Zany2/dtoken-go/errors"
	requestcontext "github.com/Zany2/dtoken-go/httpauth/context"

	adapter "github.com/Zany2/dtoken-go/contract"
	manager "github.com/Zany2/dtoken-go/internal/engine"
	hertzapp "github.com/cloudwego/hertz/pkg/app"
)

// GetTokenValueByContext gets token value from current Hertz context GetTokenValueByContext 从当前 Hertz 上下文获取 Token 值。
func GetTokenValueByContext(ctx *hertzapp.RequestContext) (string, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return "", err
	}

	tokenValue := dCtx.GetTokenValue()
	if tokenValue == "" {
		return "", derror.ErrNotLogin
	}
	return tokenValue, nil
}

// GetRequestContextByContext gets raw request context GetRequestContextByContext 获取原始请求上下文。
func GetRequestContextByContext(ctx *hertzapp.RequestContext) (adapter.RequestContext, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.GetRequestContext(), nil
}

// GetManagerByContext gets current DToken manager GetManagerByContext 获取当前 DToken 管理器。
func GetManagerByContext(ctx *hertzapp.RequestContext) (*manager.Manager, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.GetManager(), nil
}

// GetLoginIDByContext gets current login ID GetLoginIDByContext 获取当前登录 ID。
func GetLoginIDByContext(ctx *hertzapp.RequestContext) (string, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return "", err
	}
	return dCtx.Auth().GetLoginID(requestContext(ctx))
}

// GetTokenInfoByContext gets current token info GetTokenInfoByContext 获取当前 token 信息。
func GetTokenInfoByContext(ctx *hertzapp.RequestContext) (*manager.TokenInfo, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Auth().GetTokenInfo(requestContext(ctx))
}

// IntrospectTokenByContext inspects current token without renewal side effects IntrospectTokenByContext 无续期副作用地检查当前 Token 状态。
func IntrospectTokenByContext(ctx *hertzapp.RequestContext) (*manager.TokenIntrospection, error) {
	dCtx, err := requireDTokenContextByContext(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.Auth().IntrospectToken(requestContext(ctx))
}

// HasRolesOrByContext checks whether current user has any role HasRolesOrByContext 检查当前用户是否拥有任一角色。
func HasRolesOrByContext(ctx *hertzapp.RequestContext, roles []string) bool {
	return HasRolesByContext(ctx, roles)
}

// HasPermissionsOrByContext checks whether current user has any permission HasPermissionsOrByContext 检查当前用户是否拥有任一权限。
func HasPermissionsOrByContext(ctx *hertzapp.RequestContext, permissions []string) bool {
	return HasPermissionsByContext(ctx, permissions)
}

// requestContextKey stores the standard context separately from Hertz request data. requestContextKey 将标准上下文与 Hertz 请求数据分开存储。
const requestContextKey = "dtoken-go/hertz/request-context"

// bindRequestContext scopes the handler context to the current DToken call chain. bindRequestContext 在当前 DToken 调用链内绑定处理器上下文。
func bindRequestContext(c context.Context, ctx *hertzapp.RequestContext) func() {
	previous, _ := ctx.Get(requestContextKey)
	ctx.Set(requestContextKey, c)
	return func() { ctx.Set(requestContextKey, previous) }
}

// requestContext gets the context bound by DToken middleware or annotations. requestContext 获取 DToken 中间件或注解绑定的标准上下文。
func requestContext(ctx *hertzapp.RequestContext) context.Context {
	if ctx != nil {
		if value, ok := ctx.Get(requestContextKey); ok {
			if c, ok := value.(context.Context); ok && c != nil {
				return c
			}
		}
	}
	return context.Background()
}

// requireDTokenContextByContext gets required DToken context requireDTokenContextByContext 获取必需的 DToken 上下文。
func requireDTokenContextByContext(ctx *hertzapp.RequestContext) (*requestcontext.DTokenContext, error) {
	dCtx, ok := GetDTokenContext(ctx)
	if !ok {
		if ctx == nil {
			return nil, derror.ErrNotLogin
		}
		return nil, derror.ErrManagerNotFound
	}
	if mgr := dCtx.GetManager(); mgr == nil || mgr.IsClosed() {
		return nil, derror.ErrManagerNotFound
	}
	return dCtx, nil
}
