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

// TestJWTRejectsUnsafeSecrets covers signing and all public JWT validation entry points. TestJWTRejectsUnsafeSecrets 覆盖签发及所有公开 JWT 校验入口的密钥检查。
func TestJWTRejectsUnsafeSecrets(t *testing.T) {
	for _, tt := range []struct {
		name   string
		secret string
	}{
		{"empty", ""},
		{"whitespace", " \t\r\n"},
		{"default", DefaultJWTSecret},
		{"padded default", " " + DefaultJWTSecret + "\t"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			g := NewGenerator(60, tt.secret, adapter.TokenStyleJWT)
			if token, err := g.Generate("user", "web", "browser"); token != "" || !errors.Is(err, derror.ErrInvalidParam) {
				t.Fatalf("Generate() = %q, %v; want empty token and ErrInvalidParam", token, err)
			}

			// Sign externally with the key previously accepted by the generator. 使用生成器此前接受的密钥从外部签发。
			key := tt.secret
			if key == "" {
				key = DefaultJWTSecret
			}
			token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"loginId": "user"}).SignedString([]byte(key))
			if err != nil {
				t.Fatal(err)
			}
			if claims, err := g.ParseJWT(token); claims != nil || !errors.Is(err, derror.ErrInvalidParam) {
				t.Fatalf("ParseJWT() = %v, %v; want nil claims and ErrInvalidParam", claims, err)
			}
			if err := g.ValidateJWT(token); !errors.Is(err, derror.ErrInvalidParam) {
				t.Fatalf("ValidateJWT() = %v; want ErrInvalidParam", err)
			}
			if id, device, deviceID, err := g.GetLoginInfoFromJWT(token); id != "" || device != "" || deviceID != "" || !errors.Is(err, derror.ErrInvalidParam) {
				t.Fatalf("GetLoginInfoFromJWT() = %q, %q, %q, %v; want empty identity and ErrInvalidParam", id, device, deviceID, err)
			}
		})
	}
}

// TestJWTUsesExactCustomSecret ensures validation does not trim a valid signing key. TestJWTUsesExactCustomSecret 确保校验不会裁剪有效签名密钥。
func TestJWTUsesExactCustomSecret(t *testing.T) {
	const secret = " custom-secret "
	g := NewGenerator(60, secret, adapter.TokenStyleJWT)
	token, err := g.Generate("user", "web", "browser")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := jwt.Parse(token, func(*jwt.Token) (any, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()})); err != nil {
		t.Fatalf("generated token does not use the exact configured key: %v", err)
	}
	if err := g.ValidateJWT(token); err != nil {
		t.Fatalf("ValidateJWT() rejected custom key: %v", err)
	}
	if err := NewGenerator(60, "custom-secret", adapter.TokenStyleJWT).ValidateJWT(token); !errors.Is(err, jwt.ErrTokenSignatureInvalid) {
		t.Fatalf("trimmed key validation = %v; want signature error", err)
	}
}

// TestJWTOnlyAcceptsHS256 rejects other HMAC algorithms even with a matching secret. TestJWTOnlyAcceptsHS256 即使密钥匹配也拒绝其他 HMAC 算法。
func TestJWTOnlyAcceptsHS256(t *testing.T) {
	g := NewGenerator(60, "test-secret", adapter.TokenStyleJWT)
	for _, method := range []*jwt.SigningMethodHMAC{jwt.SigningMethodHS256, jwt.SigningMethodHS384, jwt.SigningMethodHS512} {
		t.Run(method.Alg(), func(t *testing.T) {
			token, err := jwt.NewWithClaims(method, jwt.MapClaims{"loginId": "user"}).SignedString([]byte("test-secret"))
			if err != nil {
				t.Fatal(err)
			}
			claims, err := g.ParseJWT(token)
			if method == jwt.SigningMethodHS256 {
				if err != nil || claims["loginId"] != "user" {
					t.Fatalf("ParseJWT(HS256) = %v, %v", claims, err)
				}
			} else if claims != nil || !errors.Is(err, jwt.ErrTokenSignatureInvalid) {
				t.Fatalf("ParseJWT(%s) = %v, %v; want nil claims and algorithm rejection", method.Alg(), claims, err)
			}
		})
	}
}

// TestJWTLoginInfoValidatesDeviceClaims distinguishes absent claims from malformed values. TestJWTLoginInfoValidatesDeviceClaims 区分缺失设备声明和类型错误的值。
func TestJWTLoginInfoValidatesDeviceClaims(t *testing.T) {
	g := NewGenerator(60, "test-secret", adapter.TokenStyleJWT)
	for _, field := range []string{"device", "deviceId"} {
		t.Run(field, func(t *testing.T) {
			for _, tt := range []struct {
				name    string
				value   any
				missing bool
				wantErr bool
			}{
				{name: "missing", missing: true},
				{name: "empty", value: ""},
				{name: "string", value: "mobile"},
				{name: "null", value: nil, wantErr: true},
				{name: "number", value: 123, wantErr: true},
				{name: "boolean", value: true, wantErr: true},
				{name: "array", value: []string{"mobile"}, wantErr: true},
				{name: "object", value: map[string]string{"name": "mobile"}, wantErr: true},
			} {
				t.Run(tt.name, func(t *testing.T) {
					claims := jwt.MapClaims{"loginId": "user", "device": "web", "deviceId": "browser"}
					claims[field] = tt.value
					if tt.missing {
						delete(claims, field)
					}
					token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("test-secret"))
					if err != nil {
						t.Fatal(err)
					}
					id, device, deviceID, err := g.GetLoginInfoFromJWT(token)
					if tt.wantErr {
						if err == nil || id != "" || device != "" || deviceID != "" {
							t.Fatalf("invalid claim returned %q, %q, %q, %v", id, device, deviceID, err)
						}
						return
					}
					wantDevice, _ := claims["device"].(string)
					wantDeviceID, _ := claims["deviceId"].(string)
					if err != nil || id != "user" || device != wantDevice || deviceID != wantDeviceID {
						t.Fatalf("valid claims returned %q, %q, %q, %v", id, device, deviceID, err)
					}
				})
			}
		})
	}
}
