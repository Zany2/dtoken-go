package config

import (
	"fmt"
	"github.com/Zany2/dtoken-go/core/adapter"
	"strings"
)

// Config 总配置结构体
type Config struct {
	// AuthType 认证体系类型
	AuthType string

	// KeyPrefix 存储键的前缀
	KeyPrefix string

	// TokenName Token 名称（同时也是 Cookie 名称）
	TokenName string

	// Timeout Token 超时时间（单位：秒，-1 表示永不过期）
	Timeout int64

	// AutoRenew 是否在每次验证 Token 时自动续期（延长有效期）
	AutoRenew bool

	// RenewMaxRefresh Token 自动续期触发阈值（单位：秒，剩余有效期低于此值时触发续期，-1 表示不限制）
	RenewMaxRefresh int64

	// RenewInterval 同一 Token 两次续期的最小间隔时间（单位：秒，-1 表示不限制）
	RenewInterval int64

	// ActiveTimeout Token 最大不活跃时长（单位：秒，超过此时间未访问则被踢出，-1 表示不限制）
	ActiveTimeout int64

	// ConcurrencyScope 并发控制的作用域（"account"=账号级，"device"=设备级）
	ConcurrencyScope ConcurrencyScope

	// IsConcurrent 是否允许同一账号并发登录（true=允许，false=新登录挤掉旧登录）
	IsConcurrent bool

	// IsShare 并发登录是否共用同一个 Token（true=共用一个，false=每次登录新建一个）
	IsShare bool

	// MaxLoginCount 同一账号最大登录数量（-1 表示不限，仅当 IsConcurrent=true 且 IsShare=false 时生效）
	MaxLoginCount int64

	// IsReadBody 是否尝试从请求体读取 Token
	IsReadBody bool

	// IsReadHeader 是否尝试从 HTTP Header 读取 Token（推荐开启）
	IsReadHeader bool

	// IsReadCookie 是否尝试从 Cookie 读取 Token
	IsReadCookie bool

	// TokenStyle Token 生成风格
	TokenStyle adapter.TokenStyle

	// JwtSecretKey JWT 模式的密钥（仅当 TokenStyle 为 JWT 时生效）
	JwtSecretKey string

	// IsLog 是否开启操作日志
	IsLog bool

	// IsPrintBanner 是否打印启动 Banner
	IsPrintBanner bool

	// AsyncEvent 是否异步触发事件（true=异步，false=同步）
	AsyncEvent bool

	// CookieConfig Cookie 配置
	CookieConfig *CookieConfig
}

// CookieConfig Cookie 配置结构体
type CookieConfig struct {
	// Domain Cookie 作用域
	Domain string

	// Path Cookie 路径
	Path string

	// Secure 是否仅在 HTTPS 下生效
	Secure bool

	// HttpOnly 是否禁止 JavaScript 访问 Cookie
	HttpOnly bool

	// SameSite 属性（Strict、Lax、None）
	SameSite SameSiteMode

	// MaxAge Cookie 过期时间（单位：秒）
	MaxAge int64
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		AuthType:         DefaultAuthType,
		KeyPrefix:        DefaultKeyPrefix,
		TokenName:        DefaultTokenName,
		Timeout:          DefaultTimeout,
		AutoRenew:        true,
		RenewMaxRefresh:  DefaultTimeout / 2,
		RenewInterval:    NoLimit,
		ActiveTimeout:    NoLimit,
		ConcurrencyScope: ConcurrencyScopeAccount,
		IsConcurrent:     true,
		IsShare:          true,
		MaxLoginCount:    DefaultMaxLoginCount,
		IsReadBody:       false,
		IsReadHeader:     true,
		IsReadCookie:     false,
		TokenStyle:       adapter.TokenStyleUUID,
		JwtSecretKey:     "",
		IsLog:            false,
		IsPrintBanner:    true,
		AsyncEvent:       true,
		CookieConfig:     DefaultCookieConfig(),
	}
}

// DefaultCookieConfig 返回默认的 Cookie 配置
func DefaultCookieConfig() *CookieConfig {
	return &CookieConfig{
		Domain:   "",
		Path:     DefaultCookiePath,
		Secure:   false,
		HttpOnly: true,
		SameSite: SameSiteLax,
		MaxAge:   0,
	}
}

