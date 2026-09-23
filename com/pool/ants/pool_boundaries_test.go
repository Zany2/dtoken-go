package ants

import (
	"errors"
	"math"
	"strconv"
	"testing"
	"time"

	antslib "github.com/panjf2000/ants/v2"
)

// TestRenewPoolCapacityBoundaries rejects capacities that overflow ants' internal storage. TestRenewPoolCapacityBoundaries 拒绝超出 ants 内部存储范围的容量。
func TestRenewPoolCapacityBoundaries(t *testing.T) {
	cfg := DefaultRenewPoolConfig().SetMinSize(math.MaxInt32).SetMaxSize(math.MaxInt32)
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate(MaxInt32) = %v", err)
	}
	if strconv.IntSize == 32 {
		return
	}

	tooLarge := int64(math.MaxInt32) + 1
	for _, minSize := range []int{1, int(tooLarge)} {
		cfg.SetMinSize(minSize).SetMaxSize(int(tooLarge))
		if mgr, err := NewRenewPoolManagerWithConfig(cfg); err == nil {
			mgr.Stop()
			t.Fatalf("constructor accepted capacity outside int32 range: %+v", cfg)
		}
	}
}

// TestRenewPoolIdleWorkersDoNotCountAsLoad verifies idle cached workers neither inflate stats nor prevent shrinking. TestRenewPoolIdleWorkersDoNotCountAsLoad 验证缓存的空闲协程不会虚增负载或阻止缩容。
func TestRenewPoolIdleWorkersDoNotCountAsLoad(t *testing.T) {
	cfg := DefaultRenewPoolConfig().SetMinSize(1).SetMaxSize(2).
		SetCheckInterval(5 * time.Millisecond).SetExpiry(time.Hour)
	mgr, err := NewRenewPoolManagerWithConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer mgr.Stop()
	release := make(chan struct{})
	defer func() {
		if release != nil {
			close(release)
		}
	}()
	started := make(chan struct{})
	// Capture a stable channel before clearing the cleanup reference. 清理引用置空前捕获固定通道。
	taskRelease := release
	if err := mgr.Submit(func() {
		close(started)
		<-taskRelease
	}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("task did not start")
	}
	waitForPoolCondition(t, func() bool {
		running, capacity, usage := mgr.Stats()
		return running == 1 && capacity == 2 && usage == 0.5
	})
	close(release)
	release = nil

	waitForPoolCondition(t, func() bool {
		running, capacity, usage := mgr.Stats()
		return running == 0 && capacity == 1 && usage == 0
	})
	if mgr.pool.Running() == 0 {
		t.Fatal("expected an idle cached worker before expiry")
	}
}

// TestRenewPoolNonBlockingNestedSubmit verifies saturation returns an error instead of blocking a Manager-style nested event. TestRenewPoolNonBlockingNestedSubmit 验证饱和时返回错误，而非阻塞 Manager 风格的嵌套事件提交。
func TestRenewPoolNonBlockingNestedSubmit(t *testing.T) {
	cfg := DefaultRenewPoolConfig().SetMinSize(1).SetMaxSize(1)
	mgr, err := NewRenewPoolManagerWithConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer mgr.Stop()
	if err := mgr.Submit(nil); err == nil {
		t.Fatal("Submit(nil) must reject the task")
	}
	result := make(chan error, 1)
	if err := mgr.Submit(func() {
		result <- mgr.Submit(func() {})
	}); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-result:
		if !errors.Is(err, antslib.ErrPoolOverload) {
			t.Fatalf("nested Submit() = %v, want ErrPoolOverload", err)
		}
	case <-time.After(time.Second):
		t.Fatal("nested Submit blocked")
	}
}

// TestRenewPoolStopUnblocksSubmit verifies shutdown rejects waiting work and lets accepted work finish. TestRenewPoolStopUnblocksSubmit 验证关闭时拒绝等待中的任务，并允许已接收任务完成。
func TestRenewPoolStopUnblocksSubmit(t *testing.T) {
	cfg := DefaultRenewPoolConfig().SetMinSize(1).SetMaxSize(1).SetNonBlocking(false)
	mgr, err := NewRenewPoolManagerWithConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer mgr.Stop()
	release := make(chan struct{})
	defer func() {
		if release != nil {
			close(release)
		}
	}()
	taskRelease := release
	if err := mgr.Submit(func() { <-taskRelease }); err != nil {
		t.Fatal(err)
	}
	waitForPoolCondition(t, func() bool {
		running, _, _ := mgr.Stats()
		return running == 1
	})
	submitted := make(chan error, 1)
	rejectedRan := make(chan struct{}, 1)
	go func() {
		submitted <- mgr.Submit(func() { rejectedRan <- struct{}{} })
	}()
	waitForPoolCondition(t, func() bool { return mgr.pool.Waiting() == 1 })
	stopped := make(chan struct{})
	go func() {
		mgr.Stop()
		close(stopped)
	}()
	select {
	case err := <-submitted:
		if !errors.Is(err, antslib.ErrPoolClosed) {
			t.Fatalf("waiting Submit() = %v, want ErrPoolClosed", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Stop did not unblock Submit")
	}
	select {
	case <-stopped:
		t.Fatal("Stop returned before the active task finished")
	default:
	}
	close(release)
	release = nil
	select {
	case <-stopped:
	case <-time.After(DefaultStopTimeout + time.Second):
		t.Fatal("Stop did not finish")
	}
	select {
	case <-rejectedRan:
		t.Fatal("rejected task executed")
	default:
	}
	if running, _, usage := mgr.Stats(); running != 0 || usage != 0 {
		t.Fatalf("Stats after Stop = (%d, %f), want zero load", running, usage)
	}
}

// TestRenewPoolPanicReleasesLoad verifies panic recovery cleans up load and allows subsequent work. TestRenewPoolPanicReleasesLoad 验证 panic 恢复会清理负载并允许后续任务运行。
func TestRenewPoolPanicReleasesLoad(t *testing.T) {
	mgr, err := NewRenewPoolManagerWithConfig(DefaultRenewPoolConfig().SetMinSize(1).SetMaxSize(1))
	if err != nil {
		t.Fatal(err)
	}
	defer mgr.Stop()
	panicked := make(chan struct{})
	if err := mgr.Submit(func() {
		defer close(panicked)
		panic("test task panic")
	}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-panicked:
	case <-time.After(time.Second):
		t.Fatal("panic task did not run")
	}
	waitForPoolCondition(t, func() bool {
		running, _, usage := mgr.Stats()
		return running == 0 && usage == 0 && mgr.pool.Running() == 0
	})
	done := make(chan struct{})
	if err := mgr.Submit(func() { close(done) }); err != nil {
		t.Fatalf("Submit after panic = %v", err)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("pool did not execute work after panic")
	}
}

// waitForPoolCondition bounds asynchronous lifecycle assertions. waitForPoolCondition 为异步生命周期断言设置等待上限。
func waitForPoolCondition(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatal("pool condition was not reached")
		}
		time.Sleep(time.Millisecond)
	}
}
