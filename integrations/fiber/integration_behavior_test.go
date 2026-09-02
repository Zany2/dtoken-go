package fiber

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Zany2/dtoken-go/dtoken"
	gofiber "github.com/gofiber/fiber/v2"
)

// TestRegisterContextMiddlewareStopsAfterManagerFailure verifies manager resolution failures stop Fiber chains. TestRegisterContextMiddlewareStopsAfterManagerFailure 验证 Manager 解析失败时 Fiber 链路会停止。
func TestRegisterContextMiddlewareStopsAfterManagerFailure(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)

	app := gofiber.New()
	downstreamCalled := false
	app.Use(RegisterDTokenContextMiddleware(context.Background(), WithAuthType("missing-fiber-manager")))
	app.Get("/protected", func(c *gofiber.Ctx) error {
		downstreamCalled = true
		return c.SendStatus(http.StatusNoContent)
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/protected", nil))
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	defer resp.Body.Close()
	if downstreamCalled {
		t.Fatal("downstream handler ran after manager resolution failure")
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

// TestAuthMiddlewareSuccessAndFailure verifies authenticated and rejected requests. TestAuthMiddlewareSuccessAndFailure 验证认证成功与失败请求。
func TestAuthMiddlewareSuccessAndFailure(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)

	ctx := context.Background()
	mgr, err := dtoken.NewBuilder().IsPrintBanner(false).AutoRenew(false).Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	token, err := mgr.Login(ctx, "fiber-middleware-user")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	app := gofiber.New()
	downstreamCalled := false
	app.Use(AuthMiddleware(ctx, WithManager(mgr)))
	app.Get("/protected", func(c *gofiber.Ctx) error {
		downstreamCalled = true
		return c.SendStatus(http.StatusNoContent)
	})

	successReq := httptest.NewRequest(http.MethodGet, "/protected", nil)
	successReq.Header.Set(mgr.GetConfig().TokenName, token)
	successResp, err := app.Test(successReq)
	if err != nil {
		t.Fatalf("success app.Test() error = %v", err)
	}
	successResp.Body.Close()
	if successResp.StatusCode != http.StatusNoContent || !downstreamCalled {
		t.Fatalf("success status=%d downstream=%v", successResp.StatusCode, downstreamCalled)
	}

	downstreamCalled = false
	failureResp, err := app.Test(httptest.NewRequest(http.MethodGet, "/protected", nil))
	if err != nil {
		t.Fatalf("failure app.Test() error = %v", err)
	}
	failureResp.Body.Close()
	if failureResp.StatusCode != http.StatusUnauthorized || downstreamCalled {
		t.Fatalf("failure status=%d downstream=%v", failureResp.StatusCode, downstreamCalled)
	}
}

// TestAnnotationHandlerControlFlow verifies Fiber annotation success and failure paths. TestAnnotationHandlerControlFlow 验证 Fiber 注解处理器成功与失败路径。
func TestAnnotationHandlerControlFlow(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)

	ctx := context.Background()
	mgr, err := dtoken.NewBuilder().IsPrintBanner(false).AutoRenew(false).AuthType("fiber-annotation").Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	dtoken.SetManager(mgr)
	token, err := mgr.Login(ctx, "fiber-annotation-user")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if err = mgr.AddPermissions(ctx, "fiber-annotation-user", []string{"report:read"}); err != nil {
		t.Fatalf("AddPermissions() error = %v", err)
	}

	app := gofiber.New()
	handled := false
	app.Get("/reports", CheckPermissionMiddleware(ctx, []string{"report:read"}, func(c *gofiber.Ctx) error {
		handled = true
		return c.SendStatus(http.StatusNoContent)
	}, nil, "fiber-annotation"))
	successReq := httptest.NewRequest(http.MethodGet, "/reports", nil)
	successReq.Header.Set(mgr.GetConfig().TokenName, token)
	successResp, err := app.Test(successReq)
	if err != nil {
		t.Fatalf("success app.Test() error = %v", err)
	}
	successResp.Body.Close()
	if successResp.StatusCode != http.StatusNoContent || !handled {
		t.Fatalf("success status=%d handled=%v", successResp.StatusCode, handled)
	}

	app = gofiber.New()
	handled = false
	app.Get("/reports", CheckRoleMiddleware(ctx, []string{"admin"}, func(c *gofiber.Ctx) error {
		handled = true
		return c.SendStatus(http.StatusNoContent)
	}, nil, "fiber-annotation"))
	failureReq := httptest.NewRequest(http.MethodGet, "/reports", nil)
	failureReq.Header.Set(mgr.GetConfig().TokenName, token)
	failureResp, err := app.Test(failureReq)
	if err != nil {
		t.Fatalf("failure app.Test() error = %v", err)
	}
	failureResp.Body.Close()
	if failureResp.StatusCode != http.StatusForbidden || handled {
		t.Fatalf("failure status=%d handled=%v", failureResp.StatusCode, handled)
	}
}

// TestGetDTokenContextNilIsSafe verifies nil context lookup does not panic. TestGetDTokenContextNilIsSafe 验证空上下文查询不会 panic。
func TestGetDTokenContextNilIsSafe(t *testing.T) {
	if value, ok := GetDTokenContext(nil); value != nil || ok {
		t.Fatalf("GetDTokenContext(nil) = %v, %v, want nil,false", value, ok)
	}
	if _, err := GetTokenValueByContext(nil); !errors.Is(err, ErrNotLogin) {
		t.Fatalf("GetTokenValueByContext(nil) error = %v, want ErrNotLogin", err)
	}
}