// Validate 验证配置是否合法
func (c *Config) Validate() error {
	// 阶段1：基础格式验证
	if c.TokenName == "" {
		return fmt.Errorf("TokenName 不能为空")
	}
	if strings.ContainsAny(c.TokenName, "\t\r\n") {
		return fmt.Errorf("TokenName 不能包含制表符或换行符，当前值：%q", c.TokenName)
	}
	if len(c.TokenName) > 64 {
		return fmt.Errorf("TokenName 长度不能超过 64 个字符，当前长度：%d", len(c.TokenName))
	}

	if c.AuthType == "" {
		return fmt.Errorf("AuthType 不能为空")
	}
	if strings.ContainsAny(c.AuthType, "\t\r\n") {
		return fmt.Errorf("AuthType 不能包含制表符或换行符，当前值：%q", c.AuthType)
	}
	if len(c.AuthType) > 64 {
		return fmt.Errorf("AuthType 长度不能超过 64 个字符，当前长度：%d", len(c.AuthType))
	}

	// 阶段2：数值范围验证
	if err := c.checkNoLimits(); err != nil {
		return err
	}

	// 阶段3：并发作用域验证
	switch c.ConcurrencyScope {
	case ConcurrencyScopeAccount, ConcurrencyScopeDevice:
		// 合法值
	default:
		return fmt.Errorf("ConcurrencyScope 必须为 %q 或 %q，当前值：%q",
			ConcurrencyScopeAccount, ConcurrencyScopeDevice, c.ConcurrencyScope)
	}

	// 阶段4：Token 风格相关验证
	if c.TokenStyle == adapter.TokenStyleJWT && c.JwtSecretKey == "" {
		return fmt.Errorf("TokenStyle 为 JWT 时，JwtSecretKey 不能为空")
	}

	// 阶段5：关键参数自动修正
	if c.AutoRenew && c.Timeout != NoLimit && c.RenewMaxRefresh != NoLimit && c.RenewMaxRefresh > c.Timeout {
		c.RenewMaxRefresh = c.Timeout / 2
		if c.RenewMaxRefresh <= 0 {
			c.RenewMaxRefresh = c.Timeout
		}
	}

	// 阶段6：时间关系验证
	if c.AutoRenew && c.ActiveTimeout != NoLimit && c.RenewInterval != NoLimit && c.RenewInterval >= c.ActiveTimeout {
		return fmt.Errorf("RenewInterval (%d) 必须小于 ActiveTimeout (%d)，否则活跃用户可能被踢出", c.RenewInterval, c.ActiveTimeout)
	}

	// 阶段7：Token 读取来源验证
	if !c.IsReadHeader && !c.IsReadCookie && !c.IsReadBody {
		return fmt.Errorf("至少需要启用 IsReadHeader、IsReadCookie 或 IsReadBody 中的一项")
	}

	// 阶段8：Cookie 配置验证
	if c.IsReadCookie && c.CookieConfig == nil {
		return fmt.Errorf("启用 IsReadCookie 时，CookieConfig 不能为空")
	}
	if c.CookieConfig != nil {
		if err := c.validateCookieConfig(); err != nil {
			return err
		}
	}

	return nil
}

// validateCookieConfig 验证 Cookie 配置是否合法
func (c *Config) validateCookieConfig() error {
	cc := c.CookieConfig

	if cc.Path == "" {
		return fmt.Errorf("CookieConfig.Path 不能为空")
	}

	switch cc.SameSite {
	case SameSiteLax, SameSiteStrict, SameSiteNone, "":
	default:
		return fmt.Errorf("无效的 CookieConfig.SameSite 值：%v", cc.SameSite)
	}

	if cc.SameSite == SameSiteNone && !cc.Secure {
		return fmt.Errorf("SameSite 为 None 时，Secure 必须为 true（浏览器强制要求）")
	}

	return nil
}

// Clone 克隆配置
func (c *Config) Clone() *Config {
	newConfig := *c
	if c.CookieConfig != nil {
		cookieConfig := *c.CookieConfig
		newConfig.CookieConfig = &cookieConfig
	}
	return &newConfig
}

