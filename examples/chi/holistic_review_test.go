package main

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	chidt "github.com/Zany2/dtoken-go/integrations/chi"
	"github.com/go-chi/chi/v5"
)

// TestProductionRouteFailures covers the actual registration, authentication and access failure callbacks. TestProductionRouteFailures 覆盖实际注册、认证和权限过滤器的失败回调。
func TestProductionRouteFailures(t *testing.T) {
	setupChiManager(t)
	ctx := context.Background()
	router := chi.NewRouter()
	registerRoutes(router)
	check := func(method, path, body, token string, wantStatus, wantCode int) {
		t.Helper()
		rec := requestChi(t, router, method, path, body, token)
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
		check(method, path, "", "", http.StatusUnauthorized, chidt.CodeNotLogin)
		check(method, path, "", "invalid-token", http.StatusUnauthorized, chidt.CodeNotLogin)
	}
	token, err := chidt.Login(ctx, "reader", "web")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/admin", "/articles"} {
		check(http.MethodGet, path, "", token, http.StatusForbidden, chidt.CodePermissionDenied)
	}
	if err := chidt.DisableDevice(ctx, "reader", "web", time.Hour); err != nil {
		t.Fatal(err)
	}
	check(http.MethodGet, "/me", "", token, http.StatusForbidden, chidt.CodeAccountDisabled)

	chidt.DeleteAllManager()
	check(http.MethodPost, "/login", `{"username":"alice","password":"123456"}`, "", http.StatusInternalServerError, chidt.CodeServerError)
	check(http.MethodGet, "/admin", "", "", http.StatusInternalServerError, chidt.CodeServerError)

	mgr, err := chidt.NewBuilder().SetStorage(&chiFailingStorage{Storage: chidt.NewMemoryStorage()}).
		AutoRenew(false).AsyncEvent(false).IsPrintBanner(false).Build()
	if err != nil {
		t.Fatal(err)
	}
	chidt.SetManager(mgr)
	for _, path := range []string{"/me", "/admin", "/articles", "/logout"} {
		method := http.MethodGet
		if path == "/logout" {
			method = http.MethodPost
		}
		check(method, path, "", "some-token", http.StatusInternalServerError, chidt.CodeServerError)
	}
}
