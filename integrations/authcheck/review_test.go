package authcheck

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/Zany2/dtoken-go/core/adapter"
	corecontext "github.com/Zany2/dtoken-go/core/context"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/manager"
	"github.com/Zany2/dtoken-go/dtoken"
)

// TestResolveManagerFromContextMissingCache verifies annotation requests can fall back without a cached context. TestResolveManagerFromContextMissingCache 验证注解请求没有缓存上下文时可正常回退。
func TestResolveManagerFromContextMissingCache(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	mgr := newAuthcheckTestManager(t)
	dtoken.SetManager(mgr)

	var missingContext *corecontext.DTokenContext
	for _, cached := range []any{nil, missingContext, (*managerSource)(nil), struct{}{}} {
		if got, err := ResolveManagerFromContext("", cached); err != nil || got != mgr {
			t.Fatalf("cached %T: manager = %v, error = %v", cached, got, err)
		}
	}
	dtoken.DeleteAllManager()
	if got, err := ResolveManagerFromContext("", missingContext); got != nil || !errors.Is(err, derror.ErrManagerNotFound) {
		t.Fatalf("missing cache and registry: manager = %v, error = %v", got, err)
	}
}

// TestResolveManagerFromContextRejectsEmptyScope keeps invalid request scopes out of the global namespace. TestResolveManagerFromContextRejectsEmptyScope 防止无效请求作用域回退到全局认证空间。
func TestResolveManagerFromContextRejectsEmptyScope(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	global := newAuthcheckTestManager(t)
	dtoken.SetManager(global)
	for _, cached := range []any{managerSource{}, &managerSource{}, corecontext.NewContext(nil, nil)} {
		if got, err := ResolveManagerFromContext("", cached); got != nil || !errors.Is(err, derror.ErrManagerNotFound) {
			t.Fatalf("cached %T: manager=%p error=%v", cached, got, err)
		}
		if got, err := ResolveManagerFromContext(global.GetConfig().AuthType, cached); err != nil || got != global {
			t.Fatalf("explicit selection with cached %T: manager=%p error=%v", cached, got, err)
		}
	}
}

// TestCheckRejectsUnavailableManager verifies protected requests reject missing and closed managers. TestCheckRejectsUnavailableManager 验证受保护请求拒绝缺失或已关闭的管理器。
func TestCheckRejectsUnavailableManager(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	active := newAuthcheckTestManager(t)
	dtoken.SetManager(active)
	closed := newAuthcheckTestManager(t)
	ctx := context.Background()
	token, err := closed.Login(ctx, "user")
	if err != nil {
		t.Fatal(err)
	}
	closed.CloseManager()

	if got, err := ResolveManager(closed, ""); got != nil || !errors.Is(err, derror.ErrManagerNotFound) {
		t.Fatalf("closed explicit manager = %v, error = %v", got, err)
	}
	if got, err := ResolveManagerFromContext("", managerSource{manager: closed}); got != nil || !errors.Is(err, derror.ErrManagerNotFound) {
		t.Fatalf("closed cached manager = %v, error = %v", got, err)
	}
	if got, err := ResolveManagerFromContext(active.GetConfig().AuthType, managerSource{manager: closed}); err != nil || got != active {
		t.Fatalf("explicit auth type manager = %v, error = %v", got, err)
	}
	for _, mgr := range []*manager.Manager{nil, closed} {
		for _, req := range []Request{
			{TokenValue: token, CheckLogin: true},
			{TokenValue: token, CheckDisable: true},
			{TokenValue: token, Permissions: []string{"read"}},
			{TokenValue: token, Roles: []string{"member"}},
		} {
			if result, err := Check(ctx, mgr, req); result != nil || !errors.Is(err, derror.ErrManagerNotFound) {
				t.Fatalf("unavailable manager Check(%+v) = %v, %v", req, result, err)
			}
		}
		if result, err := Check(ctx, mgr, Request{}); err != nil || result == nil {
			t.Fatalf("unprotected Check() = %v, %v", result, err)
		}
	}
}

