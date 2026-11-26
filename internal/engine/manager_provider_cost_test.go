package engine

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	derror "github.com/Zany2/dtoken-go/errors"
)

// TestTokenAuthorizationSkipsSession covers every token authorization entry with unreadable Session data. TestTokenAuthorizationSkipsSession 使用不可解码的 Session 覆盖全部 Token 授权入口。
func TestTokenAuthorizationSkipsSession(t *testing.T) {
	ctx := context.Background()
	var calls int
	var subjects []AccessSubject
	resolve := func(_ context.Context, subject AccessSubject) ([]string, error) {
		calls++
		subjects = append(subjects, subject)
		return []string{"allowed"}, nil
	}
	mgr := newTestManagerWithAccessProvider(t, nil, AccessProviderFunc{PermissionFunc: resolve, RoleFunc: resolve})
	token, err := mgr.Login(ctx, "provider-cost", "web", "browser")
	if err != nil {
		t.Fatal(err)
	}
	if err = mgr.storage.Set(ctx, mgr.getSessionKey("provider-cost"), "{", time.Hour); err != nil {
		t.Fatal(err)
	}
	storage := &managerGetCountingStorage{Storage: mgr.storage}
	mgr.storage = storage
	wantSubject := AccessSubject{AuthType: mgr.config.AuthType, LoginID: "provider-cost", Device: "web", DeviceID: "browser", Token: token}

	assertCost := func(before int) {
		t.Helper()
		if calls != before+1 || subjects[len(subjects)-1] != wantSubject {
			t.Fatalf("provider calls/subject = %d/%+v", calls, subjects)
		}
		if storage.getCount(mgr.getTokenKey(token)) != calls || storage.getCount(mgr.getSessionKey("provider-cost")) != 0 {
			t.Fatal("authorization did not use one token read and zero Session reads")
		}
	}
	for _, get := range []func(context.Context, string) ([]string, error){mgr.GetPermissionsByToken, mgr.GetRolesByToken} {
		before := calls
		if values, err := get(ctx, token); err != nil || !reflect.DeepEqual(values, []string{"allowed"}) {
			t.Fatalf("get = %v, %v", values, err)
		}
		assertCost(before)
	}
	for _, has := range []func() bool{
		func() bool { return mgr.HasPermissionByToken(ctx, token, "allowed") },
		func() bool { return mgr.HasPermissionsAndByToken(ctx, token, []string{"allowed"}) },
		func() bool { return mgr.HasPermissionsOrByToken(ctx, token, []string{"allowed"}) },
		func() bool { return mgr.HasRoleByToken(ctx, token, "allowed") },
		func() bool { return mgr.HasRolesAndByToken(ctx, token, []string{"allowed"}) },
		func() bool { return mgr.HasRolesOrByToken(ctx, token, []string{"allowed"}) },
	} {
		before := calls
		if !has() {
			t.Fatal("provider-authorized token was rejected")
		}
		assertCost(before)
	}
	for _, check := range []func() error{
		func() error { return mgr.CheckPermissionByToken(ctx, token, "allowed") },
		func() error { return mgr.CheckPermissionAndByToken(ctx, token, []string{"allowed"}) },
		func() error { return mgr.CheckPermissionOrByToken(ctx, token, []string{"allowed"}) },
		func() error { return mgr.CheckRoleByToken(ctx, token, "allowed") },
		func() error { return mgr.CheckRoleAndByToken(ctx, token, []string{"allowed"}) },
		func() error { return mgr.CheckRoleOrByToken(ctx, token, []string{"allowed"}) },
	} {
		before := calls
		if err := check(); err != nil {
			t.Fatal(err)
		}
		assertCost(before)
	}

	// Credential validation must still precede provider calls. 凭据校验仍必须先于提供器调用。
	before := calls
	if _, err = mgr.GetPermissionsByToken(ctx, "missing"); !errors.Is(err, derror.ErrInvalidToken) {
		t.Fatalf("missing token = %v", err)
	}
	if _, err = mgr.GetRolesByToken(ctx, "missing"); !errors.Is(err, derror.ErrInvalidToken) {
		t.Fatalf("missing token = %v", err)
	}
	if calls != before {
		t.Fatal("invalid credentials reached provider")
	}
}

