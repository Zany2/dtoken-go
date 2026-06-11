package dtoken

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/oauth2"
)

// TestRegistryHelpersAndBuildAndSetManager verifies auth type parsing and build registration. TestRegistryHelpersAndBuildAndSetManager 验证认证类型解析和构建注册。
func TestRegistryHelpersAndBuildAndSetManager(t *testing.T) {
	DeleteAllManager()
	t.Cleanup(DeleteAllManager)

	if got := getAutoType(" admin "); got != "admin:" {
		t.Fatalf("getAutoType(admin) = %q, want admin:", got)
	}
	if got := getAutoType(""); got != config.DefaultAuthType {
		t.Fatalf("getAutoType(empty) = %q, want %q", got, config.DefaultAuthType)
	}
	if got := resolveAuthType("option", "override"); got != "override" {
		t.Fatalf("resolveAuthType() = %q, want override", got)
	}
	if got := resolveAuthType("option"); got != "option" {
		t.Fatalf("resolveAuthType(option) = %q, want option", got)
	}
	device, deviceID, authType := parseDeviceAndAuthType("web", "browser", "admin")
	if device != "web" || deviceID != "browser" || authType != "admin" {
		t.Fatalf("parseDeviceAndAuthType() = %q, %q, %q", device, deviceID, authType)
	}

	mgr, err := BuildAndSetManager(NewBuilder().IsPrintBanner(false).AutoRenew(false), "built")
	if err != nil {
		t.Fatalf("BuildAndSetManager() error = %v", err)
	}
	if mgr.GetConfig().AuthType != "built:" {
		t.Fatalf("AuthType = %q, want built:", mgr.GetConfig().AuthType)
	}
	if got, err := GetManager("built"); err != nil || got != mgr {
		t.Fatalf("GetManager(built) = %v, %v, want built manager", got, err)
	}
	if events, err := GetEventManager("built"); err != nil || events != mgr.GetEventManager() {
		t.Fatalf("GetEventManager(built) = %v, %v, want manager event manager", events, err)
	}
}

// TestFacadeReturnsManagerNotFoundWithoutRegisteredManager verifies global helpers fail clearly without a manager. TestFacadeReturnsManagerNotFoundWithoutRegisteredManager 验证未注册管理器时全局方法返回明确错误。
func TestFacadeReturnsManagerNotFoundWithoutRegisteredManager(t *testing.T) {
	DeleteAllManager()
	t.Cleanup(DeleteAllManager)

	ctx := context.Background()
	errChecks := map[string]error{
		"Default":              func() error { _, err := Default(); return err }(),
		"NewByAuthType":        func() error { _, err := NewByAuthType("missing"); return err }(),
		"GetEventManager":      func() error { _, err := GetEventManager(); return err }(),
		"Login":                func() error { _, err := Login(ctx, "u"); return err }(),
		"CheckLogin":           CheckLogin(ctx, "token"),
		"AddPermissions":       AddPermissions(ctx, "u", []string{"p"}),
		"GetPermissions":       func() error { _, err := GetPermissions(ctx, "u"); return err }(),
		"AddRoles":             AddRoles(ctx, "u", []string{"r"}),
		"GetRoles":             func() error { _, err := GetRoles(ctx, "u"); return err }(),
		"Disable":              Disable(ctx, "u", time.Minute, "risk"),
		"GetDisableInfo":       func() error { _, err := GetDisableInfo(ctx, "u"); return err }(),
		"GetSession":           func() error { _, err := GetSession(ctx, "u"); return err }(),
		"GenerateNonce":        func() error { _, err := GenerateNonce(ctx); return err }(),
		"CreateTicket":         func() error { _, err := CreateTicket(ctx, "u"); return err }(),
		"CreateShortKey":       func() error { _, err := CreateShortKey(ctx); return err }(),
		"RegisterOAuth2Client": RegisterOAuth2Client(&oauth2.Client{ClientID: "client"}),
	}
	for name, err := range errChecks {
		if !errors.Is(err, derror.ErrManagerNotFound) {
			t.Fatalf("%s error = %v, want ErrManagerNotFound", name, err)
		}
	}

	if IsLogin(ctx, "token") {
		t.Fatal("IsLogin() = true without manager, want false")
	}
	if HasPermission(ctx, "u", "p") {
		t.Fatal("HasPermission() = true without manager, want false")
	}
	if HasRole(ctx, "u", "r") {
		t.Fatal("HasRole() = true without manager, want false")
	}
}

