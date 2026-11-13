package chi

import (
	"github.com/Zany2/dtoken-go"
	derror "github.com/Zany2/dtoken-go/errors"
	"net/http"
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
func Context(auth *dtoken.Auth) func(http.Handler) http.Handler {
	return RegisterDTokenContextMiddleware(WithAuth(auth))
}

// RequireLogin validates credentials from the current request. RequireLogin 校验当前请求的登录凭证。
func RequireLogin() func(http.Handler) http.Handler { return AuthMiddleware() }
