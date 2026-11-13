package engine

import (
	"testing"
	"time"
)

// TestManagerBackgroundIsolation verifies instance-owned status tasks close independently. TestManagerBackgroundIsolation 验证实例自有状态任务独立关闭。
func TestManagerBackgroundIsolation(t *testing.T) {
	first := newTestManagerWithRuntime(t, &managerLifecycleTestPool{}, &managerLifecycleTestLogger{})
	second := newTestManagerWithRuntime(t, &managerLifecycleTestPool{}, &managerLifecycleTestLogger{})
	t.Cleanup(first.CloseManager)
	t.Cleanup(second.CloseManager)

	// Start long-lived tasks without relying on ticker timing. 启动长期任务，避免依赖定时器触发时序。
	first.StartRenewPoolStatusLogger(time.Hour)
	second.StartRenewPoolStatusLogger(time.Hour)
	firstBackground, secondBackground := first.background, second.background
	if firstBackground == nil || secondBackground == nil || firstBackground == secondBackground {
		t.Fatal("managers must own distinct background state")
	}
	first.StartRenewPoolStatusLogger(time.Hour)
	if first.background != firstBackground {
		t.Fatal("repeated startup replaced background state")
	}

	// Closing one instance must preserve the other instance's worker. 关闭一个实例时保留另一实例的任务。
	first.CloseManager()
	select {
	case <-firstBackground.closeCh:
	default:
		t.Fatal("closed manager background was not stopped")
	}
	select {
	case <-secondBackground.closeCh:
		t.Fatal("closing one manager stopped another manager")
	default:
	}
	if first.background != nil || second.background != secondBackground {
		t.Fatal("background ownership changed unexpectedly")
	}

	// Closed managers cannot recreate their background tasks. 已关闭的管理器拒绝重建后台任务。
	first.StartRenewPoolStatusLogger(time.Hour)
	if first.background != nil {
		t.Fatal("closed manager recreated background state")
	}
}
