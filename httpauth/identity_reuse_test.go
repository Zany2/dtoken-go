package httpauth

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/Zany2/dtoken-go/config"
	adapter "github.com/Zany2/dtoken-go/contract"
	derror "github.com/Zany2/dtoken-go/errors"
	manager "github.com/Zany2/dtoken-go/internal/engine"
	"github.com/Zany2/dtoken-go/internal/utils"
)

// TestCheckReusesValidatedIdentity guards storage reads and the lazy identity result contract. TestCheckReusesValidatedIdentity 保护存储读取次数与延迟身份输出约定。
func TestCheckReusesValidatedIdentity(t *testing.T) {
	cfg := newAuthcheckTestManager(t).GetConfig()
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
	namespace := utils.StorageNamespace(cfg.KeyPrefix, cfg.AuthType)
	storage.tokenKey = namespace + config.TokenKeyPrefix + token
	storage.sessionKey = namespace + manager.SessionKeyPrefix + "user"

	for mask := 0; mask < 16; mask++ {
		t.Run(fmt.Sprintf("checks_%04b", mask), func(t *testing.T) {
			req := Request{TokenValue: token, CheckLogin: mask&1 != 0, CheckDisable: mask&2 != 0}
			if mask&4 != 0 {
				req.Permissions = []string{"read"}
			}
			if mask&8 != 0 {
				req.Roles = []string{"reader"}
			}
			var wantTokens, wantSessions int64
			if mask != 0 {
				wantTokens = 1
			}
			if mask&12 != 0 {
				wantSessions = 1
			}
			wantID := ""
			if req.CheckDisable || (req.CheckLogin && wantSessions == 1) {
				wantID = "user"
			}
			storage.tokenReads.Store(0)
			storage.sessionReads.Store(0)
			result, err := Check(ctx, mgr, req)
			if err != nil || result == nil || result.LoginID != wantID {
				t.Fatalf("Check() = %v, %v, want login ID %q", result, err, wantID)
			}
			if got := storage.tokenReads.Load(); got != wantTokens {
				t.Fatalf("token reads = %d, want %d", got, wantTokens)
			}
			if got := storage.sessionReads.Load(); got != wantSessions {
				t.Fatalf("session reads = %d, want %d", got, wantSessions)
			}
		})
	}

	// Reusing a request value must not cache authentication across calls. 复用请求参数不能跨调用缓存登录态。
	req := Request{TokenValue: token, CheckLogin: true, Permissions: []string{"read"}, Roles: []string{"reader"}}
	if _, err := Check(ctx, mgr, req); err != nil {
		t.Fatal(err)
	}
	if err := mgr.Kickout(ctx, token); err != nil {
		t.Fatal(err)
	}
	if _, err := Check(ctx, mgr, req); !errors.Is(err, derror.ErrNotLogin) {
		t.Fatalf("Check after kickout = %v, want ErrNotLogin", err)
	}
}

// TestCheckLoadsSessionOnlyForAccess preserves the lightweight login path and access failure causes. TestCheckLoadsSessionOnlyForAccess 保留轻量登录路径及访问检查的错误原因。
func TestCheckLoadsSessionOnlyForAccess(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name  string
		value any
		want  error
	}{
		{"missing", nil, derror.ErrInvalidToken},
		{"corrupt", []byte("{broken"), derror.ErrSerializeFailed},
		{"foreign", []byte(`{"authType":"other:","loginId":"user"}`), derror.ErrInvalidToken},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mgr := newAuthcheckTestManager(t)
			token, err := mgr.Login(ctx, "user", "web")
			if err != nil {
				t.Fatal(err)
			}
			cfg := mgr.GetConfig()
			key := utils.StorageNamespace(cfg.KeyPrefix, cfg.AuthType) + manager.SessionKeyPrefix + "user"
			if tc.value == nil {
				err = mgr.GetStorage().Delete(ctx, key)
			} else {
				err = mgr.GetStorage().Set(ctx, key, tc.value, adapter.TTLNoExpire)
			}
			if err != nil {
				t.Fatal(err)
			}
			loginError := errors.New("custom login error")
			req := Request{TokenValue: token, CheckLogin: true, LoginError: loginError}
			if _, err := Check(ctx, mgr, req); err != nil {
				t.Fatalf("login-only check = %v", err)
			}
			req.Permissions = []string{"read"}
			req.Roles = []string{"reader"}
			if _, err := Check(ctx, mgr, req); !errors.Is(err, tc.want) {
				t.Fatalf("access check = %v, want %v", err, tc.want)
			}
		})
	}
}

// identityReadStorage counts token and session reads separately from disable checks. identityReadStorage 分别统计 Token 和 Session 读取，排除封禁检查。
type identityReadStorage struct {
	adapter.Storage
	tokenKey     string
	sessionKey   string
	tokenReads   atomic.Int64
	sessionReads atomic.Int64
}

// Get counts reads while preserving the wrapped storage behavior. Get 统计读取次数并保持被包装存储的行为。
func (s *identityReadStorage) Get(ctx context.Context, key string) (any, error) {
	if key == s.tokenKey {
		s.tokenReads.Add(1)
	}
	if key == s.sessionKey {
		s.sessionReads.Add(1)
	}
	return s.Storage.Get(ctx, key)
}
