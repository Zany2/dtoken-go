package dtoken

import (
	"context"
	"testing"
	"time"
)

// TestDefaultMemoryDisableScopes checks account, business and device restrictions through default APIs. TestDefaultMemoryDisableScopes 通过默认 API 验证账号、业务与设备封禁。
func TestDefaultMemoryDisableScopes(t *testing.T) {
	DeleteAllManager()
	t.Cleanup(CloseAll)
	_, err := NewBuilder().IsPrintBanner(false).IsLog(false).AutoRenew(false).IsShare(false).Build()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	t.Run("account", func(t *testing.T) {
		token, err := LoginID(ctx, "account")
		if err != nil {
			t.Fatal(err)
		}
		if err = Disable(ctx, DisableOptions{LoginID: "account", Duration: time.Minute, Reason: "review"}); err != nil {
			t.Fatal(err)
		}
		if !IsDisable(ctx, "account") || CheckDisable(ctx, "account") == nil || IsLogin(ctx, token) {
			t.Fatal("account restriction failed")
		}
		if info, err := GetDisableInfo(ctx, "account"); err != nil || info == nil || info.DisableReason != "review" {
			t.Fatalf("disable info=%+v,%v", info, err)
		}
		if ttl, err := GetDisableTTL(ctx, "account"); err != nil || ttl <= 0 || ttl > 60 {
			t.Fatalf("disable ttl=%d,%v", ttl, err)
		}
		if _, err = LoginID(ctx, "account"); err == nil {
			t.Fatal("disabled account logged in")
		}
		if err = Untie(ctx, "account"); err != nil {
			t.Fatal(err)
		}
		if IsDisable(ctx, "account") || CheckDisable(ctx, "account") != nil || IsLogin(ctx, token) {
			t.Fatal("untie restored an old token or retained restriction")
		}
		if _, err = LoginID(ctx, "account"); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("service", func(t *testing.T) {
		token, err := LoginID(ctx, "service")
		if err != nil {
			t.Fatal(err)
		}
		for _, disable := range []func() error{
			func() error {
				return DisableService(ctx, ServiceDisableOptions{LoginID: "service", Service: "publish", Level: 2, Duration: time.Minute, Reason: "review"})
			},
			func() error { return DisableServiceWithReason(ctx, "service", "publish", time.Minute, "review") },
			func() error { return DisableServiceLevel(ctx, "service", "publish", 2, time.Minute) },
			func() error {
				return DisableServiceLevelWithReason(ctx, "service", "publish", 2, time.Minute, "review")
			},
		} {
			if err = disable(); err != nil {
				t.Fatal(err)
			}
			if !IsDisableService(ctx, "service", "publish") || !IsDisableServiceLevel(ctx, "service", "publish", 0) || CheckDisableService(ctx, "service", "publish") == nil || CheckDisableServiceLevel(ctx, "service", "publish", 0) == nil {
				t.Fatal("service restriction missing")
			}
			if IsDisableServiceLevel(ctx, "service", "publish", 3) || CheckDisableServiceLevel(ctx, "service", "publish", 3) != nil || CheckDisableService(ctx, "service", "read") != nil || !IsLogin(ctx, token) {
				t.Fatal("service restriction escaped its scope")
			}
			if info, err := GetDisableServiceInfo(ctx, "service", "publish"); err != nil || info == nil {
				t.Fatalf("service info=%+v,%v", info, err)
			}
			if ttl, err := GetDisableServiceTTL(ctx, "service", "publish"); err != nil || ttl <= 0 || ttl > 60 {
				t.Fatalf("service ttl=%d,%v", ttl, err)
			}
			if err = UntieService(ctx, "service", "publish"); err != nil {
				t.Fatal(err)
			}
			if IsDisableService(ctx, "service", "publish") || CheckDisableService(ctx, "service", "publish") != nil {
				t.Fatal("service untie failed")
			}
		}
	})
	t.Run("device", func(t *testing.T) {
		for _, tc := range []struct {
			disable  func() error
			concrete bool
		}{
			{func() error {
				return DisableDevice(ctx, DeviceDisableOptions{LoginID: "device", Device: "web", Duration: time.Minute})
			}, false},
			{func() error { return DisableDeviceWithReason(ctx, "device", "web", time.Minute, "review") }, false},
			{func() error { return DisableDeviceAndDeviceID(ctx, "device", "web", "one", time.Minute) }, true},
			{func() error {
				return DisableDeviceAndDeviceIDWithReason(ctx, "device", "web", "one", time.Minute, "review")
			}, true},
		} {
			tokens := []string{}
			for _, opts := range []LoginOptions{{LoginID: "device", Device: "web", DeviceID: "one"}, {LoginID: "device", Device: "web", DeviceID: "two"}, {LoginID: "device", Device: "phone"}} {
				token, err := Login(ctx, opts)
				if err != nil {
					t.Fatal(err)
				}
				tokens = append(tokens, token)
			}
			if err = tc.disable(); err != nil {
				t.Fatal(err)
			}
			if !IsLogin(ctx, tokens[0]) || !IsLogin(ctx, tokens[1]) || !IsLogin(ctx, tokens[2]) {
				t.Fatal("reversible device restriction retired credentials")
			}
			if IsDisableDeviceAndDeviceID(ctx, "device", "web", "two") == tc.concrete || IsDisableDevice(ctx, "device", "phone") {
				t.Fatal("device restriction escaped its scope")
			}
			if _, err = Login(ctx, LoginOptions{LoginID: "device", Device: "web", DeviceID: "one"}); err == nil {
				t.Fatal("disabled device logged in")
			}
			if tc.concrete {
				if !IsDisableDeviceAndDeviceID(ctx, "device", "web", "one") || CheckDisableDeviceAndDeviceID(ctx, "device", "web", "one") == nil {
					t.Fatal("terminal restriction missing")
				}
				if info, err := GetDisableDeviceAndDeviceIDInfo(ctx, "device", "web", "one"); err != nil || info == nil {
					t.Fatalf("terminal info=%+v,%v", info, err)
				}
				if ttl, err := GetDisableDeviceAndDeviceIDTTL(ctx, "device", "web", "one"); err != nil || ttl <= 0 || ttl > 60 {
					t.Fatalf("terminal ttl=%d,%v", ttl, err)
				}
				if err = UntieDeviceAndDeviceID(ctx, "device", "web", "one"); err != nil {
					t.Fatal(err)
				}
				if IsDisableDeviceAndDeviceID(ctx, "device", "web", "one") || CheckDisableDeviceAndDeviceID(ctx, "device", "web", "one") != nil {
					t.Fatal("terminal untie failed")
				}
			} else {
				if !IsDisableDevice(ctx, "device", "web") || CheckDisableDevice(ctx, "device", "web") == nil {
					t.Fatal("device restriction missing")
				}
				if info, err := GetDisableDeviceInfo(ctx, "device", "web"); err != nil || info == nil {
					t.Fatalf("device info=%+v,%v", info, err)
				}
				if ttl, err := GetDisableDeviceTTL(ctx, "device", "web"); err != nil || ttl <= 0 || ttl > 60 {
					t.Fatalf("device ttl=%d,%v", ttl, err)
				}
				if err = UntieDevice(ctx, "device", "web"); err != nil {
					t.Fatal(err)
				}
				if IsDisableDevice(ctx, "device", "web") || CheckDisableDevice(ctx, "device", "web") != nil {
					t.Fatal("device untie failed")
				}
			}
			if !IsLogin(ctx, tokens[0]) {
				t.Fatal("untie lost a preserved token")
			}
			if err = LogoutByLoginID(ctx, "device"); err != nil {
				t.Fatal(err)
			}
		}
	})
}
