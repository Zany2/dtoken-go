package httpauth

import (
	"context"
	"sync/atomic"
	"testing"

	adapter "github.com/Zany2/dtoken-go/contract"
	manager "github.com/Zany2/dtoken-go/internal/engine"
)

// TestCheckReusesValidatedIdentity guards storage reads and the lazy identity result contract. TestCheckReusesValidatedIdentity 保护存储读取次数与延迟身份输出约定。
func TestCheckReusesValidatedIdentity(t *testing.T) {
	cfg := newAuthcheckTestManager(t).GetConfig().Clone()
	storage := &identityReadStorage{Storage: newAuthcheckTestStorage()}
	mgr := manager.NewManager(cfg, &authcheckTestGenerator{}, storage, authcheckTestCodec{}, adapter.NewNopLogger(), nil, nil)
	t.Cleanup(mgr.CloseManager)
	ctx := context.Background()
	token, err := mgr.Login(ctx, "user", "web", "browser")
	if err != nil {
		t.Fatal(err)
	}
	if err := mgr.AddPermissions(ctx, "user", []string{"read"}); err != nil {
		t.Fatal(err)
	}
	if err := mgr.AddRoles(ctx, "user", []string{"reader"}); err != nil {
		t.Fatal(err)
	}

	// A login-only request performs one complete validation and keeps identity output empty. 仅检查登录时执行一次完整校验并保持身份输出为空。
	storage.reads.Store(0)
	if _, err := mgr.GetLoginID(ctx, token); err != nil {
		t.Fatal(err)
	}
	loginReads := storage.reads.Load()
	storage.reads.Store(0)
	result, err := Check(ctx, mgr, Request{TokenValue: token, CheckLogin: true})
	if err != nil || result == nil || result.LoginID != "" {
		t.Fatalf("login-only Check() = %v, %v", result, err)
	}
	if got := storage.reads.Load(); got != loginReads {
		t.Fatalf("login-only reads = %d, want %d", got, loginReads)
	}

	// Compare combined checks with the same access and disable paths. 比较相同权限与封禁路径的组合检查。
	for _, tc := range []struct {
		name         string
		checkDisable bool
		permissions  []string
		roles        []string
	}{
		{name: "disable", checkDisable: true},
		{name: "permissions", permissions: []string{"read"}},
		{name: "roles", roles: []string{"reader"}},
		{name: "permissions_and_roles", permissions: []string{"read"}, roles: []string{"reader"}},
		{name: "disable_permissions", checkDisable: true, permissions: []string{"read"}},
		{name: "disable_roles", checkDisable: true, roles: []string{"reader"}},
		{name: "disable_permissions_and_roles", checkDisable: true, permissions: []string{"read"}, roles: []string{"reader"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := Request{TokenValue: token, CheckDisable: tc.checkDisable, Permissions: tc.permissions, Roles: tc.roles}
			storage.reads.Store(0)
			if _, err := Check(ctx, mgr, req); err != nil {
				t.Fatal(err)
			}
			baselineReads := storage.reads.Load()
			if !req.CheckDisable {
				baselineReads += loginReads
			}

			req.CheckLogin = true
			storage.reads.Store(0)
			result, err := Check(ctx, mgr, req)
			if err != nil || result == nil || result.LoginID != "user" {
				t.Fatalf("combined Check() = %v, %v", result, err)
			}
			if got := storage.reads.Load(); got != baselineReads {
				t.Fatalf("combined reads = %d, want %d", got, baselineReads)
			}
		})
	}
}

// identityReadStorage counts reads through the ordinary storage contract. identityReadStorage 通过普通存储契约统计读取次数。
type identityReadStorage struct {
	adapter.Storage
	reads atomic.Int64
}

// Get counts reads while preserving the wrapped storage behavior. Get 统计读取次数并保持被包装存储的行为。
func (s *identityReadStorage) Get(ctx context.Context, key string) (any, error) {
	s.reads.Add(1)
	return s.Storage.Get(ctx, key)
}
