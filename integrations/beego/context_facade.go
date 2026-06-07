// @Author daixk 2026/06/06
package beego

import (
	"context"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/manager"
	"github.com/Zany2/dtoken-go/integrations/authcheck"
	beegocontext "github.com/beego/beego/v2/server/web/context"
)

// GetTokenValueByContext gets token value from current Beego context GetTokenValueByContext 从当前 Beego 上下文获取 token 值
func GetTokenValueByContext(c *beegocontext.Context) (string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return "", err
	}

	tokenValue := dCtx.GetTokenValue()
	if tokenValue == "" {
		return "", ErrNotLogin
	}
	return tokenValue, nil
}

// GetRequestContextByContext gets raw request context GetRequestContextByContext 获取原始请求上下文
func GetRequestContextByContext(c *beegocontext.Context) (adapter.RequestContext, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.GetRequestContext(), nil
}

// GetManagerByContext gets current DToken manager GetManagerByContext 获取当前 DToken 管理器
func GetManagerByContext(c *beegocontext.Context) (*manager.Manager, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.GetManager(), nil
}

// IsLoginByContext checks current request login state IsLoginByContext 检查当前请求登录状态
func IsLoginByContext(c *beegocontext.Context) bool {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return false
	}
	return dCtx.Auth().IsLogin(requestContext(c))
}

// CheckLoginByContext checks current request login state CheckLoginByContext 校验当前请求登录状态
func CheckLoginByContext(c *beegocontext.Context) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Auth().CheckLogin(requestContext(c))
}

// GetLoginIDByContext gets current login ID GetLoginIDByContext 获取当前登录 ID
func GetLoginIDByContext(c *beegocontext.Context) (string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return "", err
	}
	return dCtx.Auth().GetLoginID(requestContext(c))
}

// LoginByTokenByContext renews current token login state LoginByTokenByContext 使用当前 token 续期登录状态
func LoginByTokenByContext(c *beegocontext.Context) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Auth().LoginByToken(requestContext(c))
}

// LogoutByContext logs out current request token LogoutByContext 登出当前请求 token
func LogoutByContext(c *beegocontext.Context) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Auth().Logout(requestContext(c))
}

// KickoutByContext kicks out current request token KickoutByContext 踢出当前请求 token
func KickoutByContext(c *beegocontext.Context) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Auth().Kickout(requestContext(c))
}

// ReplaceByContext replaces current request token ReplaceByContext 顶替当前请求 token
func ReplaceByContext(c *beegocontext.Context) error {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return err
	}
	return dCtx.Auth().Replace(requestContext(c))
}

// GetTokenInfoByContext gets current token info GetTokenInfoByContext 获取当前 token 信息
func GetTokenInfoByContext(c *beegocontext.Context) (*manager.TokenInfo, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Auth().GetTokenInfo(requestContext(c))
}

// IntrospectTokenByContext inspects current token without renewal side effects IntrospectTokenByContext 无续期副作用地检查当前 token 状态
func IntrospectTokenByContext(c *beegocontext.Context) (*manager.TokenIntrospection, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return nil, err
	}
	return dCtx.Auth().IntrospectToken(requestContext(c))
}

// GetDeviceByContext gets current token device GetDeviceByContext 获取当前 token 设备类型
func GetDeviceByContext(c *beegocontext.Context) (string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return "", err
	}
	return dCtx.Auth().GetDevice(requestContext(c))
}

// GetDeviceIDByContext gets current token device id GetDeviceIDByContext 获取当前 token 设备 ID
func GetDeviceIDByContext(c *beegocontext.Context) (string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return "", err
	}
	return dCtx.Auth().GetDeviceId(requestContext(c))
}

// GetTokenTTLByContext gets current token TTL GetTokenTTLByContext 获取当前 token 剩余有效期
func GetTokenTTLByContext(c *beegocontext.Context) (int64, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return 0, err
	}
	return dCtx.Auth().GetTokenTTL(requestContext(c))
}

// GetTokenCreateTimeByContext gets current token create time GetTokenCreateTimeByContext 获取当前 token 创建时间
func GetTokenCreateTimeByContext(c *beegocontext.Context) (int64, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil {
		return 0, err
	}
	return dCtx.Auth().GetTokenCreateTime(requestContext(c))
}

// RenewTimeoutByContext renews current token timeout RenewTimeoutByContext 续期当前 token 过期时间
func RenewTimeoutByContext(c *beegocontext.Context, timeout time.Duration) error {
	tokenValue, err := GetTokenValueByContext(c)
	if err != nil {
		return err
	}
	return RenewTimeout(requestContext(c), tokenValue, timeout)
}

// requestContext gets standard context from Beego request requestContext 从 Beego 请求获取标准上下文
func requestContext(c *beegocontext.Context) context.Context {
	if c != nil && c.Request != nil {
		return c.Request.Context()
	}
	return context.Background()
}

// requireDTokenContextByContext gets required DToken context requireDTokenContextByContext 获取必需的 DToken 上下文
func requireDTokenContextByContext(c *beegocontext.Context) (*DTokenContext, error) {
	dCtx, ok := GetDTokenContext(c)
	if !ok {
		if c == nil {
			return nil, ErrNotLogin
		}
		mgr, err := authcheck.GetManager("")
		if err != nil {
			return nil, err
		}
		return getDContext(c, mgr), nil
	}
	return dCtx, nil
}
