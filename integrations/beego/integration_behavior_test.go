package beego

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	corecontext "github.com/Zany2/dtoken-go/core/context"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/manager"
	"github.com/Zany2/dtoken-go/dtoken"
	beegocontext "github.com/beego/beego/v2/server/web/context"
)

// TestBeegoContextMiddlewareManagerFailure verifies manager resolution failures invoke the failure callback. TestBeegoContextMiddlewareManagerFailure 验证 Manager 解析失败会触发失败回调。
func TestBeegoContextMiddlewareManagerFailure(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)

	c, _ := newBeegoBehaviorContext(http.MethodGet, "/protected", "", "")
	var gotErr error
	RegisterDTokenContextMiddleware(
		context.Background(),
		WithAuthType("missing-beego-manager"),
		WithFailFunc(func(_ *beegocontext.Context, err error) { gotErr = err }),
	)(c)
	if !errors.Is(gotErr, derror.ErrManagerNotFound) {
		t.Fatalf("failure error = %v, want ErrManagerNotFound", gotErr)
	}
	if _, ok := GetDTokenContext(c); ok {
		t.Fatal("context should not be cached after manager resolution failure")
	}
}

// TestBeegoAuthMiddlewareSuccessAndFailure verifies authenticated and rejected requests. TestBeegoAuthMiddlewareSuccessAndFailure 验证认证中间件的成功与失败请求。
func TestBeegoAuthMiddlewareSuccessAndFailure(t *testing.T) {
	ctx := context.Background()
	mgr := newBeegoBehaviorManager(t)
	token, err := mgr.Login(ctx, "beego-auth-user")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	success, _ := newBeegoBehaviorContext(http.MethodGet, "/protected", mgr.GetConfig().TokenName, token)
	var successErr error
	AuthMiddleware(ctx, WithManager(mgr), WithFailFunc(func(_ *beegocontext.Context, err error) {
		successErr = err
	}))(success)
	if successErr != nil {
		t.Fatalf("authenticated request failed: %v", successErr)
	}
	if dCtx, ok := GetDTokenContext(success); !ok || dCtx.GetManager() != mgr {
		t.Fatal("authenticated request did not cache the configured manager")
	}

	failure, _ := newBeegoBehaviorContext(http.MethodGet, "/protected", mgr.GetConfig().TokenName, "")
	var failureErr error
	AuthMiddleware(ctx, WithManager(mgr), WithFailFunc(func(_ *beegocontext.Context, err error) {
		failureErr = err
	}))(failure)
	if !errors.Is(failureErr, derror.ErrTokenExpired) {
		t.Fatalf("failure error = %v, want ErrTokenExpired", failureErr)
	}
}

// TestBeegoPermissionPathMiddleware verifies the request path participates in permission checks. TestBeegoPermissionPathMiddleware 验证请求路径会参与权限校验。
func TestBeegoPermissionPathMiddleware(t *testing.T) {
	ctx := context.Background()
	mgr := newBeegoBehaviorManager(t)
	token, err := mgr.Login(ctx, "beego-path-user")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if err = mgr.AddPermissions(ctx, "beego-path-user", []string{"/beego-protected"}); err != nil {
		t.Fatalf("AddPermissions() error = %v", err)
	}

	success, _ := newBeegoBehaviorContext(http.MethodGet, "/beego-protected", mgr.GetConfig().TokenName, token)
	var successErr error
	PermissionPathMiddleware(ctx, nil, WithManager(mgr), WithFailFunc(func(_ *beegocontext.Context, err error) {
		successErr = err
	}))(success)
	if successErr != nil {
		t.Fatalf("path permission request failed: %v", successErr)
	}

	failure, _ := newBeegoBehaviorContext(http.MethodGet, "/other", mgr.GetConfig().TokenName, token)
	var failureErr error
	PermissionPathMiddleware(ctx, nil, WithManager(mgr), WithFailFunc(func(_ *beegocontext.Context, err error) {
		failureErr = err
	}))(failure)
	if !errors.Is(failureErr, derror.ErrPermissionDenied) {
		t.Fatalf("failure error = %v, want ErrPermissionDenied", failureErr)
	}
}

