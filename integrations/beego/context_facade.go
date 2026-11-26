// @Author daixk 2026/06/06
package beego

import (
	"context"
	derror "github.com/Zany2/dtoken-go/errors"
	requestcontext "github.com/Zany2/dtoken-go/httpauth/context"

	adapter "github.com/Zany2/dtoken-go/contract"
	manager "github.com/Zany2/dtoken-go/internal/engine"
	beegocontext "github.com/beego/beego/v2/server/web/context"
)

// GetTokenValueByContext gets token value from current Beego context GetTokenValueByContext 从当前 Beego 上下文获取 Token。
func GetTokenValueByContext(c *beegocontext.Context) (string, error) {
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
func GetRequestContextByContext(c *beegocontext.Context) (adapter.RequestContext, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.GetRequestContext(), nil
}

// GetManagerByContext gets current DToken manager GetManagerByContext 获取当前 DToken 管理。
func GetManagerByContext(c *beegocontext.Context) (*manager.Manager, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.GetManager(), nil
}

// GetLoginIDByContext gets current login ID GetLoginIDByContext 获取当前登录 ID
func GetLoginIDByContext(c *beegocontext.Context) (string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return "", err
	}
	return dCtx.Auth().GetLoginID(requestContext(c))
}

// GetTokenInfoByContext gets current token info GetTokenInfoByContext 获取当前 token 信息
func GetTokenInfoByContext(c *beegocontext.Context) (*manager.TokenInfo, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Auth().GetTokenInfo(requestContext(c))
}

// IntrospectTokenByContext inspects current token without renewal side effects IntrospectTokenByContext 无续期副作用地检查当前 Token 状态。
func IntrospectTokenByContext(c *beegocontext.Context) (*manager.TokenIntrospection, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Auth().IntrospectToken(requestContext(c))
}

// requestContext gets standard context from Beego request requestContext 从 Beego 请求获取标准上下文。
func requestContext(c *beegocontext.Context) context.Context {
	if c != nil && c.Request != nil {
		return c.Request.Context()
	}
	return context.Background()
}

// requireDTokenContextByContext gets required DToken context requireDTokenContextByContext 获取必需的 DToken 上下文。
func requireDTokenContextByContext(c *beegocontext.Context) (*requestcontext.DTokenContext, error) {
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
