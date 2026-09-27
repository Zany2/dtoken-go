package main

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	fiberdt "github.com/Zany2/dtoken-go/integrations/fiber"
	gofiber "github.com/gofiber/fiber/v2"
)

// TestProductionRouteFailures covers the actual registration, authentication and access failure callbacks. TestProductionRouteFailures 覆盖实际注册、认证和权限过滤器的失败回调。
func TestProductionRouteFailures(t *testing.T) {
	setupFiberManager(t)
	ctx := context.Background()
	router := gofiber.New()
	registerRoutes(router)
	check := func(method, path, body, token string, wantStatus, wantCode int) {
		t.Helper()
		rec := requestFiber(t, router, method, path, body, token)
		defer rec.Body.Close()
		var response Response
		if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
			t.Fatal(err)
		}
		status := rec.StatusCode
		if status != wantStatus || response.Code != wantCode || response.Data != nil {
			t.Fatalf("response=%d %+v, want status=%d code=%d", status, response, wantStatus, wantCode)
		}
		if wantStatus == http.StatusInternalServerError && response.Message != "internal server error" {
			t.Fatalf("internal details leaked: %+v", response)
		}
	}
	for _, path := range []string{"/me", "/admin", "/articles", "/logout"} {
		method := http.MethodGet
		if path == "/logout" {
			method = http.MethodPost
		}
		check(method, path, "", "", http.StatusUnauthorized, fiberdt.CodeNotLogin)
		check(method, path, "", "invalid-token", http.StatusUnauthorized, fiberdt.CodeNotLogin)
	}
	token, err := fiberdt.Login(ctx, "reader", "web")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/admin", "/articles"} {
		check(http.MethodGet, path, "", token, http.StatusForbidden, fiberdt.CodePermissionDenied)
	}
	if err := fiberdt.DisableDevice(ctx, "reader", "web", time.Hour); err != nil {
		t.Fatal(err)
	}
	check(http.MethodGet, "/me", "", token, http.StatusForbidden, fiberdt.CodeAccountDisabled)

	fiberdt.DeleteAllManager()
	check(http.MethodPost, "/login", `{"username":"alice","password":"123456"}`, "", http.StatusInternalServerError, fiberdt.CodeServerError)
	check(http.MethodGet, "/admin", "", "", http.StatusInternalServerError, fiberdt.CodeServerError)

	mgr, err := fiberdt.NewBuilder().SetStorage(&fiberFailingStorage{Storage: fiberdt.NewMemoryStorage()}).
		AutoRenew(false).AsyncEvent(false).IsPrintBanner(false).Build()
	if err != nil {
		t.Fatal(err)
	}
	fiberdt.SetManager(mgr)
	for _, path := range []string{"/me", "/admin", "/articles", "/logout"} {
		method := http.MethodGet
		if path == "/logout" {
			method = http.MethodPost
		}
		check(method, path, "", "some-token", http.StatusInternalServerError, fiberdt.CodeServerError)
	}
}
