package dtoken

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/manager"
)

// TestFacadeRejectsClosedManager verifies external closure invalidates existing facades. TestFacadeRejectsClosedManager 验证外部关闭后已有门面不再使用旧管理器。
func TestFacadeRejectsClosedManager(t *testing.T) {
	for _, scenario := range []string{"direct", "replace", "delete", "shared-auth"} {
		t.Run(scenario, func(t *testing.T) {
			DeleteAllManager()
			t.Cleanup(DeleteAllManager)
			mgr := newRegistryTestManager("closed-facade", &registryTestPool{})
			SetManager(mgr)
			auth := New(mgr)
			t.Cleanup(auth.Close)
			if got, err := auth.requireManager(); err != nil || got != mgr {
				t.Fatalf("open requireManager() = %v, %v", got, err)
			}

			var replacement *manager.Manager
			switch scenario {
			case "direct":
				mgr.CloseManager()
			case "replace":
				replacement = newRegistryTestManager("closed-facade", &registryTestPool{})
				SetManager(replacement)
			case "delete":
				if err := DeleteManager("closed-facade"); err != nil {
					t.Fatal(err)
				}
			case "shared-auth":
				New(mgr).Close()
			}

			ctx := context.Background()
			if got, err := auth.requireManager(); got != nil || !errors.Is(err, derror.ErrManagerNotFound) {
				t.Fatalf("closed requireManager() = %v, %v", got, err)
			}
			if token, err := auth.LoginID(ctx, "user"); token != "" || !errors.Is(err, derror.ErrManagerNotFound) {
				t.Fatalf("closed LoginID() = %q, %v", token, err)
			}
			if auth.IsLogin(ctx, "token") {
				t.Fatal("closed IsLogin() = true")
			}
			if auth.EventManager() != nil {
				t.Fatal("closed EventManager() must be nil")
			}

			if replacement != nil {
				auth.Close()
				if got, err := GetManager("closed-facade"); err != nil || got != replacement || got.IsClosed() {
					t.Fatalf("replacement GetManager() = %v, %v", got, err)
				}
				return
			}
			if got, err := GetManager("closed-facade"); got != nil || !errors.Is(err, derror.ErrManagerNotFound) {
				t.Fatalf("closed GetManager() = %v, %v", got, err)
			}
			if got, err := NewByAuthType("closed-facade"); got != nil || !errors.Is(err, derror.ErrManagerNotFound) {
				t.Fatalf("closed NewByAuthType() = %v, %v", got, err)
			}
			if got, err := GetEventManager("closed-facade"); got != nil || !errors.Is(err, derror.ErrManagerNotFound) {
				t.Fatalf("closed GetEventManager() = %v, %v", got, err)
			}
			if err := CheckLogin(ctx, "token", "closed-facade"); !errors.Is(err, derror.ErrManagerNotFound) {
				t.Fatalf("closed CheckLogin() error = %v", err)
			}
		})
	}
}

// TestDisableFacadePreservesValidation verifies wrappers never broaden invalid disable requests. TestDisableFacadePreservesValidation 验证门面不会扩大非法封禁请求的范围。
func TestDisableFacadePreservesValidation(t *testing.T) {
	DeleteAllManager()
	t.Cleanup(DeleteAllManager)
	mgr, err := NewBuilder().IsPrintBanner(false).AutoRenew(false).AuthType("disable-validation").Build()
	if err != nil {
		t.Fatal(err)
	}
	SetManager(mgr)
	auth := New(mgr)
	ctx := context.Background()

	serviceCalls := []struct {
		name string
		call func(int) error
	}{
		{"instance-options", func(level int) error {
			return auth.DisableService(ctx, ServiceDisableOptions{LoginID: "user", Service: "service", Level: level, Duration: time.Minute})
		}},
		{"global-options", func(level int) error {
			return DisableServiceWithOptions(ctx, ServiceDisableOptions{AuthType: "disable-validation", LoginID: "user", Service: "service", Level: level, Duration: time.Minute})
		}},
		{"instance-level", func(level int) error {
			return auth.DisableServiceLevel(ctx, "user", "service", level, time.Minute)
		}},
		{"instance-level-reason", func(level int) error {
			return auth.DisableServiceLevelWithReason(ctx, "user", "service", level, time.Minute, "reason")
		}},
	}
	for _, tc := range serviceCalls {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.call(-1); !errors.Is(err, derror.ErrInvalidParam) {
				t.Fatalf("negative level error = %v, want ErrInvalidParam", err)
			}
			if info, err := auth.GetDisableServiceInfo(ctx, "user", "service"); !errors.Is(err, derror.ErrServiceNotDisabled) || info != nil {
				t.Fatalf("invalid request created service disable: %v, %v", info, err)
			}
			for _, level := range []int{0, 2} {
				if err := tc.call(level); err != nil {
					t.Fatalf("level %d error = %v", level, err)
				}
				if info, err := auth.GetDisableServiceInfo(ctx, "user", "service"); err != nil || info == nil || info.Level != level {
					t.Fatalf("level %d info = %v, %v", level, info, err)
				}
			}
			if err := auth.UntieService(ctx, "user", "service"); err != nil {
				t.Fatal(err)
			}
		})
	}

	for _, withReason := range []bool{false, true} {
		for _, deviceID := range []string{"", "   "} {
			var err error
			if withReason {
				err = auth.DisableDeviceAndDeviceIDWithReason(ctx, "user", "web", deviceID, time.Minute, "reason")
			} else {
				err = auth.DisableDeviceAndDeviceID(ctx, "user", "web", deviceID, time.Minute)
			}
			if !errors.Is(err, derror.ErrInvalidParam) {
				t.Fatalf("deviceID %q, withReason %v: error = %v, want ErrInvalidParam", deviceID, withReason, err)
			}
			if info, err := auth.GetDisableDeviceInfo(ctx, "user", "web"); !errors.Is(err, derror.ErrDeviceNotDisabled) || info != nil {
				t.Fatalf("invalid request disabled device type: %v, %v", info, err)
			}
		}
	}
}