// TestCheckAccessLogic verifies invalid logic cannot silently become OR. TestCheckAccessLogic 验证非法逻辑不会静默退化为 OR。
func TestCheckAccessLogic(t *testing.T) {
	ctx := context.Background()
	mgr := newAuthcheckTestManager(t)
	token, err := mgr.Login(ctx, "user")
	if err != nil {
		t.Fatal(err)
	}
	if err := mgr.AddPermissions(ctx, "user", []string{"read"}); err != nil {
		t.Fatal(err)
	}
	if err := mgr.AddRoles(ctx, "user", []string{"member"}); err != nil {
		t.Fatal(err)
	}
	for _, checkLogin := range []bool{false, true} {
		for _, roles := range []bool{false, true} {
			for _, logic := range []LogicType{"", LogicOr, LogicAnd, "and", "XOR"} {
				req := Request{TokenValue: token, CheckLogin: checkLogin, LogicType: logic}
				denied := derror.ErrPermissionDenied
				if roles {
					req.Roles = []string{"member", "missing"}
					denied = derror.ErrRoleDenied
				} else {
					req.Permissions = []string{"read", "missing"}
				}
				var want error
				switch logic {
				case "", LogicOr:
				case LogicAnd:
					want = denied
				default:
					want = derror.ErrInvalidParam
				}
				result, err := Check(ctx, mgr, req)
				if !errors.Is(err, want) || (result == nil) != (want != nil) {
					t.Fatalf("Check(%+v) = %v, %v, want error %v", req, result, err, want)
				}
			}
		}
	}
}

// TestErrorMappingNilAndWrapped verifies nil safety without losing wrapped error mappings. TestErrorMappingNilAndWrapped 验证空错误安全处理且保留包装错误映射。
func TestErrorMappingNilAndWrapped(t *testing.T) {
	var typedNil *derror.DTokenError
	for _, tc := range []struct {
		name    string
		err     error
		code    int
		message string
	}{
		{"nil", nil, derror.CodeSuccess, "success"},
		{"typed nil", typedNil, derror.CodeServerError, "<nil>"},
		{"wrapped typed nil", fmt.Errorf("wrapped: %w", typedNil), derror.CodeServerError, "wrapped: <nil>"},
		{"wrapped custom", fmt.Errorf("wrapped: %w", derror.NewDTokenError(499, "custom", derror.ErrInvalidParam)), 499, "custom"},
		{"wrapped sentinel", fmt.Errorf("wrapped: %w", derror.ErrPermissionDenied), derror.CodePermissionDenied, "wrapped: " + derror.ErrPermissionDenied.Error()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if code, message := GetErrorCodeAndMessage(tc.err); code != tc.code || message != tc.message {
				t.Fatalf("mapping = %d, %q, want %d, %q", code, message, tc.code, tc.message)
			}
		})
	}
}

// TestCheckDisablePropagatesReadFailure verifies the additional disable check cannot hide a storage failure. TestCheckDisablePropagatesReadFailure 验证追加的封禁检查不会吞掉存储错误。
func TestCheckDisablePropagatesReadFailure(t *testing.T) {
	base := newAuthcheckTestManager(t)
	cfg := base.GetConfig()
	storage := &disableReadFailureStorage{
		authcheckTestStorage: newAuthcheckTestStorage(),
		disableKey:           cfg.KeyPrefix + cfg.AuthType + manager.DisableKeyPrefix + "user",
	}
	mgr := manager.NewManager(cfg, &authcheckTestGenerator{}, storage, authcheckTestCodec{}, adapter.NewNopLogger(), nil, nil)
	t.Cleanup(mgr.CloseManager)
	ctx := context.Background()
	token, err := mgr.Login(ctx, "user")
	if err != nil {
		t.Fatal(err)
	}
	storage.armed = true
	result, err := Check(ctx, mgr, Request{TokenValue: token, CheckDisable: true})
	if result != nil || !errors.Is(err, derror.ErrStorageUnavailable) {
		t.Fatalf("Check(disable read failure) = %v, %v", result, err)
	}
	if storage.disableReads != 2 {
		t.Fatalf("disable reads = %d, want 2", storage.disableReads)
	}
}

type disableReadFailureStorage struct {
	*authcheckTestStorage
	disableKey   string
	armed        bool
	disableReads int
}

// Get fails after login validation has read the disable marker once. Get 在登录校验读取一次封禁标记后模拟故障。
func (s *disableReadFailureStorage) Get(ctx context.Context, key string) (any, error) {
	if s.armed && key == s.disableKey {
		s.disableReads++
		if s.disableReads > 1 {
			return nil, errors.New("disable storage unavailable")
		}
	}
	return s.authcheckTestStorage.Get(ctx, key)
}

// Exists models an unavailable boolean lookup as false. Exists 模拟布尔查询在存储故障时返回 false。
func (s *disableReadFailureStorage) Exists(ctx context.Context, key string) bool {
	if s.armed && key == s.disableKey {
		return false
	}
	return s.authcheckTestStorage.Exists(ctx, key)
}
