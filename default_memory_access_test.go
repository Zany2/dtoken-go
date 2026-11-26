package dtoken

import (
	"context"
	"slices"
	"testing"
)

// TestDefaultMemoryAccess checks mutation and AND/OR authorization through the default registry. TestDefaultMemoryAccess 通过默认注册表验证变更和 AND/OR 授权。
func TestDefaultMemoryAccess(t *testing.T) {
	DeleteAllManager()
	t.Cleanup(CloseAll)
	_, err := NewBuilder().IsPrintBanner(false).IsLog(false).AutoRenew(false).Build()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	token, err := LoginID(ctx, "user")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("Permissions", func(t *testing.T) {
		if err := AddPermissions(ctx, PermissionOptions{LoginID: "user", Permissions: []string{"read"}}); err != nil {
			t.Fatal(err)
		}
		if err := AddPermissionsByToken(ctx, token, []string{"write"}); err != nil {
			t.Fatal(err)
		}
		for _, get := range []func() ([]string, error){
			func() ([]string, error) { return GetPermissions(ctx, "user") },
			func() ([]string, error) { return GetPermissionsByToken(ctx, token) },
		} {
			if values, err := get(); err != nil || len(values) != 2 || !slices.Contains(values, "read") || !slices.Contains(values, "write") {
				t.Fatalf("grants = %v, %v", values, err)
			}
		}
		checks := []struct {
			name    string
			call    func() bool
			granted bool
		}{
			{"single account", func() bool { return HasPermission(ctx, "user", "read") }, true},
			{"single token", func() bool { return HasPermissionByToken(ctx, token, "read") }, true},
			{"and account", func() bool { return HasPermissionsAnd(ctx, "user", []string{"read", "write"}) }, true},
			{"and token", func() bool { return HasPermissionsAndByToken(ctx, token, []string{"read", "write"}) }, true},
			{"or account", func() bool { return HasPermissionsOr(ctx, "user", []string{"read", "missing"}) }, true},
			{"or token", func() bool { return HasPermissionsOrByToken(ctx, token, []string{"read", "missing"}) }, true},
			{"missing account", func() bool { return HasPermissionsAnd(ctx, "user", []string{"read", "missing"}) }, false},
			{"missing token", func() bool { return HasPermissionsOrByToken(ctx, token, []string{"missing"}) }, false},
		}
		errors := []struct {
			name string
			call func() error
		}{
			{"single options", func() error { return CheckPermission(ctx, PermissionOptions{LoginID: "user", Permission: "read"}) }},
			{"and options", func() error {
				return CheckPermissionsAnd(ctx, PermissionOptions{LoginID: "user", Permissions: []string{"read", "write"}})
			}},
			{"or options", func() error {
				return CheckPermissionsOr(ctx, PermissionOptions{Token: token, Permissions: []string{"read", "missing"}})
			}},
			{"single token", func() error { return CheckPermissionByToken(ctx, token, "read") }},
			{"and account", func() error { return CheckPermissionAnd(ctx, "user", []string{"read", "write"}) }},
			{"and token", func() error { return CheckPermissionAndByToken(ctx, token, []string{"read", "write"}) }},
			{"or account", func() error { return CheckPermissionOr(ctx, "user", []string{"read", "missing"}) }},
			{"or token", func() error { return CheckPermissionOrByToken(ctx, token, []string{"read", "missing"}) }},
		}
		for _, check := range checks {
			if got := check.call(); got != check.granted {
				t.Errorf("%s = %v", check.name, got)
			}
		}
		for _, check := range errors {
			if err := check.call(); err != nil {
				t.Errorf("%s = %v", check.name, err)
			}
		}
		if err := RemovePermissions(ctx, PermissionOptions{LoginID: "user", Permissions: []string{"read"}}); err != nil {
			t.Fatal(err)
		}
		if err := RemovePermissionsByToken(ctx, token, []string{"write"}); err != nil {
			t.Fatal(err)
		}
		for _, check := range checks {
			if check.call() {
				t.Errorf("%s accepted removed grants", check.name)
			}
		}
		for _, check := range errors {
			if check.call() == nil {
				t.Errorf("%s accepted removed grants", check.name)
			}
		}
	})

	t.Run("Roles", func(t *testing.T) {
		if err := AddRoles(ctx, RoleOptions{LoginID: "user", Roles: []string{"read"}}); err != nil {
			t.Fatal(err)
		}
		if err := AddRolesByToken(ctx, token, []string{"write"}); err != nil {
			t.Fatal(err)
		}
		for _, get := range []func() ([]string, error){
			func() ([]string, error) { return GetRoles(ctx, "user") },
			func() ([]string, error) { return GetRolesByToken(ctx, token) },
		} {
			if values, err := get(); err != nil || len(values) != 2 || !slices.Contains(values, "read") || !slices.Contains(values, "write") {
				t.Fatalf("grants = %v, %v", values, err)
			}
		}
		checks := []struct {
			name    string
			call    func() bool
			granted bool
		}{
			{"single account", func() bool { return HasRole(ctx, "user", "read") }, true},
			{"single token", func() bool { return HasRoleByToken(ctx, token, "read") }, true},
			{"and account", func() bool { return HasRolesAnd(ctx, "user", []string{"read", "write"}) }, true},
			{"and token", func() bool { return HasRolesAndByToken(ctx, token, []string{"read", "write"}) }, true},
			{"or account", func() bool { return HasRolesOr(ctx, "user", []string{"read", "missing"}) }, true},
			{"or token", func() bool { return HasRolesOrByToken(ctx, token, []string{"read", "missing"}) }, true},
			{"missing account", func() bool { return HasRolesAnd(ctx, "user", []string{"read", "missing"}) }, false},
			{"missing token", func() bool { return HasRolesOrByToken(ctx, token, []string{"missing"}) }, false},
		}
		errors := []struct {
			name string
			call func() error
		}{
			{"single options", func() error { return CheckRole(ctx, RoleOptions{LoginID: "user", Role: "read"}) }},
			{"and options", func() error {
				return CheckRolesAnd(ctx, RoleOptions{LoginID: "user", Roles: []string{"read", "write"}})
			}},
			{"or options", func() error { return CheckRolesOr(ctx, RoleOptions{Token: token, Roles: []string{"read", "missing"}}) }},
			{"single token", func() error { return CheckRoleByToken(ctx, token, "read") }},
			{"and account", func() error { return CheckRoleAnd(ctx, "user", []string{"read", "write"}) }},
			{"and token", func() error { return CheckRoleAndByToken(ctx, token, []string{"read", "write"}) }},
			{"or account", func() error { return CheckRoleOr(ctx, "user", []string{"read", "missing"}) }},
			{"or token", func() error { return CheckRoleOrByToken(ctx, token, []string{"read", "missing"}) }},
		}
		for _, check := range checks {
			if got := check.call(); got != check.granted {
				t.Errorf("%s = %v", check.name, got)
			}
		}
		for _, check := range errors {
			if err := check.call(); err != nil {
				t.Errorf("%s = %v", check.name, err)
			}
		}
		if err := RemoveRoles(ctx, RoleOptions{LoginID: "user", Roles: []string{"read"}}); err != nil {
			t.Fatal(err)
		}
		if err := RemoveRolesByToken(ctx, token, []string{"write"}); err != nil {
			t.Fatal(err)
		}
		for _, check := range checks {
			if check.call() {
				t.Errorf("%s accepted removed grants", check.name)
			}
		}
		for _, check := range errors {
			if check.call() == nil {
				t.Errorf("%s accepted removed grants", check.name)
			}
		}
	})
}
