package kratos

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Zany2/dtoken-go/core/adapter"
	corecontext "github.com/Zany2/dtoken-go/core/context"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/manager"
	"github.com/Zany2/dtoken-go/dtoken"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

// TestKratosContextMiddlewareManagerFailure verifies manager resolution failures return through FailFunc. TestKratosContextMiddlewareManagerFailure 验证 Manager 解析失败会通过 FailFunc 返回。
func TestKratosContextMiddlewareManagerFailure(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)

	var gotErr error
	handler := RegisterDTokenContextMiddleware(
		WithAuthType("missing-kratos-manager"),
		WithFailFunc(func(_ context.Context, err error) error { gotErr = err; return err }),
	)(func(_ context.Context, _ any) (any, error) {
		t.Fatal("downstream handler ran after manager resolution failure")
		return nil, nil
	})
	if _, err := handler(context.Background(), nil); !errors.Is(err, derror.ErrManagerNotFound) {
		t.Fatalf("middleware error = %v, want ErrManagerNotFound", err)
	}
	if !errors.Is(gotErr, derror.ErrManagerNotFound) {
		t.Fatalf("failure callback error = %v, want ErrManagerNotFound", gotErr)
	}
}

// TestKratosAuthMiddlewareSuccessAndFailure verifies authenticated and rejected handler chains. TestKratosAuthMiddlewareSuccessAndFailure 验证认证中间件的成功与失败链路。
func TestKratosAuthMiddlewareSuccessAndFailure(t *testing.T) {
	ctx := context.Background()
	mgr := newKratosBehaviorManager(t)
	token, err := mgr.Login(ctx, "kratos-auth-user")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	base := newKratosBehaviorContext(mgr, token, "/protected")
	called := false
	handler := AuthMiddleware(WithManager(mgr))(func(ctx context.Context, _ any) (any, error) {
		called = true
		if _, ok := GetDTokenContext(ctx); !ok {
			t.Error("GetDTokenContext() = false in downstream handler")
		}
		return "ok", nil
	})
	result, err := handler(base, nil)
	if err != nil || result != "ok" || !called {
		t.Fatalf("success result=%v err=%v called=%v", result, err, called)
	}

	failureCtx := newKratosBehaviorContext(mgr, "", "/protected")
	called = false
	var failureErr error
	failureHandler := AuthMiddleware(
		WithManager(mgr),
		WithFailFunc(func(_ context.Context, err error) error { failureErr = err; return err }),
	)(func(_ context.Context, _ any) (any, error) {
		called = true
		return "unexpected", nil
	})
	result, err = failureHandler(failureCtx, nil)
	if !errors.Is(err, derror.ErrTokenExpired) || !errors.Is(failureErr, derror.ErrTokenExpired) {
		t.Fatalf("failure result=%v err=%v callback=%v, want ErrTokenExpired", result, err, failureErr)
	}
	if called {
		t.Fatal("downstream handler ran for an unauthenticated request")
	}
}

// TestKratosPermissionPathMiddleware verifies the request path participates in permission checks. TestKratosPermissionPathMiddleware 验证请求路径会参与权限校验。
func TestKratosPermissionPathMiddleware(t *testing.T) {
	ctx := context.Background()
	mgr := newKratosBehaviorManager(t)
	token, err := mgr.Login(ctx, "kratos-path-user")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if err = mgr.AddPermissions(ctx, "kratos-path-user", []string{"/kratos-protected"}); err != nil {
		t.Fatalf("AddPermissions() error = %v", err)
	}

	server := khttp.NewServer()
	var called bool
	var failureErr error
	requestCount := 0
	handle := func(_ http.ResponseWriter, req *http.Request) {
		requestCount++
		called = false
		failureErr = nil
		handler := PermissionPathMiddleware(nil,
			WithManager(mgr),
			WithFailFunc(func(_ context.Context, err error) error { failureErr = err; return err }),
		)(func(_ context.Context, _ any) (any, error) {
			called = true
			return "ok", nil
		})
		result, err := handler(req.Context(), nil)
		if req.URL.Path == "/kratos-protected" {
			if err != nil || result != "ok" || !called {
				t.Errorf("success result=%v err=%v called=%v", result, err, called)
			}
			return
		}
		if !errors.Is(err, derror.ErrPermissionDenied) || !errors.Is(failureErr, derror.ErrPermissionDenied) {
			t.Errorf("failure result=%v err=%v callback=%v, want ErrPermissionDenied", result, err, failureErr)
		}
		if called {
			t.Error("downstream handler ran after a path permission failure")
		}
	}
	server.HandleFunc("/kratos-protected", handle)
	server.HandleFunc("/other", handle)

	successReq := httptest.NewRequest(http.MethodGet, "http://example.com/kratos-protected", nil)
	successReq.Header.Set(mgr.GetConfig().TokenName, token)
	server.ServeHTTP(httptest.NewRecorder(), successReq)
	failureReq := httptest.NewRequest(http.MethodGet, "http://example.com/other", nil)
	failureReq.Header.Set(mgr.GetConfig().TokenName, token)
	server.ServeHTTP(httptest.NewRecorder(), failureReq)
	if requestCount != 2 {
		t.Fatalf("handled requests = %d, want 2", requestCount)
	}
}

