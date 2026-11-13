// @Author daixk 2025/12/22 15:56:00
package ants

import (
	"sync"
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

// TestNewRenewPoolManagerWithDefaultConfig verifies the default constructor returns a usable manager. TestNewRenewPoolManagerWithDefaultConfig 验证默认构造器返回可用管理器。
func TestNewRenewPoolManagerWithDefaultConfig(t *testing.T) {
	mgr := NewRenewPoolManagerWithDefaultConfig()
	if mgr == nil {
		t.Fatal("NewRenewPoolManagerWithDefaultConfig() returned nil")
	}
	defer mgr.Stop()

	if mgr.config == nil || mgr.config.MinSize != DefaultMinSize || mgr.config.MaxSize != DefaultMaxSize {
		t.Fatalf("default manager config = %+v, want default bounds", mgr.config)
	}
	if _, capacity, _ := mgr.Stats(); capacity <= 0 {
		t.Fatal("default manager should initialize a positive-capacity pool")
	}
}

// TestRenewPoolManagerScalesUpFromSingleWorker verifies small pools actually grow past capacity one. TestRenewPoolManagerScalesUpFromSingleWorker 验证单协程池可以实际扩容。
func TestRenewPoolManagerScalesUpFromSingleWorker(t *testing.T) {
	mgr, err := NewRenewPoolManagerWithConfig(&RenewPoolConfig{
		MinSize:       1,
		MaxSize:       2,
		ScaleUpRate:   0.8,
		ScaleDownRate: 0.2,
		CheckInterval: 5 * time.Millisecond,
		Expiry:        time.Second,
		NonBlocking:   true,
	})
	if err != nil {
		t.Fatalf("NewRenewPoolManagerWithConfig() error = %v", err)
	}
	defer mgr.Stop()

	started := make(chan struct{})
	release := make(chan struct{})
	if err := mgr.Submit(func() {
		close(started)
		<-release
	}); err != nil {
		t.Fatalf("Submit() error = %v", err)
	}
	<-started

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if _, capacity, _ := mgr.Stats(); capacity >= 2 {
			close(release)
			return
		}
		time.Sleep(time.Millisecond)
	}
	close(release)
	running, capacity, usage := mgr.Stats()
	t.Fatalf("pool did not scale up from capacity one; final stats = (%d, %d, %.2f)", running, capacity, usage)
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

// TestRenewPoolManagerStatsAfterStop verifies stopping preserves safe statistics access. TestRenewPoolManagerStatsAfterStop 验证停止后仍可安全读取统计信息。
func TestRenewPoolManagerStatsAfterStop(t *testing.T) {
	mgr, err := NewRenewPoolManagerWithConfig(&RenewPoolConfig{
		MinSize:       1,
		MaxSize:       2,
		ScaleUpRate:   0.8,
		ScaleDownRate: 0.2,
		CheckInterval: time.Second,
		Expiry:        time.Second,
		NonBlocking:   true,
	})
	if err != nil {
		t.Fatalf("NewRenewPoolManagerWithConfig() error = %v", err)
	}
	mgr.Stop()
	mgr.Stop()

	running, capacity, usage := mgr.Stats()
	if running < 0 || capacity < 0 || usage < 0 || usage > 1 {
		t.Fatalf("Stats(after Stop) = (%d, %d, %f), want non-negative bounded values", running, capacity, usage)
	}
}

// TestRenewPoolManagerConcurrentSubmitAndStop verifies concurrent stop safety TestRenewPoolManagerConcurrentSubmitAndStop 验证并发提交和停止安全。
func TestRenewPoolManagerConcurrentSubmitAndStop(t *testing.T) {
	mgr, err := NewRenewPoolManagerWithConfig(&RenewPoolConfig{
		MinSize:       1,
		MaxSize:       4,
		ScaleUpRate:   0.8,
		ScaleDownRate: 0.2,
		CheckInterval: 10 * time.Millisecond,
		Expiry:        time.Second,
		NonBlocking:   true,
	})
	if err != nil {
		t.Fatalf("NewRenewPoolManagerWithConfig() error = %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = mgr.Submit(func() {})
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		mgr.Stop()
	}()
	wg.Wait()

	if err := mgr.Submit(func() {}); err == nil {
		t.Fatal("Submit() should fail after concurrent Stop()")
	}
}

// TestRenewPoolManagerAutoScaleCycle verifies bounded growth and return to minimum capacity. TestRenewPoolManagerAutoScaleCycle 验证有界扩容及空闲后恢复最小容量。
func TestRenewPoolManagerAutoScaleCycle(t *testing.T) {
	cfg := DefaultRenewPoolConfig().SetMinSize(2).SetMaxSize(4).SetCheckInterval(10 * time.Millisecond)
	mgr, err := NewRenewPoolManagerWithConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer mgr.Stop()

	// Keep workers occupied while observing both growth steps. 保持工作协程占用，观察两次扩容。
	release := make(chan struct{})
	defer func() {
		if release != nil {
			close(release)
		}
	}()
	taskRelease := release
	for i := 0; i < cfg.MinSize; i++ {
		if err := mgr.Submit(func() { <-taskRelease }); err != nil {
			t.Fatal(err)
		}
	}
	waitForPoolCondition(t, func() bool {
		_, capacity, _ := mgr.Stats()
		return capacity > cfg.MinSize
	})
	if err := mgr.Submit(func() { <-taskRelease }); err != nil {
		t.Fatal(err)
	}
	waitForPoolCondition(t, func() bool {
		_, capacity, _ := mgr.Stats()
		return capacity == cfg.MaxSize
	})

	// Release accepted work and require scaling back to the configured minimum. 释放已接收任务并要求缩容到配置的最小值。
	close(release)
	release = nil
	waitForPoolCondition(t, func() bool {
		running, capacity, usage := mgr.Stats()
		return running == 0 && capacity == cfg.MinSize && usage == 0
	})
}
