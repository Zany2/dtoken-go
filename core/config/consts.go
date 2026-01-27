// @Author daixk 2025/12/7 15:34:00
package config

// SameSiteMode Cookie 的 SameSite 属性值
type SameSiteMode string

const (
	// SameSiteStrict 严格模式
	SameSiteStrict SameSiteMode = "Strict"
	// SameSiteLax 宽松模式
	SameSiteLax SameSiteMode = "Lax"
	// SameSiteNone 无限制模式（需配合 Secure=true）
	SameSiteNone SameSiteMode = "None"
)

// ConcurrencyScope 并发控制的作用域
type ConcurrencyScope string

const (
	// ConcurrencyScopeAccount 账号级别
	ConcurrencyScopeAccount ConcurrencyScope = "account"
	// ConcurrencyScopeDevice 设备级别
	ConcurrencyScopeDevice ConcurrencyScope = "device"
)

// Default configuration constants | 默认配置常量
const (
	// DefaultTokenName 默认 Token 名称
	DefaultTokenName = "dtoken"
	// DefaultKeyPrefix 默认存储键前缀
	DefaultKeyPrefix = "dtoken:"
	// DefaultAuthType 默认认证体系类型
	DefaultAuthType = "auth:"
	// DefaultTimeout 默认 Token 超时时间（30 天，单位：秒）
	DefaultTimeout = 2592000
	// DefaultMaxLoginCount 默认最大并发登录数
	DefaultMaxLoginCount = 12
	// DefaultCookiePath 默认 Cookie 路径
	DefaultCookiePath = "/"
	// NoLimit 不限制标志（用于超时、数量等字段，值为 -1）
	NoLimit = -1
)
