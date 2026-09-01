// @Author daixk 2026/08/31
package config

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Zany2/dtoken-go/core/adapter"
)

// TestDefaultConfigValues verifies every default runtime option. TestDefaultConfigValues 验证运行时配置的全部默认值。
func TestDefaultConfigValues(t *testing.T) {
	want := &Config{
		AuthType:              DefaultAuthType,
		KeyPrefix:             DefaultKeyPrefix,
		TokenName:             DefaultTokenName,
		Timeout:               DefaultTimeout,
		RefreshTokenTimeout:   DefaultRefreshTokenTimeout,
		AutoRenew:             true,
		RenewMaxRefresh:       DefaultTimeout / 2,
		RenewInterval:         NoLimit,
		ActiveTimeout:         NoLimit,
		ConcurrencyScope:      ConcurrencyScopeAccount,
		IsConcurrent:          true,
		IsShare:               true,
		MaxLoginCount:         DefaultMaxLoginCount,
		ReplacedLoginExitMode: ReplacedLoginExitModeOldDevice,
		OverflowLogoutMode:    LogoutModeKickout,
		IsReadBody:            false,
		IsReadQuery:           false,
		IsReadHeader:          true,
		IsReadCookie:          false,
		TokenStyle:            adapter.TokenStyleUUID,
		JwtSecretKey:          DefaultJWTSecretKey,
		IsLog:                 false,
		IsPrintBanner:         true,
		AsyncEvent:            true,
		CookieConfig: &CookieConfig{
			Domain:   "",
			Path:     DefaultCookiePath,
			Secure:   false,
			HttpOnly: true,
			SameSite: SameSiteLax,
			MaxAge:   0,
		},
	}

	if got := DefaultConfig(); !reflect.DeepEqual(got, want) {
		t.Fatalf("DefaultConfig() = %+v, want %+v", got, want)
	}
}

// TestDefaultConfigReturnsIndependentInstances verifies default configs do not share mutable state. TestDefaultConfigReturnsIndependentInstances 验证默认配置实例之间不共享可变状态。
func TestDefaultConfigReturnsIndependentInstances(t *testing.T) {
	first := DefaultConfig()
	second := DefaultConfig()

	if first == second {
		t.Fatal("DefaultConfig() returned the same config pointer")
	}
	if first.CookieConfig == second.CookieConfig {
		t.Fatal("DefaultConfig() shared CookieConfig pointer")
	}

	first.AuthType = "admin:"
	first.CookieConfig.Path = "/admin"
	if second.AuthType != DefaultAuthType || second.CookieConfig.Path != DefaultCookiePath {
		t.Fatalf("mutating one default config changed another: %+v", second)
	}
}

// TestDefaultCookieConfigReturnsIndependentInstances verifies cookie defaults are independently allocated. TestDefaultCookieConfigReturnsIndependentInstances 验证 Cookie 默认配置会独立分配。
func TestDefaultCookieConfigReturnsIndependentInstances(t *testing.T) {
	first := DefaultCookieConfig()
	second := DefaultCookieConfig()

	if first == second {
		t.Fatal("DefaultCookieConfig() returned the same pointer")
	}

	first.Domain = "example.com"
	first.Path = "/admin"
	if second.Domain != "" || second.Path != DefaultCookiePath {
		t.Fatalf("mutating one cookie config changed another: %+v", second)
	}
}

// TestValidateReportsInvalidNumericField verifies numeric errors identify the failing field. TestValidateReportsInvalidNumericField 验证数值错误会明确指出失败字段。
func TestValidateReportsInvalidNumericField(t *testing.T) {
	fields := []struct {
		name string
		set  func(*Config)
	}{
		{name: "Timeout", set: func(cfg *Config) { cfg.Timeout = 0 }},
		{name: "RefreshTokenTimeout", set: func(cfg *Config) { cfg.RefreshTokenTimeout = 0 }},
		{name: "RenewMaxRefresh", set: func(cfg *Config) { cfg.RenewMaxRefresh = 0 }},
		{name: "RenewInterval", set: func(cfg *Config) { cfg.RenewInterval = 0 }},
		{name: "ActiveTimeout", set: func(cfg *Config) { cfg.ActiveTimeout = 0 }},
		{name: "MaxLoginCount", set: func(cfg *Config) { cfg.MaxLoginCount = 0 }},
	}

	for _, field := range fields {
		field := field
		t.Run(field.name, func(t *testing.T) {
			cfg := DefaultConfig()
			field.set(cfg)

			err := cfg.Validate()
			if err == nil {
				t.Fatal("Validate() error = nil, want numeric validation error")
			}
			if !strings.Contains(err.Error(), "Config."+field.name) {
				t.Fatalf("Validate() error = %q, want field %q", err, field.name)
			}
		})
	}
}

// TestConfigNamespaceSettersNormalizeImmediately verifies namespace setters normalize before validation. TestConfigNamespaceSettersNormalizeImmediately 验证命名空间 Setter 会立即规范化。
func TestConfigNamespaceSettersNormalizeImmediately(t *testing.T) {
	cfg := DefaultConfig().SetAuthType(" admin ").SetKeyPrefix(" dtoken: ")

	if cfg.AuthType != "admin:" {
		t.Fatalf("AuthType = %q, want %q", cfg.AuthType, "admin:")
	}
	if cfg.KeyPrefix != "dtoken:" {
		t.Fatalf("KeyPrefix = %q, want %q", cfg.KeyPrefix, "dtoken:")
	}
}

// TestValidateNamespaceNormalizationIsIdempotent verifies repeated validation does not change namespaces. TestValidateNamespaceNormalizationIsIdempotent 验证重复校验不会重复修改命名空间。
func TestValidateNamespaceNormalizationIsIdempotent(t *testing.T) {
	cfg := DefaultConfig().SetAuthType("user").SetKeyPrefix("app")

	if err := cfg.Validate(); err != nil {
		t.Fatalf("first Validate() error = %v", err)
	}
	firstAuthType, firstKeyPrefix := cfg.AuthType, cfg.KeyPrefix

	if err := cfg.Validate(); err != nil {
		t.Fatalf("second Validate() error = %v", err)
	}
	if cfg.AuthType != firstAuthType || cfg.KeyPrefix != firstKeyPrefix {
		t.Fatalf("namespace changed after repeated validation: AuthType=%q KeyPrefix=%q", cfg.AuthType, cfg.KeyPrefix)
	}
}
