// @Author daixk 2026/05/15
package nonce

import (
	"testing"
	"time"
)

// TestConfigValidateAndClone verifies nonce config validation and clone TestConfigValidateAndClone 验证 Nonce 配置校验和克隆
func TestConfigValidateAndClone(t *testing.T) {
	cfg := DefaultConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate(default) error = %v", err)
	}

	clone := cfg.Clone()
	clone.TTL = time.Minute
	if cfg.TTL == clone.TTL {
		t.Fatal("Clone() should return independent copy")
	}
	if (*Config)(nil).Clone() != nil {
		t.Fatal("Clone() on nil should return nil")
	}
	if err := (*Config)(nil).Validate(); err != nil {
		t.Fatalf("Validate(nil) error = %v", err)
	}

	cfg.TTL = 0
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want invalid ttl error")
	}
}
