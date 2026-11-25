package main

import (
	dtoken "github.com/Zany2/dtoken-go"
	dterror "github.com/Zany2/dtoken-go/errors"
	dt1 "github.com/Zany2/dtoken-go/storage/memory"

	"context"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"testing"
	"time"
)

// TestProductionRouteFailures covers the actual registration, authentication and access failure callbacks. TestProductionRouteFailures 覆盖实际注册、认证和权限过滤器的失败回调。
func TestProductionRouteFailures(t *testing.T) {
	setupGinManager(t)
	ctx := context.Background()
	router := gin.New()
	registerTestRoutes(router)
	check := func(method, path, body, token string, wantStatus, wantCode int) {
		t.Helper()
		rec := requestGin(t, router, method, path, body, token)
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
	token, err := dtoken.Login(ctx, dtoken.LoginOptions{LoginID: "reader", Device: "web"})
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/admin", "/articles"} {
		check(http.MethodGet, path, "", token, http.StatusForbidden, dterror.CodePermissionDenied)
	}
	if err := dtoken.DisableDevice(ctx, dtoken.DeviceDisableOptions{LoginID: "reader", Device: "web", Duration: time.Hour}); err != nil {
		t.Fatal(err)
	}
	check(http.MethodGet, "/me", "", token, http.StatusForbidden, dterror.CodeAccountDisabled)

	dtoken.DeleteAllManager()
	check(http.MethodPost, "/login", `{"username":"alice","password":"123456"}`, "", http.StatusInternalServerError, dterror.CodeServerError)
	check(http.MethodGet, "/admin", "", "", http.StatusInternalServerError, dterror.CodeServerError)
	builtAuth1, err := dtoken.NewBuilder().Storage(&ginFailingStorage{Storage: dt1.NewStorage()}).
		AutoRenew(false).AsyncEvent(false).IsPrintBanner(false).BuildInstance()
	mgr := builtAuth1.Manager()

	if err != nil {
		t.Fatal(err)
	}
	dtoken.SetManager(mgr)
	for _, path := range []string{"/me", "/admin", "/articles", "/logout"} {
		method := http.MethodGet
		if path == "/logout" {
			method = http.MethodPost
		}
		check(method, path, "", "some-token", http.StatusInternalServerError, dterror.CodeServerError)
	}
}

// TestJSONValidationPrecedesTokenMutation rejects trailing values before issuing or rotating credentials. TestJSONValidationPrecedesTokenMutation 在签发或轮换凭证前拒绝多余 JSON。
func TestJSONValidationPrecedesTokenMutation(t *testing.T) {
	setupGinManager(t)
	router := gin.New()
	registerTestRoutes(router)
	valid := `{"username":"alice","password":"123456"}`
	for _, suffix := range []string{" {}", " null", " trailing"} {
		rec := requestGin(t, router, http.MethodPost, "/login", valid+suffix, "")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("invalid login accepted: %d %s", rec.Code, rec.Body.String())
		}
	}
	if _, err := dtoken.GetSession(context.Background(), "alice"); !errors.Is(err, dterror.ErrSessionNotFound) {
		t.Fatalf("invalid input created a session: %v", err)
	}
	if rec := requestGin(t, router, http.MethodPost, "/login", valid+"\n\t", ""); rec.Code != http.StatusOK {
		t.Fatalf("valid login rejected: %d %s", rec.Code, rec.Body.String())
	}
	pair, err := dtoken.LoginWithRefreshTokenOptions(context.Background(), dtoken.RefreshTokenOptions{LoginOptions: dtoken.LoginOptions{LoginID: "reader"}})
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(RefreshRequest{RefreshToken: pair.RefreshToken})
	if err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{" {}", " null", " trailing"} {
		rec := requestGin(t, router, http.MethodPost, "/refresh", string(body)+suffix, "")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("invalid refresh accepted: %d %s", rec.Code, rec.Body.String())
		}
	}
	if err := dtoken.CheckLogin(context.Background(), pair.AccessToken); err != nil {
		t.Fatalf("invalid input revoked access: %v", err)
	}
	rec := requestGin(t, router, http.MethodPost, "/refresh", string(body)+"\n\t", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("invalid input consumed refresh token: %d %s", rec.Code, rec.Body.String())
	}
}