// TestCombinedProviderFallbackCosts preserves nil/empty/error semantics and one Session read per operation. TestCombinedProviderFallbackCosts 保留 nil、空列表和错误语义，并限制单次操作最多读取一次 Session。
func TestCombinedProviderFallbackCosts(t *testing.T) {
	ctx := context.Background()
	providerErr := errors.New("provider unavailable")
	for _, tc := range []struct {
		name                string
		permissions, roles  []string
		providerErr         error
		want                error
		sessions, roleCalls int
	}{
		{"provider only", []string{"read"}, []string{"reader"}, nil, nil, 0, 1},
		{"both fallback", nil, nil, nil, nil, 1, 1},
		{"permission fallback", nil, []string{"reader"}, nil, nil, 1, 1},
		{"role fallback", []string{"read"}, nil, nil, nil, 1, 1},
		{"empty permissions", []string{}, nil, nil, derror.ErrPermissionDenied, 0, 0},
		{"blank permissions", []string{""}, nil, nil, derror.ErrPermissionDenied, 0, 0},
		{"empty roles", []string{"read"}, []string{}, nil, derror.ErrRoleDenied, 0, 1},
		{"provider error", nil, nil, providerErr, providerErr, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			permissions, roles := 0, 0
			mgr := newTestManagerWithAccessProvider(t, nil, AccessProviderFunc{
				PermissionFunc: func(context.Context, AccessSubject) ([]string, error) {
					permissions++
					return tc.permissions, tc.providerErr
				},
				RoleFunc: func(context.Context, AccessSubject) ([]string, error) { roles++; return tc.roles, nil },
			})
			token, err := mgr.Login(ctx, "fallback-cost", "web")
			if err != nil {
				t.Fatal(err)
			}
			if err = mgr.AddPermissions(ctx, "fallback-cost", []string{"read"}); err != nil {
				t.Fatal(err)
			}
			if err = mgr.AddRoles(ctx, "fallback-cost", []string{"reader"}); err != nil {
				t.Fatal(err)
			}
			storage := &managerGetCountingStorage{Storage: mgr.storage}
			mgr.storage = storage
			for attempt := 1; attempt <= 2; attempt++ {
				id, err := CheckTokenAccess(ctx, mgr, token, []string{"read"}, []string{"reader"}, true, false)
				if !errors.Is(err, tc.want) || id != "fallback-cost" {
					t.Fatalf("check = %q, %v", id, err)
				}
				if permissions != attempt || roles != attempt*tc.roleCalls {
					t.Fatalf("provider calls = %d/%d", permissions, roles)
				}
				if storage.getCount(mgr.getTokenKey(token)) != attempt || storage.getCount(mgr.getSessionKey(id)) != attempt*tc.sessions {
					t.Fatal("unexpected reads or cross-request caching")
				}
			}
		})
	}
}

// TestProviderFallbackPreservesSessionFailures checks Session validation only when fallback is required. TestProviderFallbackPreservesSessionFailures 验证仅在回退时执行 Session 校验。
func TestProviderFallbackPreservesSessionFailures(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name  string
		value any
		want  error
	}{
		{"missing", nil, derror.ErrInvalidToken},
		{"corrupt", "{", derror.ErrSerializeFailed},
		{"foreign", `{"authType":"other:","loginId":"fallback-failure"}`, derror.ErrInvalidToken},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mgr := newTestManagerWithAccessProvider(t, nil, AccessProviderFunc{})
			token, err := mgr.Login(ctx, "fallback-failure")
			if err != nil {
				t.Fatal(err)
			}
			key := mgr.getSessionKey("fallback-failure")
			if tc.value == nil {
				err = mgr.storage.Delete(ctx, key)
			} else {
				err = mgr.storage.Set(ctx, key, tc.value, time.Hour)
			}
			if err != nil {
				t.Fatal(err)
			}
			for _, get := range []func(context.Context, string) ([]string, error){mgr.GetPermissionsByToken, mgr.GetRolesByToken} {
				if _, err = get(ctx, token); !errors.Is(err, tc.want) {
					t.Fatalf("fallback = %v, want %v", err, tc.want)
				}
			}
		})
	}
}
