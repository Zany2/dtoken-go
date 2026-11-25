// @Author daixk 2025/12/22 15:56:00
package chi

import (
	"context"
	derror "github.com/Zany2/dtoken-go/errors"
	requestcontext "github.com/Zany2/dtoken-go/httpauth/context"

	adapter "github.com/Zany2/dtoken-go/contract"
	manager "github.com/Zany2/dtoken-go/internal/engine"
)

// GetTokenValueByCtx gets token value from current Chi context GetTokenValueByCtx 从当前 Chi 上下文获取 Token 值。
func GetTokenValueByCtx(ctx context.Context) (string, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return "", err
	}

	tokenValue := dCtx.GetTokenValue()
	if tokenValue == "" {
		return "", derror.ErrNotLogin
	}
	return tokenValue, nil
}

// GetRequestContextByCtx gets raw request context GetRequestContextByCtx 获取原始请求上下文。
func GetRequestContextByCtx(ctx context.Context) (adapter.RequestContext, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.GetRequestContext(), nil
}

// GetManagerByCtx gets current DToken manager GetManagerByCtx 获取当前 DToken 管理器。
func GetManagerByCtx(ctx context.Context) (*manager.Manager, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.GetManager(), nil
}

// HasRolesOrByCtx checks whether current user has any role HasRolesOrByCtx 检查当前用户是否拥有任一角色。
func HasRolesOrByCtx(ctx context.Context, roles []string) bool {
	return HasRolesByCtx(ctx, roles)
}

// HasPermissionsOrByCtx checks whether current user has any permission HasPermissionsOrByCtx 检查当前用户是否拥有任一权限。
func HasPermissionsOrByCtx(ctx context.Context, permissions []string) bool {
	return HasPermissionsByCtx(ctx, permissions)
}

// requireDTokenContextByCtx gets required DToken context requireDTokenContextByCtx 获取必需的 DToken 上下文。
func requireDTokenContextByCtx(ctx context.Context) (*requestcontext.DTokenContext, error) {
	dCtx, ok := GetDTokenContextByCtx(ctx)
	if !ok {
		return nil, derror.ErrNotLogin
	}
	if mgr := dCtx.GetManager(); mgr == nil || mgr.IsClosed() {
		return nil, derror.ErrManagerNotFound
	}
	return dCtx, nil
}
