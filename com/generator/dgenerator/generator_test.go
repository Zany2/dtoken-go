// @Author daixk 2025/12/22 15:56:00
package dgenerator

import (
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/golang-jwt/jwt/v5"
)

// TestGeneratorConstructorsVerifyDefaults verifies generator constructor defaults. TestGeneratorConstructorsVerifyDefaults 验证生成器构造器默认配置。
func TestGeneratorConstructorsVerifyDefaults(t *testing.T) {
	custom := NewGenerator(120, "custom-secret", adapter.TokenStyleRandom32)
	if custom == nil {
		t.Fatal("NewGenerator() returned nil")
	}
	if custom.timeout != 120 || custom.jwtSecretKey != "custom-secret" || custom.tokenStyle != adapter.TokenStyleRandom32 {
		t.Fatalf("NewGenerator() = %+v, want custom configuration", custom)
	}

	defaultGenerator := NewDefaultGenerator()
	if defaultGenerator == nil {
		t.Fatal("NewDefaultGenerator() returned nil")
	}
	if defaultGenerator.timeout != DefaultTimeout || defaultGenerator.jwtSecretKey != DefaultJWTSecret || defaultGenerator.tokenStyle != adapter.TokenStyleUUID {
		t.Fatalf("NewDefaultGenerator() = %+v, want defaults", defaultGenerator)
	}
}

// TestGenerateRejectsEmptyLoginID verifies empty loginID validation 测试。loginID 校验
func TestGenerateRejectsEmptyLoginID(t *testing.T) {
	g := NewDefaultGenerator()
	if _, err := g.Generate("", "web", "device-1"); !errors.Is(err, derror.ErrEmptyLoginID) {
		t.Fatalf("Generate() error = %v, want %v", err, derror.ErrEmptyLoginID)
	}
}

// TestGenerateTokenStyles verifies configured token style outputs 测试不同 Token 风格输出
func TestGenerateTokenStyles(t *testing.T) {
	tests := []struct {
		name      string
		style     adapter.TokenStyle
		wantLen   int
		wantParts int
	}{
		{name: "uuid", style: adapter.TokenStyleUUID, wantLen: 36},
		{name: "simple", style: adapter.TokenStyleSimple, wantLen: DefaultSimpleLength},
		{name: "random32", style: adapter.TokenStyleRandom32, wantLen: 32},
		{name: "random64", style: adapter.TokenStyleRandom64, wantLen: 64},
		{name: "random128", style: adapter.TokenStyleRandom128, wantLen: 128},
		{name: "hash", style: adapter.TokenStyleHash, wantLen: 64},
		{name: "timestamp", style: adapter.TokenStyleTimestamp, wantParts: 3},
		{name: "tik", style: adapter.TokenStyleTik, wantLen: TikTokenLength},
		{name: "unknown fallback", style: adapter.TokenStyle("unknown"), wantLen: 36},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewGenerator(60, "secret", tt.style)
			token, err := g.Generate("user-1", "web", "device-1")
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}
			if tt.wantLen > 0 && len(token) != tt.wantLen {
				t.Fatalf("Generate() len = %d, want %d, token=%q", len(token), tt.wantLen, token)
			}
			if tt.wantParts > 0 && len(strings.Split(token, "_")) != tt.wantParts {
				t.Fatalf("Generate() parts = %v, want %d", strings.Split(token, "_"), tt.wantParts)
			}
		})
	}
}

// TestJWTLifecycle verifies JWT generation, parsing and validation 测试 JWT 生成、解析与校验
func TestJWTLifecycle(t *testing.T) {
	g := NewGenerator(60, "secret", adapter.TokenStyleJWT)
	token, err := g.Generate("user-1", "web", "device-1")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	claims, err := g.ParseJWT(token)
	if err != nil {
		t.Fatalf("ParseJWT() error = %v", err)
	}
	if claims["loginId"] != "user-1" {
		t.Fatalf("loginId claim = %v, want user-1", claims["loginId"])
	}
	if err = g.ValidateJWT(token); err != nil {
		t.Fatalf("ValidateJWT() error = %v", err)
	}
	loginID, device, deviceID, err := g.GetLoginInfoFromJWT(token)
	if err != nil {
		t.Fatalf("GetLoginInfoFromJWT() error = %v", err)
	}
	if loginID != "user-1" {
		t.Fatalf("GetLoginInfoFromJWT() loginID = %q, want %q", loginID, "user-1")
	}
	if device != "web" {
		t.Fatalf("GetLoginInfoFromJWT() device = %q, want %q", device, "web")
	}
	if deviceID != "device-1" {
		t.Fatalf("GetLoginInfoFromJWT() deviceId = %q, want %q", deviceID, "device-1")
	}
}

// TestJWTInvalidInputs verifies invalid JWT paths 测试非法 JWT 路径
func TestJWTInvalidInputs(t *testing.T) {
	g := NewGenerator(60, "secret", adapter.TokenStyleJWT)
	if _, err := g.ParseJWT(""); err == nil {
		t.Fatal("ParseJWT() should fail for empty token")
	}
	if _, err := g.ParseJWT("bad-token"); err == nil {
		t.Fatal("ParseJWT() should fail for malformed token")
	}
	if _, _, _, err := g.GetLoginInfoFromJWT("bad-token"); err == nil {
		t.Fatal("GetLoginInfoFromJWT() should fail for malformed token")
	}
}

