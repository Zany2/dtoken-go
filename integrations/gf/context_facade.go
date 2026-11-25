// @Author daixk 2025/12/22 15:56:00
package gf

import (
	"context"
	derror "github.com/Zany2/dtoken-go/errors"
	requestcontext "github.com/Zany2/dtoken-go/httpauth/context"

	adapter "github.com/Zany2/dtoken-go/contract"
	authcheck "github.com/Zany2/dtoken-go/httpauth"
	manager "github.com/Zany2/dtoken-go/internal/engine"
	"github.com/gogf/gf/v2/frame/g"
)

// GetTokenValueByCtx gets token value from current GF context GetTokenValueByCtx 从当前 GF 上下文获取 Token。
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

// GetManagerByCtx gets current DToken manager GetManagerByCtx 获取当前 DToken 管理。
func GetManagerByCtx(ctx context.Context) (*manager.Manager, error) {
	dCtx, err := requireDTokenContextByCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dCtx.GetManager(), nil
}

// HasRolesOrByCtx checks whether current user has any role HasRolesOrByCtx 检查当前用户是否拥有任一角色
func HasRolesOrByCtx(ctx context.Context, roles []string) bool {
	return HasRolesByCtx(ctx, roles)
}

// HasPermissionsOrByCtx checks whether current user has any permission HasPermissionsOrByCtx 检查当前用户是否拥有任一权限
func HasPermissionsOrByCtx(ctx context.Context, permissions []string) bool {
	return HasPermissionsByCtx(ctx, permissions)
}

// requireDTokenContextByCtx gets required DToken context requireDTokenContextByCtx 获取必需的 DToken 上下文。
func requireDTokenContextByCtx(ctx context.Context, authType ...string) (*requestcontext.DTokenContext, error) {
	if ctx == nil {
		return nil, derror.ErrNotLogin
	}

	dCtx, ok := GetDTokenContextByCtx(ctx)
	if ok && len(authType) == 0 {
		if mgr := dCtx.GetManager(); mgr == nil || mgr.IsClosed() {
			return nil, derror.ErrManagerNotFound
		}
		return dCtx, nil
	}

	r := g.RequestFromCtx(ctx)
	if r == nil {
		return nil, derror.ErrNotLogin
	}

	mgr, err := authcheck.ResolveManagerFromContext(firstAuthType(authType...), dCtx)
	if err != nil {
		return nil, err
	}
	if ok && dCtx.GetManager() == mgr {
		return dCtx, nil
	}
	return getDContext(r, mgr), nil
}
