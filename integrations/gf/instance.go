package gf

import (
	"context"
	"github.com/Zany2/dtoken-go"
	derror "github.com/Zany2/dtoken-go/errors"
	"github.com/gogf/gf/v2/net/ghttp"
)

// WithAuth selects an authentication instance for middleware. WithAuth 为中间件选择认证实例。
func WithAuth(auth *dtoken.Auth) AuthOption {
	manager := auth.Manager()
	if manager == nil {
		panic(derror.ErrManagerNotFound)
	}
	return WithManager(manager)
}

// Context binds an instance to each request. Context 为每个请求绑定认证实例。
func Context(auth *dtoken.Auth) ghttp.HandlerFunc {
	return RegisterDTokenContextMiddleware(context.Background(), WithAuth(auth))
}

// RequireLogin validates credentials from the current request. RequireLogin 校验当前请求的登录凭证。
func RequireLogin() ghttp.HandlerFunc { return AuthMiddleware(context.Background()) }
