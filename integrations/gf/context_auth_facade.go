// @Author daixk 2026/06/05
package gf

import (
	"context"

	manager "github.com/Zany2/dtoken-go/internal/engine"
)

// GetLoginIDByCtx gets login ID by context GetLoginIDByCtx 从上下文获取登录 ID
func GetLoginIDByCtx(ctx context.Context, authType ...string) (string, error) {
	dCtx, err := requireDTokenContextByCtx(ctx, authType...)
	if err != nil {
		return "", err
	}
	return dCtx.Auth().GetLoginID(ctx)
}

// GetTokenInfoByCtx gets token info by context GetTokenInfoByCtx 从上下文获取 Token 信息
func GetTokenInfoByCtx(ctx context.Context, authType ...string) (*manager.TokenInfo, error) {
	dCtx, err := requireDTokenContextByCtx(ctx, authType...)
	if err != nil {
		return nil, err
	}
	return dCtx.Auth().GetTokenInfo(ctx)
}

// IntrospectTokenByCtx inspects current token without renewal side effects IntrospectTokenByCtx 无续期副作用地检查当前 Token 状态。
func IntrospectTokenByCtx(ctx context.Context, authType ...string) (*manager.TokenIntrospection, error) {
	dCtx, err := requireDTokenContextByCtx(ctx, authType...)
	if err != nil {
		return nil, err
	}
	return dCtx.Auth().IntrospectToken(ctx)
}
