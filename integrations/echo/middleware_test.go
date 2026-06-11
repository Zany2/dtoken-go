package echo

import (
	"context"
	"errors"
	"testing"

	echo4 "github.com/labstack/echo/v4"
)

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
	WithRouteAccessHandler(func(_ context.Context, _ echo4.Context, req *RouteAccessRequest) {
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

	wantErr := errors.New("next failed")
	options := defaultAuthOptions()
	WithBeforeAuthHandler(func(_ context.Context, _ echo4.Context, req *AuthHandleRequest) {
		req.Next()
	})(options)
	nextReq := newAuthHandleRequest(options, func() error { return wantErr })
	if !runBeforeAuthHandler(context.Background(), nil, options, nextReq) {
		t.Fatal("Next() should mark request as handled")
	}
	if !errors.Is(nextReq.result, wantErr) {
		t.Fatalf("result = %v, want %v", nextReq.result, wantErr)
	}

	WithBeforeAuthHandler(func(_ context.Context, _ echo4.Context, req *AuthHandleRequest) {
		req.Exit()
	})(options)
	exitReq := newAuthHandleRequest(options, nil)
	if !runBeforeAuthHandler(context.Background(), nil, options, exitReq) {
		t.Fatal("Exit() should mark request as handled")
	}
}

// TestFailFuncOption verifies custom failure callback wiring. TestFailFuncOption 验证自定义失败回调装配。
func TestFailFuncOption(t *testing.T) {
	wantErr := errors.New("auth failed")
	var gotErr error
	options := defaultAuthOptions()
	WithFailFunc(func(_ echo4.Context, err error) error {
		gotErr = err
		return err
	})(options)

	if err := options.FailFunc(nil, wantErr); !errors.Is(err, wantErr) {
		t.Fatalf("FailFunc() error = %v, want %v", err, wantErr)
	}
	if !errors.Is(gotErr, wantErr) {
		t.Fatalf("gotErr = %v, want %v", gotErr, wantErr)
	}
}
