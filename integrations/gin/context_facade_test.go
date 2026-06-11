package gin

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/dtoken"
	"github.com/gin-gonic/gin"
)

// TestContextFacadeUsesRegisteredManager verifies Gin context shortcut helpers. TestContextFacadeUsesRegisteredManager 验证 Gin 上下文快捷方法。
func TestContextFacadeUsesRegisteredManager(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)

	ctx := context.Background()
	mgr, err := dtoken.NewBuilder().
		IsPrintBanner(false).
		AutoRenew(false).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	dtoken.SetManager(mgr)

	token, err := mgr.Login(ctx, "gin-user", "web", "browser")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set(mgr.GetConfig().TokenName, token)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = req

	RegisterDTokenContextMiddleware(ctx)(c)
	if recorder.Code != http.StatusOK {
		t.Fatalf("RegisterDTokenContextMiddleware wrote status %d, want default OK", recorder.Code)
	}
	if _, ok := GetDTokenContext(c); !ok {
		t.Fatal("GetDTokenContext() = false, want true")
	}

	if got, err := GetTokenValueByContext(c); err != nil || got != token {
		t.Fatalf("GetTokenValueByContext() = %q, %v, want token", got, err)
	}
	if got, err := GetLoginIDByContext(c); err != nil || got != "gin-user" {
		t.Fatalf("GetLoginIDByContext() = %q, %v, want gin-user", got, err)
	}
	if got, err := GetDeviceByContext(c); err != nil || got != "web" {
		t.Fatalf("GetDeviceByContext() = %q, %v, want web", got, err)
	}
	if got, err := GetDeviceIDByContext(c); err != nil || got != "browser" {
		t.Fatalf("GetDeviceIDByContext() = %q, %v, want browser", got, err)
	}
	if !IsLoginByContext(c) {
		t.Fatal("IsLoginByContext() = false, want true")
	}
	if err = AddPermissionsByContext(c, []string{"article:read"}); err != nil {
		t.Fatalf("AddPermissionsByContext() error = %v", err)
	}
	if !HasPermissionByContext(c, "article:read") {
		t.Fatal("HasPermissionByContext() = false, want true")
	}
	if err = AddRolesByContext(c, []string{"admin"}); err != nil {
		t.Fatalf("AddRolesByContext() error = %v", err)
	}
	if !HasRoleByContext(c, "admin") {
		t.Fatal("HasRoleByContext() = false, want true")
	}
	if ttl, err := GetTokenTTLByContext(c); err != nil || ttl <= 0 {
		t.Fatalf("GetTokenTTLByContext() = %d, %v, want positive", ttl, err)
	}
	if err = RenewTimeoutByContext(c, time.Minute); err != nil {
		t.Fatalf("RenewTimeoutByContext() error = %v", err)
	}
}

// TestContextFacadeMissingManagerAndToken verifies clear errors on missing request state. TestContextFacadeMissingManagerAndToken 验证缺失请求状态时错误清晰。
func TestContextFacadeMissingManagerAndToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/protected", nil)
	if _, err := GetTokenValueByContext(c); !errors.Is(err, derror.ErrManagerNotFound) {
		t.Fatalf("GetTokenValueByContext without manager error = %v, want ErrManagerNotFound", err)
	}

	mgr, err := dtoken.NewBuilder().IsPrintBanner(false).AutoRenew(false).Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	dtoken.SetManager(mgr)
	if _, err = GetTokenValueByContext(c); !errors.Is(err, ErrNotLogin) {
		t.Fatalf("GetTokenValueByContext without token error = %v, want ErrNotLogin", err)
	}
}

// TestAnnotationHandlerControlFlow verifies annotation ignore, success, and failure paths. TestAnnotationHandlerControlFlow 验证注解式处理器的忽略、成功和失败路径。
func TestAnnotationHandlerControlFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)

	ctx := context.Background()
	mgr, err := dtoken.NewBuilder().
		IsPrintBanner(false).
		AutoRenew(false).
		AuthType("gin-annotation").
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	dtoken.SetManager(mgr)

	token, err := mgr.Login(ctx, "ann-user")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if err = mgr.AddPermissions(ctx, "ann-user", []string{"report:read"}); err != nil {
		t.Fatalf("AddPermissions() error = %v", err)
	}

	handled := false
	failCalled := false
	success := newGinTestContext(http.MethodGet, "/reports", mgr.GetConfig().TokenName, token)
	CheckPermissionMiddleware(ctx, []string{"report:read"}, func(c *gin.Context) {
		handled = true
	}, func(c *gin.Context, err error) {
		failCalled = true
	}, "gin-annotation")(success)
	if !handled || failCalled || success.IsAborted() {
		t.Fatalf("success path handled=%v failCalled=%v aborted=%v", handled, failCalled, success.IsAborted())
	}

	ignored := false
	ignoreCtx := newGinTestContext(http.MethodGet, "/public", "", "")
	IgnoreMiddleware(ctx, func(c *gin.Context) {
		ignored = true
	}, nil)(ignoreCtx)
	if !ignored || ignoreCtx.IsAborted() {
		t.Fatalf("ignore path ignored=%v aborted=%v", ignored, ignoreCtx.IsAborted())
	}

	var gotErr error
	failCtx := newGinTestContext(http.MethodGet, "/reports", mgr.GetConfig().TokenName, token)
	CheckRoleMiddleware(ctx, []string{"admin"}, nil, func(c *gin.Context, err error) {
		gotErr = err
	}, "gin-annotation")(failCtx)
	if !errors.Is(gotErr, derror.ErrRoleDenied) {
		t.Fatalf("failure error = %v, want ErrRoleDenied", gotErr)
	}
	if !failCtx.IsAborted() {
		t.Fatal("failure path should abort request")
	}
}

func newGinTestContext(method, path, tokenName, token string) *gin.Context {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest(method, path, nil)
	if tokenName != "" {
		req.Header.Set(tokenName, token)
	}
	c.Request = req
	return c
}
