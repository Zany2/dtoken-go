package dgenerator

import (
	"strings"
	"testing"

	"github.com/Zany2/dtoken-go/core/adapter"
)

// TestGenerateRejectsEmptyLoginID verifies empty loginID validation 测试空 loginID 校验
func TestGenerateRejectsEmptyLoginID(t *testing.T) {
	g := NewDefaultGenerator()
	if _, err := g.Generate("", "web", "device-1"); err != ErrEmptyLoginID {
		t.Fatalf("Generate() error = %v, want %v", err, ErrEmptyLoginID)
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
	loginID, err := g.GetLoginIDFromJWT(token)
	if err != nil {
		t.Fatalf("GetLoginIDFromJWT() error = %v", err)
	}
	if loginID != "user-1" {
		t.Fatalf("GetLoginIDFromJWT() = %q, want %q", loginID, "user-1")
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
	if _, err := g.GetLoginIDFromJWT("bad-token"); err == nil {
		t.Fatal("GetLoginIDFromJWT() should fail for malformed token")
	}
}

// TestRandomStringFromCharsetValidation verifies random string input validation 测试随机字符串输入校验
func TestRandomStringFromCharsetValidation(t *testing.T) {
	if _, err := randomStringFromCharset("", 1); err == nil {
		t.Fatal("randomStringFromCharset() should fail for empty charset")
	}
	if _, err := randomStringFromCharset(TikCharset, 0); err == nil {
		t.Fatal("randomStringFromCharset() should fail for non-positive length")
	}
}
