package ants

import (
	"testing"
	"time"
)

// TestRenewPoolManagerSubmitAndStop verifies task submission and stop behavior 测试任务提交与停止行为
func TestRenewPoolManagerSubmitAndStop(t *testing.T) {
	mgr, err := NewRenewPoolManagerWithConfig(&RenewPoolConfig{
		MinSize:       1,
		MaxSize:       2,
		ScaleUpRate:   0.8,
		ScaleDownRate: 0.2,
		CheckInterval: 10 * time.Millisecond,
		Expiry:        time.Second,
		NonBlocking:   false,
	})
	if err != nil {
		t.Fatalf("NewRenewPoolManagerWithConfig() error = %v", err)
	}
	defer mgr.Stop()

	done := make(chan struct{})
	if err = mgr.Submit(func() { close(done) }); err != nil {
		t.Fatalf("Submit() error = %v", err)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("submitted task did not run")
	}

	_, capacity, usage := mgr.Stats()
	if capacity <= 0 {
		t.Fatalf("Stats() capacity = %d, want positive", capacity)
	}
	if usage < 0 || usage > 1 {
		t.Fatalf("Stats() usage = %f, want [0,1]", usage)
	}

	mgr.Stop()
	if err = mgr.Submit(func() {}); err == nil {
		t.Fatal("Submit() should fail after Stop()")
	}
}

// TestRenewPoolManagerNormalizesConfig verifies config normalization during construction 测试构造时配置归一化
func TestRenewPoolManagerNormalizesConfig(t *testing.T) {
	cfg := &RenewPoolConfig{
		MinSize:       0,
		MaxSize:       0,
		ScaleUpRate:   DefaultScaleUpRate,
		ScaleDownRate: DefaultScaleDownRate,
		CheckInterval: DefaultCheckInterval,
		Expiry:        DefaultExpiry,
		NonBlocking:   true,
	}
	mgr, err := NewRenewPoolManagerWithConfig(cfg)
	if err != nil {
		t.Fatalf("NewRenewPoolManagerWithConfig() error = %v", err)
	}
	defer mgr.Stop()

	if cfg.MinSize != DefaultMinSize || cfg.MaxSize != DefaultMinSize {
		t.Fatalf("normalized config = %+v", cfg)
	}
}