// TestGlobalOptionAuthTypePrecedence verifies variadic auth type overrides option auth type. TestGlobalOptionAuthTypePrecedence 验证可变认证类型优先于选项中的认证类型。
func TestGlobalOptionAuthTypePrecedence(t *testing.T) {
	DeleteAllManager()
	t.Cleanup(DeleteAllManager)

	ctx := context.Background()
	optionMgr, err := NewBuilder().IsPrintBanner(false).AutoRenew(false).AuthType("option").Build()
	if err != nil {
		t.Fatalf("Build(option) error = %v", err)
	}
	overrideMgr, err := NewBuilder().IsPrintBanner(false).AutoRenew(false).AuthType("override").Build()
	if err != nil {
		t.Fatalf("Build(override) error = %v", err)
	}
	SetManager(optionMgr)
	SetManager(overrideMgr)

	token, err := LoginWithOptions(ctx, LoginOptions{
		AuthType: "option",
		LoginID:  "user-override",
		Device:   "web",
	}, "override")
	if err != nil {
		t.Fatalf("LoginWithOptions() error = %v", err)
	}
	if _, err = GetLoginID(ctx, token, "option"); !errors.Is(err, derror.ErrInvalidToken) {
		t.Fatalf("GetLoginID(option) error = %v, want ErrInvalidToken", err)
	}
	loginID, err := GetLoginID(ctx, token, "override")
	if err != nil {
		t.Fatalf("GetLoginID(override) error = %v", err)
	}
	if loginID != "user-override" {
		t.Fatalf("loginID = %q, want user-override", loginID)
	}
}

// TestGlobalTypedOptionFacades verifies option-based global helpers dispatch to manager methods. TestGlobalTypedOptionFacades 验证全局选项门面分发到管理器方法。
func TestGlobalTypedOptionFacades(t *testing.T) {
	DeleteAllManager()
	t.Cleanup(DeleteAllManager)

	ctx := context.Background()
	mgr, err := NewBuilder().
		IsPrintBanner(false).
		AutoRenew(false).
		AuthType("typed").
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	SetManager(mgr)

	token, err := LoginWithOptions(ctx, LoginOptions{
		AuthType: "typed",
		LoginID:  "typed-user",
		Device:   "web",
		DeviceID: "browser",
		Timeout:  time.Minute,
	})
	if err != nil {
		t.Fatalf("LoginWithOptions() error = %v", err)
	}
	if err = AddPermissionsWithOptions(ctx, PermissionOptions{AuthType: "typed", Token: token, Permission: "article:read"}); err != nil {
		t.Fatalf("AddPermissionsWithOptions() error = %v", err)
	}
	if err = CheckPermissionWithOptions(ctx, PermissionOptions{AuthType: "typed", LoginID: "typed-user", Permission: "article:read"}); err != nil {
		t.Fatalf("CheckPermissionWithOptions() error = %v", err)
	}
	if err = AddRolesWithOptions(ctx, RoleOptions{AuthType: "typed", LoginID: "typed-user", Roles: []string{"admin", "operator"}}); err != nil {
		t.Fatalf("AddRolesWithOptions() error = %v", err)
	}
	if err = CheckRolesOrWithOptions(ctx, RoleOptions{AuthType: "typed", Token: token, Roles: []string{"missing", "admin"}}); err != nil {
		t.Fatalf("CheckRolesOrWithOptions() error = %v", err)
	}
	if err = DisableServiceWithOptions(ctx, ServiceDisableOptions{
		AuthType: "typed",
		LoginID:  "typed-user",
		Service:  "pay",
		Level:    3,
		Duration: time.Minute,
	}); err != nil {
		t.Fatalf("DisableServiceWithOptions() error = %v", err)
	}
	if !IsDisableServiceLevel(ctx, "typed-user", "pay", 2, "typed") {
		t.Fatal("IsDisableServiceLevel(level 2) = false, want true")
	}
	if err = DisableDeviceWithOptions(ctx, DeviceDisableOptions{
		AuthType: "typed",
		LoginID:  "typed-user",
		Device:   "web",
		DeviceID: "browser",
		Duration: time.Minute,
		Reason:   "risk",
	}); err != nil {
		t.Fatalf("DisableDeviceWithOptions() error = %v", err)
	}
	if !IsDisableDeviceAndDeviceId(ctx, "typed-user", "web", "browser", "typed") {
		t.Fatal("IsDisableDeviceAndDeviceId() = false, want true")
	}
	if err = LogoutWithOptions(ctx, LogoutOptions{AuthType: "typed", Token: token}); err != nil {
		t.Fatalf("LogoutWithOptions() error = %v", err)
	}
	if IsLogin(ctx, token, "typed") {
		t.Fatal("IsLogin() after LogoutWithOptions = true, want false")
	}
}

// TestInstanceNilAuthReturnsManagerNotFound verifies nil instance facades fail consistently. TestInstanceNilAuthReturnsManagerNotFound 验证空实例门面返回一致的管理器缺失错误。
func TestInstanceNilAuthReturnsManagerNotFound(t *testing.T) {
	ctx := context.Background()
	var auth *Auth

	if auth.Manager() != nil {
		t.Fatal("nil Auth Manager() should return nil")
	}
	if auth.EventManager() != nil {
		t.Fatal("nil Auth EventManager() should return nil")
	}
	if _, err := auth.LoginID(ctx, "user"); !errors.Is(err, derror.ErrManagerNotFound) {
		t.Fatalf("LoginID() error = %v, want ErrManagerNotFound", err)
	}
	if err := auth.CheckLogin(ctx, "token"); !errors.Is(err, derror.ErrManagerNotFound) {
		t.Fatalf("CheckLogin() error = %v, want ErrManagerNotFound", err)
	}
	if auth.IsLogin(ctx, "token") {
		t.Fatal("IsLogin() = true for nil Auth, want false")
	}
}
