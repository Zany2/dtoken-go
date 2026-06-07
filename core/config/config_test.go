// @Author daixk 2025/12/22 15:56:00
package config

import (
	"testing"

	"github.com/Zany2/dtoken-go/core/adapter"
)

// TestValidateNormalizesNamespaces verifies namespace fields are normalized TestValidateNormalizesNamespaces 验证命名空间字段会被统一格式化
func TestValidateNormalizesNamespaces(t *testing.T) {
	cfg := DefaultConfig()
	cfg.AuthType = " user "
	cfg.KeyPrefix = " custom "

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if cfg.AuthType != "user:" {
		t.Fatalf("AuthType = %q, want %q", cfg.AuthType, "user:")
	}
	if cfg.KeyPrefix != "custom:" {
		t.Fatalf("KeyPrefix = %q, want %q", cfg.KeyPrefix, "custom:")
	}
}

// TestValidateRejectsInvalidTokenStyle verifies unsupported token styles fail fast TestValidateRejectsInvalidTokenStyle 验证不支持的 Token 风格会被提前拒绝
func TestValidateRejectsInvalidTokenStyle(t *testing.T) {
	cfg := DefaultConfig()
	cfg.TokenStyle = adapter.TokenStyle("unknown")

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want invalid token style error")
	}
}

// TestValidateRejectsEmptyKeyPrefix verifies empty key prefix is rejected TestValidateRejectsEmptyKeyPrefix 验证空 KeyPrefix 会被拒绝
func TestValidateRejectsEmptyKeyPrefix(t *testing.T) {
	cfg := DefaultConfig()
	cfg.KeyPrefix = ""

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want empty key prefix error")
	}
}

// TestValidateRejectsWhitespaceTokenName verifies token name cannot be whitespace TestValidateRejectsWhitespaceTokenName 验证 Token 名称不能是空白
func TestValidateRejectsWhitespaceTokenName(t *testing.T) {
	cfg := DefaultConfig()
	cfg.TokenName = "   "

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want whitespace token name error")
	}
}

// TestValidateRejectsSeparatorOnlyNamespaces verifies namespaces need real content TestValidateRejectsSeparatorOnlyNamespaces 验证命名空间必须包含实际内容
func TestValidateRejectsSeparatorOnlyNamespaces(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(cfg *Config)
	}{
		{
			name: "auth type",
			mutate: func(cfg *Config) {
				cfg.AuthType = ":"
			},
		},
		{
			name: "key prefix",
			mutate: func(cfg *Config) {
				cfg.KeyPrefix = "::"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.mutate(cfg)

			if err := cfg.Validate(); err == nil {
				t.Fatal("Validate() error = nil, want separator-only namespace error")
			}
		})
	}
}

// TestValidateRejectsInvalidAutoRenewRelations verifies auto renew timing must be meaningful TestValidateRejectsInvalidAutoRenewRelations 验证自动续期时序必须有效
func TestValidateRejectsInvalidAutoRenewRelations(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(cfg *Config)
	}{
		{
			name: "no limit timeout",
			mutate: func(cfg *Config) {
				cfg.Timeout = NoLimit
			},
		},
		{
			name: "refresh threshold exceeds timeout",
			mutate: func(cfg *Config) {
				cfg.Timeout = 60
				cfg.RenewMaxRefresh = 61
			},
		},
		{
			name: "renew interval reaches timeout",
			mutate: func(cfg *Config) {
				cfg.Timeout = 60
				cfg.RenewInterval = 60
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.mutate(cfg)

			if err := cfg.Validate(); err == nil {
				t.Fatal("Validate() error = nil, want auto renew relation error")
			}
		})
	}
}

// TestValidateRejectsWhitespaceJWTSecret verifies JWT secret cannot be blank TestValidateRejectsWhitespaceJWTSecret 验证 JWT 密钥不能是空白
func TestValidateRejectsWhitespaceJWTSecret(t *testing.T) {
	cfg := DefaultConfig()
	cfg.TokenStyle = adapter.TokenStyleJWT
	cfg.JwtSecretKey = "   "

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want whitespace jwt secret error")
	}
}

