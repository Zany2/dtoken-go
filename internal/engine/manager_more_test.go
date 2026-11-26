package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/config"

	derror "github.com/Zany2/dtoken-go/errors"
)

// TestManagerAccessors verifies manager component and optional-module accessors. TestManagerAccessors 验证 Manager 组件与可选模块访问器。

// TestManagerLoginRenewAndTokenDetails verifies login renewal and token metadata queries. TestManagerLoginRenewAndTokenDetails 验证登录续期与 Token 元数据查询。
func TestManagerLoginRenewAndTokenDetails(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, func(cfg *config.Config) {
		cfg.Timeout = 30
		cfg.AutoRenew = false
	})

	token, err := mgr.Login(ctx, "detail-user", "web", "browser-1")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	device, err := mgr.GetDevice(ctx, token)
	if err != nil {
		t.Fatalf("GetDevice() error = %v", err)
	}
	if device != "web" {
		t.Fatalf("GetDevice() = %q, want web", device)
	}
	deviceID, err := mgr.GetDeviceID(ctx, token)
	if err != nil {
		t.Fatalf("GetDeviceID() error = %v", err)
	}
	if deviceID != "browser-1" {
		t.Fatalf("GetDeviceID() = %q, want browser-1", deviceID)
	}
	combinedDevice, combinedDeviceID, err := mgr.GetDeviceAndDeviceID(ctx, token)
	if err != nil {
		t.Fatalf("GetDeviceAndDeviceID() error = %v", err)
	}
	if combinedDevice != "web" || combinedDeviceID != "browser-1" {
		t.Fatalf("GetDeviceAndDeviceID() = %q/%q, want web/browser-1", combinedDevice, combinedDeviceID)
	}
	createdAt, err := mgr.GetTokenCreateTime(ctx, token)
	if err != nil {
		t.Fatalf("GetTokenCreateTime() error = %v", err)
	}
	if createdAt <= 0 {
		t.Fatalf("GetTokenCreateTime() = %d, want positive unix time", createdAt)
	}

	if err = mgr.RenewTimeout(ctx, token, 75*time.Second); err != nil {
		t.Fatalf("RenewTimeout() error = %v", err)
	}
	info, err := mgr.GetTokenInfo(ctx, token)
	if err != nil {
		t.Fatalf("GetTokenInfo() error = %v", err)
	}
	if info.Timeout != 75 {
		t.Fatalf("TokenInfo.Timeout = %d, want 75", info.Timeout)
	}
	ttl, err := mgr.GetTokenTTL(ctx, token)
	if err != nil {
		t.Fatalf("GetTokenTTL() error = %v", err)
	}
	if ttl <= 0 || ttl > 75 {
		t.Fatalf("GetTokenTTL() = %d, want 1..75", ttl)
	}

	if err = mgr.LoginByToken(ctx, token); err != nil {
		t.Fatalf("LoginByToken() error = %v", err)
	}
	if err = mgr.LoginByToken(ctx, ""); !errors.Is(err, derror.ErrInvalidToken) {
		t.Fatalf("LoginByToken(empty) error = %v, want ErrInvalidToken", err)
	}
	if err = mgr.RenewTimeout(ctx, "", time.Second); !errors.Is(err, derror.ErrInvalidToken) {
		t.Fatalf("RenewTimeout(empty) error = %v, want ErrInvalidToken", err)
	}
}

