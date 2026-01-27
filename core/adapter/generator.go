// @Author daixk 2025/12/5 15:52:00
package adapter

// Generator Token生成接口
type Generator interface {
	// Generate 生成Token
	Generate(loginId, device, deviceId string) (string, error)
}

// TokenStyle Token生成风格
type TokenStyle string

const (
	// TokenStyleUUID UUID风格
	TokenStyleUUID TokenStyle = "uuid"
	// TokenStyleSimple 简单随机字符串
	TokenStyleSimple TokenStyle = "simple"
	// TokenStyleRandom32 32位随机字符串
	TokenStyleRandom32 TokenStyle = "random32"
	// TokenStyleRandom64 64位随机字符串
	TokenStyleRandom64 TokenStyle = "random64"
	// TokenStyleRandom128 128位随机字符串
	TokenStyleRandom128 TokenStyle = "random128"
	// TokenStyleJWT JWT风格
	TokenStyleJWT TokenStyle = "jwt"
	// TokenStyleHash SHA256哈希风格
	TokenStyleHash TokenStyle = "hash"
	// TokenStyleTimestamp 时间戳风格
	TokenStyleTimestamp TokenStyle = "timestamp"
	// TokenStyleTik Tik风格短ID（类似抖音）
	TokenStyleTik TokenStyle = "tik"
)