// TestJWTValidationBoundaries verifies no-expiry, expired, wrong-secret, and non-HMAC JWT paths. TestJWTValidationBoundaries 验证无过期、已过期、错误密钥和非 HMAC JWT 路径。
func TestJWTValidationBoundaries(t *testing.T) {
	noExpiry := NewGenerator(0, "secret", adapter.TokenStyleJWT)
	token, err := noExpiry.Generate("user-1", "web", "device-1")
	if err != nil {
		t.Fatalf("Generate(no expiry) error = %v", err)
	}
	claims, err := noExpiry.ParseJWT(token)
	if err != nil {
		t.Fatalf("ParseJWT(no expiry) error = %v", err)
	}
	if _, exists := claims["exp"]; exists {
		t.Fatal("JWT with non-positive timeout should not contain exp")
	}

	expired := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"loginId": "user-1",
		"exp":     time.Now().Add(-time.Minute).Unix(),
	})
	expiredToken, err := expired.SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("SignedString(expired) error = %v", err)
	}
	if err := noExpiry.ValidateJWT(expiredToken); err == nil {
		t.Fatal("ValidateJWT() should reject expired tokens")
	}

	wrongSecret := NewGenerator(0, "other-secret", adapter.TokenStyleJWT)
	if err := wrongSecret.ValidateJWT(token); err == nil {
		t.Fatal("ValidateJWT() should reject a token signed with another secret")
	}

	none := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{"loginId": "user-1"})
	noneToken, err := none.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("SignedString(none) error = %v", err)
	}
	if err := noExpiry.ValidateJWT(noneToken); err == nil {
		t.Fatal("ValidateJWT() should reject non-HMAC tokens")
	}
}

// TestGetLoginInfoFromJWTRequiresLoginID verifies required login ID claims. TestGetLoginInfoFromJWTRequiresLoginID 验证 loginId Claim 必须存在。
func TestGetLoginInfoFromJWTRequiresLoginID(t *testing.T) {
	g := NewGenerator(60, "secret", adapter.TokenStyleJWT)
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"device":   "web",
		"deviceId": "device-1",
	}).SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("SignedString() error = %v", err)
	}
	if _, _, _, err := g.GetLoginInfoFromJWT(token); err == nil {
		t.Fatal("GetLoginInfoFromJWT() should reject a token without loginId")
	}
}

// TestGeneratorHashAndTimestampFormats verifies generated hash and timestamp token formats. TestGeneratorHashAndTimestampFormats 验证 hash 和 timestamp Token 格式。
func TestGeneratorHashAndTimestampFormats(t *testing.T) {
	g := NewDefaultGenerator()

	first, err := g.generateHash("user-1", "web", "device-1")
	if err != nil {
		t.Fatalf("generateHash() error = %v", err)
	}
	if len(first) != sha256HexLength || !isHex(first) {
		t.Fatalf("generateHash() = %q, want %d hex characters", first, sha256HexLength)
	}
	second, err := g.generateHash("user-1", "web", "device-1")
	if err != nil {
		t.Fatalf("generateHash(second) error = %v", err)
	}
	if first == second {
		t.Fatal("generateHash() returned the same token twice")
	}

	timestampToken, err := g.generateTimestamp("user-1")
	if err != nil {
		t.Fatalf("generateTimestamp() error = %v", err)
	}
	parts := strings.Split(timestampToken, "_")
	if len(parts) != 3 || parts[1] != "user-1" || len(parts[2]) != TimestampRandomLen*2 || !isHex(parts[2]) {
		t.Fatalf("generateTimestamp() = %q, want timestamp_loginID_random format", timestampToken)
	}
	if _, err := strconv.ParseInt(parts[0], 10, 64); err != nil {
		t.Fatalf("generateTimestamp() timestamp = %q, want integer: %v", parts[0], err)
	}
}

// TestGeneratorHelpersVerifyFallbacks verifies helper defaults and charset constraints. TestGeneratorHelpersVerifyFallbacks 验证辅助函数默认值和字符集约束。
func TestGeneratorHelpersVerifyFallbacks(t *testing.T) {
	g := NewGenerator(60, "", adapter.TokenStyleJWT)
	if got := g.getJWTSecret(); got != DefaultJWTSecret {
		t.Fatalf("getJWTSecret() = %q, want %q", got, DefaultJWTSecret)
	}

	if token, err := g.generateSimple(0); err != nil || len(token) != DefaultSimpleLength {
		t.Fatalf("generateSimple(0) = %q, %v, want default length %d", token, err, DefaultSimpleLength)
	}

	const charset = "ab"
	token, err := randomStringFromCharset(charset, 128)
	if err != nil {
		t.Fatalf("randomStringFromCharset() error = %v", err)
	}
	if len(token) != 128 {
		t.Fatalf("randomStringFromCharset() length = %d, want 128", len(token))
	}
	for _, char := range token {
		if !strings.ContainsRune(charset, char) {
			t.Fatalf("randomStringFromCharset() contains %q outside charset %q", char, charset)
		}
	}
}

// TestRandomStringFromCharsetValidation verifies random string input validation 测试随机字符串输入校。
func TestRandomStringFromCharsetValidation(t *testing.T) {
	if _, err := randomStringFromCharset("", 1); err == nil {
		t.Fatal("randomStringFromCharset() should fail for empty charset")
	}
	if _, err := randomStringFromCharset(TikCharset, 0); err == nil {
		t.Fatal("randomStringFromCharset() should fail for non-positive length")
	}
}

const sha256HexLength = 64

// isHex reports whether a string is valid hexadecimal. isHex 判断字符串是否为有效十六进制。
func isHex(value string) bool {
	_, err := hex.DecodeString(value)
	return err == nil
}
