package chi

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
	chiRouter "github.com/go-chi/chi/v5"
)

// TestChiRegisterContextMiddlewareStopsAfterManagerFailure verifies manager resolution failures stop the HTTP chain. TestChiRegisterContextMiddlewareStopsAfterManagerFailure 验证 Manager 解析失败时 HTTP 链路会停止。
func TestChiRegisterContextMiddlewareStopsAfterManagerFailure(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)

	router := chiRouter.NewRouter()
	downstreamCalled := false
	router.Use(RegisterDTokenContextMiddleware(WithAuthType("missing-chi-manager")))
	router.Get("/protected", func(w http.ResponseWriter, _ *http.Request) {
		downstreamCalled = true
		w.WriteHeader(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/protected", nil))
	if downstreamCalled {
		t.Fatal("downstream handler ran after manager resolution failure")
	}
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
}

// TestChiMiddlewareSuccessAndFailure verifies the authenticated HTTP chain. TestChiMiddlewareSuccessAndFailure 验证认证 HTTP 链路的成功与失败。
func TestChiMiddlewareSuccessAndFailure(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)

	ctx := context.Background()
	mgr, err := dtoken.NewBuilder().IsPrintBanner(false).AutoRenew(false).Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	token, err := mgr.Login(ctx, "chi-middleware-user")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	router := chiRouter.NewRouter()
	downstreamCalled := false
	router.Use(RegisterDTokenContextMiddleware(WithManager(mgr)))
	router.Use(AuthMiddleware(WithManager(mgr)))
	router.Get("/protected", func(w http.ResponseWriter, r *http.Request) {
		downstreamCalled = true
		if _, ok := GetDTokenContextByCtx(r.Context()); !ok {
			t.Error("GetDTokenContextByCtx() = false in downstream handler")
		}
		w.WriteHeader(http.StatusNoContent)
	})

	successReq := httptest.NewRequest(http.MethodGet, "/protected", nil)
	successReq.Header.Set(mgr.GetConfig().TokenName, token)
	successRecorder := httptest.NewRecorder()
	router.ServeHTTP(successRecorder, successReq)
	if successRecorder.Code != http.StatusNoContent || !downstreamCalled {
		t.Fatalf("success status=%d downstream=%v", successRecorder.Code, downstreamCalled)
	}

	downstreamCalled = false
	failureRecorder := httptest.NewRecorder()
	router.ServeHTTP(failureRecorder, httptest.NewRequest(http.MethodGet, "/protected", nil))
	if failureRecorder.Code != http.StatusUnauthorized || downstreamCalled {
		t.Fatalf("failure status=%d downstream=%v", failureRecorder.Code, downstreamCalled)
	}
}

// TestChiPermissionPathMiddleware verifies path permissions are appended and enforced. TestChiPermissionPathMiddleware 验证路径权限会被追加并执行校验。
func TestChiPermissionPathMiddleware(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)

	ctx := context.Background()
	mgr, err := dtoken.NewBuilder().IsPrintBanner(false).AutoRenew(false).Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	token, err := mgr.Login(ctx, "chi-path-user")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if err = mgr.AddPermissions(ctx, "chi-path-user", []string{"/path-protected"}); err != nil {
		t.Fatalf("AddPermissions() error = %v", err)
	}

	router := chiRouter.NewRouter()
	router.Use(PermissionPathMiddleware(nil, WithManager(mgr)))
	router.Get("/path-protected", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/path-protected", nil)
	req.Header.Set(mgr.GetConfig().TokenName, token)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", recorder.Code)
	}
}

