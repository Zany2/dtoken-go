package echo

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Zany2/dtoken-go/dtoken"
	echo4 "github.com/labstack/echo/v4"
)

// TestRegisterContextMiddlewareStopsAfterManagerFailure verifies manager resolution failures stop Echo chains. TestRegisterContextMiddlewareStopsAfterManagerFailure 验证 Manager 解析失败时 Echo 链路会停止。
func TestRegisterContextMiddlewareStopsAfterManagerFailure(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)

	e := echo4.New()
	downstreamCalled := false
	e.Use(RegisterDTokenContextMiddleware(context.Background(), WithAuthType("missing-echo-manager")))
	e.GET("/protected", func(c echo4.Context) error {
		downstreamCalled = true
		return c.NoContent(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	e.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/protected", nil))
	if downstreamCalled {
		t.Fatal("downstream handler ran after manager resolution failure")
	}
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
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
	token, err := mgr.Login(ctx, "echo-middleware-user")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	e := echo4.New()
	downstreamCalled := false
	e.Use(AuthMiddleware(ctx, WithManager(mgr)))
	e.GET("/protected", func(c echo4.Context) error {
		downstreamCalled = true
		return c.NoContent(http.StatusNoContent)
	})

	successReq := httptest.NewRequest(http.MethodGet, "/protected", nil)
	successReq.Header.Set(mgr.GetConfig().TokenName, token)
	successRecorder := httptest.NewRecorder()
	e.ServeHTTP(successRecorder, successReq)
	if successRecorder.Code != http.StatusNoContent || !downstreamCalled {
		t.Fatalf("success status=%d downstream=%v", successRecorder.Code, downstreamCalled)
	}

	downstreamCalled = false
	failureRecorder := httptest.NewRecorder()
	e.ServeHTTP(failureRecorder, httptest.NewRequest(http.MethodGet, "/protected", nil))
	if failureRecorder.Code != http.StatusUnauthorized || downstreamCalled {
		t.Fatalf("failure status=%d downstream=%v", failureRecorder.Code, downstreamCalled)
	}
}

// TestAnnotationHandlerControlFlow verifies Echo annotation success and failure paths. TestAnnotationHandlerControlFlow 验证 Echo 注解处理器成功与失败路径。
func TestAnnotationHandlerControlFlow(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)

	ctx := context.Background()
	mgr, err := dtoken.NewBuilder().IsPrintBanner(false).AutoRenew(false).AuthType("echo-annotation").Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	dtoken.SetManager(mgr)
	token, err := mgr.Login(ctx, "echo-annotation-user")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if err = mgr.AddPermissions(ctx, "echo-annotation-user", []string{"report:read"}); err != nil {
		t.Fatalf("AddPermissions() error = %v", err)
	}

	engine := echo4.New()
	successReq := httptest.NewRequest(http.MethodGet, "/reports", nil)
	successReq.Header.Set(mgr.GetConfig().TokenName, token)
	successCtx := engine.NewContext(successReq, httptest.NewRecorder())
	handled := false
	err = CheckPermissionMiddleware(ctx, []string{"report:read"}, func(c echo4.Context) error {
		handled = true
		return c.NoContent(http.StatusNoContent)
	}, nil, "echo-annotation")(successCtx)
	if err != nil || !handled || successCtx.Response().Status != http.StatusNoContent {
		t.Fatalf("success err=%v handled=%v status=%d", err, handled, successCtx.Response().Status)
	}

	failureReq := httptest.NewRequest(http.MethodGet, "/reports", nil)
	failureReq.Header.Set(mgr.GetConfig().TokenName, token)
	failureRecorder := httptest.NewRecorder()
	failureCtx := engine.NewContext(failureReq, failureRecorder)
	handled = false
	err = CheckRoleMiddleware(ctx, []string{"admin"}, func(echo4.Context) error {
		handled = true
		return nil
	}, nil, "echo-annotation")(failureCtx)
	if err != nil || handled || failureRecorder.Code != http.StatusForbidden {
		t.Fatalf("failure err=%v handled=%v status=%d", err, handled, failureRecorder.Code)
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
