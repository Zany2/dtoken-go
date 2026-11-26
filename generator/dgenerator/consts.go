// @Author daixk 2025/12/22 16:08:00
package dgenerator

// Generator default values and token style constants 生成器默认值和 Token 风格常量
const (
	// DefaultTimeout defines the default timeout 默认超时时间，30天，单位：秒。
	DefaultTimeout = 2592000

	// DefaultJWTSecret is a legacy placeholder rejected by JWT operations. DefaultJWTSecret 是旧版占位密钥，JWT 操作会拒绝使用。
	DefaultJWTSecret = "dtoken-go"

	// TikTokenLength defines the TikTok style short ID length TikTok 风格短 ID 的长度
	TikTokenLength = 11

	// TikCharset defines the TikTok style short ID charset TikTok 风格短 ID 的字符集（数字 + 大小写字母）
	TikCharset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

	// HashRandomBytesLen defines the random byte length for hash tokens 哈希 Token 使用的随机字节长度
	HashRandomBytesLen = 16

	// TimestampRandomLen defines the random byte length for timestamp tokens 时间戳 Token 使用的随机字节长度
	TimestampRandomLen = 8

	// DefaultSimpleLength defines the default simple token length 默认简单 Token 的长度
	DefaultSimpleLength = 16
)
