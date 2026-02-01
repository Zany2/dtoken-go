package context

import (
	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/manager"
	"strings"
)

const (
	bearerPrefix = "Bearer "
	authHeader   = "Authorization"
)

// DTokenContext 表示当前请求的 Sa-Token 上下文
type DTokenContext struct {
	reqCtx  adapter.RequestContext
	manager *manager.Manager
}

// NewContext 创建新的 DTokenContext 上下文
func NewContext(reqCtx adapter.RequestContext, mgr *manager.Manager) *DTokenContext {
	return &DTokenContext{
		reqCtx:  reqCtx,
		manager: mgr,
	}
}

// GetTokenValue 从当前请求中获取 Token 值，按 Header → Cookie → Query 顺序尝试
func (c *DTokenContext) GetTokenValue() string {
	cfg := c.manager.GetConfig()

	// 1. 尝试从 Header 获取
	if cfg.IsReadHeader {
		// 优先从配置的 Token 名称对应的 Header 中读取
		if token := strings.TrimSpace(c.reqCtx.GetHeader(cfg.TokenName)); token != "" {
			return token
		}

		// 其次尝试从 Authorization 头中提取 Bearer Token
		if auth := c.reqCtx.GetHeader(authHeader); auth != "" {
			if token := extractBearerToken(auth); token != "" {
				return token
			}
		}
	}

	// 2. 尝试从 Cookie 获取
	if cfg.IsReadCookie {
		if token := strings.TrimSpace(c.reqCtx.GetCookie(cfg.TokenName)); token != "" {
			return token
		}
	}

	// 3. 尝试从 URL 查询参数获取
	if token := strings.TrimSpace(c.reqCtx.GetQuery(cfg.TokenName)); token != "" {
		return token
	}

	return ""
}

// GetRequestContext 获取原始请求上下文
func (c *DTokenContext) GetRequestContext() adapter.RequestContext {
	return c.reqCtx
}

// GetManager 获取关联的认证管理器
func (c *DTokenContext) GetManager() *manager.Manager {
	return c.manager
}

// extractBearerToken 从 Authorization 头中提取 Bearer Token（忽略大小写）
func extractBearerToken(auth string) string {
	auth = strings.TrimSpace(auth)
	if auth == "" {
		return ""
	}

	// 检查是否以 "Bearer " 开头（不区分大小写）
	if len(auth) > 7 && strings.EqualFold(auth[:7], bearerPrefix) {
		return strings.TrimSpace(auth[7:])
	}

	// 若不符合 Bearer 格式，直接返回原值（兼容自定义格式）
	return auth
}
