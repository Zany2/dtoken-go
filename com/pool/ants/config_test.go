package ants

import (
	"testing"
	"time"
)

// TestRenewPoolConfigDefaultsAndSetters verifies defaults and fluent setters 测试默认配置与链式设置
func TestRenewPoolConfigDefaultsAndSetters(t *testing.T) {
	cfg := DefaultRenewPoolConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate(default) error = %v", err)
	}

	cfg.SetMinSize(2).
		SetMaxSize(4).
		SetScaleUpRate(0.8).
		SetScaleDownRate(0.2).
		SetCheckInterval(time.Second).
		SetExpiry(2 * time.Second).
		SetPrintStatusInterval(time.Minute).
		SetPreAlloc(true).
		SetNonBlocking(false)

	if cfg.MinSize != 2 || cfg.MaxSize != 4 || !cfg.PreAlloc || cfg.NonBlocking {
		t.Fatalf("setters produced config = %+v", cfg)
	}

	clone := cfg.Clone()
	clone.SetMinSize(3)
	if cfg.MinSize == clone.MinSize {
		t.Fatal("Clone() should return independent copy")
	}
	if (*RenewPoolConfig)(nil).Clone() != nil {
		t.Fatal("Clone() on nil should return nil")
	}
	if err := (*RenewPoolConfig)(nil).Validate(); err != nil {
		t.Fatalf("Validate(nil) error = %v", err)
	}
}

// TestRenewPoolConfigValidateInvalid verifies invalid config detection 测试非法配置校验
func TestRenewPoolConfigValidateInvalid(t *testing.T) {
	tests := []RenewPoolConfig{
		{MinSize: 0, MaxSize: 1, ScaleUpRate: 0.8, ScaleDownRate: 0.2, CheckInterval: time.Second, Expiry: time.Second},
		{MinSize: 2, MaxSize: 1, ScaleUpRate: 0.8, ScaleDownRate: 0.2, CheckInterval: time.Second, Expiry: time.Second},
		{MinSize: 1, MaxSize: 1, ScaleUpRate: 0, ScaleDownRate: 0.2, CheckInterval: time.Second, Expiry: time.Second},
		{MinSize: 1, MaxSize: 1, ScaleUpRate: 0.8, ScaleDownRate: -0.1, CheckInterval: time.Second, Expiry: time.Second},
		{MinSize: 1, MaxSize: 1, ScaleUpRate: 0.8, ScaleDownRate: 0.2, CheckInterval: 0, Expiry: time.Second},
		{MinSize: 1, MaxSize: 1, ScaleUpRate: 0.8, ScaleDownRate: 0.2, CheckInterval: time.Second, Expiry: 0},
	}

	for _, tt := range tests {
		cfg := tt
		if err := cfg.Validate(); err == nil {
			t.Fatalf("Validate(%+v) should fail", cfg)
		}
	}
}