// SetAuthType 设置认证体系类型
func (c *Config) SetAuthType(authType string) *Config {
	c.AuthType = authType
	return c
}

// SetKeyPrefix 设置存储键的前缀
func (c *Config) SetKeyPrefix(keyPrefix string) *Config {
	c.KeyPrefix = keyPrefix
	return c
}

// SetTokenName 设置 Token 名称
func (c *Config) SetTokenName(name string) *Config {
	c.TokenName = name
	return c
}

// SetTimeout 设置超时时间
func (c *Config) SetTimeout(timeout int64) *Config {
	c.Timeout = timeout
	return c
}

// SetRenewMaxRefresh 设置自动续期触发阈值
func (c *Config) SetRenewMaxRefresh(refresh int64) *Config {
	c.RenewMaxRefresh = refresh
	return c
}

// SetRenewInterval 设置最小续期间隔
func (c *Config) SetRenewInterval(interval int64) *Config {
	c.RenewInterval = interval
	return c
}

// SetActiveTimeout 设置最大不活跃时长
func (c *Config) SetActiveTimeout(timeout int64) *Config {
	c.ActiveTimeout = timeout
	return c
}

// SetIsConcurrent 设置是否允许并发登录
func (c *Config) SetIsConcurrent(isConcurrent bool) *Config {
	c.IsConcurrent = isConcurrent
	return c
}

// SetIsShare 设置是否共享 Token
func (c *Config) SetIsShare(isShare bool) *Config {
	c.IsShare = isShare
	return c
}

// SetMaxLoginCount 设置最大登录数量
func (c *Config) SetMaxLoginCount(count int64) *Config {
	c.MaxLoginCount = count
	return c
}

// SetIsReadBody 设置是否从请求体读取 Token
func (c *Config) SetIsReadBody(isReadBody bool) *Config {
	c.IsReadBody = isReadBody
	return c
}

// SetIsReadHeader 设置是否从 Header 读取 Token
func (c *Config) SetIsReadHeader(isReadHeader bool) *Config {
	c.IsReadHeader = isReadHeader
	return c
}

// SetIsReadCookie 设置是否从 Cookie 读取 Token
func (c *Config) SetIsReadCookie(isReadCookie bool) *Config {
	c.IsReadCookie = isReadCookie
	return c
}

// SetTokenStyle 设置 Token 生成风格
func (c *Config) SetTokenStyle(style adapter.TokenStyle) *Config {
	c.TokenStyle = style
	return c
}

// SetJwtSecretKey 设置 JWT 密钥
func (c *Config) SetJwtSecretKey(key string) *Config {
	c.JwtSecretKey = key
	return c
}

// SetAutoRenew 设置是否自动续期
func (c *Config) SetAutoRenew(autoRenew bool) *Config {
	c.AutoRenew = autoRenew
	return c
}

// SetIsLog 设置是否开启日志
func (c *Config) SetIsLog(isLog bool) *Config {
	c.IsLog = isLog
	return c
}

// SetIsPrintBanner 设置是否打印启动 Banner
func (c *Config) SetIsPrintBanner(isPrint bool) *Config {
	c.IsPrintBanner = isPrint
	return c
}

// SetAsyncEvent 设置是否异步触发事件
func (c *Config) SetAsyncEvent(asyncEvent bool) *Config {
	c.AsyncEvent = asyncEvent
	return c
}

// SetCookieConfig 设置 Cookie 配置
func (c *Config) SetCookieConfig(cookieConfig *CookieConfig) *Config {
	if cookieConfig != nil {
		c.CookieConfig = cookieConfig
	}
	return c
}

// checkNoLimits 验证所有数值型配置必须为 -1（无限制）或大于 0（有效值）
func (c *Config) checkNoLimits() error {
	fields := map[string]int64{
		"Timeout":         c.Timeout,
		"RenewMaxRefresh": c.RenewMaxRefresh,
		"RenewInterval":   c.RenewInterval,
		"ActiveTimeout":   c.ActiveTimeout,
		"MaxLoginCount":   c.MaxLoginCount,
	}

	for name, value := range fields {
		if value == -1 || value > 0 {
			continue
		}
		return fmt.Errorf("%s 必须为 -1（无限制）或大于 0，当前值：%d", name, value)
	}
	return nil
}
