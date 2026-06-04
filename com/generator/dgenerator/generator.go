// @Author daixk 2025/12/17 9:39:00
package dgenerator

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Generator implements token generation. Generator 实现 Token 生成器。
type Generator struct {
	timeout      int64              // timeout stores token ttl in seconds for JWT exp claims. timeout 存储 JWT exp 使用的 Token 有效秒数。
	jwtSecretKey string             // jwtSecretKey stores JWT signing secret. jwtSecretKey 存储 JWT 签名密钥。
	tokenStyle   adapter.TokenStyle // tokenStyle stores token generation style. tokenStyle 存储 Token 生成风格。
}

// Interface assertion keeps generator contract checked at compile time. 接口断言在编译期检查生成器契约。
var _ adapter.Generator = (*Generator)(nil)

// NewGenerator creates a token generator. NewGenerator 创建新的 Token 生成器。
func NewGenerator(timeout int64, jwtSecretKey string, tokenStyle adapter.TokenStyle) *Generator {
	return &Generator{
		timeout:      timeout,
		jwtSecretKey: jwtSecretKey,
		tokenStyle:   tokenStyle,
	}
}

// NewDefaultGenerator creates the default token generator. NewDefaultGenerator 创建默认 Token 生成器。
func NewDefaultGenerator() *Generator {
	return &Generator{
		timeout:      DefaultTimeout,
		jwtSecretKey: DefaultJWTSecret,
		tokenStyle:   adapter.TokenStyleUUID,
	}
}

// Generate creates a token by configured style. Generate 根据配置的风格生成 Token。
func (g *Generator) Generate(loginID, device, deviceId string) (string, error) {
	if loginID == "" {
		return "", derror.ErrEmptyLoginID
	}

	switch g.tokenStyle {
	case adapter.TokenStyleUUID:
		return g.generateUUID()
	case adapter.TokenStyleSimple:
		return g.generateSimple(DefaultSimpleLength)
	case adapter.TokenStyleRandom32:
		return g.generateSimple(32)
	case adapter.TokenStyleRandom64:
		return g.generateSimple(64)
	case adapter.TokenStyleRandom128:
		return g.generateSimple(128)
	case adapter.TokenStyleJWT:
		return g.generateJWT(loginID, device, deviceId)
	case adapter.TokenStyleHash:
		return g.generateHash(loginID, device, deviceId)
	case adapter.TokenStyleTimestamp:
		return g.generateTimestamp(loginID)
	case adapter.TokenStyleTik:
		return g.generateTik()
	default:
		return g.generateUUID()
	}
}

// ParseJWT parses a JWT token and returns claims. ParseJWT 解析 JWT Token 并返回声明。
func (g *Generator) ParseJWT(tokenStr string) (jwt.MapClaims, error) {
	if tokenStr == "" {
		return nil, fmt.Errorf("token string cannot be empty")
	}

	secretKey := g.getJWTSecret()

	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
		// Verify the signing method. 验证签名方法。
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secretKey), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse JWT: %w", err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// ValidateJWT validates the JWT token. ValidateJWT 验证 JWT Token。
func (g *Generator) ValidateJWT(tokenStr string) error {
	_, err := g.ParseJWT(tokenStr)
	return err
}

// GetLoginInfoFromJWT extracts loginID, device, and deviceId from a JWT token. GetLoginInfoFromJWT 从 JWT Token 中提取登录 ID、设备和设备 ID。
func (g *Generator) GetLoginInfoFromJWT(tokenStr string) (loginID, device, deviceId string, err error) {
	claims, err := g.ParseJWT(tokenStr)
	if err != nil {
		return "", "", "", err
	}

	loginID, ok := claims["loginId"].(string)
	if !ok {
		return "", "", "", fmt.Errorf("loginId not found in token claims")
	}

	device, _ = claims["device"].(string)
	deviceId, _ = claims["deviceId"].(string)

	return loginID, device, deviceId, nil
}

// generateUUID creates a UUID token. generateUUID 生成 UUID Token。
func (g *Generator) generateUUID() (string, error) {
	u, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("failed to generate UUID: %w", err)
	}
	return u.String(), nil
}

// generateSimple creates a random string token with fixed length. generateSimple 生成指定长度的简单随机字符串 Token。
func (g *Generator) generateSimple(length int) (string, error) {
	if length <= 0 {
		length = DefaultSimpleLength
	}
	return randomStringFromCharset(TikCharset, length)
}

// generateJWT creates a JWT token. generateJWT 生成 JWT Token。
func (g *Generator) generateJWT(loginID, device, deviceId string) (string, error) {
	now := time.Now()

	claims := jwt.MapClaims{
		"loginId":  loginID,
		"device":   device,
		"deviceId": deviceId,
		"iat":      now.Unix(),
	}

	// Add expiration when timeout is configured. 配置超时时间时添加过期时间。
	if g.timeout > 0 {
		claims["exp"] = now.Add(time.Duration(g.timeout) * time.Second).Unix()
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	secretKey := g.getJWTSecret()

	signedToken, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT token: %w", err)
	}

	return signedToken, nil
}

// getJWTSecret returns the JWT secret with fallback. getJWTSecret 获取 JWT 密钥，空值时使用默认密钥。
func (g *Generator) getJWTSecret() string {
	if g.jwtSecretKey != "" {
		return g.jwtSecretKey
	}
	return DefaultJWTSecret
}

// generateHash creates a SHA256 hash style token. generateHash 生成 SHA256 哈希风格 Token。
func (g *Generator) generateHash(loginID, device, deviceId string) (string, error) {
	randomBytes := make([]byte, HashRandomBytesLen)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Build the hash input. 创建哈希输入。
	data := fmt.Sprintf(
		"%s:%s:%s:%d:%s",
		loginID,
		device,
		deviceId,
		time.Now().UnixNano(),
		hex.EncodeToString(randomBytes),
	)

	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:]), nil
}

// generateTimestamp creates a timestamp style token. generateTimestamp 生成时间戳风格 Token。
func (g *Generator) generateTimestamp(loginID string) (string, error) {
	randomBytes := make([]byte, TimestampRandomLen)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	timestamp := time.Now().UnixMilli()
	random := hex.EncodeToString(randomBytes)
	return fmt.Sprintf("%d_%s_%s", timestamp, loginID, random), nil
}

// generateTik creates a TikTok style short ID token. generateTik 生成 TikTok 风格的短 ID Token。
func (g *Generator) generateTik() (string, error) {
	return randomStringFromCharset(TikCharset, TikTokenLength)
}

// randomStringFromCharset creates a random string from the charset. randomStringFromCharset 使用指定字符集生成随机字符串。
func randomStringFromCharset(charset string, length int) (string, error) {
	if length <= 0 || charset == "" {
		return "", fmt.Errorf("invalid length or charset")
	}

	charsetLen := int64(len(charset))
	result := make([]byte, length)

	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(charsetLen))
		if err != nil {
			return "", fmt.Errorf("failed to generate random string: %w", err)
		}
		result[i] = charset[num.Int64()]
	}

	return string(result), nil
}
