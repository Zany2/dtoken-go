// @Author daixk 2025/12/22 15:56:00
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

// TestRenewPoolManagerRejectsInvalidConfig verifies invalid config returns error TestRenewPoolManagerRejectsInvalidConfig 验证非法配置会返回错误
func TestRenewPoolManagerRejectsInvalidConfig(t *testing.T) {
	cfg := &RenewPoolConfig{
		MinSize:       0,
		MaxSize:       0,
		ScaleUpRate:   DefaultScaleUpRate,
		ScaleDownRate: DefaultScaleDownRate,
		CheckInterval: DefaultCheckInterval,
		Expiry:        DefaultExpiry,
		NonBlocking:   true,
	}
	if mgr, err := NewRenewPoolManagerWithConfig(cfg); err == nil {
		mgr.Stop()
		t.Fatal("NewRenewPoolManagerWithConfig() error = nil, want invalid config error")
	}
}

// TestRenewPoolManagerNilSafety verifies nil manager methods are safe TestRenewPoolManagerNilSafety 验证空管理器方法安全
func TestRenewPoolManagerNilSafety(t *testing.T) {
	var mgr *RenewPoolManager
	if err := mgr.Submit(func() {}); err == nil {
		t.Fatal("Submit(nil manager) error = nil, want error")
	}
	if err := mgr.Submit(nil); err == nil {
		t.Fatal("Submit(nil task) error = nil, want error")
	}
	mgr.Stop()
	running, capacity, usage := mgr.Stats()
	if running != 0 || capacity != 0 || usage != 0 {
		t.Fatalf("Stats(nil manager) = (%d, %d, %f), want zero", running, capacity, usage)
	}
}
