package hertz

import (
	"context"
	"errors"
	"net/http"
	"testing"

	corecontext "github.com/Zany2/dtoken-go/core/context"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/manager"
	"github.com/Zany2/dtoken-go/dtoken"
	hertzapp "github.com/cloudwego/hertz/pkg/app"
)

// TestHertzRegisterContextMiddlewareStopsAfterManagerFailure verifies manager resolution failures abort the request. TestHertzRegisterContextMiddlewareStopsAfterManagerFailure 验证 Manager 解析失败时请求会中断。
func TestHertzRegisterContextMiddlewareStopsAfterManagerFailure(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)

	requestContext := hertzapp.NewContext(0)
	var gotErr error
	handler := RegisterDTokenContextMiddleware(
		context.Background(),
		WithAuthType("missing-hertz-manager"),
		WithFailFunc(func(_ context.Context, _ *hertzapp.RequestContext, err error) { gotErr = err }),
	)
	handler(context.Background(), requestContext)
	if gotErr == nil {
		t.Fatal("failure callback was not called")
	}
	if !requestContext.IsAborted() {
		t.Fatal("request context should be aborted after manager resolution failure")
	}
}

// TestHertzAnnotationHandlerControlFlow verifies annotation success and failure callbacks. TestHertzAnnotationHandlerControlFlow 验证注解成功路径与失败回调。
func TestHertzAnnotationHandlerControlFlow(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)

	ctx := context.Background()
	mgr, err := dtoken.NewBuilder().IsPrintBanner(false).AutoRenew(false).Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	token, err := mgr.Login(ctx, "hertz-annotation-user")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if err = mgr.AddPermissions(ctx, "hertz-annotation-user", []string{"report:read"}); err != nil {
		t.Fatalf("AddPermissions() error = %v", err)
	}

	successCtx := hertzapp.NewContext(0)
	successCtx.Request.SetRequestURI("/reports")
	successCtx.Request.Header.Set(mgr.GetConfig().TokenName, token)
	getDTokenContext(successCtx, mgr)
	handled := false
	CheckPermissionMiddleware(ctx, []string{"report:read"}, func(context.Context, *hertzapp.RequestContext) {
		handled = true
	}, nil)(ctx, successCtx)
	if !handled {
		t.Fatal("permission annotation did not run handler")
	}

	failureCtx := hertzapp.NewContext(0)
	failureCtx.Request.SetRequestURI("/reports")
	failureCtx.Request.Header.Set(mgr.GetConfig().TokenName, token)
	getDTokenContext(failureCtx, mgr)
	var gotErr error
	CheckRoleMiddleware(ctx, []string{"admin"}, nil, func(_ context.Context, _ *hertzapp.RequestContext, err error) {
		gotErr = err
	})(ctx, failureCtx)
	if !errors.Is(gotErr, derror.ErrRoleDenied) {
		t.Fatalf("failure error = %v, want ErrRoleDenied", gotErr)
	}
	if !failureCtx.IsAborted() {
		t.Fatal("failure context should be aborted")
	}
}

// TestHertzContextLookupNilIsSafe verifies nil lookup arguments do not panic. TestHertzContextLookupNilIsSafe 验证空查询参数不会 panic。
func TestHertzContextLookupNilIsSafe(t *testing.T) {
	if value, ok := GetDTokenContext(nil); value != nil || ok {
		t.Fatalf("GetDTokenContext(nil) = %v, %v, want nil,false", value, ok)
	}
	if _, err := GetTokenValueByContext(nil); !errors.Is(err, derror.ErrNotLogin) {
		t.Fatalf("GetTokenValueByContext(nil) error = %v, want ErrNotLogin", err)
	}
}

// TestHertzTypedNilContextIsSafe verifies typed nil cache entries are ignored. TestHertzTypedNilContextIsSafe 验证类型正确但为空的缓存会被忽略。
func TestHertzTypedNilContextIsSafe(t *testing.T) {
	var cached *corecontext.DTokenContext
	ctx := hertzapp.NewContext(0)
	ctx.Set(DTokenCtxKey, cached)
	if value, ok := GetDTokenContext(ctx); value != nil || ok {
		t.Fatalf("GetDTokenContext(typed nil) = %v, %v, want nil,false", value, ok)
	}

	if value := getDTokenContext(ctx, &manager.Manager{}); value == nil {
		t.Fatal("getDTokenContext() returned nil after typed nil cache entry")
	}
}

// TestHertzContextFacadeUsesCachedManager verifies request-context facade operations use the cached manager. TestHertzContextFacadeUsesCachedManager 验证请求上下文门面会复用缓存的 Manager。
func TestHertzContextFacadeUsesCachedManager(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)

	ctx := context.Background()
	mgr, err := dtoken.NewBuilder().IsPrintBanner(false).AutoRenew(false).Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	token, err := mgr.Login(ctx, "hertz-facade-user")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	requestContext := hertzapp.NewContext(0)
	requestContext.Request.SetRequestURI("/")
	requestContext.Request.Header.Set(mgr.GetConfig().TokenName, token)
	getDTokenContext(requestContext, mgr)

	if got, err := GetTokenValueByContext(requestContext); err != nil || got != token {
		t.Fatalf("GetTokenValueByContext() = %q, %v, want token", got, err)
	}
	if got, err := GetLoginIDByContext(requestContext); err != nil || got != "hertz-facade-user" {
		t.Fatalf("GetLoginIDByContext() = %q, %v, want hertz-facade-user", got, err)
	}
	if err = AddPermissionsByContext(requestContext, []string{"article:read"}); err != nil {
		t.Fatalf("AddPermissionsByContext() error = %v", err)
	}
	if !HasPermissionByContext(requestContext, "article:read") {
		t.Fatal("HasPermissionByContext() = false, want true")
	}
	if err = RemovePermissionsByContext(requestContext, []string{"article:read"}); err != nil {
		t.Fatalf("RemovePermissionsByContext() error = %v", err)
	}
	if HasPermissionByContext(requestContext, "article:read") {
		t.Fatal("HasPermissionByContext() = true after removal, want false")
	}
}

// TestHertzContextRequestCollections verifies repeated query values and basic request metadata. TestHertzContextRequestCollections 验证重复查询参数与基础请求元数据。
func TestHertzContextRequestCollections(t *testing.T) {
	hertzCtx := hertzapp.NewContext(0)
	hertzCtx.Request.SetMethod(http.MethodGet)
	hertzCtx.Request.SetRequestURI("/demo?tag=one&tag=two")
	hertzCtx.Request.Header.Set("X-Test", "one")
	hertzCtx.Request.Header.Add("X-Test", "two")
	ctx := NewHertzContext(hertzCtx)

	if values := ctx.GetQueryAll()["tag"]; len(values) != 2 || values[0] != "one" || values[1] != "two" {
		t.Fatalf("GetQueryAll()[tag] = %v, want [one two]", values)
	}
	if values := ctx.GetHeaders()["X-Test"]; len(values) != 2 || values[0] != "one" || values[1] != "two" {
		t.Fatalf("GetHeaders()[X-Test] = %v, want [one two]", values)
	}
	if ctx.IsTLS() {
		t.Fatal("IsTLS() = true for HTTP request")
	}
}
