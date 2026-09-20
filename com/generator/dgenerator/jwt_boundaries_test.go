package dgenerator

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// TestJWTIndependentIssuancesHaveUniqueIDs verifies identical subjects receive distinct credentials across generator instances. TestJWTIndependentIssuancesHaveUniqueIDs 验证相同主体跨生成器实例独立签发时获得不同凭证。
func TestJWTIndependentIssuancesHaveUniqueIDs(t *testing.T) {
	seenTokens := make(map[string]bool)
	seenIDs := make(map[string]bool)
	for i := 0; i < 32; i++ {
		g := NewGenerator(60, "test-secret", adapter.TokenStyleJWT)
		token, err := g.Generate("user", "web", "browser")
		if err != nil {
			t.Fatal(err)
		}
		claims, err := g.ParseJWT(token)
		if err != nil {
			t.Fatal(err)
		}
		id, ok := claims["jti"].(string)
		if !ok || id == "" {
			t.Fatal("generated JWT is missing its unique ID")
		}
		if _, err := uuid.Parse(id); err != nil {
			t.Fatalf("invalid JWT ID: %v", err)
		}
		if seenTokens[token] || seenIDs[id] {
			t.Fatal("independent issuance reused a token or JWT ID")
		}
		seenTokens[token], seenIDs[id] = true, true
		if claims["loginId"] != "user" || claims["device"] != "web" || claims["deviceId"] != "browser" {
			t.Fatalf("identity claims changed: %v", claims)
		}
	}
}

// TestJWTTimeoutBoundaries verifies direct generator use cannot overflow expiration and retains no-expiry compatibility. TestJWTTimeoutBoundaries 验证直接使用生成器时过期时间不会溢出，并保留不过期兼容行为。
func TestJWTTimeoutBoundaries(t *testing.T) {
	const maxSeconds = math.MaxInt64 / int64(time.Second)
	for _, timeout := range []int64{-1, 0, 60, maxSeconds} {
		g := NewGenerator(timeout, "test-secret", adapter.TokenStyleJWT)
		token, err := g.Generate("user", "", "")
		if err != nil {
			t.Fatalf("Generate(timeout=%d): %v", timeout, err)
		}
		claims, err := g.ParseJWT(token)
		if err != nil {
			t.Fatalf("ParseJWT(timeout=%d): %v", timeout, err)
		}
		expires, exists := claims["exp"]
		if timeout <= 0 {
			if exists {
				t.Fatalf("timeout=%d unexpectedly added exp", timeout)
			}
			continue
		}
		exp, expOK := expires.(float64)
		iat, iatOK := claims["iat"].(float64)
		if !exists || !expOK || !iatOK || exp-iat != float64(timeout) {
			t.Fatalf("timeout=%d has incorrect lifetime: exp=%v iat=%v", timeout, expires, claims["iat"])
		}
	}
	for _, timeout := range []int64{maxSeconds + 1, math.MaxInt64} {
		token, err := NewGenerator(timeout, "test-secret", adapter.TokenStyleJWT).Generate("user", "", "")
		if token != "" || !errors.Is(err, derror.ErrInvalidParam) {
			t.Fatalf("Generate(timeout=%d) = %q, %v; want empty token and ErrInvalidParam", timeout, token, err)
		}
	}
}

// TestJWTLoginInfoRejectsInvalidIdentity verifies signed but invalid identity claims cannot be reported as successful login information. TestJWTLoginInfoRejectsInvalidIdentity 验证已签名但身份声明无效的 Token 不会返回成功的登录信息。
func TestJWTLoginInfoRejectsInvalidIdentity(t *testing.T) {
	g := NewGenerator(60, "test-secret", adapter.TokenStyleJWT)
	for _, loginID := range []any{"", nil, 123} {
		token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"loginId": loginID}).SignedString([]byte("test-secret"))
		if err != nil {
			t.Fatal(err)
		}
		id, device, deviceID, err := g.GetLoginInfoFromJWT(token)
		if err == nil || id != "" || device != "" || deviceID != "" {
			t.Fatalf("invalid loginId=%v returned %q, %q, %q, %v", loginID, id, device, deviceID, err)
		}
	}

	// Older JWTs without jti remain readable. 不含 jti 的旧 JWT 仍可读取。
	legacy, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"loginId": "legacy"}).SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatal(err)
	}
	if id, _, _, err := g.GetLoginInfoFromJWT(legacy); err != nil || id != "legacy" {
		t.Fatalf("legacy JWT identity = %q, %v", id, err)
	}
}
