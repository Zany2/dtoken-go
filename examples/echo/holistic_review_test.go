package main

import (
	dtoken "github.com/Zany2/dtoken-go"
	dterror "github.com/Zany2/dtoken-go/errors"
	dt2 "github.com/Zany2/dtoken-go/global"
	dt1 "github.com/Zany2/dtoken-go/storage/memory"

	"context"
	"encoding/json"
	echo4 "github.com/labstack/echo/v4"
	"net/http"
	"testing"
	"time"
)

// TestProductionRouteFailures covers the actual registration, authentication and access failure callbacks. TestProductionRouteFailures 覆盖实际注册、认证和权限过滤器的失败回调。
func TestProductionRouteFailures(t *testing.T) {
	setupEchoManager(t)
	ctx := context.Background()
	router := echo4.New()
	registerRoutes(router)
	check := func(method, path, body, token string, wantStatus, wantCode int) {
		t.Helper()
		rec := requestEcho(t, router, method, path, body, token)
		var response Response
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		status := rec.Code
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
		check(method, path, "", "", http.StatusUnauthorized, dterror.CodeNotLogin)
		check(method, path, "", "invalid-token", http.StatusUnauthorized, dterror.CodeNotLogin)
	}
	token, err := dt2.Login(ctx, "reader", "web")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/admin", "/articles"} {
		check(http.MethodGet, path, "", token, http.StatusForbidden, dterror.CodePermissionDenied)
	}
	if err := dt2.DisableDevice(ctx, "reader", "web", time.Hour); err != nil {
		t.Fatal(err)
	}
	check(http.MethodGet, "/me", "", token, http.StatusForbidden, dterror.CodeAccountDisabled)

	dt2.DeleteAllManager()
	check(http.MethodPost, "/login", `{"username":"alice","password":"123456"}`, "", http.StatusInternalServerError, dterror.CodeServerError)
	check(http.MethodGet, "/admin", "", "", http.StatusInternalServerError, dterror.CodeServerError)
	builtAuth1, err := dtoken.NewBuilder().Storage(&echoFailingStorage{Storage: dt1.NewStorage()}).
		AutoRenew(false).AsyncEvent(false).IsPrintBanner(false).Build()
	mgr := builtAuth1.Manager()

	if err != nil {
		t.Fatal(err)
	}
	dt2.SetManager(mgr)
	for _, path := range []string{"/me", "/admin", "/articles", "/logout"} {
		method := http.MethodGet
		if path == "/logout" {
			method = http.MethodPost
		}
		check(method, path, "", "some-token", http.StatusInternalServerError, dterror.CodeServerError)
	}
}
