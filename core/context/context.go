// @Author daixk 2025/12/22 15:56:00
package context

import (
	"context"
	"strings"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/manager"
)

const (
	bearerPrefix = "Bearer "
	authHeader   = "Authorization"
)

// DTokenContext defines current request dtoken context DTokenContext 定义当前请求 DToken 上下文
type DTokenContext struct {
	reqCtx  adapter.RequestContext
	manager *manager.Manager
}

// NewContext creates dtoken context NewContext 创建新的 DToken 上下文
func NewContext(reqCtx adapter.RequestContext, mgr *manager.Manager) *DTokenContext {
	return &DTokenContext{
		reqCtx:  reqCtx,
		manager: mgr,
	}
}

// GetTokenValue gets token from current request GetTokenValue 从当前请求读取 Token
func (c *DTokenContext) GetTokenValue() string {
	cfg := c.manager.GetConfig()

	if cfg.IsReadHeader {
		if token := strings.TrimSpace(c.reqCtx.GetHeader(cfg.TokenName)); token != "" {
			return extractBearerToken(token)
		}
		if auth := c.reqCtx.GetHeader(authHeader); auth != "" {
			if token := extractBearerToken(auth); token != "" {
				return token
			}
		}
	}
	if cfg.IsReadCookie {
		if token := strings.TrimSpace(c.reqCtx.GetCookie(cfg.TokenName)); token != "" {
			return token
		}
	}
	if cfg.IsReadQuery {
		if token := strings.TrimSpace(c.reqCtx.GetQuery(cfg.TokenName)); token != "" {
			return token
		}
	}
	if cfg.IsReadBody {
		if token := strings.TrimSpace(c.reqCtx.GetPostForm(cfg.TokenName)); token != "" {
			return token
		}
	}

	return ""
}

// GetRequestContext returns raw request context GetRequestContext 获取原始请求上下文
func (c *DTokenContext) GetRequestContext() adapter.RequestContext {
	return c.reqCtx
}

// GetManager returns related manager GetManager 获取关联的认证管理器
func (c *DTokenContext) GetManager() *manager.Manager {
	return c.manager
}

// requireToken returns current token or not-login error requireToken 获取当前 Token，不存在时返回未登录错误
func (c *DTokenContext) requireToken() (string, error) {
	token := c.GetTokenValue()
	if token == "" {
		return "", derror.ErrNotLogin
	}
	return token, nil
}

// currentLoginID resolves login ID from current token currentLoginID 通过当前 Token 解析登录 ID
func (c *DTokenContext) currentLoginID(ctx context.Context) (string, error) {
	token, err := c.requireToken()
	if err != nil {
		return "", err
	}
	return c.manager.GetLoginID(ctx, token)
}

// setTokenCookie writes current token cookie setTokenCookie 写入当前 Token Cookie
func (c *DTokenContext) setTokenCookie(token string) {
	if token == "" {
		return
	}
	c.setCookie(token, c.tokenCookieMaxAge())
}

// clearTokenCookie clears current token cookie clearTokenCookie 清理当前 Token Cookie
func (c *DTokenContext) clearTokenCookie() {
	c.setCookie("", -1)
}

// setCookie writes token cookie with configured options setCookie 按配置写入 Token Cookie
func (c *DTokenContext) setCookie(value string, maxAge int64) {
	cfg := c.manager.GetConfig()
	cookieCfg := cfg.CookieConfig
	if cookieCfg == nil {
		c.reqCtx.SetCookie(cfg.TokenName, value, int(maxAge), "/", "", false, true)
		return
	}
	c.reqCtx.SetCookieWithOptions(&adapter.CookieOptions{
		Name:     cfg.TokenName,
		Value:    value,
		MaxAge:   int(maxAge),
		Path:     cookieCfg.Path,
		Domain:   cookieCfg.Domain,
		Secure:   cookieCfg.Secure,
		HttpOnly: cookieCfg.HttpOnly,
		SameSite: string(cookieCfg.SameSite),
	})
}

// tokenCookieMaxAge returns configured cookie max age tokenCookieMaxAge 返回配置的 Cookie 最大有效期
func (c *DTokenContext) tokenCookieMaxAge() int64 {
	cfg := c.manager.GetConfig()
	if cfg.CookieConfig == nil {
		return 0
	}
	return cfg.CookieConfig.MaxAge
}

// extractBearerToken extracts bearer token extractBearerToken 从 Authorization 头中提取 Bearer Token
func extractBearerToken(auth string) string {
	auth = strings.TrimSpace(auth)
	if auth == "" {
		return ""
	}
	if strings.EqualFold(auth, strings.TrimSpace(bearerPrefix)) {
		return ""
	}
	if len(auth) > len(bearerPrefix) && strings.EqualFold(auth[:len(bearerPrefix)], bearerPrefix) {
		return strings.TrimSpace(auth[len(bearerPrefix):])
	}
	return auth
}
