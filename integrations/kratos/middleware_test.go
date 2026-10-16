// @Author daixk 2026/06/07
package kratos

import (
	"context"
	"errors"
	"testing"

	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/manager"
)

// TestWithManagerOption verifies explicit manager injection is retained by options. TestWithManagerOption 验证选项会保留显式注入的 Manager。
func TestWithManagerOption(t *testing.T) {
	expected := &manager.Manager{}
	options := defaultAuthOptions()
	WithManager(expected)(options)
	if options.Manager != expected {
		t.Fatalf("Manager = %p, want %p", options.Manager, expected)
	}
}

// TestAuthMiddlewareUsesTokenExpiredLoginError verifies auth failure semantics TestAuthMiddlewareUsesTokenExpiredLoginError 验证认证失败语义
func TestAuthMiddlewareUsesTokenExpiredLoginError(t *testing.T) {
	if err := authMiddlewareLoginError(); !errors.Is(err, derror.ErrTokenExpired) {
		t.Fatalf("authMiddlewareLoginError() = %v, want ErrTokenExpired", err)
	}
}

// TestRouteAccessRequestMutations verifies route access rule mutation. TestRouteAccessRequestMutations 验证路由访问规则变更。
func TestRouteAccessRequestMutations(t *testing.T) {
	options := defaultAuthOptions()
	WithAuthType("admin")(options)
	WithLogicType(LogicOr)(options)

	req := newRouteAccessRequest(options)
	if req.AuthType != "admin" || req.LogicType != LogicOr || req.CheckDisable {
		t.Fatalf("newRouteAccessRequest() = %+v", req)
	}

	req.RequirePermissions("article:read")
	req.RequireRoles("admin")
	if req.skipPermission {
		t.Fatal("RequirePermissions/RequireRoles should enable permission checks")
	}
	if len(req.Permissions) != 1 || req.Permissions[0] != "article:read" {
		t.Fatalf("Permissions = %v", req.Permissions)
	}
	if len(req.Roles) != 1 || req.Roles[0] != "admin" {
		t.Fatalf("Roles = %v", req.Roles)
	}

	req.SkipPermission()
	if !req.skipPermission || len(req.Permissions) != 0 || len(req.Roles) != 0 {
		t.Fatalf("SkipPermission() = %+v", req)
	}

	req.RequireRoles("operator")
	if req.skipPermission || len(req.Roles) != 1 || req.Roles[0] != "operator" {
		t.Fatalf("RequireRoles(after skip) = %+v", req)
	}

	req.SkipAuth()
	if !req.skipAuth {
		t.Fatal("SkipAuth() should mark auth as skipped")
	}
}

// TestRouteAccessHandlerOption verifies custom route access handler execution. TestRouteAccessHandlerOption 验证自定义路由访问处理器执行。
func TestRouteAccessHandlerOption(t *testing.T) {
	options := defaultAuthOptions()
	WithRouteAccessHandler(func(_ context.Context, _ any, req *RouteAccessRequest) {
		req.AuthType = "tenant:"
		req.CheckDisable = true
		req.RequirePermissions("report:read")
		req.SetLogicType(LogicOr)
	})(options)

	req := newRouteAccessRequest(options)
	options.RouteAccessHandler(context.Background(), nil, req)
	if req.AuthType != "tenant:" {
		t.Fatalf("AuthType = %q, want tenant:", req.AuthType)
	}
	if !req.CheckDisable {
		t.Fatal("CheckDisable = false, want true")
	}
	if req.LogicType != LogicOr {
		t.Fatalf("LogicType = %v, want %v", req.LogicType, LogicOr)
	}
	if len(req.Permissions) != 1 || req.Permissions[0] != "report:read" {
		t.Fatalf("Permissions = %v", req.Permissions)
	}
}

// TestBeforeAuthHandlerNextAndExit verifies custom before-auth control flow. TestBeforeAuthHandlerNextAndExit 验证认证前置处理流程。
func TestBeforeAuthHandlerNextAndExit(t *testing.T) {
	if runBeforeAuthHandler(context.Background(), nil, defaultAuthOptions(), nil) {
		t.Fatal("runBeforeAuthHandler without handler should return false")
	}

	wantResult := "next-result"
	wantErr := errors.New("next failed")
	options := defaultAuthOptions()
	WithBeforeAuthHandler(func(_ context.Context, _ any, req *AuthHandleRequest) {
		req.Next()
	})(options)
	nextReq := newAuthHandleRequest(options, func() (any, error) {
		return wantResult, wantErr
	})
	if !runBeforeAuthHandler(context.Background(), nil, options, nextReq) {
		t.Fatal("Next() should mark request as handled")
	}
	if nextReq.result != wantResult || !errors.Is(nextReq.err, wantErr) {
		t.Fatalf("Next() result = %v, %v, want %v, %v", nextReq.result, nextReq.err, wantResult, wantErr)
	}

	WithBeforeAuthHandler(func(_ context.Context, _ any, req *AuthHandleRequest) {
		req.Exit()
	})(options)
	exitReq := newAuthHandleRequest(options, nil)
	if !runBeforeAuthHandler(context.Background(), nil, options, exitReq) {
		t.Fatal("Exit() should mark request as handled")
	}
}

// TestDispatchFailUsesCustomFailFunc verifies custom failure dispatch. TestDispatchFailUsesCustomFailFunc 验证自定义失败处理分发。
func TestDispatchFailUsesCustomFailFunc(t *testing.T) {
	wantErr := errors.New("auth failed")
	var gotErr error
	failFunc := func(_ context.Context, err error) error {
		gotErr = err
		return err
	}

	if err := dispatchFail(context.Background(), failFunc, wantErr); !errors.Is(err, wantErr) {
		t.Fatalf("dispatchFail() error = %v, want %v", err, wantErr)
	}
	if !errors.Is(gotErr, wantErr) {
		t.Fatalf("gotErr = %v, want %v", gotErr, wantErr)
	}
}
