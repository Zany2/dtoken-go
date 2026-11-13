// @Author daixk 2026/05/15
package oauth2

import (
	"testing"
	"time"
)

// TestConfigValidateAndClone verifies OAuth2 config validation and clone TestConfigValidateAndClone 验证 OAuth2 配置校验和克隆
func TestConfigValidateAndClone(t *testing.T) {
	cfg := DefaultConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate(default) error = %v", err)
	}

	clone := cfg.Clone()
	clone.TokenExpiration = time.Minute
	if cfg.TokenExpiration == clone.TokenExpiration {
		t.Fatal("Clone() should return independent copy")
	}
	if (*Config)(nil).Clone() != nil {
		t.Fatal("Clone() on nil should return nil")
	}
	if err := (*Config)(nil).Validate(); err != nil {
		t.Fatalf("Validate(nil) error = %v", err)
	}

	tests := []Config{
		{CodeExpiration: 0, TokenExpiration: time.Hour, RefreshExpiration: time.Hour},
		{CodeExpiration: time.Minute, TokenExpiration: 0, RefreshExpiration: time.Hour},
		{CodeExpiration: time.Minute, TokenExpiration: time.Hour, RefreshExpiration: 0},
		{CodeExpiration: time.Minute, TokenExpiration: time.Hour, RefreshExpiration: time.Hour},
	}
	for _, tt := range tests {
		cfg := tt
		if err := cfg.Validate(); err == nil {
			t.Fatalf("Validate(%+v) should fail", cfg)
		}
	}
}
