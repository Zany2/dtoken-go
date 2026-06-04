// @Author daixk 2025/12/22 15:56:00
package ants

import (
	"sync"
	"sync/atomic"
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

// TestRenewPoolManagerAutoScale verifies scale up and scale down behavior TestRenewPoolManagerAutoScale 验证自动扩缩容行为。
func TestRenewPoolManagerAutoScale(t *testing.T) {
	const (
		minSize    = 2
		maxSize    = 64
		maxRunTime = 30 * time.Minute
	)

	mgr, err := NewRenewPoolManagerWithConfig(&RenewPoolConfig{
		MinSize:       minSize,
		MaxSize:       maxSize,
		ScaleUpRate:   0.8,
		ScaleDownRate: 0.3,
		CheckInterval: time.Second,
		Expiry:        time.Second,
		NonBlocking:   true,
	})
	if err != nil {
		t.Fatalf("NewRenewPoolManagerWithConfig() error = %v", err)
	}
	defer mgr.Stop()

	stopTasks := make(chan struct{})
	defer close(stopTasks)

	stopLogging := make(chan struct{})
	logDone := make(chan struct{})
	go logPoolStats(t, mgr, time.Second, stopLogging, logDone)
	defer func() {
		close(stopLogging)
		<-logDone
	}()

	deadline := time.Now().Add(maxRunTime)
	for cycle := 1; time.Now().Before(deadline); cycle++ {
		var started atomic.Int64
		var completed atomic.Int64
		phaseStarted := time.Now()
		reachedMax := false

		t.Logf("cycle %d high-load phase: slowly increasing variable-duration tasks", cycle)
		for pressure := 1; time.Now().Before(deadline); pressure++ {
			running, capacity, usage := mgr.Stats()
			submits := capacity + pressure
			if submits < minSize {
				submits = minSize
			}

			for i := 0; i < submits; i++ {
				duration := simulatedRenewTaskDuration(started.Load() + int64(i) + 1)
				_ = mgr.Submit(func() {
					started.Add(1)
					select {
					case <-time.After(duration):
					case <-stopTasks:
					}
					completed.Add(1)
				})
			}

			t.Logf(
				"cycle %d pressure step: pressure=%d submitted=%d started=%d completed=%d running=%d capacity=%d usage=%.2f%%",
				cycle,
				pressure,
				submits,
				started.Load(),
				completed.Load(),
				running,
				capacity,
				usage*100,
			)

			if capacity >= maxSize {
				reachedMax = true
			}
			if reachedMax && time.Since(phaseStarted) >= 15*time.Second {
				break
			}
			if !sleepUntilDeadline(500*time.Millisecond, deadline) {
				break
			}
		}

		t.Logf("cycle %d idle phase: stop submitting and wait for tasks to finish naturally, started=%d completed=%d", cycle, started.Load(), completed.Load())
		for time.Now().Before(deadline) {
			running, capacity, usage := mgr.Stats()
			if running == 0 && capacity <= minSize {
				break
			}

			t.Logf(
				"cycle %d idle wait: started=%d completed=%d running=%d capacity=%d usage=%.2f%%",
				cycle,
				started.Load(),
				completed.Load(),
				running,
				capacity,
				usage*100,
			)
			if !sleepUntilDeadline(2*time.Second, deadline) {
				break
			}
		}

		t.Logf("cycle %d idle hold: capacity returned to min", cycle)
		sleepUntilDeadline(5*time.Second, deadline)
	}

	t.Logf("auto scale observation finished after max run time: %s", maxRunTime)
}

func simulatedRenewTaskDuration(sequence int64) time.Duration {
	durations := []time.Duration{
		200 * time.Millisecond,
		350 * time.Millisecond,
		500 * time.Millisecond,
		800 * time.Millisecond,
		1200 * time.Millisecond,
		2 * time.Second,
	}
	return durations[(sequence-1)%int64(len(durations))]
}

func logPoolStats(t *testing.T, mgr *RenewPoolManager, interval time.Duration, stop <-chan struct{}, done chan<- struct{}) {
	defer close(done)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			running, capacity, usage := mgr.Stats()
			t.Logf("renew pool status: running=%d capacity=%d usage=%.2f%%", running, capacity, usage*100)
		case <-stop:
			running, capacity, usage := mgr.Stats()
			t.Logf("renew pool final status: running=%d capacity=%d usage=%.2f%%", running, capacity, usage*100)
			return
		}
	}
}

func sleepUntilDeadline(duration time.Duration, deadline time.Time) bool {
	remaining := time.Until(deadline)
	if remaining <= 0 {
		return false
	}
	if remaining < duration {
		time.Sleep(remaining)
		return false
	}
	time.Sleep(duration)
	return true
}
