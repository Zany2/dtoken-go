package fiber

import (
	"context"
	"github.com/Zany2/dtoken-go"
	derror "github.com/Zany2/dtoken-go/errors"
	gofiber "github.com/gofiber/fiber/v2"
)

// WithAuth selects an open authentication instance or returns ErrManagerNotFound. WithAuth 选择未关闭的认证实例，否则返回 ErrManagerNotFound。
func WithAuth(auth *dtoken.Auth) (AuthOption, error) {
	manager := auth.Manager()
	if manager == nil {
		return nil, derror.ErrManagerNotFound
	}
	return WithManager(manager), nil
}

// Context creates request middleware or returns an invalid-instance error. Context 创建请求中间件，实例无效时返回错误。
func Context(auth *dtoken.Auth) (gofiber.Handler, error) {
	option, err := WithAuth(auth)
	if err != nil {
		return nil, err
	}
	return RegisterDTokenContextMiddleware(context.Background(), option), nil
}

// RequireLogin validates credentials from the current request. RequireLogin 校验当前请求的登录凭证。
func RequireLogin() gofiber.Handler { return AuthMiddleware(context.Background()) }
