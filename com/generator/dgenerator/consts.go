// @Author daixk 2025/12/22 16:08:00
package dgenerator

// Constants for token generation Token生成常量
const (
	// DefaultTimeout 默认超时时间（30天，单位：秒）
	DefaultTimeout = 2592000
	// DefaultJWTSecret 默认 JWT 密钥（生产环境应覆盖）
	DefaultJWTSecret = "dtoken-go"
	// TikTokenLength TikTok 风格短 ID 的长度
	TikTokenLength = 11
	// TikCharset TikTok 风格短 ID 的字符集（数字 + 大小写字母）
	TikCharset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	// HashRandomBytesLen 哈希 Token 使用的随机字节长度
	HashRandomBytesLen = 16
	// TimestampRandomLen 时间戳 Token 使用的随机字节长度
	TimestampRandomLen = 8
	// DefaultSimpleLength 默认简单 Token 的长度
	DefaultSimpleLength = 16
)
