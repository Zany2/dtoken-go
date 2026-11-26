package dtoken

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/config"
	derror "github.com/Zany2/dtoken-go/errors"
)

// TestRegistryHelpersAndBuild verifies auth type parsing and build registration. TestRegistryHelpersAndBuild 验证认证类型解析和构建注册。
func TestRegistryHelpersAndBuild(t *testing.T) {
	DeleteAllManager()
	t.Cleanup(DeleteAllManager)

	if got := getAutoType(" admin "); got != "admin:" {
		t.Fatalf("getAutoType(admin) = %q, want admin:", got)
	}
	if got := getAutoType(""); got != config.DefaultAuthType {
		t.Fatalf("getAutoType(empty) = %q, want %q", got, config.DefaultAuthType)
	}

	auth, err := NewBuilder().IsPrintBanner(false).AutoRenew(false).AuthType("built").Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	mgr := auth.Manager()
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

// TestRootTypedOptionFacades verifies option-based global helpers dispatch to manager methods. TestRootTypedOptionFacades 验证全局选项门面分发到管理器方法。
func TestRootTypedOptionFacades(t *testing.T) {
	DeleteAllManager()
	t.Cleanup(DeleteAllManager)

	ctx := context.Background()
	mgr, err := buildTestManager(NewBuilder().
		IsPrintBanner(false).
		AutoRenew(false).
		AuthType("typed"),
	)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if err := SetManager(mgr); err != nil {
		t.Fatalf("register authentication: %v", err)
	}

	token, err := AuthType("typed").Login(ctx, LoginOptions{
		LoginID:  "typed-user",
		Device:   "web",
		DeviceID: "browser",
		Timeout:  time.Minute,
	})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if err = AuthType("typed").AddPermissions(ctx, PermissionOptions{Token: token, Permission: "article:read"}); err != nil {
		t.Fatalf("AddPermissions() error = %v", err)
	}
	if err = AuthType("typed").CheckPermission(ctx, PermissionOptions{LoginID: "typed-user", Permission: "article:read"}); err != nil {
		t.Fatalf("CheckPermission() error = %v", err)
	}
	if err = AuthType("typed").AddRoles(ctx, RoleOptions{LoginID: "typed-user", Roles: []string{"admin", "operator"}}); err != nil {
		t.Fatalf("AddRoles() error = %v", err)
	}
	if err = AuthType("typed").CheckRolesOr(ctx, RoleOptions{Token: token, Roles: []string{"missing", "admin"}}); err != nil {
		t.Fatalf("CheckRolesOr() error = %v", err)
	}
	if err = AuthType("typed").DisableService(ctx, ServiceDisableOptions{
		LoginID:  "typed-user",
		Service:  "pay",
		Level:    3,
		Duration: time.Minute,
	}); err != nil {
		t.Fatalf("DisableService() error = %v", err)
	}
	if !AuthType("typed").IsDisableServiceLevel(ctx, "typed-user", "pay", 2) {
		t.Fatal("IsDisableServiceLevel(level 2) = false, want true")
	}
	if err = AuthType("typed").DisableDevice(ctx, DeviceDisableOptions{
		LoginID:  "typed-user",
		Device:   "web",
		DeviceID: "browser",
		Duration: time.Minute,
		Reason:   "risk",
	}); err != nil {
		t.Fatalf("DisableDevice() error = %v", err)
	}
	if !AuthType("typed").IsDisableDeviceAndDeviceID(ctx, "typed-user", "web", "browser") {
		t.Fatal("IsDisableDeviceAndDeviceID() = false, want true")
	}
	if err = AuthType("typed").Logout(ctx, LogoutOptions{Token: token}); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if AuthType("typed").IsLogin(ctx, token) {
		t.Fatal("IsLogin() after Logout = true, want false")
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