// TestManagerLogoutKickoutReplaceScopes verifies logout, kickout, and replacement across terminal scopes. TestManagerLogoutKickoutReplaceScopes 验证不同终端作用域下的登出、踢下线与顶替操作。
func TestManagerLogoutKickoutReplaceScopes(t *testing.T) {
	ctx := context.Background()

	t.Run("logout by device removes only that device", func(t *testing.T) {
		mgr := newTestManager(t, func(cfg *config.Config) {
			cfg.IsConcurrent = true
			cfg.IsShare = false
		})
		web, err := mgr.Login(ctx, "scope-logout", "web", "a")
		if err != nil {
			t.Fatalf("Login(web) error = %v", err)
		}
		mobile, err := mgr.Login(ctx, "scope-logout", "mobile", "a")
		if err != nil {
			t.Fatalf("Login(mobile) error = %v", err)
		}
		if err = mgr.LogoutByDevice(ctx, "scope-logout", "web"); err != nil {
			t.Fatalf("LogoutByDevice() error = %v", err)
		}
		if err = mgr.CheckLogin(ctx, web); !errors.Is(err, derror.ErrInvalidToken) {
			t.Fatalf("web CheckLogin() error = %v, want ErrInvalidToken", err)
		}
		if err = mgr.CheckLogin(ctx, mobile); err != nil {
			t.Fatalf("mobile CheckLogin() error = %v", err)
		}
	})

	t.Run("kickout and replace by login id preserve causes", func(t *testing.T) {
		kickMgr := newTestManager(t, nil)
		kickToken, err := kickMgr.Login(ctx, "scope-kick")
		if err != nil {
			t.Fatalf("Login(kick) error = %v", err)
		}
		if err = kickMgr.KickoutByLoginID(ctx, "scope-kick"); err != nil {
			t.Fatalf("KickoutByLoginID() error = %v", err)
		}
		if err = kickMgr.CheckLogin(ctx, kickToken); !errors.Is(err, derror.ErrTokenKickout) {
			t.Fatalf("kick token CheckLogin() error = %v, want ErrTokenKickout", err)
		}

		replaceMgr := newTestManager(t, nil)
		replaceToken, err := replaceMgr.Login(ctx, "scope-replace")
		if err != nil {
			t.Fatalf("Login(replace) error = %v", err)
		}
		if err = replaceMgr.ReplaceByLoginID(ctx, "scope-replace"); err != nil {
			t.Fatalf("ReplaceByLoginID() error = %v", err)
		}
		if err = replaceMgr.CheckLogin(ctx, replaceToken); !errors.Is(err, derror.ErrTokenReplaced) {
			t.Fatalf("replace token CheckLogin() error = %v, want ErrTokenReplaced", err)
		}
	})

	t.Run("invalid scoped arguments", func(t *testing.T) {
		mgr := newTestManager(t, nil)
		if err := mgr.LogoutByDevice(ctx, "", "web"); !errors.Is(err, derror.ErrIDIsEmpty) {
			t.Fatalf("LogoutByDevice(empty id) error = %v, want ErrIDIsEmpty", err)
		}
		if err := mgr.KickoutByDevice(ctx, "u", " "); !errors.Is(err, derror.ErrInvalidParam) {
			t.Fatalf("KickoutByDevice(empty device) error = %v, want ErrInvalidParam", err)
		}
		if err := mgr.ReplaceByDeviceAndDeviceID(ctx, "u", "web"); !errors.Is(err, derror.ErrInvalidParam) {
			t.Fatalf("ReplaceByDeviceAndDeviceID(missing id) error = %v, want ErrInvalidParam", err)
		}
	})
}