// TestKratosAnnotationHandlerControlFlow verifies nil, success, ignore, and failure annotation paths. TestKratosAnnotationHandlerControlFlow 验证 nil、成功、忽略和失败注解路径。
func TestKratosAnnotationHandlerControlFlow(t *testing.T) {
	ctx := context.Background()
	mgr := newKratosBehaviorManager(t)
	token, err := mgr.Login(ctx, "kratos-annotation-user")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if err = mgr.AddPermissions(ctx, "kratos-annotation-user", []string{"report:read"}); err != nil {
		t.Fatalf("AddPermissions() error = %v", err)
	}

	called := false
	nilAnnotation := GetHandler(nil, (*Annotation)(nil))(func(_ context.Context, _ any) (any, error) {
		called = true
		return "ok", nil
	})
	if result, err := nilAnnotation(context.Background(), nil); err != nil || result != "ok" || !called {
		t.Fatalf("nil annotation result=%v err=%v called=%v", result, err, called)
	}

	called = false
	successHandler := GetHandler(nil, &Annotation{CheckPermission: []string{"report:read"}})(func(_ context.Context, _ any) (any, error) {
		called = true
		return "ok", nil
	})
	result, err := successHandler(newKratosBehaviorContext(mgr, token, "/reports"), nil)
	if err != nil || result != "ok" || !called {
		t.Fatalf("annotation success result=%v err=%v called=%v", result, err, called)
	}

	called = false
	accessHandler := AccessMiddleware(
		WithManager(mgr),
		WithRouteAccessHandler(func(_ context.Context, _ any, req *RouteAccessRequest) {
			req.RequirePermissions("report:read")
		}),
	)(func(_ context.Context, _ any) (any, error) {
		called = true
		return "access-ok", nil
	})
	result, err = accessHandler(newKratosBehaviorContext(mgr, token, "/reports"), nil)
	if err != nil || result != "access-ok" || !called {
		t.Fatalf("access result=%v err=%v called=%v", result, err, called)
	}

	called = false
	ignoreHandler := IgnoreMiddleware(nil)(func(_ context.Context, _ any) (any, error) {
		called = true
		return "ignored", nil
	})
	result, err = ignoreHandler(context.Background(), nil)
	if err != nil || result != "ignored" || !called {
		t.Fatalf("ignore result=%v err=%v called=%v", result, err, called)
	}

	var failureErr error
	failureHandler := CheckRoleMiddleware([]string{"admin"}, func(_ context.Context, err error) error {
		failureErr = err
		return err
	}, "")(
		func(_ context.Context, _ any) (any, error) {
			t.Fatal("downstream handler ran after role failure")
			return nil, nil
		},
	)
	result, err = failureHandler(newKratosBehaviorContext(mgr, token, "/reports"), nil)
	if !errors.Is(err, derror.ErrRoleDenied) || !errors.Is(failureErr, derror.ErrRoleDenied) {
		t.Fatalf("annotation failure result=%v err=%v callback=%v, want ErrRoleDenied", result, err, failureErr)
	}
}

