// @Author daixk 2025/12/22 15:56:00
package config

// SameSiteMode defines cookie sameSite mode SameSiteMode 定义 Cookie SameSite 属性值
type SameSiteMode string

const (
	// SameSiteStrict uses strict mode SameSiteStrict 使用严格模式
	SameSiteStrict SameSiteMode = "Strict"
	// SameSiteLax uses lax mode SameSiteLax 使用宽松模式
	SameSiteLax SameSiteMode = "Lax"
	// SameSiteNone uses none mode SameSiteNone 使用无限制模式
	SameSiteNone SameSiteMode = "None"
)

// ConcurrencyScope defines concurrency scope ConcurrencyScope 定义并发控制作用域
type ConcurrencyScope string

const (
	// ConcurrencyScopeAccount uses account scope ConcurrencyScopeAccount 使用账号级别作用域
	ConcurrencyScopeAccount ConcurrencyScope = "account"
	// ConcurrencyScopeDevice uses device scope ConcurrencyScopeDevice 使用设备级别作用域
	ConcurrencyScopeDevice ConcurrencyScope = "device"
)

// ReplacedLoginExitMode defines non-concurrent login strategy ReplacedLoginExitMode 定义非并发登录处理策略
type ReplacedLoginExitMode string

const (
	// ReplacedLoginExitModeOldDevice exits old terminals ReplacedLoginExitModeOldDevice 让旧终端下线
	ReplacedLoginExitModeOldDevice ReplacedLoginExitMode = "old_device"
	// ReplacedLoginExitModeNewDevice rejects new login ReplacedLoginExitModeNewDevice 拒绝新终端登录
	ReplacedLoginExitModeNewDevice ReplacedLoginExitMode = "new_device"
)

// LogoutMode defines token exit mode LogoutMode 定义 Token 下线模式
type LogoutMode string

const (
	// LogoutModeLogout deletes token mapping LogoutModeLogout 删除 Token 映射
	LogoutModeLogout LogoutMode = "logout"
	// LogoutModeKickout marks token as kicked out LogoutModeKickout 标记 Token 被踢下线
	LogoutModeKickout LogoutMode = "kickout"
	// LogoutModeReplaced marks token as replaced LogoutModeReplaced 标记 Token 被顶下线
	LogoutModeReplaced LogoutMode = "replaced"
)

const (
	// DefaultTokenName stores default token name DefaultTokenName 存储默认 Token 名称
	DefaultTokenName = "dtoken"
	// DefaultKeyPrefix stores default key prefix DefaultKeyPrefix 存储默认存储键前缀
	DefaultKeyPrefix = "dtoken:"
	// DefaultAuthType stores default auth type DefaultAuthType 存储默认认证体系类型
	DefaultAuthType = "auth:"
	// TokenKeyPrefix stores token key prefix TokenKeyPrefix 存储 Token 键前缀
	TokenKeyPrefix = "token:"
	// DefaultTimeout stores default timeout DefaultTimeout 存储默认 Token 超时时间
	DefaultTimeout = 2592000
	// DefaultJWTSecretKey stores default jwt secret key DefaultJWTSecretKey 存储默认 JWT 密钥
	DefaultJWTSecretKey = "dtoken-go"
	// DefaultMaxLoginCount stores default max login count DefaultMaxLoginCount 存储默认最大并发登录数
	DefaultMaxLoginCount = 12
	// DefaultCookiePath stores default cookie path DefaultCookiePath 存储默认 Cookie 路径
	DefaultCookiePath = "/"
	// NoLimit marks unlimited value NoLimit 标记无限制取值
	NoLimit = -1
)