// TestManagerDisableBoundaries verifies account, service, and device disable boundaries. TestManagerDisableBoundaries 验证账号、服务与设备封禁边界。
func TestManagerDisableBoundaries(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, nil)

	if err := mgr.Disable(ctx, "", time.Minute); !errors.Is(err, derror.ErrIDIsEmpty) {
		t.Fatalf("Disable(empty id) error = %v, want ErrIDIsEmpty", err)
	}
	if err := mgr.Untie(ctx, ""); !errors.Is(err, derror.ErrIDIsEmpty) {
		t.Fatalf("Untie(empty id) error = %v, want ErrIDIsEmpty", err)
	}
	if mgr.IsDisable(ctx, "") {
		t.Fatal("IsDisable(empty id) = true, want false")
	}
	if _, err := mgr.GetDisableInfo(ctx, "missing"); !errors.Is(err, derror.ErrAccountNotDisabled) {
		t.Fatalf("GetDisableInfo(missing) error = %v, want ErrAccountNotDisabled", err)
	}
	if ttl, err := mgr.GetDisableTTL(ctx, "missing"); err != nil || ttl != -2 {
		t.Fatalf("GetDisableTTL(missing) = %d, %v, want -2, nil", ttl, err)
	}
	if err := mgr.CheckDisable(ctx, "missing"); err != nil {
		t.Fatalf("CheckDisable(missing) error = %v", err)
	}

	if err := mgr.DisableDevice(ctx, "device-user", " web ", time.Minute, "risk"); err != nil {
		t.Fatalf("DisableDevice() error = %v", err)
	}
	if !mgr.IsDisableDevice(ctx, "device-user", "web") {
		t.Fatal("IsDisableDevice() = false, want true")
	}
	if err := mgr.CheckDisableDevice(ctx, "device-user", "web"); !errors.Is(err, derror.ErrDeviceDisabled) {
		t.Fatalf("CheckDisableDevice() error = %v, want ErrDeviceDisabled", err)
	}
	info, err := mgr.GetDisableDeviceInfo(ctx, "device-user", "web")
	if err != nil {
		t.Fatalf("GetDisableDeviceInfo() error = %v", err)
	}
	if info.Device != "web" || info.DisableReason != "risk" {
		t.Fatalf("device disable info = %+v, want web risk", info)
	}
	ttl, err := mgr.GetDisableDeviceTTL(ctx, "device-user", "web")
	if err != nil {
		t.Fatalf("GetDisableDeviceTTL() error = %v", err)
	}
	if ttl <= 0 || ttl > 60 {
		t.Fatalf("GetDisableDeviceTTL() = %d, want 1..60", ttl)
	}
	if err = mgr.UntieDevice(ctx, "device-user", "web"); err != nil {
		t.Fatalf("UntieDevice() error = %v", err)
	}
	if _, err = mgr.GetDisableDeviceInfo(ctx, "device-user", "web"); !errors.Is(err, derror.ErrDeviceNotDisabled) {
		t.Fatalf("GetDisableDeviceInfo(after untie) error = %v, want ErrDeviceNotDisabled", err)
	}

	if err = mgr.DisableDeviceAndDeviceID(ctx, "device-user", "web", "a", time.Minute); err != nil {
		t.Fatalf("DisableDeviceAndDeviceID() error = %v", err)
	}
	if !mgr.IsDisableDeviceAndDeviceID(ctx, "device-user", "web", "a") {
		t.Fatal("IsDisableDeviceAndDeviceID() = false, want true")
	}
	if err = mgr.CheckDisableDeviceAndDeviceID(ctx, "device-user", "web", "a"); !errors.Is(err, derror.ErrDeviceDisabled) {
		t.Fatalf("CheckDisableDeviceAndDeviceID() error = %v, want ErrDeviceDisabled", err)
	}
	concreteInfo, err := mgr.GetDisableDeviceAndDeviceIDInfo(ctx, "device-user", "web", "a")
	if err != nil {
		t.Fatalf("GetDisableDeviceAndDeviceIDInfo() error = %v", err)
	}
	if concreteInfo.Device != "web" || concreteInfo.DeviceID != "a" {
		t.Fatalf("concrete device disable info = %+v, want web/a", concreteInfo)
	}
	if err = mgr.UntieDeviceAndDeviceID(ctx, "device-user", "web", "a"); err != nil {
		t.Fatalf("UntieDeviceAndDeviceID() error = %v", err)
	}
	if ttl, err = mgr.GetDisableDeviceAndDeviceIDTTL(ctx, "device-user", "web", "a"); err != nil || ttl != -2 {
		t.Fatalf("GetDisableDeviceAndDeviceIDTTL(after untie) = %d, %v, want -2, nil", ttl, err)
	}

	for name, testErr := range map[string]error{
		"DisableService empty service":          mgr.DisableService(ctx, "u", " ", time.Minute),
		"DisableServiceLevel empty service":     mgr.DisableServiceLevel(ctx, "u", " ", 1, time.Minute),
		"UntieService empty service":            mgr.UntieService(ctx, "u", " "),
		"CheckDisableService empty service":     mgr.CheckDisableService(ctx, "u", " "),
		"CheckDisableDevice empty device":       mgr.CheckDisableDevice(ctx, "u", " "),
		"CheckDisableConcrete empty device id":  mgr.CheckDisableDeviceAndDeviceID(ctx, "u", "web", " "),
		"GetDisableServiceInfo empty service":   func() error { _, err := mgr.GetDisableServiceInfo(ctx, "u", " "); return err }(),
		"GetDisableDeviceInfo empty device":     func() error { _, err := mgr.GetDisableDeviceInfo(ctx, "u", " "); return err }(),
		"GetDisableConcreteInfo empty device":   func() error { _, err := mgr.GetDisableDeviceAndDeviceIDInfo(ctx, "u", " ", "a"); return err }(),
		"GetDisableServiceTTL empty service":    func() error { _, err := mgr.GetDisableServiceTTL(ctx, "u", " "); return err }(),
		"GetDisableDeviceTTL empty device":      func() error { _, err := mgr.GetDisableDeviceTTL(ctx, "u", " "); return err }(),
		"GetDisableConcreteTTL empty device id": func() error { _, err := mgr.GetDisableDeviceAndDeviceIDTTL(ctx, "u", "web", " "); return err }(),
	} {
		if !errors.Is(testErr, derror.ErrInvalidParam) {
			t.Fatalf("%s error = %v, want ErrInvalidParam", name, testErr)
		}
	}
}

// TestManagerOptionalModulesDisabled verifies disabled optional modules fail explicitly. TestManagerOptionalModulesDisabled 验证未启用的可选模块会明确失败。

// TestManagerNonceFacade verifies nonce generation, validation, consumption, and TTL access. TestManagerNonceFacade 验证 Nonce 生成、校验、消费与 TTL 查询。
