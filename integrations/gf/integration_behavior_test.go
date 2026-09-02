package gf

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	corecontext "github.com/Zany2/dtoken-go/core/context"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/manager"
	"github.com/Zany2/dtoken-go/dtoken"
	"github.com/gogf/gf/v2/net/ghttp"
)

// TestGFRegisterContextMiddlewareUsesFailureCallback verifies manager resolution failures stop before request traversal. TestGFRegisterContextMiddlewareUsesFailureCallback 验证 Manager 解析失败时会触发失败回调并停止请求遍历。
func TestGFRegisterContextMiddlewareUsesFailureCallback(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)

	var gotErr error
	handler := RegisterDTokenContextMiddleware(
		context.Background(),
		WithAuthType("missing-gf-manager"),
		WithFailFunc(func(_ *ghttp.Request, err error) { gotErr = err }),
	)
	handler(nil)
	if gotErr == nil {
		t.Fatal("failure callback was not called")
	}
}

// TestGFContextAdapterRequestValues verifies request data exposed by the GoFrame adapter. TestGFContextAdapterRequestValues 验证 GoFrame 适配器暴露的请求数据。
func TestGFContextAdapterRequestValues(t *testing.T) {
	req := &ghttp.Request{Request: httptest.NewRequest(http.MethodPost, "http://example.com/demo?foo=bar&foo=baz", strings.NewReader("hello"))}
	req.Header.Set("X-Token", "token")
	req.Header.Set("User-Agent", "gf-test")
	req.Cookie = ghttp.GetCookie(req)
	ctx := NewGFContext(req)

	if got := ctx.GetHeader("X-Token"); got != "token" {
		t.Fatalf("GetHeader() = %q, want token", got)
	}
	if got := ctx.GetHeaders()["X-Token"]; len(got) != 1 || got[0] != "token" {
		t.Fatalf("GetHeaders()[X-Token] = %v, want [token]", got)
	}
	if got := ctx.GetQuery("foo"); got != "bar" {
		t.Fatalf("GetQuery() = %q, want bar", got)
	}
	if values := ctx.GetQueryAll()["foo"]; len(values) != 2 || values[0] != "bar" || values[1] != "baz" {
		t.Fatalf("GetQueryAll()[foo] = %v, want [bar baz]", values)
	}
	body, err := ctx.GetBody()
	if err != nil || string(body) != "hello" {
		t.Fatalf("GetBody() = %q, %v, want hello", body, err)
	}
	if got := ctx.GetPostForm("missing"); got != "" {
		t.Fatalf("GetPostForm(missing) = %q, want empty", got)
	}
	if got := ctx.GetMethod(); got != http.MethodPost {
		t.Fatalf("GetMethod() = %q, want POST", got)
	}
	if got := ctx.GetPath(); got != "/demo" {
		t.Fatalf("GetPath() = %q, want /demo", got)
	}
	if got := ctx.GetURL(); got != "http://example.com/demo?foo=bar&foo=baz" {
		t.Fatalf("GetURL() = %q, want full URL", got)
	}
	if got := ctx.GetUserAgent(); got != "gf-test" {
		t.Fatalf("GetUserAgent() = %q, want gf-test", got)
	}
	if ctx.IsTLS() {
		t.Fatal("IsTLS() = true for HTTP request")
	}
}

// TestGFAnnotationHandlerControlFlow verifies annotation success and failure callbacks. TestGFAnnotationHandlerControlFlow 验证注解成功路径与失败回调。
func TestGFAnnotationHandlerControlFlow(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)

	ctx := context.Background()
	mgr, err := dtoken.NewBuilder().IsPrintBanner(false).AutoRenew(false).Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	token, err := mgr.Login(ctx, "gf-annotation-user")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if err = mgr.AddPermissions(ctx, "gf-annotation-user", []string{"report:read"}); err != nil {
		t.Fatalf("AddPermissions() error = %v", err)
	}

	successReq := &ghttp.Request{Request: httptest.NewRequest(http.MethodGet, "/reports", nil)}
	successReq.Header.Set(mgr.GetConfig().TokenName, token)
	getDContext(successReq, mgr)
	handled := false
	CheckPermissionMiddleware(ctx, []string{"report:read"}, func(*ghttp.Request) {
		handled = true
	}, nil)(successReq)
	if !handled {
		t.Fatal("permission annotation did not run handler")
	}

	failureReq := &ghttp.Request{Request: httptest.NewRequest(http.MethodGet, "/reports", nil)}
	failureReq.Header.Set(mgr.GetConfig().TokenName, token)
	getDContext(failureReq, mgr)
	var gotErr error
	CheckRoleMiddleware(ctx, []string{"admin"}, nil, func(_ *ghttp.Request, err error) {
		gotErr = err
	})(failureReq)
	if !errors.Is(gotErr, derror.ErrRoleDenied) {
		t.Fatalf("failure error = %v, want ErrRoleDenied", gotErr)
	}
}

// TestGFContextLookupNilIsSafe verifies nil lookup arguments do not panic. TestGFContextLookupNilIsSafe 验证空查询参数不会 panic。
func TestGFContextLookupNilIsSafe(t *testing.T) {
	if value, ok := GetDTokenContext(nil); value != nil || ok {
		t.Fatalf("GetDTokenContext(nil) = %v, %v, want nil,false", value, ok)
	}
	if value, ok := GetDTokenContextByCtx(nil); value != nil || ok {
		t.Fatalf("GetDTokenContextByCtx(nil) = %v, %v, want nil,false", value, ok)
	}
	if _, err := GetTokenValueByCtx(nil); !errors.Is(err, derror.ErrNotLogin) {
		t.Fatalf("GetTokenValueByCtx(nil) error = %v, want ErrNotLogin", err)
	}
}

// TestGFTypedNilContextIsSafe verifies typed nil cache entries are ignored. TestGFTypedNilContextIsSafe 验证类型正确但为空的缓存会被忽略。
func TestGFTypedNilContextIsSafe(t *testing.T) {
	var cached *corecontext.DTokenContext
	req := &ghttp.Request{Request: httptest.NewRequest(http.MethodGet, "/", nil)}
	req.SetCtxVar(DTokenCtxKey, cached)
	if value, ok := GetDTokenContext(req); value != nil || ok {
		t.Fatalf("GetDTokenContext(typed nil) = %v, %v, want nil,false", value, ok)
	}

	if value := getDContext(req, &manager.Manager{}); value == nil {
		t.Fatal("getDContext() returned nil after typed nil cache entry")
	}
}

// TestGFContextFacadeUsesCachedManager verifies request-context facade operations use the cached manager. TestGFContextFacadeUsesCachedManager 验证请求上下文门面会复用缓存的 Manager。
func TestGFContextFacadeUsesCachedManager(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)

	ctx := context.Background()
	mgr, err := dtoken.NewBuilder().IsPrintBanner(false).AutoRenew(false).Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	token, err := mgr.Login(ctx, "gf-facade-user")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	request := &ghttp.Request{Request: httptest.NewRequest(http.MethodGet, "/", nil)}
	request.Header.Set(mgr.GetConfig().TokenName, token)
	getDContext(request, mgr)
	requestContext := request.Context()

	if got, err := GetTokenValueByCtx(requestContext); err != nil || got != token {
		t.Fatalf("GetTokenValueByCtx() = %q, %v, want token", got, err)
	}
	if got, err := GetLoginIDByCtx(requestContext); err != nil || got != "gf-facade-user" {
		t.Fatalf("GetLoginIDByCtx() = %q, %v, want gf-facade-user", got, err)
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
