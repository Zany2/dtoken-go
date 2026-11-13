package global

import (
	"context"
	"errors"
	"testing"
	"time"

	derror "github.com/Zany2/dtoken-go/errors"
)

// TestRegistryNormalizesAuthType verifies manager registry auth type normalization. TestRegistryNormalizesAuthType 验证管理器注册表会规范化认证类型。
func TestRegistryNormalizesAuthType(t *testing.T) {
	DeleteAllManager()
	t.Cleanup(DeleteAllManager)

	mgr, err := buildTestManager(NewBuilder().
		IsPrintBanner(false).
		AutoRenew(false).
		AuthType("registry"),
	)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	SetManager(mgr)

	withoutSuffix, err := GetManager("registry")
	if err != nil {
		t.Fatalf("GetManager(registry) error = %v", err)
	}
	withSuffix, err := GetManager("registry:")
	if err != nil {
		t.Fatalf("GetManager(registry:) error = %v", err)
	}
	if withoutSuffix != withSuffix || withoutSuffix != mgr {
		t.Fatal("GetManager() did not return the registered manager for normalized auth types")
	}

	if err = DeleteManager("registry"); err != nil {
		t.Fatalf("DeleteManager() error = %v", err)
	}
	if _, err = GetManager("registry"); !errors.Is(err, derror.ErrManagerNotFound) {
		t.Fatalf("GetManager() after delete error = %v, want ErrManagerNotFound", err)
	}
}

// TestGlobalFacadeLoginAndAccessFlow verifies global functions route to selected manager. TestGlobalFacadeLoginAndAccessFlow 验证全局函数会路由到指定管理器。
func TestGlobalFacadeLoginAndAccessFlow(t *testing.T) {
	DeleteAllManager()
	t.Cleanup(DeleteAllManager)

	ctx := context.Background()
	mgr, err := buildTestManager(NewBuilder().
		IsPrintBanner(false).
		AutoRenew(false).
		AuthType("facade"),
	)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	SetManager(mgr)

	token, err := Login(ctx, "user-1", "web", "browser-1", "facade")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if !IsLogin(ctx, token, "facade") {
		t.Fatal("IsLogin() = false, want true")
	}
	device, err := GetDevice(ctx, token, "facade")
	if err != nil {
		t.Fatalf("GetDevice() error = %v", err)
	}
	if device != "web" {
		t.Fatalf("GetDevice() = %q, want web", device)
	}
	combinedDevice, combinedDeviceID, err := GetDeviceAndDeviceID(ctx, token, "facade")
	if err != nil {
		t.Fatalf("GetDeviceAndDeviceID() error = %v", err)
	}
	if combinedDevice != "web" || combinedDeviceID != "browser-1" {
		t.Fatalf("GetDeviceAndDeviceID() = %q/%q, want web/browser-1", combinedDevice, combinedDeviceID)
	}

	if err = AddPermissions(ctx, "user-1", []string{"profile:read"}, "facade"); err != nil {
		t.Fatalf("AddPermissions() error = %v", err)
	}
	if !HasPermission(ctx, "user-1", "profile:read", "facade") {
		t.Fatal("HasPermission() = false, want true")
	}
	if err = AddRolesByToken(ctx, token, []string{"member"}, "facade"); err != nil {
		t.Fatalf("AddRolesByToken() error = %v", err)
	}
	if !HasRoleByToken(ctx, token, "member", "facade") {
		t.Fatal("HasRoleByToken() = false, want true")
	}

	if err = LogoutByDeviceAndDeviceID(ctx, "user-1", "web", "browser-1", "facade"); err != nil {
		t.Fatalf("LogoutByDeviceAndDeviceID() error = %v", err)
	}
	if IsLogin(ctx, token, "facade") {
		t.Fatal("IsLogin() after logout = true, want false")
	}
}

// TestInstanceFacadeOptions verifies typed instance options dispatch by token or login ID. TestInstanceFacadeOptions 验证实例门面按 token 或登录 ID 分发类型化选项。
func TestInstanceFacadeOptions(t *testing.T) {
	ctx := context.Background()
	mgr, err := buildTestManager(NewBuilder().
		IsPrintBanner(false).
		AutoRenew(false).
		AuthType("instance"),
	)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	auth := New(mgr)
	t.Cleanup(auth.Close)

	token, err := auth.Login(ctx, LoginOptions{
		LoginID:  "user-2",
		Device:   "mobile",
		DeviceID: "phone-1",
		Timeout:  time.Minute,
	})
	if err != nil {
		t.Fatalf("Auth.Login() error = %v", err)
	}
	if err = auth.AddPermissions(ctx, PermissionOptions{Token: token, Permission: "article:edit"}); err != nil {
		t.Fatalf("Auth.AddPermissions() error = %v", err)
	}
	if err = auth.CheckPermission(ctx, PermissionOptions{Token: token, Permission: "article:edit"}); err != nil {
		t.Fatalf("Auth.CheckPermission() error = %v", err)
	}
	if !auth.HasPermissionByToken(ctx, token, "article:edit") {
		t.Fatal("Auth.HasPermissionByToken() = false, want true")
	}
	if err = auth.AddRoles(ctx, RoleOptions{LoginID: "user-2", Roles: []string{"author"}}); err != nil {
		t.Fatalf("Auth.AddRoles() error = %v", err)
	}
	if err = auth.CheckRole(ctx, RoleOptions{Token: token, Role: "author"}); err != nil {
		t.Fatalf("Auth.CheckRole() error = %v", err)
	}
	if !auth.HasRoleByToken(ctx, token, "author") {
		t.Fatal("Auth.HasRoleByToken() = false, want true")
	}
	if count, err := auth.GetOnlineTerminalCount(ctx, "user-2"); err != nil || count != 1 {
		t.Fatalf("Auth.GetOnlineTerminalCount() = %d, %v, want 1", count, err)
	}
	if device, err := auth.GetDevice(ctx, token); err != nil || device != "mobile" {
		t.Fatalf("Auth.GetDevice() = %q, %v, want mobile", device, err)
	}
	if device, deviceID, err := auth.GetDeviceAndDeviceID(ctx, token); err != nil || device != "mobile" || deviceID != "phone-1" {
		t.Fatalf("Auth.GetDeviceAndDeviceID() = %q/%q, %v, want mobile/phone-1", device, deviceID, err)
	}

	cfg := mgr.GetConfig()
	if cfg.AuthType != "instance:" {
		t.Fatalf("manager auth type = %q, want normalized instance", cfg.AuthType)
	}
}

// TestInstanceFacadeOptionalModules verifies optional module instance facade methods. TestInstanceFacadeOptionalModules 验证可选模块实例门面方法。
