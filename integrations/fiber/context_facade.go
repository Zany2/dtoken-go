// @Author daixk 2025/12/22 15:56:00
package fiber

import (
	"context"
	derror "github.com/Zany2/dtoken-go/errors"
	requestcontext "github.com/Zany2/dtoken-go/httpauth/context"

	adapter "github.com/Zany2/dtoken-go/contract"
	manager "github.com/Zany2/dtoken-go/internal/engine"
	gofiber "github.com/gofiber/fiber/v2"
)

// GetTokenValueByContext gets token value from current Fiber context GetTokenValueByContext 从当前 Fiber 上下文获取 Token。
func GetTokenValueByContext(c *gofiber.Ctx) (string, error) {
	dCtx, err := requireDTokenContextByContext(c)
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
func GetRequestContextByContext(c *gofiber.Ctx) (adapter.RequestContext, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.GetRequestContext(), nil
}

// GetManagerByContext gets current DToken manager GetManagerByContext 获取当前 DToken 管理。
func GetManagerByContext(c *gofiber.Ctx) (*manager.Manager, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.GetManager(), nil
}

// GetLoginIDByContext gets current login ID GetLoginIDByContext 获取当前登录 ID
func GetLoginIDByContext(c *gofiber.Ctx) (string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return "", err
	}
	return dCtx.Auth().GetLoginID(requestContext(c))
}

// GetTokenInfoByContext gets current token info GetTokenInfoByContext 获取当前 token 信息
func GetTokenInfoByContext(c *gofiber.Ctx) (*manager.TokenInfo, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Auth().GetTokenInfo(requestContext(c))
}

// IntrospectTokenByContext inspects current token without renewal side effects IntrospectTokenByContext 无续期副作用地检查当前 Token 状态。
func IntrospectTokenByContext(c *gofiber.Ctx) (*manager.TokenIntrospection, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Auth().IntrospectToken(requestContext(c))
}

// HasRolesOrByContext checks whether current user has any role HasRolesOrByContext 检查当前用户是否拥有任一角色
func HasRolesOrByContext(c *gofiber.Ctx, roles []string) bool {
	return HasRolesByContext(c, roles)
}

// HasPermissionsOrByContext checks whether current user has any permission HasPermissionsOrByContext 检查当前用户是否拥有任一权限
func HasPermissionsOrByContext(c *gofiber.Ctx, permissions []string) bool {
	return HasPermissionsByContext(c, permissions)
}

// requestContext gets standard context from Fiber request requestContext 从 Fiber 请求获取标准上下文。
func requestContext(c *gofiber.Ctx) context.Context {
	if c != nil {
		// UserContext carries application deadlines and is not Fiber's pooled RequestCtx. UserContext 承载业务截止时间，避免将复用的 RequestCtx 传入异步任务。
		return c.UserContext()
	}
	return context.Background()
}

// requireDTokenContextByContext gets required DToken context requireDTokenContextByContext 获取必需的 DToken 上下文。
func requireDTokenContextByContext(c *gofiber.Ctx) (*requestcontext.DTokenContext, error) {
	dCtx, ok := GetDTokenContext(c)
	if !ok {
		if c == nil {
			return nil, derror.ErrNotLogin
		}
		return nil, derror.ErrManagerNotFound
	}
	if mgr := dCtx.GetManager(); mgr == nil || mgr.IsClosed() {
		return nil, derror.ErrManagerNotFound
	}
	return dCtx, nil
}