// TestChiAnnotationHandlers verifies annotation helpers and missing context behavior. TestChiAnnotationHandlers 验证注解处理器与缺失上下文行为。
func TestChiAnnotationHandlers(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)

	ctx := context.Background()
	mgr, err := dtoken.NewBuilder().IsPrintBanner(false).AutoRenew(false).Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	dtoken.SetManager(mgr)
	token, err := mgr.Login(ctx, "chi-annotation-user")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	called := false
	successReq := httptest.NewRequest(http.MethodGet, "/annotated", nil)
	successReq.Header.Set(mgr.GetConfig().TokenName, token)
	GetHandler(func(http.ResponseWriter, *http.Request) { called = true }, &Annotation{CheckLogin: true})(httptest.NewRecorder(), successReq)
	if !called {
		t.Fatal("GetHandler() did not run handler for valid token")
	}

	failureRecorder := httptest.NewRecorder()
	CheckLoginHandler()(failureRecorder, httptest.NewRequest(http.MethodGet, "/annotated", nil))
	if failureRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("annotation failure status = %d, want 401", failureRecorder.Code)
	}

	if _, ok := GetDTokenContext(nil); ok {
		t.Fatal("GetDTokenContext(nil) = true, want false")
	}
	if _, ok := GetDTokenContextByCtx(nil); ok {
		t.Fatal("GetDTokenContextByCtx(nil) = true, want false")
	}
	if _, err = GetTokenValueByCtx(nil); !errors.Is(err, derror.ErrNotLogin) {
		t.Fatalf("GetTokenValueByCtx(nil) error = %v, want ErrNotLogin", err)
	}
}

// TestChiTypedNilContextIsSafe verifies typed nil cache entries are ignored. TestChiTypedNilContextIsSafe 验证类型正确但为空的缓存会被忽略。
func TestChiTypedNilContextIsSafe(t *testing.T) {
	var cached *corecontext.DTokenContext
	requestContext := context.WithValue(context.Background(), DTokenCtxKey, cached)
	if value, ok := GetDTokenContextByCtx(requestContext); value != nil || ok {
		t.Fatalf("GetDTokenContextByCtx(typed nil) = %v, %v, want nil,false", value, ok)
	}

	request := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(requestContext)
	chiContext := NewChiContext(httptest.NewRecorder(), request).(*ChiContext)
	if value := getDTokenContext(chiContext, &manager.Manager{}); value == nil {
		t.Fatal("getDTokenContext() returned nil after typed nil cache entry")
	}
}

// TestChiContextFacadeUsesCachedManager verifies request-context facade operations use the cached manager. TestChiContextFacadeUsesCachedManager 验证请求上下文门面会复用缓存的 Manager。
func TestChiContextFacadeUsesCachedManager(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)

	ctx := context.Background()
	mgr, err := dtoken.NewBuilder().IsPrintBanner(false).AutoRenew(false).Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	token, err := mgr.Login(ctx, "chi-facade-user")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set(mgr.GetConfig().TokenName, token)
	chiContext := NewChiContext(httptest.NewRecorder(), request).(*ChiContext)
	getDTokenContext(chiContext, mgr)
	requestContext := chiContext.r.Context()

	if got, err := GetTokenValueByCtx(requestContext); err != nil || got != token {
		t.Fatalf("GetTokenValueByCtx() = %q, %v, want token", got, err)
	}
	if got, err := GetLoginIDByCtx(requestContext); err != nil || got != "chi-facade-user" {
		t.Fatalf("GetLoginIDByCtx() = %q, %v, want chi-facade-user", got, err)
	}
	if err = AddPermissionsByCtx(requestContext, []string{"article:read"}); err != nil {
		t.Fatalf("AddPermissionsByCtx() error = %v", err)
	}
	if !HasPermissionByCtx(requestContext, "article:read") {
		t.Fatal("HasPermissionByCtx() = false, want true")
	}
	if err = RemovePermissionsByCtx(requestContext, []string{"article:read"}); err != nil {
		t.Fatalf("RemovePermissionsByCtx() error = %v", err)
	}
	if HasPermissionByCtx(requestContext, "article:read") {
		t.Fatal("HasPermissionByCtx() = true after removal, want false")
	}
}
