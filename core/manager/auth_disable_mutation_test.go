package manager

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/listener"
)

// TestUntieEventsRequireExistingDisableState verifies no-op unties do not emit lifecycle events. TestUntieEventsRequireExistingDisableState 验证无效解禁不会触发生命周期事件。
func TestUntieEventsRequireExistingDisableState(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		event   listener.Event
		disable func(*Manager) error
		untie   func(*Manager) error
	}{
		{
			name:  "account",
			event: listener.EventUntie,
			disable: func(mgr *Manager) error {
				return mgr.Disable(ctx, "untie-account", time.Minute)
			},
			untie: func(mgr *Manager) error {
				return mgr.Untie(ctx, "untie-account")
			},
		},
		{
			name:  "service",
			event: listener.EventUntieService,
			disable: func(mgr *Manager) error {
				return mgr.DisableService(ctx, "untie-service", "billing", time.Minute)
			},
			untie: func(mgr *Manager) error {
				return mgr.UntieService(ctx, "untie-service", "billing")
			},
		},
		{
			name:  "legacy service marker",
			event: listener.EventUntieService,
			disable: func(mgr *Manager) error {
				return mgr.saveToStorage(
					ctx,
					mgr.getLegacyDisableServiceKey("untie-legacy-service", "billing"),
					ServiceDisableInfo{Service: "billing"},
					time.Minute,
				)
			},
			untie: func(mgr *Manager) error {
				return mgr.UntieService(ctx, "untie-legacy-service", "billing")
			},
		},
		{
			name:  "device type",
			event: listener.EventUntieDevice,
			disable: func(mgr *Manager) error {
				return mgr.DisableDevice(ctx, "untie-device", "web", time.Minute)
			},
			untie: func(mgr *Manager) error {
				return mgr.UntieDevice(ctx, "untie-device", "web")
			},
		},
		{
			name:  "concrete device",
			event: listener.EventUntieDevice,
			disable: func(mgr *Manager) error {
				return mgr.DisableDeviceAndDeviceID(ctx, "untie-concrete-device", "web", "browser", time.Minute)
			},
			untie: func(mgr *Manager) error {
				return mgr.UntieDeviceAndDeviceID(ctx, "untie-concrete-device", "web", "browser")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := newTestManager(t, nil)
			eventCount := 0
			mgr.GetEventManager().RegisterFuncWithConfig(tt.event, func(*listener.EventData) {
				eventCount++
			}, listener.ListenerConfig{Async: false})

			if err := tt.untie(mgr); err != nil {
				t.Fatalf("initial untie error = %v", err)
			}
			if eventCount != 0 {
				t.Fatalf("initial no-op untie event count = %d, want 0", eventCount)
			}
			if err := tt.disable(mgr); err != nil {
				t.Fatalf("disable error = %v", err)
			}
			if err := tt.untie(mgr); err != nil {
				t.Fatalf("effective untie error = %v", err)
			}
			if err := tt.untie(mgr); err != nil {
				t.Fatalf("repeated untie error = %v", err)
			}
			if eventCount != 1 {
				t.Fatalf("untie event count = %d, want 1", eventCount)
			}
		})
	}
}

// TestUntieFallbackReportsDeleteFailureBeforeEvent verifies ordinary storage preserves errors and event ordering. TestUntieFallbackReportsDeleteFailureBeforeEvent 验证普通存储会保留删除错误和事件顺序。
func TestUntieFallbackReportsDeleteFailureBeforeEvent(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, nil)
	if err := mgr.DisableService(ctx, "untie-fallback", "billing", time.Minute); err != nil {
		t.Fatalf("DisableService() error = %v", err)
	}

	deleteErr := errors.New("delete failed")
	storage := &managerFailingStorage{Storage: mgr.storage, deleteErr: deleteErr}
	mgr.storage = storage
	eventCount := 0
	mgr.GetEventManager().RegisterFuncWithConfig(listener.EventUntieService, func(*listener.EventData) {
		eventCount++
	}, listener.ListenerConfig{Async: false})

	if err := mgr.UntieService(ctx, "untie-fallback", "billing"); !errors.Is(err, derror.ErrStorageUnavailable) {
		t.Fatalf("UntieService(delete failure) error = %v, want ErrStorageUnavailable", err)
	}
	if eventCount != 0 {
		t.Fatalf("failed untie event count = %d, want 0", eventCount)
	}

	storage.deleteErr = nil
	if err := mgr.UntieService(ctx, "untie-fallback", "billing"); err != nil {
		t.Fatalf("UntieService(fallback success) error = %v", err)
	}
	if err := mgr.UntieService(ctx, "untie-fallback", "billing"); err != nil {
		t.Fatalf("UntieService(fallback no-op) error = %v", err)
	}
	if eventCount != 1 {
		t.Fatalf("fallback untie event count = %d, want 1", eventCount)
	}
}