// TestValidateRejectsInvalidCookieConfig verifies cookie config basic bounds TestValidateRejectsInvalidCookieConfig 验证 Cookie 配置基础边界
func TestValidateRejectsInvalidCookieConfig(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(cfg *Config)
	}{
		{
			name: "domain with whitespace",
			mutate: func(cfg *Config) {
				cfg.CookieConfig.Domain = "example .com"
			},
		},
		{
			name: "blank path",
			mutate: func(cfg *Config) {
				cfg.CookieConfig.Path = "   "
			},
		},
		{
			name: "path without slash",
			mutate: func(cfg *Config) {
				cfg.CookieConfig.Path = "api"
			},
		},
		{
			name: "negative max age",
			mutate: func(cfg *Config) {
				cfg.CookieConfig.MaxAge = -1
			},
		},
		{
			name: "invalid same site",
			mutate: func(cfg *Config) {
				cfg.CookieConfig.SameSite = SameSiteMode("Invalid")
			},
		},
		{
			name: "same site none without secure",
			mutate: func(cfg *Config) {
				cfg.CookieConfig.SameSite = SameSiteNone
				cfg.CookieConfig.Secure = false
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.mutate(cfg)

			if err := cfg.Validate(); err == nil {
				t.Fatal("Validate() error = nil, want invalid cookie config error")
			}
		})
	}
}

// TestConfigCloneDeepCopiesCookieConfig verifies clone isolates nested cookie config TestConfigCloneDeepCopiesCookieConfig 验证克隆会隔离嵌套 Cookie 配置
func TestConfigCloneDeepCopiesCookieConfig(t *testing.T) {
	cfg := DefaultConfig()
	cfg.AuthType = "user:"
	cfg.CookieConfig.Domain = "example.com"
	cfg.CookieConfig.Path = "/api"
	cfg.CookieConfig.MaxAge = 3600

	clone := cfg.Clone()
	if clone == cfg {
		t.Fatal("Clone() returned original config pointer")
	}
	if clone.CookieConfig == cfg.CookieConfig {
		t.Fatal("Clone() reused CookieConfig pointer, want deep copy")
	}
	if clone.CookieConfig.Domain != "example.com" || clone.CookieConfig.Path != "/api" || clone.CookieConfig.MaxAge != 3600 {
		t.Fatalf("clone CookieConfig = %+v, want copied values", clone.CookieConfig)
	}

	clone.CookieConfig.Domain = "changed.example.com"
	clone.CookieConfig.Path = "/changed"
	if cfg.CookieConfig.Domain != "example.com" || cfg.CookieConfig.Path != "/api" {
		t.Fatalf("original CookieConfig changed after clone mutation: %+v", cfg.CookieConfig)
	}
}

// TestConfigClonePreservesNilCookieConfig verifies nil cookie config stays nil TestConfigClonePreservesNilCookieConfig 验证 nil Cookie 配置克隆后仍为 nil
func TestConfigClonePreservesNilCookieConfig(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CookieConfig = nil

	clone := cfg.Clone()
	if clone == cfg {
		t.Fatal("Clone() returned original config pointer")
	}
	if clone.CookieConfig != nil {
		t.Fatalf("Clone().CookieConfig = %+v, want nil", clone.CookieConfig)
	}
}

// TestValidateRejectsMissingCookieConfig verifies cookie reads need cookie config TestValidateRejectsMissingCookieConfig 验证读取 Cookie 时必须提供 Cookie 配置
func TestValidateRejectsMissingCookieConfig(t *testing.T) {
	cfg := DefaultConfig()
	cfg.IsReadCookie = true
	cfg.CookieConfig = nil

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want nil cookie config error")
	}
}

// TestSetCookieConfigAllowsNil verifies callers can explicitly clear cookie config TestSetCookieConfigAllowsNil 验证调用方可以显式清空 Cookie 配置
func TestSetCookieConfigAllowsNil(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SetCookieConfig(nil)

	if cfg.CookieConfig != nil {
		t.Fatal("CookieConfig should be nil after SetCookieConfig(nil)")
	}
}

// TestValidateRejectsNilConfig verifies nil config is rejected TestValidateRejectsNilConfig 验证空配置会被拒绝
func TestValidateRejectsNilConfig(t *testing.T) {
	if err := (*Config)(nil).Validate(); err == nil {
		t.Fatal("Validate() error = nil, want nil config error")
	}
}