// TestKratosContextFacadeUsesCachedManager verifies token, login ID, and permission facade operations. TestKratosContextFacadeUsesCachedManager 验证 Token、登录 ID 和权限门面操作。
func TestKratosContextFacadeUsesCachedManager(t *testing.T) {
	ctx := context.Background()
	mgr := newKratosBehaviorManager(t)
	token, err := mgr.Login(ctx, "kratos-facade-user")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	requestContext := newKratosBehaviorContext(mgr, token, "/")
	if got, err := GetTokenValueByCtx(requestContext); err != nil || got != token {
		t.Fatalf("GetTokenValueByCtx() = %q, %v, want token", got, err)
	}
	if got, err := GetLoginIDByCtx(requestContext); err != nil || got != "kratos-facade-user" {
		t.Fatalf("GetLoginIDByCtx() = %q, %v, want kratos-facade-user", got, err)
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

// TestKratosContextLookupHandlesNilAndTypedNil verifies safe lookup and recreation of cached contexts. TestKratosContextLookupHandlesNilAndTypedNil 验证 nil 与 typed-nil 缓存查询及重建安全。
func TestKratosContextLookupHandlesNilAndTypedNil(t *testing.T) {
	if value, ok := GetDTokenContext(nil); value != nil || ok {
		t.Fatalf("GetDTokenContext(nil) = %v, %v, want nil,false", value, ok)
	}
	if _, ok := GetDTokenContextByCtx(nil); ok {
		t.Fatal("GetDTokenContextByCtx(nil) = true, want false")
	}
	if _, err := GetTokenValueByCtx(nil); !errors.Is(err, derror.ErrNotLogin) {
		t.Fatalf("GetTokenValueByCtx(nil) error = %v, want ErrNotLogin", err)
	}

	mgr := newKratosBehaviorManager(t)
	dtoken.SetManager(mgr)
	t.Cleanup(dtoken.DeleteAllManager)
	if _, err := GetLoginIDByCtx(nil); !errors.Is(err, derror.ErrNotLogin) {
		t.Fatalf("GetLoginIDByCtx(nil) error = %v, want ErrNotLogin", err)
	}

	var cached *corecontext.DTokenContext
	typedNil := context.WithValue(context.Background(), DTokenCtxKey, cached)
	if value, ok := GetDTokenContext(typedNil); value != nil || ok {
		t.Fatalf("GetDTokenContext(typed nil) = %v, %v, want nil,false", value, ok)
	}
	if value, _ := getDTokenContext(typedNil, &manager.Manager{}); value == nil {
		t.Fatal("getDTokenContext() returned nil after typed nil cache entry")
	}
}

func newKratosBehaviorManager(t *testing.T) *manager.Manager {
	t.Helper()
	mgr, err := dtoken.NewBuilder().IsPrintBanner(false).AutoRenew(false).Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	t.Cleanup(mgr.CloseManager)
	return mgr
}

func newKratosBehaviorContext(mgr *manager.Manager, token, path string) context.Context {
	req := &kratosBehaviorRequestContext{
		headers: map[string]string{mgr.GetConfig().TokenName: token},
		path:    path,
	}
	dCtx := corecontext.NewContext(req, mgr)
	return context.WithValue(context.Background(), DTokenCtxKey, dCtx)
}

type kratosBehaviorRequestContext struct {
	headers map[string]string
	path    string
}

var _ adapter.RequestContext = (*kratosBehaviorRequestContext)(nil)

func (c *kratosBehaviorRequestContext) Get(key string) (any, bool)  { return nil, false }
func (c *kratosBehaviorRequestContext) GetClientIP() string         { return "" }
func (c *kratosBehaviorRequestContext) GetCookie(string) string     { return "" }
func (c *kratosBehaviorRequestContext) GetHeader(key string) string { return c.headers[key] }
func (c *kratosBehaviorRequestContext) GetMethod() string           { return http.MethodGet }
func (c *kratosBehaviorRequestContext) GetPath() string             { return c.path }
func (c *kratosBehaviorRequestContext) GetQuery(string) string      { return "" }
func (c *kratosBehaviorRequestContext) Set(string, any)             {}
func (c *kratosBehaviorRequestContext) SetCookie(string, string, int, string, string, bool, bool) {
}
func (c *kratosBehaviorRequestContext) SetHeader(string, string) {}
func (c *kratosBehaviorRequestContext) GetHeaders() map[string][]string {
	result := make(map[string][]string, len(c.headers))
	for key, value := range c.headers {
		result[key] = []string{value}
	}
	return result
}
func (c *kratosBehaviorRequestContext) GetQueryAll() map[string][]string { return nil }
func (c *kratosBehaviorRequestContext) GetPostForm(string) string        { return "" }
func (c *kratosBehaviorRequestContext) GetBody() ([]byte, error)         { return nil, nil }
func (c *kratosBehaviorRequestContext) GetURL() string                   { return c.path }
func (c *kratosBehaviorRequestContext) GetUserAgent() string             { return "" }
func (c *kratosBehaviorRequestContext) SetCookieWithOptions(*adapter.CookieOptions) {
}
func (c *kratosBehaviorRequestContext) GetString(string) string { return "" }
func (c *kratosBehaviorRequestContext) MustGet(string) any      { return nil }
func (c *kratosBehaviorRequestContext) Abort()                  {}
func (c *kratosBehaviorRequestContext) IsAborted() bool         { return false }
func (c *kratosBehaviorRequestContext) IsTLS() bool             { return false }
func (c *kratosBehaviorRequestContext) SetStatusCode(int)       {}
func (c *kratosBehaviorRequestContext) Write(data []byte) (int, error) {
	return len(data), nil
}