// TestBeegoAccessMiddlewareAndAnnotations verifies route annotations drive access checks. TestBeegoAccessMiddlewareAndAnnotations 验证路由注解会驱动访问校验。
func TestBeegoAccessMiddlewareAndAnnotations(t *testing.T) {
	ctx := context.Background()
	mgr := newBeegoBehaviorManager(t)
	token, err := mgr.Login(ctx, "beego-annotation-user")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if err = mgr.AddPermissions(ctx, "beego-annotation-user", []string{"report:read"}); err != nil {
		t.Fatalf("AddPermissions() error = %v", err)
	}

	annotationHandler := RouteAccessHandlerFromAnnotations(&Annotation{
		CheckLogin:      true,
		CheckPermission: []string{"report:read"},
		LogicType:       LogicAnd,
	})
	success, _ := newBeegoBehaviorContext(http.MethodGet, "/reports", mgr.GetConfig().TokenName, token)
	var successErr error
	AccessMiddleware(ctx, WithManager(mgr), WithRouteAccessHandler(annotationHandler), WithFailFunc(func(_ *beegocontext.Context, err error) {
		successErr = err
	}))(success)
	if successErr != nil {
		t.Fatalf("annotated request failed: %v", successErr)
	}

	failure, _ := newBeegoBehaviorContext(http.MethodGet, "/reports", mgr.GetConfig().TokenName, "")
	var failureErr error
	AccessMiddleware(ctx, WithManager(mgr), WithRouteAccessHandler(annotationHandler), WithFailFunc(func(_ *beegocontext.Context, err error) {
		failureErr = err
	}))(failure)
	if !errors.Is(failureErr, derror.ErrTokenExpired) {
		t.Fatalf("annotated failure error = %v, want ErrTokenExpired", failureErr)
	}

	ignoreReq := newRouteAccessRequest(defaultAuthOptions())
	RouteAccessHandlerFromAnnotations(&Annotation{Ignore: true})(ctx, success, ignoreReq)
	if !ignoreReq.skipAuth {
		t.Fatal("Ignore annotation should skip authentication")
	}
}

// TestBeegoContextFacadeUsesCachedManager verifies token, login ID, and permission facade operations. TestBeegoContextFacadeUsesCachedManager 验证 Token、登录 ID 和权限门面操作。
func TestBeegoContextFacadeUsesCachedManager(t *testing.T) {
	ctx := context.Background()
	mgr := newBeegoBehaviorManager(t)
	token, err := mgr.Login(ctx, "beego-facade-user")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	c, _ := newBeegoBehaviorContext(http.MethodGet, "/", mgr.GetConfig().TokenName, token)
	RegisterDTokenContextMiddleware(ctx, WithManager(mgr))(c)
	if got, err := GetTokenValueByContext(c); err != nil || got != token {
		t.Fatalf("GetTokenValueByContext() = %q, %v, want token", got, err)
	}
	if got, err := GetLoginIDByContext(c); err != nil || got != "beego-facade-user" {
		t.Fatalf("GetLoginIDByContext() = %q, %v, want beego-facade-user", got, err)
	}
	if got, err := GetManagerByContext(c); err != nil || got != mgr {
		t.Fatalf("GetManagerByContext() = %p, %v, want configured manager", got, err)
	}
	if err = AddPermissionsByContext(c, []string{"article:read"}); err != nil {
		t.Fatalf("AddPermissionsByContext() error = %v", err)
	}
	if !HasPermissionByContext(c, "article:read") {
		t.Fatal("HasPermissionByContext() = false, want true")
	}
	if err = RemovePermissionsByContext(c, []string{"article:read"}); err != nil {
		t.Fatalf("RemovePermissionsByContext() error = %v", err)
	}
	if HasPermissionByContext(c, "article:read") {
		t.Fatal("HasPermissionByContext() = true after removal, want false")
	}
}

// TestBeegoContextLookupHandlesNilAndTypedNil verifies safe context lookup and recreation. TestBeegoContextLookupHandlesNilAndTypedNil 验证 nil 与 typed-nil 上下文查询及重建安全。
func TestBeegoContextLookupHandlesNilAndTypedNil(t *testing.T) {
	if value, ok := GetDTokenContext(nil); value != nil || ok {
		t.Fatalf("GetDTokenContext(nil) = %v, %v, want nil,false", value, ok)
	}

	c, _ := newBeegoBehaviorContext(http.MethodGet, "/", "", "")
	var cached *corecontext.DTokenContext
	c.Input.SetData(DTokenCtxKey, cached)
	if value, ok := GetDTokenContext(c); value != nil || ok {
		t.Fatalf("GetDTokenContext(typed nil) = %v, %v, want nil,false", value, ok)
	}
	if value := getDContext(c, &manager.Manager{}); value == nil {
		t.Fatal("getDContext() returned nil after typed nil cache entry")
	}
}

func newBeegoBehaviorManager(t *testing.T) *manager.Manager {
	t.Helper()
	mgr, err := dtoken.NewBuilder().IsPrintBanner(false).AutoRenew(false).Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	t.Cleanup(mgr.CloseManager)
	return mgr
}

func newBeegoBehaviorContext(method, path, tokenName, token string) (*beegocontext.Context, *httptest.ResponseRecorder) {
	req := httptest.NewRequest(method, path, nil)
	if tokenName != "" {
		req.Header.Set(tokenName, token)
	}
	recorder := httptest.NewRecorder()
	c := beegocontext.NewContext()
	c.Reset(recorder, req)
	return c, recorder
}
